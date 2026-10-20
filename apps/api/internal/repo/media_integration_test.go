package repo_test

import (
	"context"
	"testing"
	"time"

	"grafikarsa/api/internal/dbtest"
	"grafikarsa/api/internal/repo"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// seedUser membuat user minimal.
func seedUser(t *testing.T, pool *pgxpool.Pool, username string) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	id := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, username, email, password_hash, name, role_primary)
		 VALUES ($1, $2, $3, 'x', $4, 'siswa')`,
		id, username, username+"@test.local", username)
	if err != nil {
		t.Fatalf("seed user %s: %v", username, err)
	}
	return id
}

// seedMedia membuat media dengan purpose + status tertentu.
func seedMedia(t *testing.T, pool *pgxpool.Pool, owner uuid.UUID, purpose, key, status string, age time.Duration) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	id := uuid.New()
	_, err := pool.Exec(ctx,
		`INSERT INTO media_assets (id, owner_id, purpose, object_key, mime, size_bytes,
		                          checksum_sha256, status, created_at)
		 VALUES ($1, $2, $3, $4, 'image/jpeg', 100, $5, $6, NOW() - $7::interval)`,
		id, owner, purpose, key, uuid.NewString(), status, fmtDur(age))
	if err != nil {
		t.Fatalf("seed media %s: %v", key, err)
	}
	return id
}

func fmtDur(d time.Duration) string {
	return d.String()
}

// TestListOrphanedConfirmedMedia menguji query yang mencari media 'confirmed'
// yang tidak lagi direferensikan siapa pun.
//
// Logikanya mudah salah sehingga diuji dengan banyak kasus sekaligus:
// media yang masih dipakai (avatar, thumbnail, asset block, payload block),
// media preview milik media lain, purpose yang=rUJ direferensikan non-FK,
// media yang belum cukup umur, dan media yang masih 'pending'.
func TestListOrphanedConfirmedMedia(t *testing.T) {
	pool := dbtest.Pool(t)
	dbtest.Truncate(t, pool, "media_assets", "portfolio_blocks", "portfolios", "users", "majors")
	q := repo.New(pool)
	ctx := context.Background()

	owner := seedUser(t, pool, "owner01")
	other := seedUser(t, pool, "other01")

	// Portfolio + block milik owner.
	var portfolioID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO portfolios (id, owner_id, title, slug, status)
		 VALUES ($1, $2, 'P', 'p', 'published') RETURNING id`,
		uuid.New(), owner).Scan(&portfolioID); err != nil {
		t.Fatalf("seed portfolio: %v", err)
	}
	var blockID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO portfolio_blocks (id, portfolio_id, block_type, position, payload)
		 VALUES ($1, $2, 'image', 0, '{}'::jsonb) RETURNING id`,
		uuid.New(), portfolioID).Scan(&blockID); err != nil {
		t.Fatalf("seed block: %v", err)
	}

	// Media yang DIPAKAI -> tidak boleh jadi orphan.
	avatar := seedMedia(t, pool, owner, "avatar", "k/avatar", "confirmed", 2*time.Hour)
	thumb := seedMedia(t, pool, owner, "thumbnail", "k/thumb", "confirmed", 2*time.Hour)
	asset := seedMedia(t, pool, owner, "block_image", "k/asset", "confirmed", 2*time.Hour)
	payload := seedMedia(t, pool, owner, "block_image", "k/payload", "confirmed", 2*time.Hour)
	preview := seedMedia(t, pool, owner, "block_image", "k/preview", "confirmed", 2*time.Hour)
	_, _ = pool.Exec(ctx, `UPDATE users SET avatar_media_id = $1 WHERE id = $2`, avatar, owner)
	_, _ = pool.Exec(ctx, `UPDATE portfolios SET thumbnail_media_id = $1 WHERE id = $2`, thumb, portfolioID)
	_, _ = pool.Exec(ctx, `UPDATE portfolio_blocks SET asset_media_id = $1 WHERE id = $2`, asset, blockID)
	_, _ = pool.Exec(ctx, `UPDATE portfolio_blocks SET payload = jsonb_build_object('media_id', $1::text) WHERE id = $2`, payload, blockID)
	// preview milik media lain (halaman-1 PDF)
	_, _ = pool.Exec(ctx, `UPDATE media_assets SET preview_media_id = $1 WHERE id = $2`, preview, thumb)

	// Media ORPHAN sungguhan.
	seedMedia(t, pool, owner, "block_image", "k/orphan", "confirmed", 2*time.Hour)
	// Media orphan tapi terlalu baru (upload yang baru saja dikonfirmasi).
	seedMedia(t, pool, owner, "block_image", "k/fresh", "confirmed", 5*time.Minute)
	// Media orphan tapi status pending -> milik ListPendingMedia.
	seedMedia(t, pool, owner, "block_image", "k/pending", "pending", 2*time.Hour)
	// Purpose non-FK: bisa dirujuk dari body/markdown -> tidak boleh dihapus.
	seedMedia(t, pool, owner, "document", "k/doc", "confirmed", 2*time.Hour)
	seedMedia(t, pool, owner, "feedback", "k/feedback", "confirmed", 2*time.Hour)
	// Milik user lain, juga orphan.
	seedMedia(t, pool, other, "block_image", "k/other", "confirmed", 2*time.Hour)

	rows, err := q.ListOrphanedConfirmedMedia(ctx)
	if err != nil {
		t.Fatalf("ListOrphanedConfirmedMedia: %v", err)
	}
	got := map[string]bool{}
	for _, r := range rows {
		got[r.ObjectKey] = true
	}

	mustBe := []string{"k/orphan", "k/other"}
	for _, k := range mustBe {
		if !got[k] {
			t.Errorf("media orphan %q harus terdeteksi, tetapi tidak", k)
		}
	}
	mustNot := []string{
		"k/avatar", "k/thumb", "k/asset", "k/payload", "k/preview",
		"k/fresh", "k/pending", "k/doc", "k/feedback",
	}
	for _, k := range mustNot {
		if got[k] {
			t.Errorf("media %q TIDAK boleh terdeteksi sebagai orphan", k)
		}
	}
	if len(rows) != len(mustBe) {
		t.Errorf("jumlah orphan = %d, ingin %d (detail: %v)", len(rows), len(mustBe), got)
	}

	// Preview milik media yang sudah dihapus harus ikut jadi orphan di
	// lintasan berikutnya (kalau tidak, preview-nya menumpuk selamanya).
	if _, err := pool.Exec(ctx, `UPDATE media_assets SET status = 'deleted' WHERE id = $1`, thumb); err != nil {
		t.Fatalf("hapus parent preview: %v", err)
	}
	rows2, err := q.ListOrphanedConfirmedMedia(ctx)
	if err != nil {
		t.Fatalf("ListOrphanedConfirmedMedia (kedua): %v", err)
	}
	found := false
	for _, r := range rows2 {
		if r.ObjectKey == "k/preview" {
			found = true
		}
	}
	if !found {
		t.Error("preview dari media yang sudah dihapus harus ikut terdeteksi sebagai orphan")
	}
}

// TestGetUserPermissionsRespectsExpiry membuktikan filter ur.expires_at
// bekerja terhadap data nyata.
//
// Assignment role yang sudah kedaluwarsa tidak lagi memberi permission.
// Ini yang membuat fitur role berumur (PRD §1.1) benar-benar berlaku.
func TestGetUserPermissionsRespectsExpiry(t *testing.T) {
	pool := dbtest.Pool(t)
	dbtest.Truncate(t, pool, "user_roles", "role_permissions", "permissions", "roles", "users")
	q := repo.New(pool)
	ctx := context.Background()

	uid := seedUser(t, pool, "permuser01")

	// Dua permission + satu role.
	var permLive, permExpired uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO permissions (id, key, permission_group, description)
		 VALUES ($1, 'test.live', 'test', '') RETURNING id`, uuid.New()).Scan(&permLive); err != nil {
		t.Fatalf("seed permission live: %v", err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO permissions (id, key, permission_group, description)
		 VALUES ($1, 'test.expired', 'test', '') RETURNING id`, uuid.New()).Scan(&permExpired); err != nil {
		t.Fatalf("seed permission expired: %v", err)
	}
	var roleID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO roles (id, slug, name, scope_type, is_system)
		 VALUES ($1, 'test_role', 'Test', 'global', false) RETURNING id`, uuid.New()).Scan(&roleID); err != nil {
		t.Fatalf("seed role: %v", err)
	}
	_, _ = pool.Exec(ctx, `INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2), ($1, $3)`, roleID, permLive, permExpired)

	// Assignment yang masih berlaku.
	_, _ = pool.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_id, scope_major_id, assigned_by, expires_at)
		 VALUES ($1, $2, NULL, NULL, NOW() + INTERVAL '1 day')`, uid, roleID)

	keys, err := q.GetUserPermissions(ctx, mustUUID(uid))
	if err != nil {
		t.Fatalf("GetUserPermissions: %v", err)
	}
	if !contains(keys, "test.live") {
		t.Errorf("permission yang masih berlaku harus diberikan, dapat %v", keys)
	}

	// Sekarang buat assignment-nya kedaluwarsa.
	_, _ = pool.Exec(ctx, `UPDATE user_roles SET expires_at = NOW() - INTERVAL '1 day' WHERE user_id = $1`, uid)

	keys, err = q.GetUserPermissions(ctx, mustUUID(uid))
	if err != nil {
		t.Fatalf("GetUserPermissions (kedua): %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("assignment yang sudah kedaluwarsa tidak lagi boleh memberi permission, dapat %v", keys)
	}
}

func mustUUID(id uuid.UUID) pgtype.UUID {
	var v pgtype.UUID
	_ = v.Scan(id.String())
	return v
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
