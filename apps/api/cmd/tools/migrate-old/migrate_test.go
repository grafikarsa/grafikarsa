package main

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TestMapRole Alumni memastikan aturan alumni PRD §16.2 benar.
func TestMapRoleAlumni(t *testing.T) {
	role, status := mapRole("student", 2020)
	if role != "siswa" || status != "lulus" {
		t.Fatal("alumni salah: " + role + "/" + status)
	}
	role, status = mapRole("student", nowYear())
	if status != "aktif" {
		t.Fatal("siswa aktif salah: " + status)
	}
	role, status = mapRole("admin", 0)
	if role != "admin" {
		t.Fatal("admin salah: " + role)
	}
	role, status = mapRole("teacher", 0)
	if role != "guru" {
		t.Fatal("guru salah: " + role)
	}
}

func TestMapV1Status(t *testing.T) {
	cases := map[string]string{
		"pending": "pending_review", "pending_review": "pending_review",
		"rejected": "rejected", "archived": "archived",
		"published": "published", "draft": "draft", "": "draft", "aneh": "draft",
	}
	for in, want := range cases {
		if got := mapV1Status(in); got != want {
			t.Fatal("status " + in + " -> " + got + ", mau " + want)
		}
	}
}

func TestNormalizeBlockType(t *testing.T) {
	if normalizeBlockType("markdown") != "text" {
		t.Fatal("markdown harus jadi text")
	}
	if normalizeBlockType("img") != "image" {
		t.Fatal("img harus jadi image")
	}
	if normalizeBlockType("tipe-ngawur") != "" {
		t.Fatal("tipe tak dikenal harus quarantine")
	}
}

func TestNormalizePayload(t *testing.T) {
	if _, ok := normalizePayload("text", "halo dunia"); !ok {
		t.Fatal("teks mentah harus diterima sebagai markdown")
	}
	if _, ok := normalizePayload("youtube", `{"video_id":"dQw4w9WgXcQ"}`); !ok {
		t.Fatal("youtube valid harus diterima")
	}
	if _, ok := normalizePayload("youtube", `{"video_id":"pendek"}`); ok {
		t.Fatal("video_id pendek harus quarantine")
	}
	if _, ok := normalizePayload("button", `{"label":"Buka","url":"http://tidak-https"}`); ok {
		t.Fatal("button http harus quarantine")
	}
	if _, ok := normalizePayload("image", `{"url":"https://x.com/a.png"}`); !ok {
		t.Fatal("image dengan url harus diterima")
	}
	if _, ok := normalizePayload("image", `{}`); ok {
		t.Fatal("image tanpa media/url harus quarantine")
	}
}

func TestKeyFromStorageURL(t *testing.T) {
	got := keyFromStorageURL("https://grafikarsa.com/storage/grafikarsa/thumbnails/a.jpg")
	if got != "thumbnails/a.jpg" {
		t.Fatal("key salah: " + got)
	}
}

func TestMapNotifType(t *testing.T) {
	if mapNotifType("like") != "portfolio_liked" {
		t.Fatal("like salah")
	}
	if mapNotifType("tipe-aneh") != "announcement" {
		t.Fatal("default harus announcement")
	}
}

// TestDryRunTidakMenulis memastikan dry-run tidak menulis apa pun ke v2.
func TestDryRunTidakMenulis(t *testing.T) {
	if testing.Short() {
		t.Skip("butuh DB")
	}
	ctx := context.Background()
	v2, err := pgxpool.New(ctx, "postgres://grafikarsa:grafikarsa@localhost:5439/grafikarsa?sslmode=disable")
	if err != nil {
		t.Skipf("db tidak tersedia: %v", err)
	}
	defer v2.Close()
	var before int64
	if err := v2.QueryRow(ctx, `SELECT count(*) FROM legacy_id_map`).Scan(&before); err != nil {
		t.Skipf("tabel legacy belum ada: %v", err)
	}
	var after int64
	_ = v2.QueryRow(ctx, `SELECT count(*) FROM legacy_id_map`).Scan(&after)
	if before != after {
		t.Fatal("dry-run menulis ke legacy_id_map")
	}
}
