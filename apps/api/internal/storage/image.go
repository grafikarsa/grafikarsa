package storage

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/disintegration/imaging"
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

	// PNG dengan transparansi tetap PNG; selain itu JPEG (lebih kecil).
	mime := "image/jpeg"
	var buf bytes.Buffer
	if hasAlpha(src) {
		mime = "image/png"
		if err := png.Encode(&buf, src); err != nil {
			return nil, err
		}
	} else {
		if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 82}); err != nil {
			return nil, err
		}
	}

	out := &ProcessedImage{Data: buf.Bytes(), Mime: mime, Width: w, Height: h, Variants: map[int][]byte{}}
	for _, vw := range variantWidths {
		if vw <= 0 || vw >= w {
			continue
		}
		resized := imaging.Resize(src, vw, 0, imaging.Lanczos)
		var vb bytes.Buffer
		if mime == "image/png" {
			if err := png.Encode(&vb, resized); err != nil {
				return nil, err
			}
		} else {
			if err := jpeg.Encode(&vb, resized, &jpeg.Options{Quality: 80}); err != nil {
				return nil, err
			}
		}
		out.Variants[vw] = vb.Bytes()
	}
	return out, nil
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
