package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"grafikarsa/api/internal/config"
	"grafikarsa/api/internal/storage"
)

// storageIndex mengindeks objek di object storage baru (S3-compatible).
//
// Dipakai migrasi media untuk dua hal:
//
//  1. Menyimpan metadata yang BENAR: mime, ukuran, checksum. Tanpa ini
//     setiap media terdaftar sebagai "application/octet-stream" dengan
//     size 0 dan checksum hash-URL — dan karena handler by-key memakai
//     media_assets.mime sebagai Content-Type, setiap gambar di situs
//     dilayani sebagai unduhan, bukan tampilan.
//  2. Memastikan objeknya benar-benar ADA di storage baru. Kalau tidak,
//     URL lama diarahkan ke key yang tidak ada — gambar rusak di karya
//     yang sudah tayang.
type storageIndex struct {
	store storage.Store
	sizes map[string]int64
	sums  map[string]string
	mu    sync.Mutex
}

// newStorageIndex membangun indeks seluruh objek di bucket.
func newStorageIndex(ctx context.Context, cfg *config.Config) (*storageIndex, error) {
	store := storage.New(cfg)
	ix := &storageIndex{
		store: store,
		sizes: map[string]int64{},
		sums:  map[string]string{},
	}
	// maxKeys = 0 berarti "tanpa batas" — ListObjectsV2 sudah memaginasi
	// sendiri. Kalau diberi angka, hasilnya DIPOTONG di angka itu (bukan per
	// halaman) — dulu indeks hanya berisi 1.000 objek pertama, sisanya
	// dianggap "tidak ada" dan migrasi gagal.
	objs, err := store.ListObjectsV2(ctx, "", 0)
	if err != nil {
		return nil, fmt.Errorf("listing objek storage: %w", err)
	}
	for _, o := range objs {
		ix.sizes[o.Key] = o.Size
	}
	slog.Info("indeks storage dibangun", "objek", len(ix.sizes))
	return ix, nil
}

// size mengembalikan ukuran objek (0 = tidak ada).
func (ix *storageIndex) size(key string) int64 {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	return ix.sizes[key]
}

// checksum menghitung sha256 objek (dihitung sekali per key).
func (ix *storageIndex) checksum(ctx context.Context, key string) (string, error) {
	ix.mu.Lock()
	if s, ok := ix.sums[key]; ok {
		ix.mu.Unlock()
		return s, nil
	}
	ix.mu.Unlock()

	rc, err := ix.store.Get(ctx, key)
	if err != nil {
		return "", err
	}
	defer rc.Close()
	h := sha256.New()
	if _, err := io.Copy(h, rc); err != nil {
		return "", err
	}
	sum := hex.EncodeToString(h.Sum(nil))
	ix.mu.Lock()
	ix.sums[key] = sum
	ix.mu.Unlock()
	return sum, nil
}

// mimeByExt menebak Content-Type dari ekstensi key.
//
// Dipakai karena S3 Listing tidak mengembalikan Content-Type, dan
// application/octet-stream membuat browser mengunduh file sebagai ganti
// menampilkannya.
func mimeByExt(key string) string {
	// Key objek storage tidak punya query string, tapi defensif tetap
	// baik: sisanya membuat ekstensi tidak terbaca -> mime generik.
	if j := strings.IndexAny(key, "?#"); j >= 0 {
		key = key[:j]
	}
	switch strings.ToLower(path.Ext(key)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".xls":
		return "application/vnd.ms-excel"
	case ".xlsx":
		return "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case ".pptx":
		return "application/vnd.openxmlformats-officedocument.presentationml.presentation"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".txt":
		return "text/plain; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}

// purposeFromKey menentukan purpose media dari awalan key.
//
// Sebelum ini semua media dilabeli "document", padahal key v1 sudah
// membedakan jenisnya (avatars/, thumbnails/, portfolio-images/, ...).
// Purpose dipakai UI untuk mengelompokkan media (avatar terpisah dari gambar
// karya), jadi label yang salah = galeri yang kacau.
//
// Semua nilai harus ada di CHECK constraint media_purpose_valid.
func purposeFromKey(key string) string {
	switch {
	case strings.HasPrefix(key, "avatars/"):
		return "avatar"
	case strings.HasPrefix(key, "banners/"):
		return "banner"
	case strings.HasPrefix(key, "thumbnails/"):
		return "thumbnail"
	case strings.HasPrefix(key, "portfolio-images/"):
		return "block_image"
	case strings.HasPrefix(key, "feedback/"):
		return "feedback"
	case strings.HasPrefix(key, "feedback-attachments/"):
		return "feedback"
	case strings.HasPrefix(key, "changelog-covers/"):
		return "changelog_cover"
	default:
		return "document"
	}
}

// storageURLRegex menangkap URL /storage/... di dalam teks (mis. payload
// blok portfolio yang berupa JSON).
var storageURLRegex = regexp.MustCompile(`https?://[^\s"'<>\\]+/storage/[^\s"'<>\\]+`)

// collectJSONURLs mengumpulkan URL /storage/ dari kolom JSON/JSONB.
//
// Ini yang TERLEWAT sebelumnya: URL gambar di dalam payload blok portfolio
// (2.110 blok di produksi) tidak ada di kolom URL, tapi di dalam JSON. Kalau
// tidak dikumpulkan, URL lamanya tidak terdaftar di legacy_urls dan berakhir
// 410 Gone — gambar di karya yang sudah tayang rusak.
func collectJSONURLs(ctx context.Context, v1 *pgxpool.Pool, s *v1schema, table, col string, out map[string]bool) {
	if !s.has(table, col) {
		return
	}
	rows, err := v1.Query(ctx, fmt.Sprintf(`SELECT %s::text FROM %s WHERE %s::text LIKE '%%/storage/%%'`, col, table, col))
	if err != nil {
		slog.Warn("gagal membaca kolom json", "table", table, "col", col, "err", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var txt any
		if err := rows.Scan(&txt); err != nil {
			return
		}
		for _, m := range storageURLRegex.FindAllString(fmt.Sprint(txt), -1) {
			out[m] = true
		}
	}
}
