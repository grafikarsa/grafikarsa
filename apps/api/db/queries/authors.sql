-- Queries for content collaboration: authors, invitations, and ownership.

-- name: LockContent :one
SELECT * FROM contents WHERE id = @id FOR UPDATE;

-- name: ListContentAuthors :many
SELECT
  ca.content_id,
  ca.user_id,
  ca.level,
  ca.role_label,
  ca.position,
  ca.invite_status,
  ca.invited_at,
  ca.invite_expires_at,
  u.username,
  u.display_name
FROM content_authors ca
JOIN users u ON u.id = ca.user_id
WHERE ca.content_id = @content_id
  AND ca.invite_status IN ('accepted', 'pending')
ORDER BY ca.position, ca.invited_at;

-- name: GetContentAuthorRow :one
SELECT * FROM content_authors
WHERE content_id = @content_id
  AND user_id = @user_id;

-- name: GetContentAuthorWithUser :one
SELECT
  ca.content_id,
  ca.user_id,
  ca.level,
  ca.role_label,
  ca.position,
  ca.invite_status,
  ca.invited_at,
  ca.invite_expires_at,
  u.username,
  u.display_name
FROM content_authors ca
JOIN users u ON u.id = ca.user_id
WHERE ca.content_id = @content_id
  AND ca.user_id = @user_id;

-- name: CountActiveAuthors :one
SELECT count(*)::int AS count
FROM content_authors
WHERE content_id = @content_id
  AND invite_status IN ('accepted', 'pending');

-- name: NextAuthorPosition :one
SELECT COALESCE(max(position) + 1, 0)::smallint AS position
FROM content_authors
WHERE content_id = @content_id;

-- name: GetInviteTargetUser :one
SELECT id, status, activated_at, content_blocked_at
FROM users
WHERE id = @id;

-- name: InsertContentAuthorInvite :exec
INSERT INTO content_authors (
  content_id, user_id, level, role_label, position, invite_status,
  invited_by, invited_at, invite_expires_at
) VALUES (
  @content_id, @user_id, @level, @role_label, @position, 'pending',
  @invited_by, @invited_at, @invite_expires_at
);

-- name: ReinviteContentAuthor :exec
UPDATE content_authors
SET level = @level,
    role_label = @role_label,
    invite_status = 'pending',
    invited_by = @invited_by,
    invited_at = @invited_at,
    invite_expires_at = @invite_expires_at,
    responded_at = NULL
WHERE content_id = @content_id
  AND user_id = @user_id;

-- name: UpdateContentAuthorFields :exec
UPDATE content_authors
SET level = @level,
    role_label = @role_label,
    position = @position
WHERE content_id = @content_id
  AND user_id = @user_id;

-- name: SetContentAuthorLevel :exec
UPDATE content_authors
SET level = @level
WHERE content_id = @content_id
  AND user_id = @user_id;

-- name: RemoveContentAuthor :exec
UPDATE content_authors
SET invite_status = 'removed',
    responded_at = @responded_at
WHERE content_id = @content_id
  AND user_id = @user_id;

-- name: LeaveContentAuthor :exec
UPDATE content_authors
SET invite_status = 'left',
    responded_at = @responded_at
WHERE content_id = @content_id
  AND user_id = @user_id;

-- name: AcceptContentAuthorInvite :exec
UPDATE content_authors
SET invite_status = 'accepted',
    responded_at = @responded_at
WHERE content_id = @content_id
  AND user_id = @user_id;

-- name: DeclineContentAuthorInvite :exec
UPDATE content_authors
SET invite_status = 'declined',
    responded_at = @responded_at
WHERE content_id = @content_id
  AND user_id = @user_id;

-- name: UpdateContentOwner :exec
UPDATE contents
SET owner_id = @owner_id
WHERE id = @id;

-- name: ExpirePendingInvitations :execrows
UPDATE content_authors
SET invite_status = 'expired'
WHERE invite_status = 'pending'
  AND invite_expires_at < @now;

-- name: ListPendingInvitations :many
SELECT
  ca.content_id,
  ca.level,
  ca.role_label,
  ca.invited_at,
  ca.invite_expires_at,
  c.public_id,
  c.type,
  c.title,
  c.cover_media_id,
  m.width AS cover_width,
  m.height AS cover_height,
  m.blurhash AS cover_blurhash,
  m.dominant_color AS cover_dominant_color,
  u.id AS inviter_id,
  u.username AS inviter_username,
  u.display_name AS inviter_display_name
FROM content_authors ca
JOIN contents c ON c.id = ca.content_id
JOIN users u ON u.id = ca.invited_by
LEFT JOIN media_assets m ON m.id = c.cover_media_id
WHERE ca.user_id = @user_id
  AND ca.invite_status = 'pending'
  AND ca.invite_expires_at > @now
  AND (
    sqlc.narg('cursor_time')::timestamptz IS NULL
    OR (ca.invited_at, ca.content_id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::uuid)
  )
ORDER BY ca.invited_at DESC, ca.content_id DESC
LIMIT @limit_count;
