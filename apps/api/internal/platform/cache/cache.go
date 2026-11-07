// Package cache manages the Valkey (Redis-compatible) client.
package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Open creates a Valkey client and verifies it can reach the server.
func Open(ctx context.Context, url string) (*redis.Client, error) {
	opts, err := redis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("valkey: %w", err)
	}

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("valkey: %w", err)
	}

	return client, nil
}
