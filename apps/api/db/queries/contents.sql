-- name: CountActiveDraftsByOwner :one
SELECT count(*)::int AS count
FROM contents
WHERE owner_id = @owner_id
  AND status IN ('draft', 'changes_requested');

-- name: InsertContent :one
INSERT INTO contents (public_id, type, status, owner_id, slug, title)
VALUES (@public_id, @content_type, 'draft', @owner_id, 'tanpa-judul', 'Tanpa judul')
ON CONFLICT (public_id) DO NOTHING
RETURNING *;

-- name: InsertContentDraft :one
INSERT INTO content_drafts (content_id, body, meta, updated_by)
VALUES (@content_id, @body, @meta, @updated_by)
RETURNING *;

-- name: InsertContentOwner :exec
INSERT INTO content_authors (content_id, user_id, level, position, invite_status, responded_at)
VALUES (@content_id, @user_id, 'owner', 0, 'accepted', @responded_at);

-- name: InsertContentStats :exec
INSERT INTO content_stats (content_id) VALUES (@content_id);

-- name: GetContent :one
SELECT * FROM contents WHERE id = @id;

-- name: GetContentAuthor :one
SELECT * FROM content_authors
WHERE content_id = @content_id
  AND user_id = @user_id
  AND invite_status = 'accepted';

-- name: GetContentDraft :one
SELECT * FROM content_drafts WHERE content_id = @content_id;

-- name: UpdateContentDraft :execrows
UPDATE content_drafts
SET title = @title,
    category_id = @category_id,
    cover_media_id = @cover_media_id,
    excerpt = @excerpt,
    body = @body,
    meta = @meta,
    tag_ids = @tag_ids,
    tool_ids = @tool_ids,
    skill_ids = @skill_ids,
    lock_version = lock_version + 1,
    updated_by = @updated_by,
    updated_at = now()
WHERE content_id = @content_id
  AND lock_version = @lock_version;

-- name: DeleteContentDraftMedia :exec
DELETE FROM content_draft_media WHERE content_id = @content_id;

-- name: InsertContentDraftMedia :exec
INSERT INTO content_draft_media (content_id, media_id)
VALUES (@content_id, @media_id)
ON CONFLICT DO NOTHING;

-- name: UpdateContentFromDraft :exec
UPDATE contents
SET title = @title,
    slug = @slug,
    category_id = @category_id,
    cover_media_id = @cover_media_id,
    excerpt = @excerpt,
    updated_at = now()
WHERE id = @id;

-- name: TouchContent :exec
UPDATE contents SET updated_at = now() WHERE id = @id;

-- name: GetContentCategoryByID :one
SELECT id, content_type, slug::text AS slug, name, is_active
FROM content_categories
WHERE id = @id;

-- name: GetTagBySlug :one
SELECT id, slug::text AS slug, name, is_blocked
FROM tags
WHERE slug = @slug::citext;

-- name: InsertTag :exec
INSERT INTO tags (slug, name)
VALUES (@slug::citext, @name)
ON CONFLICT (slug) DO NOTHING;

-- name: GetTagsByIDs :many
SELECT id, slug::text AS slug, name
FROM tags
WHERE id = ANY(@ids::uuid[])
ORDER BY name;

-- name: GetToolsBySlugs :many
SELECT id, slug::text AS slug, name, category
FROM tools
WHERE is_active
  AND slug::text = ANY(@slugs::text[])
ORDER BY name;

-- name: GetToolsByIDs :many
SELECT id, slug::text AS slug, name, category
FROM tools
WHERE id = ANY(@ids::uuid[])
ORDER BY name;

-- name: GetSkillsBySlugs :many
SELECT id, slug::text AS slug, name, category
FROM skills
WHERE is_active
  AND slug::text = ANY(@slugs::text[])
ORDER BY name;

-- name: GetSkillsByIDs :many
SELECT id, slug::text AS slug, name, category
FROM skills
WHERE id = ANY(@ids::uuid[])
ORDER BY name;

-- name: GetAuthorOwnedImage :one
SELECT m.id, m.width, m.height, m.blurhash, m.dominant_color
FROM media_assets m
WHERE m.id = @media_id
  AND m.kind = 'image'
  AND m.deleted_at IS NULL
  AND EXISTS (
    SELECT 1
    FROM content_authors ca
    WHERE ca.content_id = @content_id
      AND ca.user_id = m.owner_id
      AND ca.invite_status = 'accepted'
  );

-- name: GetImageRef :one
SELECT id, width, height, blurhash, dominant_color
FROM media_assets
WHERE id = @id;

-- name: GetBlogSeriesOwner :one
SELECT owner_id FROM blog_series WHERE id = @id;

-- name: GetAuthorMini :one
SELECT id, username, display_name
FROM users
WHERE id = @id;

-- name: CountAcceptedAuthors :one
SELECT count(*)::int AS count
FROM content_authors
WHERE content_id = @content_id
  AND invite_status = 'accepted';

-- name: GetLatestModerationDecision :one
SELECT
  mc.id AS case_id,
  mc.decision,
  mc.decision_note,
  mc.decided_at,
  mr.label AS reason_label,
  a.id AS appeal_id,
  a.status AS appeal_status
FROM moderation_cases mc
LEFT JOIN moderation_reasons mr ON mr.id = mc.reason_id
LEFT JOIN appeals a ON a.case_id = mc.id
WHERE mc.content_id = @content_id
  AND mc.status = 'decided'
ORDER BY mc.decided_at DESC NULLS LAST, mc.id DESC
LIMIT 1;

-- name: ListModerationAnnotations :many
SELECT target_ref, note
FROM moderation_annotations
WHERE case_id = @case_id
ORDER BY created_at, id;

-- name: HasClaimedModerationCase :one
SELECT EXISTS (
  SELECT 1
  FROM moderation_cases
  WHERE content_id = @content_id
    AND status = 'claimed'
) AS claimed;

-- name: ListStudioContents :many
SELECT
  c.id,
  c.public_id,
  c.type,
  c.status,
  c.revision_status,
  CASE
    WHEN c.status = 'published' THEN c.title
    ELSE COALESCE(NULLIF(d.title, ''), 'Tanpa judul')
  END::text AS title,
  c.cover_media_id,
  m.width AS cover_width,
  m.height AS cover_height,
  m.blurhash AS cover_blurhash,
  m.dominant_color AS cover_dominant_color,
  ca.level AS my_level,
  (
    SELECT count(*)::int
    FROM content_authors a
    WHERE a.content_id = c.id
      AND a.invite_status = 'accepted'
  ) AS authors_count,
  c.updated_at
FROM contents c
JOIN content_authors ca
  ON ca.content_id = c.id
  AND ca.user_id = @user_id
  AND ca.invite_status = 'accepted'
JOIN content_drafts d ON d.content_id = c.id
LEFT JOIN media_assets m ON m.id = c.cover_media_id
WHERE (sqlc.narg('status')::text IS NULL OR c.status = sqlc.narg('status')::text)
  AND (sqlc.narg('type')::text IS NULL OR c.type = sqlc.narg('type')::text)
  AND (
    sqlc.narg('cursor_time')::timestamptz IS NULL
    OR (c.updated_at, c.id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::uuid)
  )
ORDER BY c.updated_at DESC, c.id DESC
LIMIT @limit_count;

-- name: CountStudioContentsByStatus :many
SELECT c.status, count(*)::int AS count
FROM contents c
JOIN content_authors ca
  ON ca.content_id = c.id
  AND ca.user_id = @user_id
  AND ca.invite_status = 'accepted'
GROUP BY c.status;

-- name: CountPendingInvitations :one
SELECT count(*)::int AS count
FROM content_authors
WHERE user_id = @user_id
  AND invite_status = 'pending'
  AND invite_expires_at > now();
