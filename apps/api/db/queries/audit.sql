-- name: InsertAuditLog :one
INSERT INTO audit_logs (actor_id, actor_ip, action, target_type, target_id, scope_major_id, data)
VALUES (@actor_id, @actor_ip, @action, @target_type, @target_id, @scope_major_id, @data)
RETURNING id, occurred_at;

-- name: ListAuditLogs :many
SELECT
  al.id,
  al.occurred_at,
  al.actor_id,
  al.actor_ip,
  al.action,
  al.target_type,
  al.target_id,
  al.scope_major_id,
  al.data,
  actor.display_name AS actor_name,
  actor.username AS actor_username
FROM audit_logs al
LEFT JOIN users actor ON actor.id = al.actor_id
WHERE (sqlc.narg('actor_id')::uuid IS NULL OR al.actor_id = sqlc.narg('actor_id')::uuid)
  AND (sqlc.narg('action')::text IS NULL OR al.action = sqlc.narg('action')::text)
  AND (sqlc.narg('action_prefix')::text IS NULL OR al.action LIKE sqlc.narg('action_prefix')::text || '%')
  AND (sqlc.narg('target_type')::text IS NULL OR al.target_type = sqlc.narg('target_type')::text)
  AND (sqlc.narg('target_id')::uuid IS NULL OR al.target_id = sqlc.narg('target_id')::uuid)
  AND (sqlc.narg('scope_major_id')::uuid IS NULL OR al.scope_major_id = sqlc.narg('scope_major_id')::uuid)
  AND (sqlc.narg('from')::timestamptz IS NULL OR al.occurred_at >= sqlc.narg('from')::timestamptz)
  AND (sqlc.narg('to')::timestamptz IS NULL OR al.occurred_at < sqlc.narg('to')::timestamptz)
  AND (
    @scope_global::boolean
    OR (
      cardinality(@scope_major_ids::uuid[]) > 0
      AND al.scope_major_id = ANY(@scope_major_ids::uuid[])
    )
  )
  AND (
    sqlc.narg('cursor_time')::timestamptz IS NULL
    OR (al.occurred_at, al.id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::bigint)
  )
ORDER BY al.occurred_at DESC, al.id DESC
LIMIT @limit_count;
