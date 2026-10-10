-- Social, feed, leaderboard, search, notifikasi.

-- name: FollowUser :exec
INSERT INTO follows (follower_id, following_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UnfollowUser :exec
DELETE FROM follows WHERE follower_id = $1 AND following_id = $2;

-- name: IsFollowing :one
SELECT COUNT(*) FROM follows WHERE follower_id = $1 AND following_id = $2;

-- name: IsMutual :one
SELECT COUNT(*) FROM follows WHERE follower_id = $1 AND following_id = $2
  OR follower_id = $2 AND following_id = $1;

-- name: ListFollowingIDs :many
SELECT following_id FROM follows WHERE follower_id = $1;

-- name: LikePortfolio :exec
INSERT INTO portfolio_likes (user_id, portfolio_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: UnlikePortfolio :exec
DELETE FROM portfolio_likes WHERE user_id = $1 AND portfolio_id = $2;

-- name: IsLiked :one
SELECT COUNT(*) FROM portfolio_likes WHERE user_id = $1 AND portfolio_id = $2;

-- name: CountLikes :one
SELECT COUNT(*) FROM portfolio_likes WHERE portfolio_id = $1;

-- name: CreateComment :one
INSERT INTO comments (portfolio_id, author_id, parent_id, body_md, body_html)
VALUES ($1, $2, $3, $4, $5) RETURNING id;

-- name: ListComments :many
SELECT c.id, c.portfolio_id, c.author_id, c.parent_id, c.body_md, c.body_html,
  c.is_edited, c.created_at, u.username, u.name AS author_name
FROM comments c JOIN users u ON u.id = c.author_id
WHERE c.portfolio_id = $1 AND c.deleted_at IS NULL
ORDER BY c.created_at ASC LIMIT $2 OFFSET $3;

-- name: GetComment :one
SELECT id, portfolio_id, author_id, parent_id, body_md FROM comments
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: DeleteComment :exec
UPDATE comments SET deleted_at = NOW() WHERE id = $1;

-- name: CountComments :one
SELECT COUNT(*) FROM comments WHERE portfolio_id = $1 AND deleted_at IS NULL;

-- name: RecordView :exec
INSERT INTO portfolio_views (portfolio_id, viewer_id, session_id, viewed_date)
VALUES ($1, $2, $3, CURRENT_DATE)
ON CONFLICT DO NOTHING;

-- name: GetFeedCandidates :many
SELECT p.id, p.title, p.slug, p.thumbnail_media_id, p.published_at,
  COALESCE(ps.engagement, 0) AS engagement, COALESCE(ps.quality, 0) AS quality,
  COALESCE(ps.trending, 0) AS trending,
  u.username, u.name AS owner_name, u.major_id AS owner_major_id, u.class_id AS owner_class_id
FROM portfolios p
JOIN users u ON u.id = p.owner_id
LEFT JOIN portfolio_stats ps ON ps.portfolio_id = p.id
WHERE p.status = 'published' AND p.deleted_at IS NULL
  AND p.published_at > NOW() - INTERVAL '90 days'
ORDER BY COALESCE(ps.trending, 0) DESC LIMIT 50;

-- name: GetRecentFeed :many
SELECT p.id, p.title, p.slug, p.thumbnail_media_id, p.published_at,
  u.username, u.name AS owner_name
FROM portfolios p JOIN users u ON u.id = p.owner_id
WHERE p.status = 'published' AND p.deleted_at IS NULL
ORDER BY p.published_at DESC LIMIT $1 OFFSET $2;

-- name: GetFollowingFeed :many
SELECT p.id, p.title, p.slug, p.thumbnail_media_id, p.published_at,
  u.username, u.name AS owner_name
FROM portfolios p JOIN users u ON u.id = p.owner_id
JOIN follows f ON f.following_id = p.owner_id AND f.follower_id = $1
WHERE p.status = 'published' AND p.deleted_at IS NULL
ORDER BY p.published_at DESC LIMIT $2 OFFSET $3;

-- name: GetFeedPreference :one
SELECT user_id, algorithm FROM feed_preferences WHERE user_id = $1 LIMIT 1;

-- name: SaveFeedPreference :exec
INSERT INTO feed_preferences (user_id, algorithm) VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE SET algorithm = EXCLUDED.algorithm;

-- name: GetUserInterest :one
SELECT user_id, tag_weights, major_weights, total_likes FROM user_interests WHERE user_id = $1 LIMIT 1;

-- name: SearchPortfolios :many
SELECT p.id, p.title, p.slug, p.thumbnail_media_id, p.published_at,
  u.username, u.name AS owner_name,
  ts_rank(p.search_tsv, plainto_tsquery('simple', $1)) AS rank
FROM portfolios p JOIN users u ON u.id = p.owner_id
WHERE p.status = 'published' AND p.deleted_at IS NULL
  AND (p.search_tsv @@ plainto_tsquery('simple', $1) OR similarity(p.title, $1) > 0.2)
ORDER BY rank DESC LIMIT 20;

-- name: SearchUsers :many
SELECT id, username, name FROM users
WHERE deleted_at IS NULL AND (similarity(name, $1) > 0.2 OR username ILIKE '%' || $1 || '%')
ORDER BY similarity(name, $1) DESC LIMIT 20;

-- name: CreateNotification :one
INSERT INTO notifications (user_id, type, title, body, data)
VALUES ($1, $2, $3, $4, $5) RETURNING id;

-- name: ListNotifications :many
SELECT id, user_id, type, title, body, data, is_read, read_at, created_at
FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CountUnreadNotifications :one
SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = FALSE;

-- name: MarkNotificationRead :exec
UPDATE notifications SET is_read = TRUE, read_at = NOW() WHERE id = $1 AND user_id = $2;

-- name: MarkAllNotificationsRead :exec
UPDATE notifications SET is_read = TRUE, read_at = NOW() WHERE user_id = $1 AND is_read = FALSE;

-- name: DeleteNotification :exec
DELETE FROM notifications WHERE id = $1 AND user_id = $2;

-- name: EnqueueOutbox :one
INSERT INTO outbox (topic, payload) VALUES ($1, $2) RETURNING id;

-- name: ClaimOutbox :many
UPDATE outbox SET status = 'sent' WHERE id IN (
  SELECT id FROM outbox WHERE status = 'pending' AND next_retry_at <= NOW()
  ORDER BY id ASC LIMIT 50 FOR UPDATE SKIP LOCKED
) RETURNING id, topic, payload;

-- name: UpsertPortfolioStats :exec
INSERT INTO portfolio_stats (portfolio_id, likes, views_7d, views_30d, comments, assessment_avg, quality, engagement, trending)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (portfolio_id) DO UPDATE SET
  likes = EXCLUDED.likes, views_7d = EXCLUDED.views_7d, views_30d = EXCLUDED.views_30d,
  comments = EXCLUDED.comments, assessment_avg = EXCLUDED.assessment_avg,
  quality = EXCLUDED.quality, engagement = EXCLUDED.engagement, trending = EXCLUDED.trending,
  computed_at = NOW();

-- name: GetPortfolioStats :one
SELECT portfolio_id, likes, views_7d, views_30d, comments, assessment_avg, quality, engagement, trending
FROM portfolio_stats WHERE portfolio_id = $1 LIMIT 1;

-- name: UpsertLeaderboard :one
INSERT INTO leaderboards (period, scope, kind, computed_for)
VALUES ($1, $2, $3, CURRENT_DATE)
ON CONFLICT (period, scope, kind, computed_for) DO UPDATE SET created_at = NOW()
RETURNING id;

-- name: ClearLeaderboardEntries :exec
DELETE FROM leaderboard_entries WHERE leaderboard_id = $1;

-- name: InsertLeaderboardEntry :exec
INSERT INTO leaderboard_entries (leaderboard_id, rank, subject_user_id, subject_portfolio_id, score, breakdown, delta_rank)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetLeaderboard :many
SELECT e.rank, e.subject_user_id, e.subject_portfolio_id, e.score, e.breakdown, e.delta_rank
FROM leaderboard_entries e JOIN leaderboards l ON l.id = e.leaderboard_id
WHERE l.period = $1 AND l.scope = $2 AND l.kind = $3
ORDER BY l.computed_for DESC, e.rank ASC LIMIT 20;

-- name: InsertAuditLog :exec
INSERT INTO audit_logs (actor_id, action, entity, entity_id, diff, ip)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: StudentLeaderboardCandidates :many
SELECT u.id, u.username, u.name,
  COUNT(DISTINCT p.id) AS portfolio_count,
  COUNT(DISTINCT pl.user_id) AS total_likes,
  COUNT(DISTINCT f.follower_id) AS followers
FROM users u
LEFT JOIN portfolios p ON p.owner_id = u.id AND p.status = 'published' AND p.deleted_at IS NULL
LEFT JOIN portfolio_likes pl ON pl.portfolio_id = p.id
LEFT JOIN follows f ON f.following_id = u.id
WHERE u.role_primary = 'siswa' AND u.deleted_at IS NULL
GROUP BY u.id HAVING COUNT(DISTINCT p.id) >= 1;

-- name: ProjectLeaderboardCandidates :many
SELECT p.id, p.title, p.slug, p.owner_id, p.published_at,
  COUNT(DISTINCT pl.user_id) AS likes,
  COUNT(DISTINCT c.id) AS comments
FROM portfolios p
LEFT JOIN portfolio_likes pl ON pl.portfolio_id = p.id
LEFT JOIN comments c ON c.portfolio_id = p.id AND c.deleted_at IS NULL
WHERE p.status = 'published' AND p.deleted_at IS NULL
GROUP BY p.id;

-- name: SuggestTags :many
SELECT slug, name FROM tags WHERE similarity(name, $1) > 0.2 ORDER BY similarity(name, $1) DESC LIMIT 5;
