-- name: GetUserForLogin :one
SELECT
  id,
  account_type,
  status,
  nisn,
  staff_number,
  username,
  display_name,
  password_hash,
  password_kind,
  password_expires_at,
  activated_at,
  recovery_codes_confirmed_at,
  content_blocked_at,
  legacy_v1_id,
  deactivated_at
FROM users
WHERE nisn = @identifier::text
   OR staff_number = @identifier::text
   OR username = @identifier::citext
LIMIT 1;

-- name: GetUserByID :one
SELECT
  id,
  account_type,
  status,
  nisn,
  staff_number,
  username,
  display_name,
  password_hash,
  password_kind,
  password_expires_at,
  activated_at,
  recovery_codes_confirmed_at,
  content_blocked_at,
  legacy_v1_id,
  deactivated_at
FROM users
WHERE id = @id::uuid;

-- name: UpdatePasswordHash :exec
UPDATE users
SET password_hash = @password_hash::text
WHERE id = @id::uuid;

-- name: SetLastLogin :exec
UPDATE users
SET last_login_at = now()
WHERE id = @id::uuid;

-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, ip, user_agent, idle_expires_at, expires_at)
VALUES (
  @user_id::uuid,
  @token_hash::bytea,
  @ip::inet,
  @user_agent::text,
  @idle_expires_at::timestamptz,
  @expires_at::timestamptz
)
RETURNING
  id,
  user_id,
  token_hash,
  ip,
  user_agent,
  created_at,
  last_seen_at,
  idle_expires_at,
  expires_at,
  revoked_at,
  revoked_reason;

-- name: GetSessionByTokenHash :one
SELECT
  id,
  user_id,
  token_hash,
  ip,
  user_agent,
  created_at,
  last_seen_at,
  idle_expires_at,
  expires_at,
  revoked_at,
  revoked_reason
FROM sessions
WHERE token_hash = @token_hash::bytea
  AND revoked_at IS NULL;

-- name: TouchSession :exec
UPDATE sessions
SET last_seen_at = now(),
    idle_expires_at = @idle_expires_at::timestamptz
WHERE id = @id::uuid
  AND revoked_at IS NULL;

-- name: RevokeSession :exec
UPDATE sessions
SET revoked_at = now(),
    revoked_reason = @reason::text
WHERE id = @id::uuid
  AND revoked_at IS NULL;

-- name: RevokeUserSessions :many
UPDATE sessions
SET revoked_at = now(),
    revoked_reason = @reason::text
WHERE user_id = @user_id::uuid
  AND revoked_at IS NULL
  AND id <> @except_id::uuid
RETURNING token_hash;

-- name: ListActiveSessions :many
SELECT
  id,
  user_id,
  token_hash,
  ip,
  user_agent,
  created_at,
  last_seen_at,
  idle_expires_at,
  expires_at
FROM sessions
WHERE user_id = @user_id::uuid
  AND revoked_at IS NULL
  AND expires_at > now()
  AND idle_expires_at > now()
ORDER BY last_seen_at DESC;

-- name: ListExplicitGrants :many
SELECT rp.permission_key, ur.major_id
FROM user_roles ur
JOIN role_permissions rp ON rp.role_id = ur.role_id
WHERE ur.user_id = @user_id::uuid
  AND ur.revoked_at IS NULL
  AND (ur.expires_at IS NULL OR ur.expires_at > now());

-- name: ListRolePermissionsBySlug :many
SELECT rp.permission_key
FROM role_permissions rp
JOIN roles r ON r.id = rp.role_id
WHERE r.slug = @slug::citext;
