package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// SlidingWindow menghitung request dalam jendela waktu yang **menggeser**
// (sliding window), bukan jendela tetap.
//
// Kenapa tidak fixed window: dengan key per-menit, batas antar menit yang
// berdekatan bisa dipakai penuh dua kali berturut-turut. Contoh batas 100
// request/menit: pada 10:00:59 client boleh 100 request, lalu tepat pada
// 10:01:00 jendela berikutnya dimulai dan client boleh 100 request lagi —
// jadi 200 request dalam ~1 detik. Sliding window menutup celah itu: yang
// dihitung adalah request dalam 60 detik terakhir, selalu.
//
// Mekanisme: sorted set (ZSET) berisi timestamp tiap request. Window
// yang lewat dibuang, request baru ditambahkan, lalu dihitung dengan ZCARD.
//
// Catatan performa: tiga perintah Redis (ZREMRANGEBYSCORE, ZADD, ZCARD)
// dikirim dalam SATU pipeline, jadi hanya satu round-trip per request.
func (c *Cache) SlidingWindow(ctx context.Context, key string, window time.Duration, limit int) (int64, error) {
	now := time.Now()
	cutoff := now.Add(-window).UnixMilli()
	member := fmt.Sprintf("%d-%d", now.UnixNano(), randSeq())

	pipe := c.rdb.Pipeline()
	// Buang request yang sudah keluar dari jendela.
	pipe.ZRemRangeByScore(ctx, key, "0", strconvFormatInt(cutoff))
	// Tandai request ini.
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now.UnixMilli()), Member: member})
	// Hitung request dalam jendela.
	card := pipe.ZCard(ctx, key)
	// Jaga agar key tidak menetap kalau traffic berhenti.
	pipe.Expire(ctx, key, window+time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return card.Val(), nil
}

// ResetSlidingWindow menghapus seluruh hitungan untuk sebuah kunci.
// Dipakai saat login berhasil: batas request tidak boleh ikut mewarisi
// hitungan dari sebelum login.
func (c *Cache) ResetSlidingWindow(ctx context.Context, key string) error {
	return c.rdb.Del(ctx, key).Err()
}

// seq menghasilkan angka urut untuk membedakan member ZSET.
// Tanpa ini, dua request dalam milidetik yang sama bisa menimpa satu
// sama lain (member ZSET harus unik) dan request-nya jadi tak terhitung.
var seqCounter struct {
	last int64
}

// randSeq mengembalikan angka unik (monoton) untuk member ZSET.
func randSeq() int64 {
	now := time.Now().UnixNano()
	if now <= seqCounter.last {
		now = seqCounter.last + 1
	}
	seqCounter.last = now
	return now
}

// strconvFormatInt mengubah int64 menjadi string desimal.
func strconvFormatInt(v int64) string {
	return fmt.Sprintf("%d", v)
}
