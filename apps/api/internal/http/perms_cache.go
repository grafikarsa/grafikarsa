package http

import (
	"context"
	"time"

	"grafikarsa/api/internal/cache"
	"grafikarsa/api/internal/repo"
)

func blacklistKey(jti string) string { return "bl:" + jti }

func blacklisted(ctx context.Context, c *cache.Cache, jti string) bool {
	if jti == "" {
		return false
	}
	ok, err := c.Exists(ctx, blacklistKey(jti))
	return err == nil && ok
}

func blacklistJTI(ctx context.Context, c *cache.Cache, jti string, ttl time.Duration) {
	if jti == "" {
		return
	}
	_ = c.Set(ctx, blacklistKey(jti), "1", ttl)
}

func permsKey(userID string) string { return "perms:" + userID }

func loadPerms(ctx context.Context, c *cache.Cache, q *repo.Queries, userID string) map[string]bool {
	if cached, err := c.GetStrings(ctx, permsKey(userID)); err == nil && cached != nil {
		out := map[string]bool{}
		for _, p := range cached {
			out[p] = true
		}
		return out
	}
	keys, err := q.GetUserPermissions(ctx, mustUUID(userID))
	if err != nil {
		return map[string]bool{}
	}
	_ = c.SetStrings(ctx, permsKey(userID), keys, 5*time.Minute)
	out := map[string]bool{}
	for _, p := range keys {
		out[p] = true
	}
	return out
}

func invalidatePerms(ctx context.Context, c *cache.Cache, userID string) {
	_ = c.Del(ctx, permsKey(userID))
}
