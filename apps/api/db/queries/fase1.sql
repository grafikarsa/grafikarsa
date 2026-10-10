-- name: GetUserByUsername :one
SELECT id, username, email, password_hash, name, bio, role_primary, status,
  class_id, major_id, is_active, must_change_password, show_email, avatar_media_id, banner_media_id
FROM users WHERE username = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetUserByID :one
SELECT id, username, email, password_hash, name, bio, role_primary, status,
  class_id, major_id, is_active, must_change_password, show_email, avatar_media_id, banner_media_id
FROM users WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: UpdatePasswordHash :exec
UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1;

-- name: TouchLastLogin :exec
UPDATE users SET last_login_at = NOW() WHERE id = $1;

-- name: CreateSession :one
INSERT INTO sessions (user_id, refresh_hash, family_id, device_label, ip, user_agent, expires_at, absolute_expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id;

-- name: GetSessionByHash :one
SELECT id, user_id, refresh_hash, family_id, device_label, ip, user_agent, expires_at, absolute_expires_at, revoked_at, rotated_to
FROM sessions WHERE refresh_hash = $1 LIMIT 1;

-- name: RevokeSession :exec
UPDATE sessions SET revoked_at = NOW() WHERE id = $1;

-- name: RotateSession :exec
UPDATE sessions SET revoked_at = NOW(), rotated_to = $2 WHERE id = $1;

-- name: RevokeFamily :exec
UPDATE sessions SET revoked_at = NOW() WHERE family_id = $1 AND revoked_at IS NULL;

-- name: RevokeAllUserSessions :exec
UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL;

-- name: ListUserSessions :many
SELECT id, user_id, refresh_hash, family_id, device_label, ip, user_agent, expires_at, absolute_expires_at, revoked_at, rotated_to, created_at
FROM sessions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50;

-- name: InsertAuthAudit :exec
INSERT INTO auth_audit (user_id, event, ip, user_agent) VALUES ($1, $2, $3, $4);

-- name: GetUserPermissions :many
SELECT DISTINCT p.key FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = $1;

-- name: CreateMedia :one
INSERT INTO media_assets (owner_id, purpose, object_key, mime, size_bytes, width, height, checksum_sha256, status, variants)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id;

-- name: GetMediaByID :one
SELECT id, owner_id, purpose, object_key, mime, size_bytes, width, height, checksum_sha256, status, variants, created_at, confirmed_at
FROM media_assets WHERE id = $1 LIMIT 1;

-- name: ConfirmMedia :exec
UPDATE media_assets SET status = 'confirmed', confirmed_at = NOW() WHERE id = $1;

-- name: UpdateUserAvatar :exec
UPDATE users SET avatar_media_id = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdateUserBanner :exec
UPDATE users SET banner_media_id = $2, updated_at = NOW() WHERE id = $1;

-- name: ListUsers :many
SELECT id, username, email, name, role_primary, status, class_id, major_id, is_active, created_at
FROM users WHERE deleted_at IS NULL
ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: CountUsersAll :one
SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;

-- name: ListRoles :many
SELECT id, slug, name, scope_type, is_system FROM roles ORDER BY name ASC;

-- name: MarkMediaDeleted :exec
UPDATE media_assets SET status = 'deleted' WHERE id = $1;

-- name: CreateUser :one
INSERT INTO users (username, email, password_hash, name, role_primary, status, major_id, class_id, must_change_password)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id;

-- name: UpdateUser :exec
UPDATE users SET name = $2, email = $3, major_id = $4, class_id = $5, updated_at = NOW() WHERE id = $1;

-- name: SoftDeleteUser :exec
UPDATE users SET deleted_at = NOW() WHERE id = $1;

-- name: SetUserActive :exec
UPDATE users SET is_active = $2, updated_at = NOW() WHERE id = $1;

-- name: SetMustChangePassword :exec
UPDATE users SET must_change_password = $2 WHERE id = $1;

-- name: CreateRole :one
INSERT INTO roles (slug, name, scope_type) VALUES ($1, $2, $3) RETURNING id;

-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1 AND is_system = FALSE;

-- name: AddRolePermission :exec
INSERT INTO role_permissions (role_id, permission_id)
SELECT $1, id FROM permissions WHERE key = $2 ON CONFLICT DO NOTHING;

-- name: AssignUserRole :exec
INSERT INTO user_roles (user_id, role_id, scope_major_id, assigned_by, expires_at)
VALUES ($1, $2, $3, $4, $5) ON CONFLICT (user_id, role_id) DO UPDATE SET scope_major_id = EXCLUDED.scope_major_id, expires_at = EXCLUDED.expires_at;

-- name: UnassignUserRole :exec
DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2;

-- name: GetRoleBySlug :one
SELECT id, slug, name, scope_type, is_system FROM roles WHERE slug = $1 LIMIT 1;

-- name: UpsertUserInterest :exec
INSERT INTO user_interests (user_id, tag_weights, major_weights, total_likes)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE SET tag_weights = EXCLUDED.tag_weights, major_weights = EXCLUDED.major_weights, total_likes = EXCLUDED.total_likes, updated_at = NOW();

-- name: SumUserMedia :one
SELECT COALESCE(SUM(size_bytes), 0) FROM media_assets WHERE owner_id = $1 AND status = 'confirmed';

-- name: CountUploadsSince :one
SELECT COUNT(*) FROM media_assets WHERE owner_id = $1 AND created_at > NOW() - ($2 || ' hours')::interval;

-- name: ListPendingMedia :many
-- Hanya media pending yang TIDAK direferensikan siapa pun (aman dihapus).
SELECT m.id, m.object_key FROM media_assets m
WHERE m.status = 'pending' AND m.created_at < NOW() - INTERVAL '24 hours'
  AND NOT EXISTS (SELECT 1 FROM users u WHERE u.avatar_media_id = m.id OR u.banner_media_id = m.id)
  AND NOT EXISTS (SELECT 1 FROM portfolios p WHERE p.thumbnail_media_id = m.id)
  AND NOT EXISTS (SELECT 1 FROM portfolio_blocks b WHERE b.asset_media_id = m.id)
LIMIT 100;

-- name: FindMediaByChecksum :one
SELECT id, object_key, mime, width, height, variants FROM media_assets
WHERE owner_id = $1 AND purpose = $2 AND checksum_sha256 = $3 AND status IN ('pending', 'confirmed')
LIMIT 1;

-- name: SetBlockAsset :exec
UPDATE portfolio_blocks SET asset_media_id = $2 WHERE id = $1;

-- name: ClearUserAvatarIfMedia :exec
UPDATE users SET avatar_media_id = NULL WHERE avatar_media_id = $1;

-- name: DeleteMediaRow :exec
DELETE FROM media_assets WHERE id = $1;

-- name: UpdateMe :exec
UPDATE users SET name = $2, bio = $3, username = $4, show_email = $5, updated_at = NOW() WHERE id = $1;

-- name: CheckUsernameTaken :one
SELECT COUNT(*) FROM users WHERE username = $1 AND id != $2 AND deleted_at IS NULL;

-- name: GetUserMajor :one
SELECT major_id FROM users WHERE id = $1 LIMIT 1;

-- name: GetMajorRequireSeries :one
SELECT require_series FROM majors WHERE id = $1 LIMIT 1;

-- name: ListSocialLinks :many
SELECT id, platform, url FROM user_social_links WHERE user_id = $1 ORDER BY platform;



-- name: DeleteSocialLinks :exec
DELETE FROM user_social_links WHERE user_id = $1;

-- name: AddSocialLink :exec
INSERT INTO user_social_links (user_id, platform, url) VALUES ($1, $2, $3) ON CONFLICT (user_id, platform) DO UPDATE SET url = EXCLUDED.url;

-- name: GetNotifPrefs :one
SELECT user_id, portfolio_liked, new_comment, new_follower, digest_daily FROM notification_preferences WHERE user_id = $1 LIMIT 1;

-- name: SaveNotifPrefs :exec
INSERT INTO notification_preferences (user_id, portfolio_liked, new_comment, new_follower, digest_daily)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id) DO UPDATE SET portfolio_liked = EXCLUDED.portfolio_liked, new_comment = EXCLUDED.new_comment, new_follower = EXCLUDED.new_follower, digest_daily = EXCLUDED.digest_daily;

-- name: GetAuditLogs :many
SELECT id, actor_id, action, entity, entity_id, created_at FROM audit_logs ORDER BY id DESC LIMIT $1 OFFSET $2;


-- name: GetUserScope :one
SELECT COALESCE(
  (SELECT ur.scope_major_id FROM user_roles ur
   JOIN roles r ON r.id = ur.role_id
   WHERE ur.user_id = u.id AND ur.scope_major_id IS NOT NULL
     AND (ur.expires_at IS NULL OR ur.expires_at > NOW())
   LIMIT 1),
  CASE WHEN u.role_primary = 'admin_jurusan' THEN u.major_id ELSE NULL END
)::uuid AS scope_major_id
FROM users u WHERE u.id = $1;

-- name: ListUsersScoped :many
SELECT id, username, email, name, role_primary, status, class_id, major_id, is_active, created_at
FROM users WHERE deleted_at IS NULL
  AND ($1 = '' OR major_id::text = $1)
ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: CountUsersScoped :one
SELECT COUNT(*) FROM users WHERE deleted_at IS NULL
  AND ($1 = '' OR major_id::text = $1);

-- name: MediaUsedInPublishedPortfolio :one
SELECT (
  EXISTS(SELECT 1 FROM portfolios p WHERE p.thumbnail_media_id = $1
    AND p.status = 'published' AND p.deleted_at IS NULL)
  OR EXISTS(SELECT 1 FROM portfolio_blocks b JOIN portfolios p ON p.id = b.portfolio_id
    WHERE b.asset_media_id = $1 AND p.status = 'published' AND p.deleted_at IS NULL)
)::bool AS used;

-- name: UpdateFeedbackPriority :exec
UPDATE feedback_threads SET priority = $2, assignee_id = $3 WHERE id = $1;

-- name: RefreshFeedbackUpvotes :exec
UPDATE feedback_threads SET upvotes_cached = (SELECT COUNT(*) FROM feedback_votes v WHERE v.thread_id = $1) WHERE id = $1;

-- name: SetFeedbackPublic :exec
UPDATE feedback_threads SET is_public = $2 WHERE id = $1;

-- name: SetFeedbackSpam :exec
UPDATE feedback_threads SET is_spam = $2, status = CASE WHEN $2 THEN 'closed' ELSE status END WHERE id = $1;

-- name: UpdateTag :exec
UPDATE tags SET name = $2, slug = COALESCE(NULLIF($3, ''), slug) WHERE id = $1;

-- name: DeleteTagIfUnused :one
DELETE FROM tags t WHERE t.id = $1 AND NOT EXISTS (SELECT 1 FROM portfolio_tags pt WHERE pt.tag_id = t.id)
RETURNING t.id;

-- name: GetUserMajorAndActive :one
SELECT major_id, is_active, deleted_at FROM users WHERE id = $1 LIMIT 1;

-- name: ReapStuckOutbox :exec
UPDATE outbox SET status = 'pending', next_retry_at = NOW()
WHERE status = 'processing' AND next_retry_at <= NOW();

-- name: GetRolePermissions :many
SELECT p.key FROM role_permissions rp JOIN permissions p ON p.id = rp.permission_id WHERE rp.role_id = $1;

-- name: GetMediaByKey :one
SELECT id, owner_id, purpose, object_key, mime, size_bytes, width, height, checksum_sha256, status, variants
FROM media_assets WHERE object_key = $1 AND status <> 'deleted' LIMIT 1;

-- name: GetLegacyURL :one
SELECT id, old_url, media_id, object_key FROM legacy_urls WHERE old_url = $1 LIMIT 1;

-- name: TouchLegacyURL :exec
UPDATE legacy_urls SET hit_count = hit_count + 1, last_hit_at = NOW() WHERE id = $1;

-- name: CreateLegacyURL :exec
INSERT INTO legacy_urls (old_url, media_id, object_key) VALUES ($1, $2, $3)
ON CONFLICT (old_url) DO UPDATE SET media_id = EXCLUDED.media_id, object_key = EXCLUDED.object_key;
