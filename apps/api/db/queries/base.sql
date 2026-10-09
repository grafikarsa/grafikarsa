-- name: Ping :one
SELECT 1 AS ok;

-- name: ListMajors :many
SELECT id, code, name FROM majors ORDER BY name ASC;

-- name: CountUsers :one
SELECT COUNT(*) FROM users WHERE deleted_at IS NULL;

-- name: GetUserByEmail :one
SELECT id, username, email, password_hash, name, role_primary, status, is_active
FROM users WHERE email = $1 AND deleted_at IS NULL LIMIT 1;

-- name: CountMajors :one
SELECT COUNT(*) FROM majors;

-- name: CountPermissions :one
SELECT COUNT(*) FROM permissions;
