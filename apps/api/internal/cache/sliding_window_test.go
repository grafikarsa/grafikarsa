package cache

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// testCache membuat cache yang terhubung ke Redis test.
//
// Env: TEST_REDIS_ADDR (mis. "127.0.0.1:6390"). Kalau tidak di-set, test
// di-skip — supaya `go test ./...` tetap jalan tanpa Redis.
func testCache(t *testing.T) *Cache {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR tidak di-set — test sliding window dilewati")
	}
	c := &Cache{rdb: redis.NewClient(&redis.Options{Addr: addr})}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.rdb.Ping(ctx).Err(); err != nil {
		c.rdb.Close()
		t.Fatalf("gagal menghubungi Redis test: %v", err)
	}
	t.Cleanup(func() { c.rdb.Close() })
	return c
}

// TestSlidingWindowMenghitungRequestDalamJendela menguji dasar sliding
// window: request dihitung per jendela yang menggeser.
func TestSlidingWindowMenghitungRequestDalamJendela(t *testing.T) {
	c := testCache(t)
	ctx := context.Background()
	key := "test:sliding:dasar"
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		t.Fatalf("gagal bersihkan: %v", err)
	}

	for i := 0; i < 5; i++ {
		n, err := c.SlidingWindow(ctx, key, time.Minute, 100)
		if err != nil {
			t.Fatalf("SlidingWindow: %v", err)
		}
		if n != int64(i+1) {
			t.Errorf("hitungan ke-%d = %d, ingin %d", i+1, n, i+1)
		}
	}
}

// TestSlidingWindowBedaJendelaTidakBocorKeJendelaBerikutnya adalah inti
// alasan kenapa fixed window tidak cukup.
//
// Dengan fixed window (key per menit), batas antar menit bisa dipakai penuh
// dua kali berturut-turut: 100 request pada 10:00:59, lalu 100 request lagi
// tepat pada 10:01:00. Sliding window menutup celah itu.
func TestSlidingWindowBedaJendelaTidakBocorKeJendelaBerikutnya(t *testing.T) {
	c := testCache(t)
	ctx := context.Background()
	key := "test:sliding:antijendela"
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		t.Fatalf("gagal bersihkan: %v", err)
	}

	// Isi jendela dengan 3 request.
	for i := 0; i < 3; i++ {
		if _, err := c.SlidingWindow(ctx, key, 200*time.Millisecond, 100); err != nil {
			t.Fatalf("SlidingWindow: %v", err)
		}
	}
	// Tunggu jendela bergeser: request lama harus keluar dari hitungan.
	time.Sleep(300 * time.Millisecond)
	n, err := c.SlidingWindow(ctx, key, 200*time.Millisecond, 100)
	if err != nil {
		t.Fatalf("SlidingWindow: %v", err)
	}
	if n != 1 {
		t.Errorf("hitungan setelah jendela bergeser = %d, ingin 1 (request lama harus keluar)", n)
	}
}

// TestSlidingWindowBatasDilewati memastikan request yang melewati batas
// dihitung sebagai request ke-N+1, sehingga middleware bisa menolak.
func TestSlidingWindowBatasDilewati(t *testing.T) {
	c := testCache(t)
	ctx := context.Background()
	key := "test:sliding:batas"
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		t.Fatalf("gagal bersihkan: %v", err)
	}

	const limit = 3
	for i := 1; i <= limit; i++ {
		n, err := c.SlidingWindow(ctx, key, time.Minute, limit)
		if err != nil {
			t.Fatalf("SlidingWindow: %v", err)
		}
		if n != int64(i) {
			t.Errorf("hitungan = %d, ingin %d", n, i)
		}
	}
	// Request berikutnya harus melewati batas.
	n, err := c.SlidingWindow(ctx, key, time.Minute, limit)
	if err != nil {
		t.Fatalf("SlidingWindow: %v", err)
	}
	if n <= int64(limit) {
		t.Errorf("hitungan setelah batas = %d, harus > %d", n, limit)
	}
}

// TestSlidingWindowMemberUnik memastikan dua request dalam milidetik yang
// sama tetap dihitung dua kali.
//
// Member ZSET harus unik. Kalau tidak, request kedua menimpa request
// pertama dan hitungannya understated — rate limit jadi longgar tanpa
// disadari.
func TestSlidingWindowMemberUnik(t *testing.T) {
	c := testCache(t)
	ctx := context.Background()
	key := "test:sliding:unik"
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		t.Fatalf("gagal bersihkan: %v", err)
	}

	const n = 50
	for i := 0; i < n; i++ {
		got, err := c.SlidingWindow(ctx, key, time.Minute, 1000)
		if err != nil {
			t.Fatalf("SlidingWindow: %v", err)
		}
		if got != int64(i+1) {
			t.Fatalf("hitungan request ke-%d = %d, ingin %d (member ZSET menimpa)", i+1, got, i+1)
		}
	}
}

// TestResetSlidingWindow memastikan hitungan bisa dihapus (dipakai saat login
// berhasil, supaya batas request tidak mewarisi hitungan sebelum login).
func TestResetSlidingWindow(t *testing.T) {
	c := testCache(t)
	ctx := context.Background()
	key := "test:sliding:reset"
	if err := c.rdb.Del(ctx, key).Err(); err != nil {
		t.Fatalf("gagal bersihkan: %v", err)
	}

	for i := 0; i < 4; i++ {
		if _, err := c.SlidingWindow(ctx, key, time.Minute, 100); err != nil {
			t.Fatalf("SlidingWindow: %v", err)
		}
	}
	if err := c.ResetSlidingWindow(ctx, key); err != nil {
		t.Fatalf("ResetSlidingWindow: %v", err)
	}
	n, err := c.SlidingWindow(ctx, key, time.Minute, 100)
	if err != nil {
		t.Fatalf("SlidingWindow: %v", err)
	}
	if n != 1 {
		t.Errorf("hitungan setelah reset = %d, ingin 1", n)
	}
}
