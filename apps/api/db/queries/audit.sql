-- name: InsertAuditLog :one
INSERT INTO audit_logs (actor_id, actor_ip, action, target_type, target_id, scope_major_id, data)
VALUES (@actor_id, @actor_ip, @action, @target_type, @target_id, @scope_major_id, @data)
RETURNING id, occurred_at;

-- name: ListAuditLogs :many
SELECT id, occurred_at, actor_id, actor_ip, action, target_type, target_id, scope_major_id, data
FROM audit_logs
WHERE (sqlc.narg('actor_id')::uuid IS NULL OR actor_id = sqlc.narg('actor_id')::uuid)
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action')::text)
  AND (sqlc.narg('action_prefix')::text IS NULL OR action LIKE sqlc.narg('action_prefix')::text || '%')
  AND (sqlc.narg('target_type')::text IS NULL OR target_type = sqlc.narg('target_type')::text)
  AND (sqlc.narg('target_id')::uuid IS NULL OR target_id = sqlc.narg('target_id')::uuid)
  AND (sqlc.narg('scope_major_id')::uuid IS NULL OR scope_major_id = sqlc.narg('scope_major_id')::uuid)
  AND (sqlc.narg('from')::timestamptz IS NULL OR occurred_at >= sqlc.narg('from')::timestamptz)
  AND (sqlc.narg('to')::timestamptz IS NULL OR occurred_at < sqlc.narg('to')::timestamptz)
  AND (
    sqlc.narg('cursor_time')::timestamptz IS NULL
    OR (occurred_at, id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::bigint)
  )
ORDER BY occurred_at DESC, id DESC
LIMIT @limit_count;
