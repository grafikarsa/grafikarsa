package testutil_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/testutil"
)

func TestNewDB_HasGooseAndRiverSchema(t *testing.T) {
	pool := testutil.NewDB(t)
	ctx := context.Background()

	for _, table := range []string{"users", "river_job"} {
		var exists bool
		if err := pool.QueryRow(ctx, "SELECT to_regclass($1) IS NOT NULL", "public."+table).Scan(&exists); err != nil {
			t.Fatalf("periksa tabel %s: %v", table, err)
		}
		if !exists {
			t.Fatalf("tabel %s tidak ada di database uji", table)
		}
	}
}

func TestNewDB_DatabasesAreIsolated(t *testing.T) {
	ctx := context.Background()
	first := testutil.NewDB(t)
	second := testutil.NewDB(t)

	const key = "testutil.isolation"
	if _, err := first.Exec(ctx, "INSERT INTO settings (key, value) VALUES ($1, 'true'::jsonb)", key); err != nil {
		t.Fatalf("insert setting: %v", err)
	}

	var exists bool
	if err := second.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM settings WHERE key = $1)", key).Scan(&exists); err != nil {
		t.Fatalf("periksa setting: %v", err)
	}
	if exists {
		t.Fatal("setting dari database pertama terlihat di database kedua")
	}
}

func TestNewSeededDB_AppliesSeeds(t *testing.T) {
	pool := testutil.NewSeededDB(t)
	ctx := context.Background()

	count := func(table string) int {
		t.Helper()
		var got int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&got); err != nil {
			t.Fatalf("hitung %s: %v", table, err)
		}
		return got
	}

	if got := count("majors"); got != 9 {
		t.Fatalf("jumlah majors = %d, mau 9", got)
	}
	if got := count("permissions"); got != 35 {
		t.Fatalf("jumlah permissions = %d, mau 35", got)
	}
}

func TestNewDB_ParallelSubtests(t *testing.T) {
	for i := range 8 {
		t.Run(fmt.Sprintf("pod_%d", i), func(t *testing.T) {
			t.Parallel()

			pool := testutil.NewDB(t)
			key := fmt.Sprintf("testutil.parallel.%c", 'a'+i)
			if _, err := pool.Exec(context.Background(), "INSERT INTO settings (key, value) VALUES ($1, 'true'::jsonb)", key); err != nil {
				t.Fatalf("insert setting: %v", err)
			}
		})
	}
}
