package http

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Object key untuk media (PRD §7.1).
//
// Formatnya:
//
//	avatars/{user_id}/{uuid7}.webp
//	banners/{user_id}/{uuid7}.webp
//	thumbnails/{portfolio_id}/{uuid7}.webp   (+ varian _320 / _800)
//	blocks/{portfolio_id}/{uuid7}.webp
//	documents/{portfolio_id}/{uuid7}.pdf
//	feedback/{thread_id}/{uuid7}.bin
//
// Dua alasan dibingkai path per-entity, bukan satu folder global:
//
// 1. Pencarian: saat portfolio dihapus, semua objeknya bisa dihapus dengan
//    satu prefix, tanpa perlu melihat tabel media satu per satu.
// 2. Deduplikasi: file identik milik user yang sama untuk purpose sama
//    di-reuse, bukan diduplikasi.
//
// Dan kenapa uuid7, bukan uuid4: uuid7 terurut waktu, jadi listing objek di
// dalam satu prefix terurut kronologis dan bisa dipaginasi dengan batas atas
// (key) — persis seperti indeks pada key-value store. Dengan uuid4, listing
// selalu penuh scan.
//
// Prefix path memakai bentuk jamak yang sama dengan PRD; `block_image`
// dipetakan ke `blocks` dan `changelog_cover` ke `changelog_covers` supaya
// path yang sama tidak dipakai dua purpose berbeda.

// prefixByPurpose memetakan purpose media ke prefix path.
var prefixByPurpose = map[string]string{
	"avatar":          "avatars",
	"banner":          "banners",
	"thumbnail":       "thumbnails",
	"block_image":     "blocks",
	"document":        "documents",
	"feedback":        "feedback",
	"changelog_cover": "changelog_covers",
}

// entityScopedPurpose: purpose yang foldernya ditentukan oleh entitas, bukan
// oleh user. Kalau entitasnya belum diketahui, path-nya jatuh ke owner
// (user) — supaya tidak pernah ada objek tanpa folder yang jelas.
var entityScopedPurpose = map[string]bool{
	"thumbnail":   true,
	"block_image": true,
	"document":    true,
	"feedback":    true,
}

// uuid7String membuat UUIDv7 (time-ordered).
//
// Kalau pembuatannya gagal (prng tidak tersedia), jatuh ke uuid4 —
// lebih baik id acak daripada gagal menyimpan file.
func uuid7String() string {
	if u, err := uuid.NewV7(); err == nil {
		return u.String()
	}
	return uuid.NewString()
}

// mediaObjectKey menyusun object key untuk sebuah media.
//
// ownerID = pemilik media (user). entityID = entitas induk kalau purpose-nya
// entity-scoped dan pemanggil menyediakannya (portfolio, thread feedback).
// entityID kosong berarti path memakai ownerID.
//
// ext adalah ekstensi file termasuk titik, mis. ".webp".
func mediaObjectKey(purpose, ownerID, entityID, ext string) string {
	prefix, ok := prefixByPurpose[purpose]
	if !ok {
		// Purpose tak dikenal tidak boleh bocor ke path bebas; pakai
		// folder "misc" supaya tetap deterministik & terisolasi.
		prefix = "misc"
	}
	segment := ownerID
	if entityScopedPurpose[purpose] && entityID != "" {
		segment = entityID
	}
	if segment == "" {
		segment = "unassigned"
	}
	return fmt.Sprintf("%s/%s/%s%s", prefix, segment, uuid7String(), ext)
}

// mediaVariantKey membuat key untuk varian lebar (mis. _320, _800).
//
// Varian memakai key terpisah, bukan suffix pada key utama, supaya file
// utama dan variannya bisa dihapus tanpa saling menimpa.
func mediaVariantKey(baseKey string, width int, ext string) string {
	return fmt.Sprintf("%s_%d%s", baseKey, width, ext)
}

// mediaKeyPrefix mengembalikan prefix folder untuk sebuah purpose — dipakai
// saat menghapus semua objek di bawah satu portfolio.
func mediaKeyPrefix(purpose, entityID string) string {
	prefix, ok := prefixByPurpose[purpose]
	if !ok {
		return ""
	}
	if entityID == "" {
		return prefix + "/"
	}
	return prefix + "/" + entityID + "/"
}

// sanitizeSegment membersihkan satu segmen path supaya tidak bisa keluar dari
// foldernya.
//
// Object key ikut tersimpan di database dan dipakai untuk membentuk URL, jadi
// segmen yang berisi "/" atau ".." harus ditolak: tanpa ini, key bisa
// menunjuk ke folder lain (mis. key "../users/xxx" menimpa avatar orang).
func sanitizeSegment(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", false
	}
	if strings.ContainsAny(s, "/\\") || s == "." || s == ".." {
		return "", false
	}
	// UUID atau slug sederhana: huruf, angka, "-", "_".
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return "", false
		}
	}
	return s, true
}
