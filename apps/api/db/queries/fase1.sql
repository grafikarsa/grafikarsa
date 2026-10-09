-- name: GetUserByUsername :one
SELECT id, username, email, password_hash, name, role_primary, status,
  class_id, major_id, is_active, must_change_password, avatar_media_id, banner_media_id
FROM users WHERE username = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetUserByID :one
SELECT id, username, email, password_hash, name, role_primary, status,
  class_id, major_id, is_active, must_change_password, avatar_media_id, banner_media_id
FROM users WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: UpdatePasswordHash :exec
UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1;

-- name: TouchLastLogin :exec
UPDATE users SET last_login_at = NOW() WHERE id = $1;

-- name: CreateSession :one
INSERT INTO sessions (user_id, refresh_hash, family_id, device_label, ip, user_agent, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id;

-- name: GetSessionByHash :one
SELECT id, user_id, refresh_hash, family_id, device_label, ip, user_agent, expires_at, revoked_at, rotated_to
FROM sessions WHERE refresh_hash = $1 LIMIT 1;

-- name: RevokeSession :exec
UPDATE sessions SET revoked_at = NOW() WHERE id = $1;

-- name: RevokeFamily :exec
UPDATE sessions SET revoked_at = NOW() WHERE family_id = $1 AND revoked_at IS NULL;

-- name: RevokeAllUserSessions :exec
UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL;

-- name: ListUserSessions :many
SELECT id, user_id, refresh_hash, family_id, device_label, ip, user_agent, expires_at, revoked_at, rotated_to, created_at
FROM sessions WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50;

-- name: InsertAuthAudit :exec
INSERT INTO auth_audit (user_id, event, ip, user_agent) VALUES ($1, $2, $3, $4);

-- name: GetUserPermissions :many
SELECT DISTINCT p.key FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = $1;

-- name: CreateMedia :one
INSERT INTO media_assets (owner_id, purpose, object_key, mime, size_bytes, width, height, checksum_sha256, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id;

-- name: GetMediaByID :one
SELECT id, owner_id, purpose, object_key, mime, size_bytes, width, height, checksum_sha256, status, created_at, confirmed_at
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
