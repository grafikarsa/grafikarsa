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

func (c *Cache) GetStrings(ctx context.Context, key string) ([]string, error) {
	return c.rdb.SMembers(ctx, key).Result()
}

func (c *Cache) SetStrings(ctx context.Context, key string, vals []string, ttl time.Duration) error {
	pipe := c.rdb.Pipeline()
	pipe.Del(ctx, key)
	if len(vals) > 0 {
		pipe.SAdd(ctx, key, vals)
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
