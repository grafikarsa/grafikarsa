package process

import (
	"bytes"
	"context"
	"fmt"
	"math"

	"github.com/buckket/go-blurhash"
	"github.com/davidbyttow/govips/v2/vips"
	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/media"
)

const (
	webpQuality = 80
	// AVIF encoding is CPU heavy. Effort 4 keeps a 1920px image in the
	// low-seconds range while still producing small files.
	avifQuality = 55
	avifEffort  = 4
	webpEffort  = 4

	blurhashWidth      = 32
	blurhashComponents = 4
	blurhashRows       = 3
)

// responsiveWidths are the srcset widths from the NFR. A source narrower than
// the smallest entry yields one variant at its own width instead.
var responsiveWidths = []int{320, 640, 960, 1280, 1920}

func (p *Processor) processImage(ctx context.Context, assetID uuid.UUID, data []byte) (*processed, error) {
	result := newProcessed()

	params := vips.NewImportParams()
	if isMultiFrameContainer(data) {
		// Animated GIF/WebP: only the first frame is needed.
		params.NumPages.Set(1)
	}

	img, err := vips.LoadImageFromBuffer(data, params)
	if err != nil {
		return result, fmt.Errorf("media: decode image: %w", err)
	}
	defer img.Close()

	if err := img.AutoRotate(); err != nil {
		return result, fmt.Errorf("media: auto-rotate image: %w", err)
	}

	width, height := img.Width(), img.Height()
	if width < 1 || height < 1 {
		return result, fmt.Errorf("media: decoded image has invalid size %dx%d", width, height)
	}
	result.width = int32Ptr(width)
	result.height = int32Ptr(height)

	hash, err := blurhashFor(img)
	if err != nil {
		return result, err
	}
	result.blurhash = &hash

	color, err := dominantColor(img)
	if err != nil {
		return result, err
	}
	result.dominantColor = &color

	for _, targetWidth := range variantWidths(width) {
		if err := p.writeImageVariants(ctx, assetID, result, img, targetWidth); err != nil {
			return result, err
		}
	}

	return result, nil
}

// writeImageVariants resizes img to targetWidth and stores WebP and AVIF
// renditions. Metadata, including EXIF, is stripped from both.
func (p *Processor) writeImageVariants(ctx context.Context, assetID uuid.UUID, result *processed, img *vips.ImageRef, targetWidth int) error {
	scaled, err := img.Copy()
	if err != nil {
		return fmt.Errorf("media: copy image for %dpx variant: %w", targetWidth, err)
	}
	defer scaled.Close()

	if scaled.Width() != targetWidth {
		scale := float64(targetWidth) / float64(img.Width())
		if err := scaled.Resize(scale, vips.KernelLanczos3); err != nil {
			return fmt.Errorf("media: resize image to %dpx: %w", targetWidth, err)
		}
	}

	width, height := scaled.Width(), scaled.Height()
	if width < 1 || height < 1 {
		return fmt.Errorf("media: resized image has invalid size %dx%d", width, height)
	}

	webp, _, err := scaled.ExportWebp(&vips.WebpExportParams{
		Quality:         webpQuality,
		ReductionEffort: webpEffort,
		StripMetadata:   true,
	})
	if err != nil {
		return fmt.Errorf("media: encode %dpx webp: %w", targetWidth, err)
	}
	if err := p.storeVariant(ctx, assetID, result, targetWidth, "webp", "image/webp", width, height, webp); err != nil {
		return err
	}

	avif, _, err := scaled.ExportAvif(&vips.AvifExportParams{
		Quality:       avifQuality,
		Effort:        avifEffort,
		StripMetadata: true,
	})
	if err != nil {
		return fmt.Errorf("media: encode %dpx avif: %w", targetWidth, err)
	}
	return p.storeVariant(ctx, assetID, result, targetWidth, "avif", "image/avif", width, height, avif)
}

func (p *Processor) storeVariant(ctx context.Context, assetID uuid.UUID, result *processed, targetWidth int, format, contentType string, width, height int, data []byte) error {
	name := fmt.Sprintf("w%d.%s", targetWidth, format)
	key := fmt.Sprintf("var/%s/%s", assetID, name)

	if err := p.s3.PutObject(ctx, p.s3.PrivateBucket, key, bytes.NewReader(data), int64(len(data)), contentType); err != nil {
		return fmt.Errorf("media: store variant %s: %w", name, err)
	}

	result.keys = append(result.keys, key)
	result.variants[name] = media.Variant{
		Key:    key,
		Format: format,
		W:      width,
		H:      height,
		Bytes:  int64(len(data)),
	}
	return nil
}

// variantWidths returns the srcset widths that do not exceed width. Images
// narrower than the smallest breakpoint still get exactly one variant.
func variantWidths(width int) []int {
	widths := make([]int, 0, len(responsiveWidths))
	for _, candidate := range responsiveWidths {
		if candidate <= width {
			widths = append(widths, candidate)
		}
	}
	if len(widths) == 0 {
		widths = append(widths, width)
	}
	return widths
}

// blurhashFor builds a blurhash from a 32px-wide thumbnail.
func blurhashFor(img *vips.ImageRef) (string, error) {
	thumb, err := scaledThumbnail(img, blurhashWidth)
	if err != nil {
		return "", fmt.Errorf("media: build blurhash thumbnail: %w", err)
	}
	defer thumb.Close()

	goImage, err := thumb.ToGoImage()
	if err != nil {
		return "", fmt.Errorf("media: convert blurhash thumbnail: %w", err)
	}

	hash, err := blurhash.Encode(blurhashComponents, blurhashRows, goImage)
	if err != nil {
		return "", fmt.Errorf("media: encode blurhash: %w", err)
	}
	return hash, nil
}

// dominantColor averages the image into a single pixel and formats it as a
// lowercase #rrggbb string.
func dominantColor(img *vips.ImageRef) (string, error) {
	pixel, err := scaledThumbnail(img, 1)
	if err != nil {
		return "", fmt.Errorf("media: build dominant color thumbnail: %w", err)
	}
	defer pixel.Close()

	goImage, err := pixel.ToGoImage()
	if err != nil {
		return "", fmt.Errorf("media: convert dominant color thumbnail: %w", err)
	}

	r, g, b, _ := goImage.At(0, 0).RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r>>8, g>>8, b>>8), nil
}

// scaledThumbnail returns a copy of img whose width is targetWidth (or the
// original width when it is already narrower). It never upscales.
func scaledThumbnail(img *vips.ImageRef, targetWidth int) (*vips.ImageRef, error) {
	thumb, err := img.Copy()
	if err != nil {
		return nil, err
	}

	width := min(targetWidth, img.Width())
	height := max(1, int(math.Round(float64(img.Height())*float64(width)/float64(img.Width()))))
	if err := thumb.ThumbnailWithSize(width, height, vips.InterestingNone, vips.SizeDown); err != nil {
		thumb.Close()
		return nil, err
	}
	return thumb, nil
}

// isMultiFrameContainer reports whether data is a GIF or WebP, whose loaders
// support selecting a single page. JPEG/PNG/AVIF do not need the option.
func isMultiFrameContainer(data []byte) bool {
	if len(data) >= 6 {
		header := string(data[:6])
		if header == "GIF87a" || header == "GIF89a" {
			return true
		}
	}
	return len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP"
}
