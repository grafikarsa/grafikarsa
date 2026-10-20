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
	if cached, err := c.GetStrings(ctx, permsKey(userID)); err == nil && len(cached) > 0 {
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

func feedWeightsKey() string { return "feed:weights" }

func invalidateFeedWeights(ctx context.Context, c *cache.Cache) {
	_ = c.Del(ctx, feedWeightsKey())
}

// ---- view counter (HyperLogLog) ----

func viewHLLKey(portfolioID, day string) string { return "views:hll:" + portfolioID + ":" + day }
func viewDirtyKey(day string) string            { return "views:dirty:" + day }

// recordViewHLL mencatat satu view unik di HLL + menandai portfolio sebagai
// dirty agar worker mem-flush agregatnya. TTL HLL 40 hari (cukup untuk statistik
// 30 hari), dirty set 2 hari.
func (a *App) recordViewHLL(ctx context.Context, portfolioID, viewerKey string) {
	day := a.now().Format("2006-01-02")
	key := viewHLLKey(portfolioID, day)
	if err := a.Cache.PFAdd(ctx, key, viewerKey); err != nil {
		return
	}
	_ = a.Cache.Expire(ctx, key, 40*24*time.Hour)
	_ = a.Cache.SAdd(ctx, viewDirtyKey(day), portfolioID)
	_ = a.Cache.Expire(ctx, viewDirtyKey(day), 48*time.Hour)
}

func userActiveKey(userID string) string { return "uactive:" + userID }

// userActive memeriksa is_active + deleted_at dengan cache 60 detik agar tidak
// ada query DB di setiap request. Invalidasi dilakukan saat admin mengubah
// status user (activate/deactivate/delete).
func (a *App) userActive(ctx context.Context, userID string) bool {
	if cached, err := a.Cache.Get(ctx, userActiveKey(userID)); err == nil {
		return cached == "1"
	}
	u, err := a.Q.GetUserByID(ctx, mustUUID(userID))
	active := err == nil && u.IsActive
	if active {
		_ = a.Cache.Set(ctx, userActiveKey(userID), "1", 60*time.Second)
	} else {
		_ = a.Cache.Set(ctx, userActiveKey(userID), "0", 60*time.Second)
	}
	return active
}

func invalidateUserActive(ctx context.Context, c *cache.Cache, userID string) {
	_ = c.Del(ctx, userActiveKey(userID))
}
