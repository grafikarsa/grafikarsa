-- name: UpsertVaultEntry :exec
INSERT INTO initial_password_vault (user_id, ciphertext, expires_at)
VALUES (@user_id::uuid, @ciphertext::bytea, @expires_at::timestamptz)
ON CONFLICT (user_id) DO UPDATE
SET ciphertext = EXCLUDED.ciphertext,
    expires_at = EXCLUDED.expires_at,
    created_at = now();

-- name: DeleteVaultEntry :exec
DELETE FROM initial_password_vault
WHERE user_id = @user_id::uuid;

-- name: DeleteExpiredVaultEntries :execrows
DELETE FROM initial_password_vault
WHERE expires_at <= now();

-- name: ListVaultEntriesForClass :many
SELECT
  u.id AS user_id,
  u.display_name,
  u.nisn,
  v.ciphertext,
  v.expires_at
FROM initial_password_vault v
JOIN users u ON u.id = v.user_id
JOIN enrollments e ON e.user_id = u.id
WHERE e.class_id = @class_id::uuid
  AND u.activated_at IS NULL
  AND v.expires_at > now()
ORDER BY u.display_name, u.id;
