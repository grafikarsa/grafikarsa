package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"grafikarsa/api/internal/config"
)

type Cache struct {
	rdb *redis.Client
}

func New(cfg *config.Config) *Cache {
	return &Cache{rdb: redis.NewClient(&redis.Options{
		Addr:     cfg.CacheAddr(),
		Password: cfg.Cache.Password,
		DB:       cfg.Cache.DB,
	})}
}

func (c *Cache) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return c.rdb.Ping(ctx).Err()
}

func (c *Cache) Exists(ctx context.Context, key string) (bool, error) {
	n, err := c.rdb.Exists(ctx, key).Result()
	return n > 0, err
}

func (c *Cache) Set(ctx context.Context, key, val string, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, val, ttl).Err()
}

func (c *Cache) Get(ctx context.Context, key string) (string, error) {
	return c.rdb.Get(ctx, key).Result()
}

func (c *Cache) GetStrings(ctx context.Context, key string) ([]string, error) {
	return c.rdb.LRange(ctx, key, 0, -1).Result()
}

func (c *Cache) SetStrings(ctx context.Context, key string, vals []string, ttl time.Duration) error {
	pipe := c.rdb.Pipeline()
	pipe.Del(ctx, key)
	if len(vals) > 0 {
		// LIST (RPUSH+LRANGE) menjaga urutan peringkat; SET (SADD/SMEMBERS)
		// mengacak urutan sehingga feed ter-cache tampil acak.
		args := make([]any, len(vals))
		for i, v := range vals {
			args[i] = v
		}
		pipe.RPush(ctx, key, args...)
	}
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *Cache) Del(ctx context.Context, keys ...string) error {
	return c.rdb.Del(ctx, keys...).Err()
}

func (c *Cache) IncrWithExpire(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	pipe := c.rdb.Pipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

func (c *Cache) Close() error { return c.rdb.Close() }

func (c *Cache) GetInt(ctx context.Context, key string) (int64, error) {
	return c.rdb.Get(ctx, key).Int64()
}

func (c *Cache) SetInt(ctx context.Context, key string, val int64, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, val, ttl).Err()
}

func (c *Cache) GetBytes(ctx context.Context, key string) ([]byte, error) {
	return c.rdb.Get(ctx, key).Bytes()
}

func (c *Cache) SetBytes(ctx context.Context, key string, val []byte, ttl time.Duration) error {
	return c.rdb.Set(ctx, key, val, ttl).Err()
}

// DelPattern menghapus key dengan pola (dipakai invalidasi cache leaderboard).
// Memakai SCAN agar tidak memblokir Dragonfly.
func (c *Cache) DelPattern(ctx context.Context, pattern string) error {
	iter := c.rdb.Scan(ctx, 0, pattern, 200).Iterator()
	for iter.Next(ctx) {
		_ = c.rdb.Del(ctx, iter.Val()).Err()
	}
	return iter.Err()
}

// ---- HyperLogLog (view counter) ----

func (c *Cache) PFAdd(ctx context.Context, key, val string) error {
	return c.rdb.PFAdd(ctx, key, val).Err()
}

func (c *Cache) PFCount(ctx context.Context, key string) (int64, error) {
	return c.rdb.PFCount(ctx, key).Result()
}

func (c *Cache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return c.rdb.Expire(ctx, key, ttl).Err()
}

// ---- SET mentah (dirty marker; LIST dipakai untuk feed yang butuh urutan) ----

func (c *Cache) SAdd(ctx context.Context, key string, vals ...string) error {
	if len(vals) == 0 {
		return nil
	}
	args := make([]any, len(vals))
	for i, v := range vals {
		args[i] = v
	}
	return c.rdb.SAdd(ctx, key, args...).Err()
}

func (c *Cache) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.rdb.SMembers(ctx, key).Result()
}
