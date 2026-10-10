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
-- Retensi: 90 hari untuk notif umum; feedback & pengumuman 1 tahun.
DELETE FROM notifications WHERE is_read = TRUE
  AND (
    (type IN ('feedback_updated', 'announcement') AND created_at < NOW() - INTERVAL '365 days')
    OR (type NOT IN ('feedback_updated', 'announcement') AND created_at < NOW() - INTERVAL '90 days')
  );

-- name: ListDigestUsers :many
SELECT user_id FROM notification_preferences WHERE digest_daily = TRUE LIMIT 100;

-- name: DigestSummaryForUser :one
SELECT COUNT(*)::int AS unread,
  COALESCE(
    (SELECT string_agg(t.title, ' • ') FROM (
      SELECT title FROM notifications
      WHERE user_id = $1 AND is_read = FALSE AND type <> 'digest'
        AND created_at > NOW() - INTERVAL '24 hours'
      ORDER BY created_at DESC LIMIT 3
    ) t),
    ''
  )::text AS sample;

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

-- name: GetSystemSetting :one
SELECT key, value FROM system_settings WHERE key = $1 LIMIT 1;

-- name: ListSystemSettings :many
SELECT key, value, updated_at FROM system_settings ORDER BY key ASC;

-- name: UpsertSystemSetting :exec
INSERT INTO system_settings (key, value, updated_by) VALUES ($1, $2, $3)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = NOW();

-- name: DashboardStats :one
-- Angka ringkas untuk /admin/dashboard (scope '' = global, atau id jurusan).
SELECT
  (SELECT COUNT(*) FROM users u WHERE u.deleted_at IS NULL AND ($1::text = '' OR u.major_id::text = $1)) AS users_total,
  (SELECT COUNT(*) FROM users u WHERE u.deleted_at IS NULL AND u.is_active AND ($1::text = '' OR u.major_id::text = $1)) AS users_active,
  (SELECT COUNT(*) FROM users u WHERE u.deleted_at IS NULL AND u.status = 'lulus' AND ($1::text = '' OR u.major_id::text = $1)) AS users_alumni,
  (SELECT COUNT(*) FROM portfolios p JOIN users u ON u.id = p.owner_id
    WHERE p.deleted_at IS NULL AND ($1::text = '' OR u.major_id::text = $1)) AS portfolios_total,
  (SELECT COUNT(*) FROM portfolios p JOIN users u ON u.id = p.owner_id
    WHERE p.deleted_at IS NULL AND p.status = 'published' AND ($1::text = '' OR u.major_id::text = $1)) AS portfolios_published,
  (SELECT COUNT(*) FROM portfolios p JOIN users u ON u.id = p.owner_id
    WHERE p.deleted_at IS NULL AND p.status = 'pending_review' AND ($1::text = '' OR u.major_id::text = $1)) AS portfolios_pending,
  (SELECT COUNT(*) FROM feedback_threads f WHERE f.status IN ('open', 'triaged', 'in_progress')) AS feedback_open,
  (SELECT COUNT(*) FROM outbox WHERE status IN ('pending', 'processing')) AS jobs_queued,
  (SELECT COUNT(*) FROM outbox WHERE status = 'failed') AS jobs_failed,
  (SELECT COUNT(*) FROM quarantine_blocks) AS blocks_quarantined;

-- name: ListOutboxJobs :many
SELECT id, topic, status, attempts, next_retry_at, created_at FROM outbox
WHERE ($1::text = '' OR status = $1)
ORDER BY id DESC LIMIT $2 OFFSET $3;

-- name: CountOutboxJobs :one
SELECT COUNT(*) FROM outbox WHERE ($1::text = '' OR status = $1);

-- name: LatestChangelog :one
SELECT id, version, title, summary, channel, audience, status, publish_at, created_at
FROM changelogs
WHERE status = 'published' AND (publish_at IS NULL OR publish_at <= NOW())
ORDER BY COALESCE(publish_at, created_at) DESC LIMIT 1;

-- name: GetChangelogByVersion :one
SELECT id, version, title, summary, channel, audience, status, publish_at, created_at
FROM changelogs WHERE version = $1 LIMIT 1;

-- name: ListPublicMajors :many
SELECT id, code, name, require_series FROM majors ORDER BY name ASC;

-- name: ListPublicClasses :many
SELECT c.id, c.name, c.grade, c.class_group, m.code AS major_code, y.start_year
FROM classes c JOIN majors m ON m.id = c.major_id JOIN academic_years y ON y.id = c.academic_year_id
WHERE y.is_active ORDER BY c.name ASC;

-- name: ListChangelogContributors :many
SELECT u.id, u.username, u.name, c.role FROM changelog_contributors c
JOIN users u ON u.id = c.user_id WHERE c.changelog_id = $1 ORDER BY c.role, u.name;

-- name: AddChangelogContributor :exec
INSERT INTO changelog_contributors (changelog_id, user_id, role) VALUES ($1, $2, $3)
ON CONFLICT (changelog_id, user_id) DO UPDATE SET role = EXCLUDED.role;

-- name: RemoveChangelogContributor :exec
DELETE FROM changelog_contributors WHERE changelog_id = $1 AND user_id = $2;

-- name: SetFeedbackLinkedChangelog :exec
UPDATE feedback_threads SET linked_changelog_id = $2 WHERE id = $1;

-- name: ListFeedbackVoterIDs :many
SELECT user_id FROM feedback_votes WHERE thread_id = $1;

-- name: ListLeaderboardRankedUserIDs :many
-- Penerima notifikasi peringkat: user yang masuk top-N best_students.
SELECT DISTINCT e.subject_user_id FROM leaderboard_entries e
JOIN leaderboards l ON l.id = e.leaderboard_id
WHERE l.period = $1 AND l.scope = 'global' AND l.kind = 'best_students'
  AND e.rank <= $2 AND e.subject_user_id IS NOT NULL;

-- name: DecayUserInterests :exec
-- Peluruhan bobot minat (decay) agar like lama tidak mendominasi selamanya.
-- Nilai dibatasi minimum 0.01 lalu dibulatkan 4 desimal.
UPDATE user_interests SET
  tag_weights = COALESCE((
    SELECT jsonb_object_agg(t.k, GREATEST(0.01::numeric, round(t.v::numeric * $1::numeric, 4)))
    FROM jsonb_each_text(user_interests.tag_weights) AS t(k, v)
  ), '{}'::jsonb),
  major_weights = COALESCE((
    SELECT jsonb_object_agg(t.k, GREATEST(0.01::numeric, round(t.v::numeric * $1::numeric, 4)))
    FROM jsonb_each_text(user_interests.major_weights) AS t(k, v)
  ), '{}'::jsonb),
  total_likes = GREATEST(0, (total_likes * $1)::int),
  updated_at = NOW()
WHERE updated_at < NOW() - INTERVAL '1 day';

-- name: ListAssessmentQueue :many
-- Antrean penilaian: portfolio published + status dinilai/belum, dibatasi scope.
SELECT p.id, p.title, p.slug, u.username, u.name AS owner_name,
  (a.id IS NOT NULL)::bool AS assessed, COALESCE(a.total_score, 0) AS total_score,
  p.published_at
FROM portfolios p
JOIN users u ON u.id = p.owner_id
LEFT JOIN portfolio_assessments a ON a.portfolio_id = p.id
WHERE p.status = 'published' AND p.deleted_at IS NULL
  AND ($1::text = '' OR u.major_id::text = $1)
  AND ($2::text = '' OR ($2::text = 'assessed' AND a.id IS NOT NULL)
       OR ($2::text = 'unassessed' AND a.id IS NULL))
ORDER BY (a.id IS NULL) DESC, p.published_at DESC
LIMIT $3 OFFSET $4;

-- name: CountAssessmentQueue :one
SELECT COUNT(*) FROM portfolios p
JOIN users u ON u.id = p.owner_id
LEFT JOIN portfolio_assessments a ON a.portfolio_id = p.id
WHERE p.status = 'published' AND p.deleted_at IS NULL
  AND ($1::text = '' OR u.major_id::text = $1)
  AND ($2::text = '' OR ($2::text = 'assessed' AND a.id IS NOT NULL)
       OR ($2::text = 'unassessed' AND a.id IS NULL));
