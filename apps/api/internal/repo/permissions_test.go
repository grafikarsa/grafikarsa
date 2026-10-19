package repo

import (
	"strings"
	"testing"
)

// Guard: GetUserPermissions WAJIB memfilter ur.expires_at.
//
// Assignment role boleh punya masa berlaku (PRD §1.1 — mis. OSIS 1 tahun).
// Tanpa filter ini, assignment yang sudah kedaluwarsa tetap memberi
// permission selamanya, yaitu privilege escalation yang tidak disadari.
// GetUserScope (query di sebelahnya) SUDAH memfilter expiry — kedua query
// ini harus konsisten.
//
// Test ini memeriksa SQL-nya, bukan hasil query, karena belum ada integration
// test dengan Postgres asli. Lihat backlog P3-03 (testcontainers).
func TestGetUserPermissionsFiltersExpiredRoles(t *testing.T) {
	sql := getUserPermissions
	if !strings.Contains(sql, "ur.expires_at") {
		t.Fatalf("GetUserPermissions harus memfilter ur.expires_at:\n%s", sql)
	}

	// Buang baris komentar supaya filter tidak bisa "tersembunyi" di komentar
	// (sqlc membuang komentar deskripsi, jadi ini graded sebagai SQL nyata).
	var body []string
	for _, line := range strings.Split(sql, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		body = append(body, line)
	}
	clean := strings.Join(body, "\n")

	want := "ur.expires_at IS NULL OR ur.expires_at > NOW()"
	if !strings.Contains(clean, want) {
		t.Fatalf("filter expiry harus persis %q:\n%s", want, clean)
	}
}

// Guard: GetUserScope WAJIB memfilter ur.expires_at (sudah ada, tapi dijaga
// agar tetap sinkron dengan GetUserPermissions di atas).
func TestGetUserScopeFiltersExpiredRoles(t *testing.T) {
	sql := getUserScope
	if !strings.Contains(sql, "ur.expires_at") {
		t.Fatalf("GetUserScope harus memfilter ur.expires_at:\n%s", sql)
	}
}
