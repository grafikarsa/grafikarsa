package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/settings"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/storage"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	permissionContentCreate = "content.create"

	uploadRateLimit  = 60
	uploadRateWindow = time.Hour

	headerBytes        = 512
	maxFilenameRunes   = 255
	dailyUploadMB      = 1024
	dailyQuotaTTL      = 48 * time.Hour
	quotaKeyPrefix     = "mq:"
	uploadLimitKeyBase = "upload:"
)

// JobInserter inserts River jobs inside an existing transaction, which keeps
// the media row and its processing job in one commit.
type JobInserter interface {
	InsertTx(ctx context.Context, tx pgx.Tx, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// Service implements upload, lookup, streaming, and deletion of media assets.
type Service struct {
	st      *store.Store
	s3      *storage.Client
	jobs    JobInserter
	limiter *ratelimit.Limiter
	// rdb backs the daily upload quota counter.
	rdb *redis.Client
	now func() time.Time
}

// NewService builds a media Service.
func NewService(st *store.Store, s3 *storage.Client, jobInserter JobInserter, limiter *ratelimit.Limiter, rdb *redis.Client) *Service {
	return &Service{
		st:      st,
		s3:      s3,
		jobs:    jobInserter,
		limiter: limiter,
		rdb:     rdb,
		now:     time.Now,
	}
}

// Upload validates and stores one original file in the private bucket, records
// the asset, and enqueues the processing job in the same transaction.
func (s *Service) Upload(ctx context.Context, p *auth.Principal, purpose Purpose, filename string, r io.Reader) (Asset, error) {
	if p == nil {
		return Asset{}, apperr.Unauthenticated()
	}
	if !validPurpose(purpose) {
		return Asset{}, apperr.Validation(apperr.Field("purpose", "invalid", "Tujuan media tidak dikenal."))
	}
	if !canUpload(p, purpose) {
		return Asset{}, apperr.ContentBlocked()
	}
	if err := s.allowUpload(ctx, p.UserID); err != nil {
		return Asset{}, err
	}

	head, content, err := readHead(r)
	if err != nil {
		return Asset{}, fmt.Errorf("media: read header: %w", err)
	}
	kind, mimeType, ok := detectContent(head)
	if !ok || !purposeAllowsKind(purpose, kind) {
		return Asset{}, apperr.UnsupportedMediaType()
	}

	limit := s.maxUploadBytes(ctx, kind)
	spooled, checksum, size, err := spool(content, limit)
	if err != nil {
		return Asset{}, err
	}
	defer func() {
		_ = spooled.Close()
		_ = os.Remove(spooled.Name())
	}()

	if err := s.reserveQuota(ctx, p.UserID, size); err != nil {
		return Asset{}, err
	}
	quotaReserved := true
	releaseQuota := func() {
		if quotaReserved {
			s.releaseQuota(p.UserID, size)
			quotaReserved = false
		}
	}

	now := s.now().UTC()
	objectID, err := uuid.NewV7()
	if err != nil {
		releaseQuota()
		return Asset{}, fmt.Errorf("media: generate object id: %w", err)
	}
	key := fmt.Sprintf("orig/%s/%s/%s", now.Format("2006"), now.Format("01"), objectID)

	if err := s.s3.PutObject(ctx, s.s3.PrivateBucket, key, spooled, size, mimeType); err != nil {
		releaseQuota()
		return Asset{}, fmt.Errorf("media: store original: %w", err)
	}

	committed := false
	defer func() {
		if committed {
			return
		}
		// The row was never committed: remove the orphan object and give the
		// quota back.
		if err := s.s3.DeleteObject(context.Background(), s.s3.PrivateBucket, key); err != nil {
			// Best effort only; the caller already receives the original error.
			_ = err
		}
		releaseQuota()
	}()

	var row store.MediaAsset
	err = s.st.WithTxFunc(ctx, func(q *store.Queries, tx pgx.Tx) error {
		row, err = q.InsertMediaAsset(ctx, store.InsertMediaAssetParams{
			OwnerID:          &p.UserID,
			Kind:             string(kind),
			Purpose:          string(purpose),
			OriginalKey:      key,
			OriginalFilename: cleanFilename(filename),
			MimeType:         mimeType,
			SizeBytes:        size,
			ChecksumSha256:   checksum,
		})
		if err != nil {
			return fmt.Errorf("media: insert asset: %w", err)
		}
		if _, err := s.jobs.InsertTx(ctx, tx, ProcessArgs{AssetID: row.ID}, nil); err != nil {
			return fmt.Errorf("media: enqueue process job: %w", err)
		}
		return nil
	})
	if err != nil {
		return Asset{}, err
	}
	committed = true

	return toAsset(row)
}

// Get returns one asset owned by the principal. Other owners get NOT_FOUND so
// media ids do not leak.
func (s *Service) Get(ctx context.Context, p *auth.Principal, id uuid.UUID) (Asset, error) {
	if p == nil {
		return Asset{}, apperr.Unauthenticated()
	}
	row, err := s.ownedRow(ctx, p, id)
	if err != nil {
		return Asset{}, err
	}
	return toAsset(row)
}

// OpenVariant streams one stored variant. The original is only served to its
// owner, which is the only visibility rule in this milestone.
func (s *Service) OpenVariant(ctx context.Context, p *auth.Principal, id uuid.UUID, variant string) (io.ReadCloser, storage.ObjectInfo, error) {
	if p == nil {
		return nil, storage.ObjectInfo{}, apperr.Unauthenticated()
	}
	row, err := s.ownedRow(ctx, p, id)
	if err != nil {
		return nil, storage.ObjectInfo{}, err
	}
	asset, err := toAsset(row)
	if err != nil {
		return nil, storage.ObjectInfo{}, err
	}

	key := row.OriginalKey
	if variant != "original" {
		found, ok := asset.Variants[variant]
		if !ok {
			return nil, storage.ObjectInfo{}, apperr.NotFound()
		}
		key = found.Key
	}

	reader, info, err := s.s3.GetObject(ctx, s.s3.PrivateBucket, key)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, storage.ObjectInfo{}, apperr.NotFound()
	}
	if err != nil {
		return nil, storage.ObjectInfo{}, err
	}
	if info.ContentType == "" {
		info.ContentType = row.MimeType
	}
	return reader, info, nil
}

// Delete soft-deletes an asset owned by the principal and removes its objects.
// Referenced assets cannot be deleted.
func (s *Service) Delete(ctx context.Context, p *auth.Principal, id uuid.UUID) error {
	if p == nil {
		return apperr.Unauthenticated()
	}
	row, err := s.ownedRow(ctx, p, id)
	if err != nil {
		return err
	}
	asset, err := toAsset(row)
	if err != nil {
		return err
	}

	referenced, err := s.st.MediaAssetIsReferenced(ctx, id)
	if err != nil {
		return fmt.Errorf("media: check references: %w", err)
	}
	if referenced {
		return apperr.Conflict("Media masih dipakai.")
	}

	affected, err := s.st.SoftDeleteMediaAsset(ctx, store.SoftDeleteMediaAssetParams{
		ID:      id,
		OwnerID: &p.UserID,
	})
	if err != nil {
		return fmt.Errorf("media: soft delete asset: %w", err)
	}
	if affected == 0 {
		return apperr.NotFound()
	}

	keys := []string{row.OriginalKey}
	for _, variant := range asset.Variants {
		keys = append(keys, variant.Key)
	}
	for _, key := range keys {
		if err := s.s3.DeleteObject(ctx, s.s3.PrivateBucket, key); err != nil {
			return fmt.Errorf("media: delete object %s: %w", key, err)
		}
	}
	return nil
}

func (s *Service) ownedRow(ctx context.Context, p *auth.Principal, id uuid.UUID) (store.MediaAsset, error) {
	row, err := s.st.GetOwnedMediaAsset(ctx, store.GetOwnedMediaAssetParams{
		ID:      id,
		OwnerID: &p.UserID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return store.MediaAsset{}, apperr.NotFound()
	}
	if err != nil {
		return store.MediaAsset{}, fmt.Errorf("media: get asset: %w", err)
	}
	return row, nil
}

// canUpload reports whether the principal may upload for purpose. Avatar and
// banner are profile images and stay available while content creation is
// blocked.
func canUpload(p *auth.Principal, purpose Purpose) bool {
	if purpose == PurposeAvatar || purpose == PurposeBanner {
		return true
	}
	return p.Can(permissionContentCreate)
}

func (s *Service) allowUpload(ctx context.Context, userID uuid.UUID) error {
	result, err := s.limiter.Allow(ctx, uploadLimitKeyBase+userID.String(), ratelimit.Rule{
		Limit:  uploadRateLimit,
		Window: uploadRateWindow,
	})
	if err != nil {
		return apperr.Unavailable("rate limiter")
	}
	if !result.Allowed {
		return apperr.RateLimited()
	}
	return nil
}

// maxUploadBytes reads the size cap for kind from settings.
func (s *Service) maxUploadBytes(ctx context.Context, kind Kind) int64 {
	key := "media.max_image_mb"
	fallback := 10
	switch kind {
	case KindPDF:
		key, fallback = "media.max_pdf_mb", 25
	case KindModel3D:
		key, fallback = "media.max_model_mb", 30
	}
	megabytes := settings.Int(ctx, s.st.Queries, key, fallback)
	return int64(megabytes) * 1024 * 1024
}

// reserveQuota adds size to today's per-user counter and rejects uploads over
// the daily limit.
func (s *Service) reserveQuota(ctx context.Context, userID uuid.UUID, size int64) error {
	key := s.quotaKey(userID)
	total, err := s.rdb.IncrBy(ctx, key, size).Result()
	if err != nil {
		return fmt.Errorf("media: reserve quota: %w", err)
	}
	if err := s.rdb.Expire(ctx, key, dailyQuotaTTL).Err(); err != nil {
		_, _ = s.rdb.DecrBy(ctx, key, size).Result()
		return fmt.Errorf("media: expire quota key: %w", err)
	}

	limit := int64(settings.Int(ctx, s.st.Queries, "media.daily_upload_mb", dailyUploadMB)) * 1024 * 1024
	if total > limit {
		_, _ = s.rdb.DecrBy(ctx, key, size).Result()
		return apperr.LimitReached("Kuota unggah harian tercapai.")
	}
	return nil
}

func (s *Service) releaseQuota(userID uuid.UUID, size int64) {
	_, _ = s.rdb.DecrBy(context.Background(), s.quotaKey(userID), size).Result()
}

func (s *Service) quotaKey(userID uuid.UUID) string {
	return quotaKeyPrefix + userID.String() + ":" + s.now().UTC().Format("20060102")
}

// readHead buffers at most headerBytes and returns a reader that replays them.
func readHead(r io.Reader) ([]byte, io.Reader, error) {
	head := make([]byte, headerBytes)
	n, err := io.ReadFull(r, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return nil, nil, err
	}
	head = head[:n]
	return head, io.MultiReader(bytes.NewReader(head), r), nil
}

// spool copies the limited stream into a temporary file while hashing it.
func spool(r io.Reader, limit int64) (*os.File, []byte, int64, error) {
	file, err := os.CreateTemp("", "gk-media-*")
	if err != nil {
		return nil, nil, 0, fmt.Errorf("media: create temp file: %w", err)
	}
	ok := false
	defer func() {
		if !ok {
			_ = file.Close()
			_ = os.Remove(file.Name())
		}
	}()

	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(r, limit+1))
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return nil, nil, 0, apperr.PayloadTooLarge()
		}
		return nil, nil, 0, fmt.Errorf("media: spool upload: %w", err)
	}
	if size > limit {
		return nil, nil, 0, apperr.PayloadTooLarge()
	}
	if size == 0 {
		return nil, nil, 0, apperr.UnsupportedMediaType()
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, nil, 0, fmt.Errorf("media: rewind temp file: %w", err)
	}

	ok = true
	return file, hash.Sum(nil), size, nil
}

// cleanFilename keeps the display name only: no directory components and no
// more than 255 characters.
func cleanFilename(filename string) *string {
	name := path.Base(strings.ReplaceAll(strings.TrimSpace(filename), "\\", "/"))
	if name == "." || name == "/" || name == "" {
		return nil
	}
	if utf8.RuneCountInString(name) > maxFilenameRunes {
		name = string([]rune(name)[:maxFilenameRunes])
	}
	return &name
}

func toAsset(row store.MediaAsset) (Asset, error) {
	variants := map[string]Variant{}
	if len(row.Variants) > 0 {
		if err := json.Unmarshal(row.Variants, &variants); err != nil {
			return Asset{}, fmt.Errorf("media: decode variants: %w", err)
		}
	}

	ownerID := uuid.Nil
	if row.OwnerID != nil {
		ownerID = *row.OwnerID
	}

	return Asset{
		ID:               row.ID,
		OwnerID:          ownerID,
		Kind:             Kind(row.Kind),
		Purpose:          Purpose(row.Purpose),
		MimeType:         row.MimeType,
		SizeBytes:        row.SizeBytes,
		Width:            row.Width,
		Height:           row.Height,
		PageCount:        row.PageCount,
		Status:           row.Status,
		Variants:         variants,
		IsPublic:         row.IsPublic,
		Blurhash:         row.Blurhash,
		DominantColor:    row.DominantColor,
		OriginalFilename: row.OriginalFilename,
		CreatedAt:        row.CreatedAt,
	}, nil
}
