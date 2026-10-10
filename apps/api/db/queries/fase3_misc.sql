-- Assessment, feedback, changelog.

-- name: ListMetrics :many
SELECT id, name, sort, is_active FROM assessment_metrics ORDER BY sort ASC;

-- name: CreateMetric :one
INSERT INTO assessment_metrics (name, sort) VALUES ($1, $2) RETURNING id;

-- name: UpdateMetric :exec
UPDATE assessment_metrics SET name = $2, sort = $3, is_active = $4 WHERE id = $1;

-- name: DeleteMetric :exec
DELETE FROM assessment_metrics WHERE id = $1;

-- name: GetAssessment :one
SELECT id, portfolio_id, assessor_id, total_score, comment FROM portfolio_assessments
WHERE portfolio_id = $1 LIMIT 1;

-- name: UpsertAssessment :one
INSERT INTO portfolio_assessments (portfolio_id, assessor_id, total_score, comment)
VALUES ($1, $2, $3, $4)
ON CONFLICT (portfolio_id) DO UPDATE SET assessor_id = EXCLUDED.assessor_id, total_score = EXCLUDED.total_score, comment = EXCLUDED.comment, updated_at = NOW()
RETURNING id;

-- name: DeleteScores :exec
DELETE FROM assessment_scores WHERE assessment_id = $1;

-- name: AddScore :exec
INSERT INTO assessment_scores (assessment_id, metric_id, score, comment)
VALUES ($1, $2, $3, $4) ON CONFLICT (assessment_id, metric_id) DO UPDATE SET score = EXCLUDED.score, comment = EXCLUDED.comment;

-- name: ListScores :many
SELECT s.metric_id, s.score, s.comment, m.name AS metric_name
FROM assessment_scores s JOIN assessment_metrics m ON m.id = s.metric_id
WHERE s.assessment_id = $1 ORDER BY m.sort ASC;

-- name: CreateFeedbackThread :one
INSERT INTO feedback_threads (reporter_id, reporter_email, category, title, body_md, priority)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING id;

-- name: GetFeedbackThread :one
SELECT id, reporter_id, reporter_email, category, title, body_md, status, priority, upvotes_cached, assignee_id, is_public, is_spam, linked_changelog_id, resolved_at, created_at
FROM feedback_threads WHERE id = $1 LIMIT 1;

-- name: ListFeedbackThreads :many
SELECT id, reporter_id, category, title, status, priority, upvotes_cached, is_public, created_at
FROM feedback_threads WHERE is_spam = FALSE
  AND ($1 = '' OR status = $1) AND ($2 = '' OR category = $2)
ORDER BY created_at DESC LIMIT $3 OFFSET $4;

-- name: UpdateFeedbackThread :exec
UPDATE feedback_threads SET status = $2, priority = $3, assignee_id = $4, is_public = $5,
  resolved_at = CASE WHEN $2 IN ('resolved', 'closed') THEN NOW() ELSE resolved_at END
WHERE id = $1;

-- name: AddFeedbackMessage :one
INSERT INTO feedback_messages (thread_id, author_id, body_md, is_internal)
VALUES ($1, $2, $3, $4) RETURNING id;

-- name: ListFeedbackMessages :many
SELECT id, thread_id, author_id, body_md, is_internal, created_at
FROM feedback_messages WHERE thread_id = $1
  AND ($2 = FALSE OR is_internal = FALSE)
ORDER BY created_at ASC;

-- name: VoteFeedback :exec
INSERT INTO feedback_votes (thread_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: CountFeedbackVotes :one
SELECT COUNT(*) FROM feedback_votes WHERE thread_id = $1;

-- name: CreateChangelog :one
INSERT INTO changelogs (version, title, summary, channel, audience, status, publish_at, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id;

-- name: GetChangelog :one
SELECT id, version, title, summary, channel, audience, status, publish_at, created_at
FROM changelogs WHERE id = $1 LIMIT 1;

-- name: ListChangelogs :many
SELECT id, version, title, summary, channel, audience, status, publish_at, created_at
FROM changelogs ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: ListPublishedChangelogs :many
SELECT id, version, title, summary, channel, audience, publish_at
FROM changelogs WHERE status = 'published' AND (publish_at IS NULL OR publish_at <= NOW())
ORDER BY publish_at DESC LIMIT $1 OFFSET $2;

-- name: SetChangelogStatus :exec
UPDATE changelogs SET status = $2 WHERE id = $1;

-- name: AddChangelogSection :exec
INSERT INTO changelog_sections (changelog_id, kind, title, body_md, sort)
VALUES ($1, $2, $3, $4, $5);

-- name: ListChangelogSections :many
SELECT id, kind, title, body_md, sort FROM changelog_sections
WHERE changelog_id = $1 ORDER BY sort ASC;

-- name: MarkChangelogRead :exec
INSERT INTO changelog_reads (changelog_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: CountChangelogReads :one
SELECT COUNT(*) FROM changelog_reads WHERE changelog_id = $1;

-- name: ListScheduledChangelogs :many
SELECT id, version, title, audience FROM changelogs
WHERE status = 'scheduled' AND (publish_at IS NULL OR publish_at <= NOW()) LIMIT 20;

-- name: PublishDueChangelogs :many
UPDATE changelogs SET status = 'published' WHERE id IN (
  SELECT id FROM changelogs WHERE status = 'scheduled' AND (publish_at IS NULL OR publish_at <= NOW()) LIMIT 20 FOR UPDATE SKIP LOCKED
) RETURNING id, version, title, audience;

-- name: DeleteOldReadNotifications :exec
DELETE FROM notifications WHERE is_read = TRUE AND created_at < NOW() - INTERVAL '90 days'
  AND type NOT IN ('feedback_updated');

-- name: ListDigestUsers :many
SELECT user_id FROM notification_preferences WHERE digest_daily = TRUE LIMIT 100;

-- name: MarkChangelogPublished :exec
UPDATE changelogs SET publish_at = NOW() WHERE id = $1 AND publish_at IS NULL;

-- name: CountUnreadChangelogs :one
SELECT COUNT(*) FROM changelogs c
WHERE c.status = 'published' AND (c.publish_at IS NULL OR c.publish_at <= NOW())
  AND NOT EXISTS (SELECT 1 FROM changelog_reads r WHERE r.changelog_id = c.id AND r.user_id = $1);

-- name: MarkAllChangelogsRead :exec
INSERT INTO changelog_reads (changelog_id, user_id)
SELECT c.id, $1 FROM changelogs c
WHERE c.status = 'published' AND (c.publish_at IS NULL OR c.publish_at <= NOW())
ON CONFLICT DO NOTHING;
