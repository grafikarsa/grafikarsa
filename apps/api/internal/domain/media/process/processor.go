//go:build vips

// Package process turns uploaded media into web-ready variants. It links
// libvips through cgo, so only the worker binary may import it. The API binary
// must stay CGO_ENABLED=0 and must not depend on this package.
package process

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/media"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/storage"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	// The largest allowed upload is 30 MB; keep a little headroom.
	maxOriginalBytes = 32 << 20
	// Deleting objects after a failed attempt must not depend on the job
	// context, which River may already have cancelled.
	cleanupTimeout = 15 * time.Second
)

// Processor downloads originals from the private bucket and writes the
// processed variants back to it.
type Processor struct {
	st     *store.Store
	s3     *storage.Client
	logger *slog.Logger
}

// NewProcessor builds a Processor.
func NewProcessor(st *store.Store, s3 *storage.Client, logger *slog.Logger) *Processor {
	if logger == nil {
		logger = slog.Default()
	}
	return &Processor{st: st, s3: s3, logger: logger}
}

// Startup initialises libvips once for the worker process. The configuration
// is deliberately conservative because the media queue already limits the
// number of concurrent jobs.
func Startup() error {
	if err := vips.Startup(&vips.Config{
		ConcurrencyLevel: 1,
		MaxCacheMem:      64 * 1024 * 1024,
		MaxCacheFiles:    0,
		MaxCacheSize:     100,
	}); err != nil {
		return fmt.Errorf("media: start libvips: %w", err)
	}
	return nil
}

// Shutdown releases libvips resources. It must only be called after Startup
// succeeded and must be the last libvips call in the process.
func Shutdown() {
	vips.Shutdown()
}

// processed is the accumulated result of one processing attempt.
type processed struct {
	variants      map[string]media.Variant
	keys          []string
	width         *int32
	height        *int32
	pageCount     *int32
	blurhash      *string
	dominantColor *string
}

func newProcessed() *processed {
	return &processed{variants: map[string]media.Variant{}}
}

// Process runs one processing attempt for assetID. A missing or soft-deleted
// asset is a no-op. Variants written during a failed attempt are deleted
// before the error is returned.
func (p *Processor) Process(ctx context.Context, assetID uuid.UUID) error {
	row, err := p.st.GetMediaAssetForProcessing(ctx, assetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("media: load asset %s: %w", assetID, err)
	}

	rows, err := p.st.MarkMediaAssetProcessing(ctx, assetID)
	if err != nil {
		return fmt.Errorf("media: mark asset %s as processing: %w", assetID, err)
	}
	if rows == 0 {
		// Soft-deleted between the lookup and the update.
		return nil
	}

	data, err := p.downloadOriginal(ctx, row)
	if err != nil {
		return err
	}

	result, err := p.processAsset(ctx, row, data)
	if err != nil {
		p.discardVariants(ctx, result)
		return err
	}

	payload, err := json.Marshal(result.variants)
	if err != nil {
		p.discardVariants(ctx, result)
		return fmt.Errorf("media: encode variants for asset %s: %w", assetID, err)
	}

	updated, err := p.st.MarkMediaAssetReady(ctx, store.MarkMediaAssetReadyParams{
		Variants:      payload,
		Width:         result.width,
		Height:        result.height,
		PageCount:     result.pageCount,
		Blurhash:      result.blurhash,
		DominantColor: result.dominantColor,
		ID:            assetID,
	})
	if err != nil {
		p.discardVariants(ctx, result)
		return fmt.Errorf("media: mark asset %s as ready: %w", assetID, err)
	}
	if updated == 0 {
		// Deleted while the variants were being produced; do not leave
		// unreferenced objects behind.
		p.discardVariants(ctx, result)
		return nil
	}

	p.logger.InfoContext(ctx, "media asset processed",
		"asset_id", assetID, "kind", row.Kind, "variants", len(result.variants))
	return nil
}

// Fail marks the asset as permanently failed. It is called by the worker once
// the final attempt has been exhausted.
func (p *Processor) Fail(ctx context.Context, assetID uuid.UUID, reason string) error {
	if _, err := p.st.MarkMediaAssetFailed(ctx, store.MarkMediaAssetFailedParams{
		FailureReason: &reason,
		ID:            assetID,
	}); err != nil {
		return fmt.Errorf("media: mark asset %s as failed: %w", assetID, err)
	}
	return nil
}

// processAsset dispatches on the stored kind. The returned result is always
// non-nil so the caller can clean up any objects written before the error.
func (p *Processor) processAsset(ctx context.Context, row store.MediaAsset, data []byte) (*processed, error) {
	switch media.Kind(row.Kind) {
	case media.KindImage:
		return p.processImage(ctx, row.ID, data)
	case media.KindPDF:
		return p.processPDF(ctx, row.ID, data)
	case media.KindModel3D:
		result := newProcessed()
		if err := validateGLB(data); err != nil {
			return result, fmt.Errorf("media: validate glb asset %s: %w", row.ID, err)
		}
		return result, nil
	case media.KindDocument:
		// Legacy document uploads have no variants.
		return newProcessed(), nil
	default:
		return newProcessed(), fmt.Errorf("media: unsupported asset kind %q", row.Kind)
	}
}

// downloadOriginal reads the private original into memory. Upload limits keep
// originals at or below 30 MB.
func (p *Processor) downloadOriginal(ctx context.Context, row store.MediaAsset) ([]byte, error) {
	reader, info, err := p.s3.GetObject(ctx, p.s3.PrivateBucket, row.OriginalKey)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, fmt.Errorf("media: original object for asset %s is missing", row.ID)
	}
	if err != nil {
		return nil, fmt.Errorf("media: download original for asset %s: %w", row.ID, err)
	}
	defer func() { _ = reader.Close() }()

	if info.Size > maxOriginalBytes {
		return nil, fmt.Errorf("media: original for asset %s is %d bytes, limit is %d", row.ID, info.Size, maxOriginalBytes)
	}

	data, err := io.ReadAll(io.LimitReader(reader, maxOriginalBytes+1))
	if err != nil {
		return nil, fmt.Errorf("media: read original for asset %s: %w", row.ID, err)
	}
	if int64(len(data)) > maxOriginalBytes {
		return nil, fmt.Errorf("media: original for asset %s exceeds %d bytes", row.ID, maxOriginalBytes)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("media: original for asset %s is empty", row.ID)
	}
	return data, nil
}

// discardVariants removes objects written during a failed attempt. Cleanup is
// best effort: the original processing error is always more useful.
func (p *Processor) discardVariants(ctx context.Context, result *processed) {
	if result == nil || len(result.keys) == 0 {
		return
	}

	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()

	for _, key := range result.keys {
		if err := p.s3.DeleteObject(cleanupCtx, p.s3.PrivateBucket, key); err != nil && !errors.Is(err, storage.ErrNotFound) {
			p.logger.WarnContext(cleanupCtx, "failed to delete variant from failed attempt",
				"key", key, "error", err)
		}
	}
}

func int32Ptr(value int) *int32 {
	converted := int32(value)
	return &converted
}
