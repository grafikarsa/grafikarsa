package http

import (
	"strings"
	"testing"
)

// TestMediaObjectKey memeriksa object key mengikuti PRD §7.1.
//
// Yang dijaga:
//   - Format path: {prefix}/{entitas}/{uuid7}.{ext}
//   - Prefix sesuai purpose (avatar -> avatars, block_image -> blocks, ...)
//   - Purpose entity-scoped memakai entity id; kalau tidak ada, jatuh ke owner
//   - uuid7 dipakai (bukan uuid4): uuid7 terurut waktu, jadi listing objek di
//     satu folder bisa dipaginasi dengan batas atas
func TestMediaObjectKey(t *testing.T) {
	const owner = "11111111-1111-4111-8111-111111111111"
	const portfolio = "22222222-2222-4222-8222-222222222222"

	cases := []struct {
		purpose string
		entity  string
		ext     string
		wantPre string // prefix path yang diharapkan
	}{
		{"avatar", "", ".webp", "avatars/" + owner + "/"},
		{"banner", "", ".webp", "banners/" + owner + "/"},
		{"thumbnail", portfolio, ".webp", "thumbnails/" + portfolio + "/"},
		{"thumbnail", "", ".webp", "thumbnails/" + owner + "/"},
		{"block_image", portfolio, ".webp", "blocks/" + portfolio + "/"},
		{"document", portfolio, ".pdf", "documents/" + portfolio + "/"},
		{"feedback", portfolio, ".bin", "feedback/" + portfolio + "/"},
		{"changelog_cover", "", ".webp", "changelog_covers/" + owner + "/"},
	}

	for _, c := range cases {
		key := mediaObjectKey(c.purpose, owner, c.entity, c.ext)
		if !strings.HasPrefix(key, c.wantPre) {
			t.Errorf("mediaObjectKey(%q, entity=%q) = %q, ingin prefix %q",
				c.purpose, c.entity, key, c.wantPre)
		}
		if !strings.HasSuffix(key, c.ext) {
			t.Errorf("mediaObjectKey(%q) = %q, harus berakhiran %q", c.purpose, key, c.ext)
		}
		// Key harus punya tepat 3 segmen: prefix/entitas/nama-file.
		if n := strings.Count(key, "/"); n != 2 {
			t.Errorf("mediaObjectKey(%q) = %q, harus punya 2 garis miring (3 segmen), dapat %d",
				c.purpose, key, n)
		}
	}
}

// TestMediaObjectKeyUUIDv7 memverifikasi id yang dipakai benar-benar uuid7.
//
// UUIDv7 punya versi 7 di byte version, dan timestamp 48-bit-nya di 12 byte
// awal. Kalau suatu saat ada yang diganti ke uuid.NewString() (v4), test ini
// gagal — karena listing objek per folder kehilangan urutan waktu.
func TestMediaObjectKeyUUIDv7(t *testing.T) {
	key := mediaObjectKey("avatar", "11111111-1111-4111-8111-111111111111", "", ".webp")
	name := key[strings.LastIndex(key, "/")+1:]
	name = strings.TrimSuffix(name, ".webp")

	if len(name) != 36 {
		t.Fatalf("nama file %q bukan UUID (panjang %d)", name, len(name))
	}
	if name[14] != '7' {
		t.Errorf("UUID %q bukan versi 7 (karakter versi = %q); harus uuid.NewV7()", name, name[14])
	}
	// Varian RFC4122: dua bit teratas byte itu harus 10, jadi nilainya
	// boleh 8, 9, a, atau b.
	switch name[19] {
	case '8', '9', 'a', 'b':
	default:
		t.Errorf("UUID %q punya varian %q, harus RFC4122 (8/9/a/b)", name, name[19])
	}
}

// TestMediaObjectKeyUnik memastikan dua upload tidak menghasilkan key yang
// sama (kalau sama, file kedua menimpa yang pertama).
func TestMediaObjectKeyUnik(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 500; i++ {
		k := mediaObjectKey("avatar", "11111111-1111-4111-8111-111111111111", "", ".webp")
		if seen[k] {
			t.Fatalf("key duplikat: %q", k)
		}
		seen[k] = true
	}
}

// TestMediaVariantKey memeriksa key varian mengikuti pola PRD
// (varian _320 / _800 di belakang key utama).
func TestMediaVariantKey(t *testing.T) {
	base := mediaObjectKey("thumbnail", "11111111-1111-4111-8111-111111111111",
		"22222222-2222-4222-8222-222222222222", ".webp")
	got := mediaVariantKey(base, 320, ".webp")
	if !strings.HasPrefix(got, base) || !strings.HasSuffix(got, "_320.webp") {
		t.Errorf("mediaVariantKey = %q, ingin %q berakhiran _320.webp", got, base)
	}
}

// TestSanitizeSegment menjaga object key tetap berada di dalam foldernya.
//
// Key ikut dipakai untuk membentuk URL dan disimpan di database. Segmen yang
// berisi "/" atau ".." bisa membuat objek berada di luar folder yang
// dimaksud — termasuk menimpa folder orang lain.
func TestSanitizeSegment(t *testing.T) {
	bad := []string{"", "  ", "..", ".", "a/b", "a\\b", "../../etc", "x y", "a;b", "a/b/c"}
	for _, s := range bad {
		if got, ok := sanitizeSegment(s); ok {
			t.Errorf("sanitizeSegment(%q) diterima sebagai %q, harus ditolak", s, got)
		}
	}
	good := map[string]string{
		"11111111-1111-4111-8111-111111111111": "11111111-1111-4111-8111-111111111111",
		"admin":                                "admin",
		"user_01":                              "user_01",
		"a-b_c":                                "a-b_c",
	}
	for in, want := range good {
		got, ok := sanitizeSegment(in)
		if !ok {
			t.Errorf("sanitizeSegment(%q) ditolak, harus diterima", in)
			continue
		}
		if got != want {
			t.Errorf("sanitizeSegment(%q) = %q, ingin %q", in, got, want)
		}
	}
}

// TestMediaKeyPrefix dipakai saat menghapus semua objek milik satu entitas.
func TestMediaKeyPrefix(t *testing.T) {
	if got := mediaKeyPrefix("thumbnail", "abc"); got != "thumbnails/abc/" {
		t.Errorf("mediaKeyPrefix = %q, ingin thumbnails/abc/", got)
	}
	if got := mediaKeyPrefix("thumbnail", ""); got != "thumbnails/" {
		t.Errorf("mediaKeyPrefix = %q, ingin thumbnails/", got)
	}
	if got := mediaKeyPrefix("tidak-ada", "abc"); got != "" {
		t.Errorf("mediaKeyPrefix untuk purpose tak dikenal = %q, ingin kosong", got)
	}
}
