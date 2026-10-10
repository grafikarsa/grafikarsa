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
SELECT p.id, p.owner_id, p.title, p.slug, p.thumbnail_media_id, p.published_at,
  COALESCE(ps.engagement, 0) AS engagement, COALESCE(ps.quality, 0) AS quality,
  COALESCE(ps.trending, 0) AS trending,
  u.username, u.name AS owner_name, u.major_id AS owner_major_id, u.class_id AS owner_class_id
FROM portfolios p
JOIN users u ON u.id = p.owner_id
LEFT JOIN portfolio_stats ps ON ps.portfolio_id = p.id
WHERE p.status = 'published' AND p.deleted_at IS NULL
  AND p.published_at > NOW() - INTERVAL '90 days'
ORDER BY (COALESCE(ps.trending, 0) + CASE WHEN p.published_at > NOW() - INTERVAL '3 days' THEN 0.5 ELSE 0 END) DESC
LIMIT 50;

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
-- Tandai 'processing' (bukan 'sent') supaya crash tidak menghilangkan event.
UPDATE outbox SET status = 'processing', attempts = attempts + 1,
  next_retry_at = NOW() + INTERVAL '5 minutes'
WHERE id IN (
  SELECT id FROM outbox WHERE status = 'pending' AND next_retry_at <= NOW()
  ORDER BY id ASC LIMIT 50 FOR UPDATE SKIP LOCKED
) RETURNING id, topic, payload, attempts;

-- name: MarkOutboxSent :exec
UPDATE outbox SET status = 'sent' WHERE id = $1;

-- name: MarkOutboxFailed :exec
UPDATE outbox
SET status = CASE WHEN attempts >= 5 THEN 'failed' ELSE 'pending' END,
    next_retry_at = NOW() + (attempts * INTERVAL '1 minute')
WHERE id = $1;

-- name: UpsertPortfolioStats :exec
INSERT INTO portfolio_stats (portfolio_id, likes, views_7d, views_30d, comments, assessment_avg, quality, engagement, trending)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (portfolio_id) DO UPDATE SET
  likes = EXCLUDED.likes, views_7d = EXCLUDED.views_7d, views_30d = EXCLUDED.views_30d,
  comments = EXCLUDED.comments, assessment_avg = EXCLUDED.assessment_avg,
  quality = EXCLUDED.quality, engagement = EXCLUDED.engagement, trending = EXCLUDED.trending,
  computed_at = NOW();

-- name: EnsurePortfolioStats :exec
INSERT INTO portfolio_stats (portfolio_id) VALUES ($1) ON CONFLICT (portfolio_id) DO NOTHING;

-- name: SetPortfolioAssessmentAvg :exec
INSERT INTO portfolio_stats (portfolio_id, assessment_avg) VALUES ($1, $2)
ON CONFLICT (portfolio_id) DO UPDATE SET assessment_avg = EXCLUDED.assessment_avg, computed_at = NOW();

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
-- Hanya snapshot TERBARU untuk kombinasi period/scope/kind, supaya entri
-- hari sebelumnya tidak tercampur saat jumlah baris < limit.
SELECT e.rank, e.subject_user_id, e.subject_portfolio_id, e.score, e.breakdown, e.delta_rank
FROM leaderboard_entries e
JOIN leaderboards l ON l.id = e.leaderboard_id
WHERE l.id = (
  SELECT l2.id FROM leaderboards l2
  WHERE l2.period = $1 AND l2.scope = $2 AND l2.kind = $3
  ORDER BY l2.computed_for DESC LIMIT 1
)
ORDER BY e.rank ASC LIMIT 20;

-- name: InsertAuditLog :exec
INSERT INTO audit_logs (actor_id, action, entity, entity_id, diff, ip)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: StudentLeaderboardCandidates :many
-- $1 = jumlah hari periode (weekly 7, monthly 30).
-- Anti-gaming: like dari akun berumur <7 hari tidak dihitung.
SELECT u.id, u.username, u.name, COALESCE(m.code, '') AS major_code,
  COUNT(DISTINCT p.id) FILTER (WHERE p.published_at > NOW() - ($1::text || ' days')::interval) AS portfolio_count,
  COUNT(DISTINCT pl.user_id) FILTER (
    WHERE lu.created_at < NOW() - INTERVAL '7 days'
      AND pl.created_at > NOW() - ($1::text || ' days')::interval
  ) AS total_likes,
  COUNT(DISTINCT f.follower_id) FILTER (WHERE f.created_at > NOW() - ($1::text || ' days')::interval) AS followers,
  COALESCE((
    SELECT AVG(pa.total_score) FROM portfolio_assessments pa
    JOIN portfolios p2 ON p2.id = pa.portfolio_id
    WHERE p2.owner_id = u.id AND p2.status = 'published' AND p2.deleted_at IS NULL
      AND pa.updated_at > NOW() - ($1::text || ' days')::interval
  ), 0)::float8 AS assessment_avg
FROM users u
LEFT JOIN majors m ON m.id = u.major_id
LEFT JOIN portfolios p ON p.owner_id = u.id AND p.status = 'published' AND p.deleted_at IS NULL
LEFT JOIN portfolio_likes pl ON pl.portfolio_id = p.id
LEFT JOIN users lu ON lu.id = pl.user_id
LEFT JOIN follows f ON f.following_id = u.id
WHERE u.role_primary = 'siswa' AND u.deleted_at IS NULL
GROUP BY u.id, m.code
HAVING COUNT(DISTINCT p.id) FILTER (WHERE p.published_at > NOW() - ($1::text || ' days')::interval) >= 1;

-- name: ProjectLeaderboardCandidates :many
-- $1 = jumlah hari periode.
SELECT p.id, p.title, p.slug, p.owner_id, p.published_at, COALESCE(m.code, '') AS owner_major_code,
  COUNT(DISTINCT pl.user_id) FILTER (WHERE pl.created_at > NOW() - ($1::text || ' days')::interval) AS likes,
  COUNT(DISTINCT c.id) FILTER (WHERE c.created_at > NOW() - ($1::text || ' days')::interval) AS comments,
  COALESCE((SELECT COUNT(*) FROM portfolio_assessments pa WHERE pa.portfolio_id = p.id
    AND pa.updated_at > NOW() - ($1::text || ' days')::interval), 0)::int8 AS assessment_count,
  COALESCE((SELECT AVG(total_score) FROM portfolio_assessments pa WHERE pa.portfolio_id = p.id
    AND pa.updated_at > NOW() - ($1::text || ' days')::interval), 0)::float8 AS assessment_avg
FROM portfolios p
LEFT JOIN users u ON u.id = p.owner_id
LEFT JOIN majors m ON m.id = u.major_id
LEFT JOIN portfolio_likes pl ON pl.portfolio_id = p.id
LEFT JOIN comments c ON c.portfolio_id = p.id AND c.deleted_at IS NULL
WHERE p.status = 'published' AND p.deleted_at IS NULL
GROUP BY p.id, m.code;

-- name: SuggestTags :many
SELECT slug, name FROM tags WHERE similarity(name, $1) > 0.2 ORDER BY similarity(name, $1) DESC LIMIT 5;

-- name: CountFollowers :one
SELECT COUNT(*) FROM follows WHERE following_id = $1;

-- name: CountFollowing :one
SELECT COUNT(*) FROM follows WHERE follower_id = $1;

-- name: CountUserPublished :one
SELECT COUNT(*) FROM portfolios WHERE owner_id = $1 AND status = 'published' AND deleted_at IS NULL;

-- name: ListFollowers :many
SELECT u.id, u.username, u.name FROM users u
JOIN follows f ON f.follower_id = u.id WHERE f.following_id = $1 AND u.deleted_at IS NULL
ORDER BY f.created_at DESC LIMIT $2 OFFSET $3;

-- name: ListFollowing :many
SELECT u.id, u.username, u.name FROM users u
JOIN follows f ON f.following_id = u.id WHERE f.follower_id = $1 AND u.deleted_at IS NULL
ORDER BY f.created_at DESC LIMIT $2 OFFSET $3;

-- name: CountViewsSince :one
-- Dibaca dari agregat harian hasil flush HyperLogLog (bukan tabel mentah).
SELECT COALESCE(SUM(unique_views), 0)::bigint FROM portfolio_view_counts
WHERE portfolio_id = $1 AND viewed_date > CURRENT_DATE - $2;

-- name: UpsertViewCount :exec
INSERT INTO portfolio_view_counts (portfolio_id, viewed_date, unique_views)
VALUES ($1, $2, $3)
ON CONFLICT (portfolio_id, viewed_date) DO UPDATE
SET unique_views = GREATEST(portfolio_view_counts.unique_views, EXCLUDED.unique_views),
    updated_at = NOW();

-- name: DeleteOldViewCounts :exec
DELETE FROM portfolio_view_counts WHERE viewed_date < CURRENT_DATE - 400;

-- name: CountLikesSince :one
SELECT COUNT(*) FROM portfolio_likes WHERE portfolio_id = $1 AND created_at > NOW() - ($2::text || ' days')::interval;

-- name: CountCommentsSince :one
SELECT COUNT(*) FROM comments WHERE portfolio_id = $1 AND deleted_at IS NULL AND created_at > NOW() - ($2::text || ' days')::interval;

-- name: CountUserLikesSince :one
SELECT COUNT(*) FROM portfolio_likes pl JOIN portfolios p ON p.id = pl.portfolio_id
WHERE p.owner_id = $1 AND pl.created_at > NOW() - ($2 || ' days')::interval;

-- name: CountUserPortfoliosSince :one
SELECT COUNT(*) FROM portfolios WHERE owner_id = $1 AND status = 'published' AND deleted_at IS NULL AND published_at > NOW() - ($2::text || ' days')::interval;

-- name: TopLikedUnassessed :many
SELECT p.id FROM portfolios p
LEFT JOIN portfolio_likes pl ON pl.portfolio_id = p.id
WHERE p.status = 'published' AND p.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM portfolio_assessments pa WHERE pa.portfolio_id = p.id)
GROUP BY p.id ORDER BY COUNT(pl.user_id) DESC LIMIT $1;

-- name: CountLikesByUserSince :one
SELECT COUNT(*) FROM portfolio_likes WHERE user_id = $1 AND created_at > NOW() - INTERVAL '1 hour';

-- name: PrevLeaderboardRanks :many
-- Snapshot sebelumnya (bukan sembarang 40 baris) untuk menghitung delta_rank.
SELECT e.subject_user_id, e.subject_portfolio_id, e.rank FROM leaderboard_entries e
JOIN leaderboards l ON l.id = e.leaderboard_id
WHERE l.id = (
  SELECT l2.id FROM leaderboards l2
  WHERE l2.period = $1 AND l2.scope = $2 AND l2.kind = $3
  ORDER BY l2.computed_for DESC LIMIT 1
)
ORDER BY e.rank ASC;

-- name: StatsPortfolioCandidates :many
-- Semua metrik statistik dalam SATU query (menghilangkan 4 query per kandidat).
-- $1 = only_changed: bila true, hanya portfolio yang aktivitasnya lebih baru
-- dari stats terakhir (mode inkremental oleh worker).
WITH activity AS (
  SELECT p.id,
    (SELECT COUNT(DISTINCT pl.user_id) FROM portfolio_likes pl WHERE pl.portfolio_id = p.id) AS total_likes,
    (SELECT COUNT(*) FROM comments c WHERE c.portfolio_id = p.id AND c.deleted_at IS NULL) AS total_comments,
    (SELECT COUNT(*) FROM portfolio_likes pl WHERE pl.portfolio_id = p.id AND pl.created_at > NOW() - INTERVAL '7 days') AS likes_7d,
    (SELECT COUNT(*) FROM comments c WHERE c.portfolio_id = p.id AND c.deleted_at IS NULL AND c.created_at > NOW() - INTERVAL '7 days') AS comments_7d,
    (SELECT COALESCE(SUM(v.unique_views), 0) FROM portfolio_view_counts v
      WHERE v.portfolio_id = p.id AND v.viewed_date > CURRENT_DATE - 7) AS views_7d,
    (SELECT COALESCE(SUM(v.unique_views), 0) FROM portfolio_view_counts v
      WHERE v.portfolio_id = p.id AND v.viewed_date > CURRENT_DATE - 30) AS views_30d,
    (SELECT COALESCE(AVG(pa.total_score), 0) FROM portfolio_assessments pa WHERE pa.portfolio_id = p.id)::float8 AS assessment_avg,
    (p.thumbnail_media_id IS NOT NULL)::bool AS has_thumbnail,
    p.published_at,
    GREATEST(
      COALESCE((SELECT MAX(pl.created_at) FROM portfolio_likes pl WHERE pl.portfolio_id = p.id), to_timestamp(0)),
      COALESCE((SELECT MAX(c.created_at) FROM comments c WHERE c.portfolio_id = p.id), to_timestamp(0)),
      COALESCE((SELECT MAX(v.updated_at) FROM portfolio_view_counts v WHERE v.portfolio_id = p.id), to_timestamp(0)),
      COALESCE((SELECT MAX(pa.updated_at) FROM portfolio_assessments pa WHERE pa.portfolio_id = p.id), to_timestamp(0))
    ) AS last_activity,
    COALESCE(ps.computed_at, to_timestamp(0)) AS stats_computed_at
  FROM portfolios p
  LEFT JOIN portfolio_stats ps ON ps.portfolio_id = p.id
  WHERE p.status = 'published' AND p.deleted_at IS NULL
)
SELECT id, published_at, total_likes, total_comments, likes_7d, comments_7d,
  views_7d, views_30d, assessment_avg, has_thumbnail
FROM activity
WHERE NOT $1::bool OR stats_computed_at < last_activity
LIMIT $2;

-- name: StatsNormalization :one
-- Nilai maksimum lintas semua portfolio published, dipakai sebagai penyebut
-- normalisasi (tidak boleh dihitung dari subset inkremental).
SELECT
  GREATEST(COALESCE(MAX(total_likes), 1), 1)::float8 AS max_likes,
  GREATEST(COALESCE(MAX(views_7d), 1), 1)::float8 AS max_views_7d,
  GREATEST(COALESCE(MAX(total_comments), 1), 1)::float8 AS max_comments
FROM (
  SELECT p.id,
    (SELECT COUNT(DISTINCT pl.user_id) FROM portfolio_likes pl WHERE pl.portfolio_id = p.id) AS total_likes,
    (SELECT COUNT(*) FROM comments c WHERE c.portfolio_id = p.id AND c.deleted_at IS NULL) AS total_comments,
    (SELECT COALESCE(SUM(v.unique_views), 0) FROM portfolio_view_counts v
      WHERE v.portfolio_id = p.id AND v.viewed_date > CURRENT_DATE - 7) AS views_7d
  FROM portfolios p
  WHERE p.status = 'published' AND p.deleted_at IS NULL
) t;

-- name: ListModeratorIDs :many
SELECT id FROM users
WHERE deleted_at IS NULL AND is_active = TRUE
  AND (role_primary = 'admin' OR (role_primary = 'admin_jurusan' AND ($1::text = '' OR major_id::text = $1)));

-- name: ListAudienceUserIDs :many
SELECT id FROM users
WHERE deleted_at IS NULL AND is_active = TRUE
  AND (
    $1::text = 'all'
    OR ($1::text = 'siswa' AND role_primary = 'siswa')
    OR ($1::text = 'guru' AND role_primary = 'guru')
    OR ($1::text = 'admin' AND role_primary IN ('admin', 'admin_jurusan'))
  );

-- name: FollowMap :many
-- Relasi follow dua arah sekaligus untuk semua owner kandidat (hindari N+1 IsFollowing).
SELECT u.id AS owner_id,
  EXISTS(SELECT 1 FROM follows f WHERE f.follower_id = $1 AND f.following_id = u.id) AS i_follow,
  EXISTS(SELECT 1 FROM follows f WHERE f.follower_id = u.id AND f.following_id = $1) AS follows_me
FROM users u WHERE u.id = ANY($2::uuid[]);
