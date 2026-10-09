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

func (c *Cache) Close() error { return c.rdb.Close() }
