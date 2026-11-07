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

-- name: IsUsernameReserved :one
SELECT EXISTS (
  SELECT 1 FROM reserved_usernames WHERE username = @username::citext
) AS reserved;

-- name: IsUsernameTaken :one
SELECT EXISTS (
  SELECT 1
  FROM users
  WHERE username = @username::citext
    AND id <> @user_id::uuid
  UNION ALL
  SELECT 1
  FROM username_history
  WHERE old_username = @username::citext
    AND released_at IS NULL
    AND redirect_until > now()
    AND user_id <> @user_id::uuid
) AS taken;

-- name: SetUsername :exec
UPDATE users
SET username = @username::citext,
    username_changed_at = now()
WHERE id = @id::uuid;

-- name: InsertUsernameHistory :exec
INSERT INTO username_history (user_id, old_username, redirect_until)
VALUES (@user_id::uuid, @old_username::citext, @redirect_until::timestamptz);

-- name: SetOwnPassword :exec
UPDATE users
SET password_hash = @password_hash::text,
    password_kind = 'user',
    password_expires_at = NULL,
    password_changed_at = now()
WHERE id = @id::uuid;

-- name: RevokeActiveRecoveryCodes :exec
UPDATE recovery_codes
SET revoked_at = now()
WHERE user_id = @user_id::uuid
  AND used_at IS NULL
  AND revoked_at IS NULL;

-- name: InsertRecoveryCode :exec
INSERT INTO recovery_codes (user_id, batch_id, code_hash)
VALUES (@user_id::uuid, @batch_id::uuid, @code_hash::text);

-- name: CountActiveRecoveryCodes :one
SELECT count(*)
FROM recovery_codes
WHERE user_id = @user_id::uuid
  AND used_at IS NULL
  AND revoked_at IS NULL;

-- name: UseRecoveryCode :one
UPDATE recovery_codes
SET used_at = now()
WHERE user_id = @user_id::uuid
  AND code_hash = @code_hash::text
  AND used_at IS NULL
  AND revoked_at IS NULL
RETURNING id;

-- name: CompleteActivation :exec
UPDATE users
SET recovery_codes_confirmed_at = now(),
    activated_at = now()
WHERE id = @id::uuid
  AND username IS NOT NULL;

-- name: GetUserIdentity :one
SELECT
  username,
  nisn,
  staff_number,
  display_name,
  password_hash,
  password_kind,
  activated_at,
  status,
  deactivated_at,
  legacy_v1_id
FROM users
WHERE id = @id::uuid;

-- name: GetRoleIDBySlug :one
SELECT id FROM roles WHERE slug = @slug::citext;

-- name: InsertSystemUserRole :exec
INSERT INTO user_roles (user_id, role_id, major_id, assigned_by)
VALUES (@user_id::uuid, @role_id::uuid, NULL, NULL);

-- name: CreateStaffUser :one
INSERT INTO users (
  account_type,
  display_name,
  username,
  staff_number,
  password_hash,
  password_kind,
  password_expires_at
) VALUES (
  'staff',
  @display_name::text,
  @username::citext,
  sqlc.narg('staff_number')::text,
  @password_hash::text,
  'initial',
  @password_expires_at::timestamptz
)
RETURNING id;
