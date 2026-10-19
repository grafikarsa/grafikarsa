package http

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// sourceOf: baca file sumber di package yang sama dengan file test ini.
func sourceOf(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Skip("runtime.Caller gagal, lewati guard source-level")
	}
	src, err := os.ReadFile(filepath.Join(filepath.Dir(file), name))
	if err != nil {
		t.Skipf("tidak bisa baca sumber %s: %v", name, err)
	}
	return string(src)
}

// Guard: media privat TIDAK BOLEH keluar dengan header cache publik.
//
// Jalur serve by-key (dipakai juga oleh redirect 301 URL lama) dulu selalu
// memakai "public, max-age=31536000, immutable" tanpa memeriksa purpose.
// Akibatnya block image / thumbnail portfolio privat bisa di-cache proxy,
// CDN, atau browser orang lain — kebocoran privasi.
//
// Test ini mengunci politiknya di satu helper supaya tidak bisa lagi
// berbeda antar jalur (by-id, by-key, preview).
func TestCacheControlFor(t *testing.T) {
	t.Run("publik boleh immutable", func(t *testing.T) {
		cc := cacheControlFor(true)
		if !strings.Contains(cc, "public") {
			t.Fatalf("media publik harus public, dapat %q", cc)
		}
		if !strings.Contains(cc, "immutable") {
			t.Fatalf("media publik boleh immutable, dapat %q", cc)
		}
	})

	t.Run("privat wajib private tanpa s-maxage", func(t *testing.T) {
		cc := cacheControlFor(false)
		if !strings.Contains(cc, "private") {
			t.Fatalf("media privat harus private, dapat %q", cc)
		}
		// "s-maxage" membuat shared cache (CDN/proxy) tetap menyimpan
		// respons walau direktorinya private.
		if strings.Contains(cc, "s-maxage") {
			t.Fatalf("media privat tidak boleh memakai s-maxage, dapat %q", cc)
		}
		if strings.Contains(cc, "public") {
			t.Fatalf("media privat tidak boleh mengandung 'public', dapat %q", cc)
		}
	})
}

// Guard: purpose yang boleh di-cache publik harus terbatas. Avatar, banner,
// dan cover changelog bersifat publik; sisanya (thumbnail, block_image,
// document, feedback) privat sampai portfolio-nya published.
func TestIsPublicPurpose(t *testing.T) {
	for _, purpose := range []string{"avatar", "banner", "changelog_cover"} {
		if !isPublicPurpose(purpose) {
			t.Fatalf("purpose %q harus publik", purpose)
		}
	}
	for _, purpose := range []string{"thumbnail", "block_image", "document", "feedback"} {
		if isPublicPurpose(purpose) {
			t.Fatalf("purpose %q tidak boleh treated publik", purpose)
		}
	}
	if isPublicPurpose("") {
		t.Fatal("purpose kosong tidak boleh publik")
	}
	if isPublicPurpose("unknown") {
		t.Fatal("purpose tak dikenal tidak boleh publik (fail-closed)")
	}
}

// Guard: TIDAK BOLEH ada header Cache-Control yang di-hardcode sebagai string
// literal di luar helper.
//
// Bug aslinya: jalur by-key menulis "public, max-age=31536000, immutable"
// langsung di handler, sehingga media privat portfolio bocor ke cache bersama.
// Test helper saja TIDAK cukup — kalau ada handler yang meng-hardcode header
// lagi, helper tetap hijau tapi bug kembali.
//
// Semua header Cache-Control harus lewat:
//   - cacheControlFor(...)          -> policy media (public/private)
//   - cacheControlRedirect(...)     -> policy respons redirect (301)
//   - cacheControlRedirectGone(...) -> policy respons 410
func TestNoHardcodedCacheControlHeader(t *testing.T) {
	body := sourceOf(t, "media_handler.go")
	allowed := []string{"cacheControlFor(", "cacheControlRedirect", "cacheControlRedirectGone"}
	for i, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		// Abaikan baris komentar (komentarnya memang menyebut nama header).
		if strings.HasPrefix(trimmed, "//") {
			continue
		}
		if !strings.Contains(line, "Cache-Control") {
			continue
		}
		okLine := false
		for _, a := range allowed {
			if strings.Contains(line, a) {
				okLine = true
				break
			}
		}
		if !okLine {
			t.Errorf("media_handler.go:%d Cache-Control di-hardcode, pakai helper:\n\t%s",
				i+1, trimmed)
		}
	}
}

// Guard: handler serve media HARUS mendelegasikan ke serveObject supaya
// policy cache (dan dukungan Range) benar-benar single-source.
func TestMediaHandlersDelegateToServeObject(t *testing.T) {
	body := sourceOf(t, "media_handler.go")
	for _, fn := range []string{"handleServeMedia", "handleServeMediaByKey"} {
		start := strings.Index(body, "func (a *App) "+fn+"(")
		if start < 0 {
			t.Fatalf("fungsi %s tidak ditemukan", fn)
		}
		end := strings.Index(body[start:], "\n}\n")
		if end < 0 {
			t.Fatalf("batas fungsi %s tidak ditemukan", fn)
		}
		seg := body[start : start+end]
		if !strings.Contains(seg, "a.serveObject(") {
			t.Errorf("%s harus mendelegasikan ke serveObject (policy cache single-source)", fn)
		}
		if strings.Contains(seg, "http.ServeContent(") {
			t.Errorf("%s tidak boleh memanggil http.ServeContent langsung, pakai serveObject", fn)
		}
	}
}
