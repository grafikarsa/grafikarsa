// Package dbtest menyediakan helper untuk integration test yang butuh
// Postgres sungguhan.
//
// Kenapa Postgres asli dan bukan SQLite: skema v2 memakai ekstensi Postgres
// (citext, pg_trgm, generated column tsvector, CHECK constraint, dan unique
// partial index). Perilaku SQLite berbeda jauh, sehingga test SQLite bisa
// hijau sementara query-nya salah di produksi.
//
// Environment:
//
//	TEST_DATABASE_URL  DSN Postgres untuk test (mis. postgres://user:pass@host:5432/db?sslmode=disable)
//
// Kalau env tersebut tidak di-set, test di-skip (bukan gagal) supaya
// `go test ./...` di laptop tanpa database tetap jalan. Di CI, job `api`
// menyalakan service postgres lalu meng-set env ini.
package dbtest

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool mengembalikan pool ke database test.
//
// Test melompat (t.Skip) kalau TEST_DATABASE_URL tidak di-set atau database
// tidak bisa dihubungi. Test yang di-skip dilaporkan jelas sebagai "skipped",
// bukan diam-diam dianggap lulus.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL tidak di-set — integration test dilewati")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("gagal membuat pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("gagal menghubungi database test: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// Truncate mengosongkan tabel yang dipakai test lalu mengembalikan pool.
//
// Dipakai supaya setiap test_case saling independen.
func Truncate(t *testing.T, pool *pgxpool.Pool, tables ...string) {
	t.Helper()
	ctx := context.Background()
	for _, tbl := range tables {
		if _, err := pool.Exec(ctx, "TRUNCATE TABLE "+tbl+" CASCADE"); err != nil {
			t.Fatalf("gagal truncate %s: %v", tbl, err)
		}
	}
}
