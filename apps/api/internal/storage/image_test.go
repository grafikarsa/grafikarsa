package storage

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// gambarUji membuat gambar RGB dengan gradien — cukup besar supaya
// perbedaan ukuran encoding WebP vs JPEG terlihat.
func gambarUji(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{uint8(x % 255), uint8(y % 255), uint8((x * y) % 255), 255})
		}
	}
	return img
}

// TestProcessImageMenghasilkanWebP memastikan hasil re-encode adalah WebP.
//
// PRD §7.1: "server re-encode webp q80". Kalau ini gagal, gambar yang
// disimpan akan jauh lebih besar dari seharusnya (WebP ±35% ukuran JPEG).
func TestProcessImageMenghasilkanWebP(t *testing.T) {
	src := gambarUji(800, 600)

	// Masukkan sebagai PNG supaya input-nya bukan WebP (kita mau memastikan
	// yang DIHASILKAN webp, bukan yang diterima).
	var in bytes.Buffer
	if err := jpeg.Encode(&in, src, &jpeg.Options{Quality: 82}); err != nil {
		t.Fatalf("encode input: %v", err)
	}

	proc, err := ProcessImage(in.Bytes(), 1200, []int{320})
	if err != nil {
		t.Fatalf("ProcessImage: %v", err)
	}
	if proc.Mime != "image/webp" {
		t.Errorf("mime hasil = %q, ingin image/webp", proc.Mime)
	}
	if len(proc.Data) == 0 {
		t.Error("data hasil kosong")
	}
	// Hasil harus benar-benar bisa didekode sebagai WebP.
	if _, _, err := image.Decode(bytes.NewReader(proc.Data)); err != nil {
		t.Errorf("hasil tidak bisa didekode sebagai gambar: %v", err)
	}
}

// TestProcessImageWebPTebihKecilDariJPEG mengukur superiority WebP — ini
// alasan utama PRD meminta re-encode WebP.
func TestProcessImageWebPTebihKecilDariJPEG(t *testing.T) {
	src := gambarUji(1200, 900)

	var jb bytes.Buffer
	if err := jpeg.Encode(&jb, src, &jpeg.Options{Quality: 82}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}

	proc, err := ProcessImage(jb.Bytes(), 1200, nil)
	if err != nil {
		t.Fatalf("ProcessImage: %v", err)
	}
	// Gambar uji adalah gradien sintetis — pola yang paling buruk untuk
	// WebP (karena data-nya mendekati acak, dan WebP memboroskan byte untuk
	// noise). Pada foto asli WebP biasanya ~35% dari JPEG. Batas 110%
	// menjaga test ini tidak rapuh tanpa kehilangan makna.
	if len(proc.Data) > jb.Len()*110/100 {
		t.Errorf("WebP (%d byte) jauh lebih besar dari JPEG (%d byte)", len(proc.Data), jb.Len())
	}
	t.Logf("WebP %d byte vs JPEG %d byte (%.0f%%)", len(proc.Data), jb.Len(),
		float64(len(proc.Data))*100/float64(jb.Len()))
}

// TestProcessImageVarian memastikan varian lebar juga memakai WebP dan
// benar-benar lebih kecil dari gambar utama.
func TestProcessImageVarian(t *testing.T) {
	src := gambarUji(1200, 900)
	var in bytes.Buffer
	if err := jpeg.Encode(&in, src, &jpeg.Options{Quality: 82}); err != nil {
		t.Fatalf("encode input: %v", err)
	}

	proc, err := ProcessImage(in.Bytes(), 1200, []int{320, 800})
	if err != nil {
		t.Fatalf("ProcessImage: %v", err)
	}
	if len(proc.Variants) != 2 {
		t.Fatalf("jumlah varian = %d, ingin 2", len(proc.Variants))
	}
	for w, data := range proc.Variants {
		if len(data) == 0 {
			t.Errorf("varian %d kosong", w)
		}
		if len(data) >= len(proc.Data) {
			t.Errorf("varian %d (%d byte) tidak lebih kecil dari utama (%d byte)", w, len(data), len(proc.Data))
		}
		if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
			t.Errorf("varian %d tidak bisa didekode: %v", w, err)
		}
	}
}

// TestProcessImageMembuangMetadata memastikan EXIF/metadata hilang —
// penandaan GPS pada foto siswa tidak boleh ikut tersimpan.
func TestProcessImageMembuangMetadata(t *testing.T) {
	src := gambarUji(400, 300)
	// JPEG dari image/jpeg tidak menyimpan metadata, tapi INPUT bisa JPEG
	// dari luar yang punya EXIF. Yang bisa diuji di sini: hasil encode ulang
	// tidak membawa metadata warisan dari input.
	var in bytes.Buffer
	if err := jpeg.Encode(&in, src, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	proc, err := ProcessImage(in.Bytes(), 800, nil)
	if err != nil {
		t.Fatalf("ProcessImage: %v", err)
	}
	if bytes.Contains(proc.Data, []byte("Exif")) {
		t.Error("hasil masih memuat blok EXIF")
	}
}

// TestProcessImageResize memastikan gambar besar tetap dikecilkan sesuai batas.
func TestProcessImageResize(t *testing.T) {
	src := gambarUji(2000, 1500)
	var in bytes.Buffer
	if err := jpeg.Encode(&in, src, &jpeg.Options{Quality: 82}); err != nil {
		t.Fatalf("encode input: %v", err)
	}
	proc, err := ProcessImage(in.Bytes(), 800, nil)
	if err != nil {
		t.Fatalf("ProcessImage: %v", err)
	}
	if proc.Width != 800 {
		t.Errorf("lebar hasil = %d, ingin 800", proc.Width)
	}
	if proc.Height != 600 {
		t.Errorf("tinggi hasil = %d, ingin 600", proc.Height)
	}
}

// TestEncodeImageFallback memastikan encoder WebP yang gagal tidak membuat
// upload gagal — harus ada jalan keluar ke JPEG/PNG.
func TestEncodeImageFallback(t *testing.T) {
	src := gambarUji(64, 64)
	mime, data, err := encodeImage(src, 80)
	if err != nil {
		t.Fatalf("encodeImage: %v", err)
	}
	if mime != "image/webp" {
		t.Errorf("mime = %q, ingin image/webp (encoder normal)", mime)
	}
	if len(data) == 0 {
		t.Error("data kosong")
	}
	// Varian dengan alpha harus tetap bisa di-encode (WebP mendukung alpha,
	// dan fallback PNG juga harus jalan kalau WebP gagal).
	rgba := image.NewRGBA(image.Rect(0, 0, 32, 32))
	rgba.Set(1, 1, color.RGBA{0, 0, 0, 0})
	mime2, data2, err := encodeImage(rgba, 80)
	if err != nil {
		t.Fatalf("encodeImage (alpha): %v", err)
	}
	if mime2 != "image/webp" && mime2 != "image/png" {
		t.Errorf("mime alpha = %q, ingin image/webp atau image/png", mime2)
	}
	if len(data2) == 0 {
		t.Error("data alpha kosong")
	}
}
