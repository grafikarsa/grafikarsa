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
//	ALLOW_DESTRUCTIVE_TEST_DB=1  Izinkan test memakai database yang namanya
//	                              tidak mengandung "test" (PENTING: test ini
//	                              MENGHAPUS isi tabel dengan TRUNCATE).
//
// Kalau env tersebut tidak di-set, test di-skip (bukan gagal) supaya
// `go test ./...` di laptop tanpa database tetap jalan. Di CI, job `api`
// menyalakan service postgres lalu meng-set env ini.
package dbtest

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool mengembalikan pool ke database test.
//
// Test melompat (t.Skip) kalau TEST_DATABASE_URL tidak di-set atau database
// tidak bisa dihubungi. Test yang di-skip dilaporkan jelas sebagai "skipped",
// bukan diam-diam dianggap lulus.
//
// PERINGATAN: integration test di package ini menghapus isi tabel dengan
// TRUNCATE. Karena itu nama database wajib mengandung "test" — kalau tidak,
// test DITOLAK, bukan dijalankan. Tanpa guard ini, satu perintah
// `go test ./...` dengan DSN yang menunjuk database dev/produksi akan
// menghapus seluruh user, portfolio, dan komentar di dalamnya tanpa jejak.
// Setel ALLOW_DESTRUCTIVE_TEST_DB=1 kalau memang disengaja.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL tidak di-set — integration test dilewati")
	}
	if err := checkDBName(dsn); err != nil {
		t.Fatalf("TEST_DATABASE_URL ditolak: %v", err)
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

// checkDBName memastikan nama database terlihat seperti database test.
func checkDBName(dsn string) error {
	if os.Getenv("ALLOW_DESTRUCTIVE_TEST_DB") == "1" {
		return nil
	}
	name := dbNameFromDSN(dsn)
	if name == "" {
		return fmt.Errorf("nama database tidak bisa dibaca dari DSN")
	}
	if !strings.Contains(strings.ToLower(name), "test") {
		return fmt.Errorf(
			"nama database %q tidak mengandung \"test\"; integration test menghapus isi tabel dengan TRUNCATE, "+
				"jadi menolak agar database dev/produksi tidak ikut terhapus. "+
				"Gunakan database terpisah (mis. grafikarsa_test), atau setel ALLOW_DESTRUCTIVE_TEST_DB=1 kalau memang disengaja",
			name,
		)
	}
	return nil
}

// dbNameFromDSN mengambil nama database dari DSN Postgres.
func dbNameFromDSN(dsn string) string {
	// Format: postgres://user:pass@host:5432/dbname?params
	if i := strings.Index(dsn, "://"); i >= 0 {
		rest := dsn[i+3:]
		// Buang kredensial & host.
		if at := strings.LastIndex(rest, "@"); at >= 0 {
			rest = rest[at+1:]
		}
		// Buang query string.
		if q := strings.Index(rest, "?"); q >= 0 {
			rest = rest[:q]
		}
		// Nama database = bagian setelah host[:port].
		if slash := strings.Index(rest, "/"); slash >= 0 {
			rest = rest[slash+1:]
		}
		if rest != "" {
			return rest
		}
		return ""
	}
	// Format key=value: host=... dbname=...
	for _, kv := range strings.Fields(dsn) {
		if strings.HasPrefix(kv, "dbname=") {
			return strings.TrimPrefix(kv, "dbname=")
		}
	}
	return ""
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
