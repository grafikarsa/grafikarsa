-- Lifecycle transitions for contents (submit, withdraw, revisions, archive,
-- delete/restore) plus the data needed by the version history and purge job.

-- name: LockContent :one
SELECT * FROM contents WHERE id = @id FOR UPDATE;

-- name: CountContentVersions :one
SELECT count(*)::int AS count
FROM content_versions
WHERE content_id = @content_id;

-- name: CountInReviewByOwner :one
SELECT count(*)::int AS count
FROM contents
WHERE owner_id = @owner_id
  AND (status = 'in_review' OR revision_status = 'in_review');

-- Queue major of the owner: the major of their latest enrollment by academic
-- year. Teachers, staff, and users without enrollment yield no row (global
-- queue).
-- name: GetOwnerQueueMajor :one
SELECT c.major_id
FROM enrollments e
JOIN classes c ON c.id = e.class_id
JOIN academic_years y ON y.id = e.academic_year_id
WHERE e.user_id = @user_id
ORDER BY y.start_year DESC
LIMIT 1;

-- name: MaxContentVersionNumber :one
SELECT COALESCE(max(number), 0)::int AS max_number
FROM content_versions
WHERE content_id = @content_id;

-- name: InsertContentVersion :one
INSERT INTO content_versions (
  content_id, number, kind, state, title, slug, category_id, cover_media_id,
  excerpt, body, meta, tag_ids, tool_ids, skill_ids, body_schema, content_hash,
  submitted_by
) VALUES (
  @content_id, @number, @kind, 'pending', @title, @slug, @category_id,
  @cover_media_id, @excerpt, @body, @meta, @tag_ids, @tool_ids, @skill_ids,
  @body_schema, @content_hash, @submitted_by
)
RETURNING *;

-- name: InsertContentVersionMedia :exec
INSERT INTO content_version_media (version_id, media_id)
VALUES (@version_id, @media_id)
ON CONFLICT DO NOTHING;

-- name: InsertModerationCase :one
INSERT INTO moderation_cases (content_id, version_id, kind, queue_major_id, status)
VALUES (@content_id, @version_id, @kind, @queue_major_id, 'open')
RETURNING *;

-- name: SetContentInReview :exec
UPDATE contents
SET status = 'in_review', revision_status = NULL
WHERE id = @id;

-- name: SetContentRevisionInReview :exec
UPDATE contents
SET revision_status = 'in_review'
WHERE id = @id;

-- name: GetPendingVersionForUpdate :one
SELECT * FROM content_versions
WHERE content_id = @content_id
  AND state = 'pending'
FOR UPDATE;

-- name: GetModerationCaseByVersion :one
SELECT * FROM moderation_cases WHERE version_id = @version_id;

-- name: SetContentVersionWithdrawn :exec
UPDATE content_versions SET state = 'withdrawn' WHERE id = @id;

-- name: SetModerationCaseWithdrawn :exec
UPDATE moderation_cases SET status = 'withdrawn' WHERE id = @id;

-- name: SetContentWithdrawnToDraft :exec
UPDATE contents
SET status = 'draft', revision_status = NULL
WHERE id = @id;

-- name: SetContentWithdrawnToTakenDown :exec
UPDATE contents
SET status = 'taken_down', revision_status = NULL
WHERE id = @id;

-- name: SetContentRevisionEditing :exec
UPDATE contents
SET revision_status = 'editing'
WHERE id = @id;

-- name: GetContentVersionByID :one
SELECT * FROM content_versions
WHERE id = @id
  AND content_id = @content_id;

-- name: UpdateContentDraftFromVersion :exec
UPDATE content_drafts
SET base_version_id = @base_version_id,
    title = @title,
    category_id = @category_id,
    cover_media_id = @cover_media_id,
    excerpt = @excerpt,
    body = @body,
    meta = @meta,
    tag_ids = @tag_ids,
    tool_ids = @tool_ids,
    skill_ids = @skill_ids,
    body_schema = @body_schema,
    lock_version = lock_version + 1,
    updated_by = @updated_by,
    updated_at = now()
WHERE content_id = @content_id;

-- name: CopyContentVersionMediaToDraft :exec
INSERT INTO content_draft_media (content_id, media_id)
SELECT @content_id, media_id
FROM content_version_media
WHERE version_id = @version_id
ON CONFLICT DO NOTHING;

-- name: ClearContentRevisionStatus :exec
UPDATE contents
SET revision_status = NULL
WHERE id = @id;

-- name: ArchiveContent :exec
UPDATE contents
SET status = 'archived', archived_at = @archived_at, revision_status = NULL
WHERE id = @id;

-- name: UnarchiveContent :exec
UPDATE contents
SET status = 'published', archived_at = NULL
WHERE id = @id;

-- name: SoftDeleteContent :exec
UPDATE contents
SET status_before_delete = status,
    status = 'deleted',
    deleted_at = @deleted_at,
    purge_after = @purge_after,
    revision_status = NULL
WHERE id = @id;

-- name: RestoreContent :execrows
UPDATE contents
SET status = status_before_delete,
    deleted_at = NULL,
    purge_after = NULL,
    status_before_delete = NULL
WHERE id = @id
  AND status = 'deleted'
  AND purge_after > now();

-- name: ListContentDraftMedia :many
SELECT media_id FROM content_draft_media WHERE content_id = @content_id;

-- name: ListMediaStatuses :many
SELECT id, status, deleted_at
FROM media_assets
WHERE id = ANY(@ids::uuid[]);

-- name: GetContentCategoryRef :one
SELECT slug::text AS slug, name
FROM content_categories
WHERE id = @id;

-- name: ListContentVersions :many
SELECT
  v.id,
  v.content_id,
  v.number,
  v.kind,
  v.state,
  v.title,
  v.slug,
  v.category_id,
  v.cover_media_id,
  v.excerpt,
  v.body,
  v.meta,
  v.tag_ids,
  v.tool_ids,
  v.skill_ids,
  v.body_schema,
  v.content_hash,
  v.submitted_by,
  v.submitted_at,
  v.decided_at,
  u.username AS submitter_username,
  u.display_name AS submitter_display_name,
  mc.id AS case_id,
  mc.decision,
  mc.decision_note,
  mc.decided_at AS case_decided_at,
  mr.label AS reason_label,
  a.id AS appeal_id,
  a.status AS appeal_status
FROM content_versions v
JOIN users u ON u.id = v.submitted_by
LEFT JOIN moderation_cases mc ON mc.version_id = v.id
LEFT JOIN moderation_reasons mr ON mr.id = mc.reason_id
LEFT JOIN appeals a ON a.case_id = mc.id
WHERE v.content_id = @content_id
ORDER BY v.number DESC;

-- name: GetVersionModerationDecision :one
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
WHERE mc.version_id = @version_id;

-- name: DeleteExpiredDeletedContents :execrows
DELETE FROM contents
WHERE id IN (
  SELECT id FROM contents
  WHERE status = 'deleted' AND purge_after < now()
  ORDER BY purge_after
  LIMIT @limit_count
);
