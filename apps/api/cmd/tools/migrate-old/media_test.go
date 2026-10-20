package main

import (
	"strings"
	"testing"
)

// TestMimeByExt Content-Type hasil migrasi menentukan cara browser
// menampilkan file. Kalau mime-nya application/octet-stream, browser
// MENGUNDUH tiap gambar sebagai ganti menampilkannya — jadi seluruh gambar
// di situs rusak setelah go-live.
func TestMimeByExt(t *testing.T) {
	cases := map[string]string{
		"avatars/x/1.jpg":                  "image/jpeg",
		"avatars/x/1.JPG":                  "image/jpeg",
		"avatars/x/1.jpeg":                 "image/jpeg",
		"portfolio-images/x/1.png":         "image/png",
		"portfolio-images/x/1.PNG":         "image/png",
		"portfolio-images/x/1.webp":        "image/webp",
		"portfolio-images/x/1.gif":         "image/gif",
		"feedback/x/1.pdf":                 "application/pdf",
		"documents/x/1.docx":               "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"documents/x/1.doc":                "application/msword",
		"x/1.mp4":                          "video/mp4",
		"tan-ekstensi":                     "application/octet-stream",
		"avatars/x/1.jpg?query=1":          "image/jpeg",
		"thumbnails/91295b14/24fb814f.jpg": "image/jpeg",
	}
	for key, want := range cases {
		if got := mimeByExt(key); got != want {
			t.Fatalf("mimeByExt(%q) = %q, mau %q", key, got, want)
		}
	}
}

// TestStorageURLRegex URL gambar di dalam payload JSON blok portfolio harus
// ikut terpetakan — kalau tidak, URL lamanya tidak terdaftar dan berakhir
// 410 Gone (gambar di karya yang sudah tayang rusak).
func TestStorageURLRegex(t *testing.T) {
	payload := `{"url": "https://grafikarsa.com/storage/grafikarsa/portfolio-images/921e9374/e038ac63.jpg", "caption": ""}`
	found := storageURLRegex.FindAllString(payload, -1)
	if len(found) != 1 {
		t.Fatalf("harus ketemu 1 URL, ketemu %d", len(found))
	}
	if !strings.Contains(found[0], "portfolio-images/921e9374/e038ac63.jpg") {
		t.Fatal("URL terpotong: " + found[0])
	}

	// URL dengan tanda kutip di Payload JSON tidak ikut terambil.
	payload2 := `{"url":"https://grafikarsa.com/storage/grafikarsa/a/b.jpg","caption":"x"}`
	if n := len(storageURLRegex.FindAllString(payload2, -1)); n != 1 {
		t.Fatalf("harus ketemu 1 URL tanpa spasi, ketemu %d", n)
	}
}

// TestPurposeFromKey Purpose mengategorikan media di galeri. Semua nilai
// harus ada di CHECK constraint media_purpose_valid.
func TestPurposeFromKey(t *testing.T) {
	cases := map[string]string{
		"avatars/uuid/x.jpg":           "avatar",
		"banners/uuid/x.jpg":           "banner",
		"thumbnails/uuid/x.jpg":        "thumbnail",
		"portfolio-images/uuid/x.jpg":  "block_image",
		"feedback-attachments/u/x.pdf": "feedback",
		"documents/u/x.docx":           "document",
		"lain/u/x.bin":                 "document",
	}
	for key, want := range cases {
		if got := purposeFromKey(key); got != want {
			t.Fatalf("purposeFromKey(%q) = %q, mau %q", key, got, want)
		}
	}
}
