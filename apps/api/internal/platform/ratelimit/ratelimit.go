// Package ratelimit provides a Valkey-backed GCRA rate limiter.
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

const keyPrefix = "rl:"

// Rule describes a token bucket: Limit requests per Window.
type Rule struct {
	Limit  int
	Window time.Duration
}

// Result is the outcome of a single Allow call.
type Result struct {
	Allowed    bool
	Limit      int
	Remaining  int
	RetryAfter time.Duration
	ResetAfter time.Duration
}

// Limiter enforces rate limit rules in Valkey.
type Limiter struct {
	limiter *redis_rate.Limiter
}

// New creates a Limiter backed by rdb.
func New(rdb *redis.Client) *Limiter {
	return &Limiter{limiter: redis_rate.NewLimiter(rdb)}
}

// Allow consumes one token for key under rule. Key is namespaced internally
// as "rl:" + key.
func (l *Limiter) Allow(ctx context.Context, key string, rule Rule) (Result, error) {
	res, err := l.limiter.Allow(ctx, keyPrefix+key, redis_rate.Limit{
		Rate:   rule.Limit,
		Burst:  rule.Limit,
		Period: rule.Window,
	})
	if err != nil {
		return Result{}, fmt.Errorf("ratelimit: allow: %w", err)
	}

	return Result{
		Allowed:    res.Allowed > 0,
		Limit:      rule.Limit,
		Remaining:  res.Remaining,
		RetryAfter: res.RetryAfter,
		ResetAfter: res.ResetAfter,
	}, nil
}

// Reset clears the counter for key (e.g. after a successful login).
func (l *Limiter) Reset(ctx context.Context, key string) error {
	if err := l.limiter.Reset(ctx, keyPrefix+key); err != nil {
		return fmt.Errorf("ratelimit: reset: %w", err)
	}
	return nil
}
