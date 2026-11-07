-- name: GetSetting :one
SELECT key, value, updated_by, updated_at FROM settings WHERE key = $1;

-- name: ListSettings :many
SELECT key, value, updated_by, updated_at FROM settings ORDER BY key;

-- name: UpsertSetting :one
INSERT INTO settings (key, value, updated_by, updated_at)
VALUES (@key, @value, @updated_by, now())
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = now()
RETURNING key, value, updated_by, updated_at;
