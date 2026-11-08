//go:build vips

package process

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	pdfCommandTimeout = 30 * time.Second
	pdfPreviewDPI     = "110"
)

// processPDF renders the first page with pdftoppm and treats that PNG as an
// image, producing the same thumbnail variants and placeholders as an uploaded
// image.
func (p *Processor) processPDF(ctx context.Context, assetID uuid.UUID, data []byte) (*processed, error) {
	dir, err := os.MkdirTemp("", "gk-pdf-*")
	if err != nil {
		return newProcessed(), fmt.Errorf("media: create pdf work directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(dir) }()

	original := filepath.Join(dir, "original.pdf")
	if err := os.WriteFile(original, data, 0o600); err != nil {
		return newProcessed(), fmt.Errorf("media: write pdf work file: %w", err)
	}

	pageCount, err := pdfPageCount(ctx, original)
	if err != nil {
		return newProcessed(), err
	}

	preview, err := renderPDFFirstPage(ctx, dir, original)
	if err != nil {
		return newProcessed(), err
	}

	result, err := p.processImage(ctx, assetID, preview)
	if err != nil {
		return result, err
	}
	result.pageCount = int32Ptr(pageCount)
	return result, nil
}

// pdfPageCount reads the page count from pdfinfo.
func pdfPageCount(ctx context.Context, path string) (int, error) {
	commandCtx, cancel := context.WithTimeout(ctx, pdfCommandTimeout)
	defer cancel()

	output, err := exec.CommandContext(commandCtx, "pdfinfo", path).CombinedOutput()
	if err != nil {
		if errors.Is(commandCtx.Err(), context.DeadlineExceeded) {
			return 0, fmt.Errorf("media: pdfinfo timed out after %s", pdfCommandTimeout)
		}
		return 0, fmt.Errorf("media: pdfinfo failed: %w: %s", err, shortCommandOutput(output))
	}

	for _, line := range strings.Split(string(output), "\n") {
		if !strings.HasPrefix(line, "Pages:") {
			continue
		}
		pages, parseErr := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Pages:")))
		if parseErr != nil || pages < 1 {
			return 0, fmt.Errorf("media: pdfinfo returned an invalid page count %q", strings.TrimSpace(line))
		}
		return pages, nil
	}
	return 0, errors.New("media: pdfinfo did not report a page count")
}

// renderPDFFirstPage rasterises page one into preview.png.
func renderPDFFirstPage(ctx context.Context, dir, path string) ([]byte, error) {
	commandCtx, cancel := context.WithTimeout(ctx, pdfCommandTimeout)
	defer cancel()

	prefix := filepath.Join(dir, "preview")
	output, err := exec.CommandContext(commandCtx, "pdftoppm",
		"-png",
		"-r", pdfPreviewDPI,
		"-f", "1",
		"-l", "1",
		"-singlefile",
		path,
		prefix,
	).CombinedOutput()
	if err != nil {
		if errors.Is(commandCtx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("media: pdftoppm timed out after %s", pdfCommandTimeout)
		}
		return nil, fmt.Errorf("media: pdftoppm failed: %w: %s", err, shortCommandOutput(output))
	}

	preview, err := os.ReadFile(prefix + ".png")
	if err != nil {
		return nil, fmt.Errorf("media: read pdf preview: %w", err)
	}
	if len(preview) == 0 {
		return nil, errors.New("media: pdftoppm produced an empty preview")
	}
	return preview, nil
}

// shortCommandOutput keeps failure reasons readable inside media_assets.
func shortCommandOutput(output []byte) string {
	text := strings.Join(strings.Fields(string(output)), " ")
	if len(text) > 200 {
		text = text[:200]
	}
	return strings.ToValidUTF8(text, "")
}
