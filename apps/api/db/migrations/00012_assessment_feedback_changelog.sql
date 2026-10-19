-- +goose Up
CREATE TABLE assessment_metrics (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(100) NOT NULL,
  sort INTEGER NOT NULL DEFAULT 0,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE portfolio_assessments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  portfolio_id UUID NOT NULL UNIQUE REFERENCES portfolios (id) ON DELETE CASCADE,
  assessor_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
  total_score NUMERIC(4, 2) NOT NULL DEFAULT 0,
  comment TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_assessments_updated_at
  BEFORE UPDATE ON portfolio_assessments FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE assessment_scores (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  assessment_id UUID NOT NULL REFERENCES portfolio_assessments (id) ON DELETE CASCADE,
  metric_id UUID NOT NULL REFERENCES assessment_metrics (id) ON DELETE RESTRICT,
  score SMALLINT NOT NULL,
  comment TEXT NOT NULL DEFAULT '',
  CONSTRAINT score_range CHECK (score BETWEEN 1 AND 10),
  CONSTRAINT score_unique UNIQUE (assessment_id, metric_id)
);

CREATE TABLE feedback_threads (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  reporter_id UUID REFERENCES users (id) ON DELETE SET NULL,
  reporter_email TEXT NOT NULL DEFAULT '',
  category TEXT NOT NULL,
  title VARCHAR(200) NOT NULL,
  body_md TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open',
  priority TEXT NOT NULL DEFAULT 'medium',
  upvotes_cached INTEGER NOT NULL DEFAULT 0,
  assignee_id UUID REFERENCES users (id) ON DELETE SET NULL,
  is_public BOOLEAN NOT NULL DEFAULT FALSE,
  is_spam BOOLEAN NOT NULL DEFAULT FALSE,
  linked_changelog_id UUID,
  resolved_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT fb_cat_valid CHECK (category IN ('bug', 'idea', 'question', 'praise')),
  CONSTRAINT fb_status_valid CHECK (status IN ('open', 'triaged', 'in_progress', 'resolved', 'closed', 'duplicate')),
  CONSTRAINT fb_prio_valid CHECK (priority IN ('low', 'medium', 'high', 'critical'))
);
CREATE INDEX idx_feedback_status ON feedback_threads (status, created_at DESC) WHERE is_spam = FALSE;
CREATE TRIGGER trg_feedback_updated_at
  BEFORE UPDATE ON feedback_threads FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE feedback_messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  thread_id UUID NOT NULL REFERENCES feedback_threads (id) ON DELETE CASCADE,
  author_id UUID REFERENCES users (id) ON DELETE SET NULL,
  body_md TEXT NOT NULL,
  is_internal BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE feedback_votes (
  thread_id UUID NOT NULL REFERENCES feedback_threads (id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (thread_id, user_id)
);

CREATE TABLE changelogs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  version TEXT NOT NULL UNIQUE,
  title VARCHAR(200) NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  channel TEXT NOT NULL DEFAULT 'stable',
  audience TEXT NOT NULL DEFAULT 'all',
  status TEXT NOT NULL DEFAULT 'draft',
  publish_at TIMESTAMPTZ,
  created_by UUID REFERENCES users (id) ON DELETE SET NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT cl_channel_valid CHECK (channel IN ('stable', 'beta')),
  CONSTRAINT cl_audience_valid CHECK (audience IN ('all', 'siswa', 'guru', 'admin')),
  CONSTRAINT cl_status_valid CHECK (status IN ('draft', 'scheduled', 'published', 'archived'))
);
CREATE INDEX idx_changelogs_published ON changelogs (publish_at DESC) WHERE status = 'published';

CREATE TABLE changelog_sections (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  changelog_id UUID NOT NULL REFERENCES changelogs (id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  title VARCHAR(200) NOT NULL DEFAULT '',
  body_md TEXT NOT NULL DEFAULT '',
  sort INTEGER NOT NULL DEFAULT 0,
  CONSTRAINT section_kind_valid CHECK (kind IN ('added', 'improved', 'fixed', 'removed'))
);

CREATE TABLE changelog_contributors (
  changelog_id UUID NOT NULL REFERENCES changelogs (id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  role TEXT NOT NULL DEFAULT 'dev',
  PRIMARY KEY (changelog_id, user_id)
);

CREATE TABLE changelog_reads (
  changelog_id UUID NOT NULL REFERENCES changelogs (id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (changelog_id, user_id)
);

ALTER TABLE feedback_threads ADD CONSTRAINT fb_changelog_fk
  FOREIGN KEY (linked_changelog_id) REFERENCES changelogs (id) ON DELETE SET NULL;

COMMENT ON TABLE portfolio_assessments IS 'Nilai guru per portfolio; total = AVG(scores)';
COMMENT ON TABLE feedback_threads IS 'Thread laporan/masukan; guest boleh via email+turnstile';

-- +goose Down
ALTER TABLE feedback_threads DROP CONSTRAINT IF EXISTS fb_changelog_fk;
DROP TABLE IF EXISTS changelog_reads;
DROP TABLE IF EXISTS changelog_contributors;
DROP TABLE IF EXISTS changelog_sections;
DROP TABLE IF EXISTS changelogs;
DROP TABLE IF EXISTS feedback_votes;
DROP TABLE IF EXISTS feedback_messages;
DROP TABLE IF EXISTS feedback_threads;
DROP TABLE IF EXISTS assessment_scores;
DROP TABLE IF EXISTS portfolio_assessments;
DROP TABLE IF EXISTS assessment_metrics;
