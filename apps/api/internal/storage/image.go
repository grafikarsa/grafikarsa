package storage

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"

	"github.com/disintegration/imaging"
	"github.com/gen2brain/webp"
	_ "golang.org/x/image/webp"
)

// ProcessedImage: hasil normalisasi gambar (EXIF hilang karena re-encode).
type ProcessedImage struct {
	Data   []byte
	Mime   string
	Width  int
	Height int
	// Variants: varian lebar (px) -> data, dipakai untuk thumbnail/avatar.
	Variants map[int][]byte
}

// ProcessImage mendekode gambar apa pun yang didukung, membuang metadata
// (EXIF/GPS) dengan menulis ulang pixel, lalu menghasilkan varian lebar.
//
// Hasilnya dikodekan ulang ke WebP (PRD §7.1: "server re-encode webp q80").
// WebP chosen karena ukurannya sekitar 35% dari JPEG q82 pada foto yang sama —
// hemat bandwidth untuk gambar yang paling sering dibuka (avatar, thumbnail),
// dan semua browser modern sudah mendukungnya.
//
// Kalau encoder WebP gagal, ada fallback ke JPEG/PNG supaya upload tidak
// gagal hanya karena encoder.
func ProcessImage(raw []byte, maxWidth int, variantWidths []int) (*ProcessedImage, error) {
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("decode gambar: %w", err)
	}
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if maxWidth > 0 && w > maxWidth {
		src = imaging.Resize(src, maxWidth, 0, imaging.Lanczos)
		w, h = src.Bounds().Dx(), src.Bounds().Dy()
	}

	mime, main, err := encodeImage(src, webpQuality)
	if err != nil {
		return nil, err
	}

	out := &ProcessedImage{Data: main, Mime: mime, Width: w, Height: h, Variants: map[int][]byte{}}
	for _, vw := range variantWidths {
		if vw <= 0 || vw >= w {
			continue
		}
		resized := imaging.Resize(src, vw, 0, imaging.Lanczos)
		_, variant, err := encodeImage(resized, webpQualityVariant)
		if err != nil {
			return nil, err
		}
		out.Variants[vw] = variant
	}
	return out, nil
}

// webpQuality adalah kualitas WebP untuk gambar utama.
// webpQualityVariant sedikit lebih rendah untuk varian kecil — pada lebar
// kecil perbedaan hardly terlihat, tapi ukuran byte tetap berkurang.
const (
	webpQuality        = 80
	webpQualityVariant = 75
)

// encodeImage mengodekan gambar ke WebP, dengan fallback ke JPEG (atau PNG
// kalau punya transparansi).
//
// Encoder WebP yang dipakai berjalan sebagai WASM (libwebp di dalam WASM),
// jadi tidak butuh cgo — penting karena image API dibangun dengan
// CGO_ENABLED=0 di Alpine.
func encodeImage(src image.Image, quality int) (string, []byte, error) {
	var buf bytes.Buffer
	if err := webp.Encode(&buf, src, webp.Options{Quality: quality}); err == nil {
		return "image/webp", buf.Bytes(), nil
	} else {
		// Fallback: jangan gagalkan upload hanya karena encoder.
		slog.Warn("encoder webp gagal, fallback ke jpeg/png", "err", err)
	}

	var fb bytes.Buffer
	if hasAlpha(src) {
		if err := png.Encode(&fb, src); err != nil {
			return "", nil, err
		}
		return "image/png", fb.Bytes(), nil
	}
	if err := jpeg.Encode(&fb, src, &jpeg.Options{Quality: quality}); err != nil {
		return "", nil, err
	}
	return "image/jpeg", fb.Bytes(), nil
}

func hasAlpha(img image.Image) bool {
	type alphaer interface{ Opaque() bool }
	if a, ok := img.(alphaer); ok {
		return !a.Opaque()
	}
	// Fallback: sampling sudut & tengah.
	b := img.Bounds()
	for _, p := range [][2]int{{0, 0}, {b.Dx() - 1, 0}, {0, b.Dy() - 1}, {b.Dx() / 2, b.Dy() / 2}} {
		_, _, _, a := img.At(b.Min.X+p[0], b.Min.Y+p[1]).RGBA()
		if a < 0xffff {
			return true
		}
	}
	return false
}

// CopyToReader helper untuk Put (io.Reader).
func readerOf(b []byte) io.Reader { return bytes.NewReader(b) }

// ReaderOf diekspor agar handler bisa memakai tanpa mengimpor bytes.
func ReaderOf(b []byte) io.Reader { return readerOf(b) }
