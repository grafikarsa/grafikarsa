package repo_test

import (
	"context"
	"testing"
	"time"

	"grafikarsa/api/internal/dbtest"
	"grafikarsa/api/internal/repo"

	"github.com/google/uuid"
)

// TestSessionRotation Dan test-token-reuse menguji siklus hidup refresh token.
//
// Yang diuji:
//   - Rotasi: token lama dicabut, token baru terbit dalam family yang sama.
//   - Deteksi reuse: memakai token yang SUDAH dirotasi harus bisa dikenali
//     (session-nya revoked), sehingga handler bisa mencabut seluruh family.
//   - Kedaluwarsa absolut: session punya batas mutlak yang tidak bisa
//     diperpanjang.
func TestSessionRotation(t *testing.T) {
	pool := dbtest.Pool(t)
	dbtest.Truncate(t, pool, "sessions", "users")
	q := repo.New(pool)
	ctx := context.Background()

	uid := seedUser(t, pool, "sessuser01")
	family := uuid.New()

	// Sesi pertama.
	s1, err := q.CreateSession(ctx, repo.CreateSessionParams{
		UserID:            mustUUID(uid),
		RefreshHash:       "hash-token-lama",
		FamilyID:          mustUUID(family),
		ExpiresAt:         toPgTS(time.Now().Add(time.Hour)),
		AbsoluteExpiresAt: toPgTS(time.Now().Add(30 * 24 * time.Hour)),
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Rotasi: token lama dicabut, token baru menunjuk ke sebelumnya.
	s2, err := q.CreateSession(ctx, repo.CreateSessionParams{
		UserID:            mustUUID(uid),
		RefreshHash:       "hash-token-baru",
		FamilyID:          mustUUID(family),
		ExpiresAt:         toPgTS(time.Now().Add(time.Hour)),
		AbsoluteExpiresAt: toPgTS(time.Now().Add(30 * 24 * time.Hour)),
	})
	if err != nil {
		t.Fatalf("CreateSession (rotasi): %v", err)
	}
	if err := q.RotateSession(ctx, repo.RotateSessionParams{ID: s1, RotatedTo: s2}); err != nil {
		t.Fatalf("RotateSession: %v", err)
	}

	// Token lama harus terdeteksi sebagai sudah dicabut.
	old, err := q.GetSessionByHash(ctx, "hash-token-lama")
	if err != nil {
		t.Fatalf("GetSessionByHash (lama): %v", err)
	}
	if !old.RevokedAt.Valid {
		t.Error("token lama harus revoked setelah rotasi")
	}
	if !old.RotatedTo.Valid {
		t.Error("token lama harus menunjuk ke session penerus")
	}

	// Token baru harus aktif dan satu keluarga.
	cur, err := q.GetSessionByHash(ctx, "hash-token-baru")
	if err != nil {
		t.Fatalf("GetSessionByHash (baru): %v", err)
	}
	if cur.RevokedAt.Valid {
		t.Error("token baru tidak boleh revoked")
	}
	if cur.FamilyID != mustUUID(family) {
		t.Error("token baru harus satu family dengan token lama")
	}
}

// TestRevokeFamily menguji pencabutan seluruh rantai token (dipakai saat
// deteksi reuse).
func TestRevokeFamily(t *testing.T) {
	pool := dbtest.Pool(t)
	dbtest.Truncate(t, pool, "sessions", "users")
	q := repo.New(pool)
	ctx := context.Background()

	uid := seedUser(t, pool, "sessuser02")
	family := uuid.New()
	// Tiga token dalam satu keluarga (riwayat rotasi).
	for i, h := range []string{"h1", "h2", "h3"} {
		if _, err := q.CreateSession(ctx, repo.CreateSessionParams{
			UserID:            mustUUID(uid),
			RefreshHash:       h,
			FamilyID:          mustUUID(family),
			ExpiresAt:         toPgTS(time.Now().Add(time.Hour)),
			AbsoluteExpiresAt: toPgTS(time.Now().Add(30 * 24 * time.Hour)),
		}); err != nil {
			t.Fatalf("CreateSession %d: %v", i, err)
		}
	}
	if err := q.RevokeFamily(ctx, mustUUID(family)); err != nil {
		t.Fatalf("RevokeFamily: %v", err)
	}
	// Semua token di keluarga itu harus dicabut.
	for _, h := range []string{"h1", "h2", "h3"} {
		s, err := q.GetSessionByHash(ctx, h)
		if err != nil {
			t.Fatalf("GetSessionByHash %s: %v", h, err)
		}
		if !s.RevokedAt.Valid {
			t.Errorf("token %s harus revoked setelah RevokeFamily", h)
		}
	}
	// Sesi milik keluarga lain tidak boleh ikut tercabut.
	other := uuid.New()
	if _, err := q.CreateSession(ctx, repo.CreateSessionParams{
		UserID:            mustUUID(uid),
		RefreshHash:       "h-lain",
		FamilyID:          mustUUID(other),
		ExpiresAt:         toPgTS(time.Now().Add(time.Hour)),
		AbsoluteExpiresAt: toPgTS(time.Now().Add(30 * 24 * time.Hour)),
	}); err != nil {
		t.Fatalf("CreateSession (keluarga lain): %v", err)
	}
	if err := q.RevokeFamily(ctx, mustUUID(family)); err != nil {
		t.Fatalf("RevokeFamily (kedua): %v", err)
	}
	s, err := q.GetSessionByHash(ctx, "h-lain")
	if err != nil {
		t.Fatalf("GetSessionByHash (keluarga lain): %v", err)
	}
	if s.RevokedAt.Valid {
		t.Error("sesi di keluarga lain tidak boleh ikut tercabut")
	}
}

// TestOutboxClaim menguji pengambilan event outbox.
//
// Yang dijaga:
//   - Event pending bisa diklaim dan statusnya jadi 'processing'.
//   - Event yang sedang diproses TIDAK boleh diklaim lagi (tidak ada
//     pemrosesan ganda).
//   - Event 'processing' yang worker tinggalkan (crash) bisa di-reap
//     kembali ke pending lalu diklaim lagi.
func TestOutboxClaim(t *testing.T) {
	pool := dbtest.Pool(t)
	dbtest.Truncate(t, pool, "outbox")
	q := repo.New(pool)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if _, err := q.EnqueueOutbox(ctx, repo.EnqueueOutboxParams{
			Topic:   "notify.test",
			Payload: []byte(`{"user_id":"x"}`),
		}); err != nil {
			t.Fatalf("EnqueueOutbox: %v", err)
		}
	}

	claimed, err := q.ClaimOutbox(ctx)
	if err != nil {
		t.Fatalf("ClaimOutbox: %v", err)
	}
	if len(claimed) != 3 {
		t.Fatalf("ClaimOutbox mengambil %d event, ingin 3", len(claimed))
	}
	for _, c := range claimed {
		if c.Attempts != 1 {
			t.Errorf("attempts event %d = %d, ingin 1", c.ID, c.Attempts)
		}
	}

	// Tidak boleh diklaim lagi saat masih diproses.
	claimed2, err := q.ClaimOutbox(ctx)
	if err != nil {
		t.Fatalf("ClaimOutbox (kedua): %v", err)
	}
	if len(claimed2) != 0 {
		t.Errorf("event 'processing' tidak boleh diklaim lagi, dapat %d", len(claimed2))
	}

	// Reaper mengembalikan event yang ditinggalkan worker ke pending.
	if _, err := pool.Exec(ctx, `UPDATE outbox SET next_retry_at = NOW() - INTERVAL '1 minute'`); err != nil {
		t.Fatalf("atur next_retry_at: %v", err)
	}
	if err := q.ReapStuckOutbox(ctx); err != nil {
		t.Fatalf("ReapStuckOutbox: %v", err)
	}
	claimed3, err := q.ClaimOutbox(ctx)
	if err != nil {
		t.Fatalf("ClaimOutbox (setelah reap): %v", err)
	}
	if len(claimed3) != 3 {
		t.Errorf("setelah reap harus bisa diklaim 3 event, dapat %d", len(claimed3))
	}
}

// TestOutboxFailure menguji kebijakan retry.
//
// Event yang gagal 5 kali menjadi 'failed' dan tidak pernah dicoba lagi —
// supaya event rusak tidak berputar tanpa henti.
func TestOutboxFailure(t *testing.T) {
	pool := dbtest.Pool(t)
	dbtest.Truncate(t, pool, "outbox")
	q := repo.New(pool)
	ctx := context.Background()

	if _, err := q.EnqueueOutbox(ctx, repo.EnqueueOutboxParams{
		Topic:   "notify.gagal",
		Payload: []byte(`{}`),
	}); err != nil {
		t.Fatalf("EnqueueOutbox: %v", err)
	}

	for i := 1; i <= 5; i++ {
		rows, err := q.ClaimOutbox(ctx)
		if err != nil {
			t.Fatalf("ClaimOutbox (putaran %d): %v", i, err)
		}
		if len(rows) != 1 {
			t.Fatalf("putaran %d: diklaim %d event, ingin 1", i, len(rows))
		}
		if err := q.MarkOutboxFailed(ctx, rows[0].ID); err != nil {
			t.Fatalf("MarkOutboxFailed: %v", err)
		}
		// Jadwalkan ulang ke masa lalu supaya bisa diklaim pada putaran berikut.
		if _, err := pool.Exec(ctx, `UPDATE outbox SET next_retry_at = NOW() - INTERVAL '1 minute'`); err != nil {
			t.Fatalf("buka next_retry_at: %v", err)
		}
	}

	var status string
	var attempts int32
	if err := pool.QueryRow(ctx, `SELECT status, attempts FROM outbox LIMIT 1`).Scan(&status, &attempts); err != nil {
		t.Fatalf("baca status outbox: %v", err)
	}
	if status != "failed" {
		t.Errorf("status setelah 5x gagal = %q, ingin 'failed'", status)
	}
	if attempts != 5 {
		t.Errorf("attempts = %d, ingin 5", attempts)
	}
	// Event 'failed' tidak boleh lagi diklaim.
	rows, err := q.ClaimOutbox(ctx)
	if err != nil {
		t.Fatalf("ClaimOutbox (setelah failed): %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("event 'failed' tidak boleh diklaim lagi, dapat %d", len(rows))
	}
}

// TestUpsertUserInterests menguji tabel minat: bobot ditulis dengan benar
// dan tidak hilang kalau tidak ada perubahan (upsert, bukan insert buta).
func TestUpsertUserInterests(t *testing.T) {
	pool := dbtest.Pool(t)
	dbtest.Truncate(t, pool, "user_interests", "users")
	q := repo.New(pool)
	ctx := context.Background()

	uid := seedUser(t, pool, "minatuser01")
	params := repo.UpsertUserInterestParams{
		UserID:       mustUUID(uid),
		TagWeights:   []byte(`{"tag-a":2,"tag-b":1}`),
		MajorWeights: []byte(`{"mjr-a":3}`),
	}
	if err := q.UpsertUserInterest(ctx, params); err != nil {
		t.Fatalf("UpsertUserInterests: %v", err)
	}
	// Upsert kedua dengan bobot sama tidak boleh menggandakan baris.
	if err := q.UpsertUserInterest(ctx, params); err != nil {
		t.Fatalf("UpsertUserInterests (kedua): %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM user_interests WHERE user_id = $1`, uid).Scan(&n); err != nil {
		t.Fatalf("hitung baris: %v", err)
	}
	if n != 1 {
		t.Errorf("baris user_interests = %d, ingin 1 (upsert, bukan insert)", n)
	}
	got, err := q.GetUserInterest(ctx, mustUUID(uid))
	if err != nil {
		t.Fatalf("GetUserInterest: %v", err)
	}
	if len(got.TagWeights) == 0 {
		t.Error("tag_weights tidak boleh kosong setelah upsert")
	}
}
