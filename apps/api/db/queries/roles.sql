-- name: ListPermissions :many
SELECT key, description, scopable
FROM permissions
ORDER BY key;

-- name: ListRoles :many
SELECT
  r.id,
  r.slug::text AS slug,
  r.name,
  r.description,
  r.is_system,
  (
    SELECT count(*)
    FROM user_roles ur
    WHERE ur.role_id = r.id
      AND ur.revoked_at IS NULL
      AND (ur.expires_at IS NULL OR ur.expires_at > now())
  )::bigint AS holder_count
FROM roles r
ORDER BY r.slug;

-- name: GetRoleByID :one
SELECT
  r.id,
  r.slug::text AS slug,
  r.name,
  r.description,
  r.is_system,
  (
    SELECT count(*)
    FROM user_roles ur
    WHERE ur.role_id = r.id
      AND ur.revoked_at IS NULL
      AND (ur.expires_at IS NULL OR ur.expires_at > now())
  )::bigint AS holder_count
FROM roles r
WHERE r.id = @id::uuid;

-- name: GetRoleBySlug :one
SELECT
  r.id,
  r.slug::text AS slug,
  r.name,
  r.description,
  r.is_system,
  (
    SELECT count(*)
    FROM user_roles ur
    WHERE ur.role_id = r.id
      AND ur.revoked_at IS NULL
      AND (ur.expires_at IS NULL OR ur.expires_at > now())
  )::bigint AS holder_count
FROM roles r
WHERE r.slug = @slug::citext;

-- name: ListAllRolePermissions :many
SELECT role_id, permission_key
FROM role_permissions
ORDER BY role_id, permission_key;

-- name: ListRolePermissions :many
SELECT p.key, p.scopable
FROM role_permissions rp
JOIN permissions p ON p.key = rp.permission_key
WHERE rp.role_id = @role_id::uuid
ORDER BY p.key;

-- name: InsertRole :one
INSERT INTO roles (slug, name, description, is_system, created_by)
VALUES (
  @slug::citext,
  @name::text,
  sqlc.narg('description')::text,
  false,
  @created_by::uuid
)
RETURNING id;

-- name: UpdateRole :one
UPDATE roles
SET name        = COALESCE(sqlc.narg('name')::text, name),
    description = CASE
                    WHEN @description_set::boolean THEN sqlc.narg('description')::text
                    ELSE description
                  END
WHERE id = @id::uuid
RETURNING id;

-- name: DeleteRole :exec
DELETE FROM roles
WHERE id = @id::uuid;

-- name: DeleteInactiveRoleAssignments :exec
DELETE FROM user_roles
WHERE role_id = @role_id::uuid
  AND (revoked_at IS NOT NULL OR expires_at <= now());

-- name: DeleteRolePermissions :exec
DELETE FROM role_permissions
WHERE role_id = @role_id::uuid;

-- name: InsertRolePermission :exec
INSERT INTO role_permissions (role_id, permission_key)
VALUES (@role_id::uuid, @permission_key::text);

-- name: GetRoleAssignmentTarget :one
SELECT id, status
FROM users
WHERE id = @id::uuid;

-- name: GetRoleMajor :one
SELECT id
FROM majors
WHERE id = @id::uuid;

-- name: InsertUserRole :one
INSERT INTO user_roles (user_id, role_id, major_id, assigned_by, expires_at)
VALUES (
  @user_id::uuid,
  @role_id::uuid,
  sqlc.narg('major_id')::uuid,
  @assigned_by::uuid,
  sqlc.narg('expires_at')::timestamptz
)
RETURNING id, assigned_at;

-- name: GetUserRoleAssignment :one
SELECT
  ur.id,
  ur.user_id,
  ur.role_id,
  r.slug::text AS role_slug,
  r.name AS role_name,
  ur.major_id,
  ur.assigned_by,
  ur.assigned_at,
  ur.expires_at
FROM user_roles ur
JOIN roles r ON r.id = ur.role_id
WHERE ur.id = @id::uuid
  AND ur.revoked_at IS NULL;

-- name: ListActiveUserRoleAssignments :many
SELECT
  ur.id,
  ur.user_id,
  ur.role_id,
  r.slug::text AS role_slug,
  r.name AS role_name,
  ur.major_id,
  ur.assigned_by,
  ur.assigned_at,
  ur.expires_at
FROM user_roles ur
JOIN roles r ON r.id = ur.role_id
WHERE ur.user_id = @user_id::uuid
  AND ur.revoked_at IS NULL
  AND (ur.expires_at IS NULL OR ur.expires_at > now())
  AND (
    @scope_global::boolean
    OR ur.major_id = ANY(@scope_major_ids::uuid[])
  )
ORDER BY ur.assigned_at, ur.id;

-- name: RevokeUserRole :one
UPDATE user_roles
SET revoked_at = now(),
    revoked_by = @revoked_by::uuid
WHERE id = @id::uuid
  AND revoked_at IS NULL
RETURNING user_id;

-- name: CountActiveRoleHolders :one
SELECT count(*)::bigint
FROM user_roles
WHERE role_id = @role_id::uuid
  AND revoked_at IS NULL
  AND (expires_at IS NULL OR expires_at > now());

-- name: ListActiveRoleHolderIDs :many
SELECT DISTINCT user_id
FROM user_roles
WHERE role_id = @role_id::uuid
  AND revoked_at IS NULL
  AND (expires_at IS NULL OR expires_at > now())
ORDER BY user_id;
