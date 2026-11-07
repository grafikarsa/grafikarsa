-- +goose Up
-- =============================================================================
-- 14. PENUGASAN & PENILAIAN (fase 2)
-- =============================================================================
CREATE TABLE rubrics (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  name        text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 100),
  created_by  uuid NOT NULL REFERENCES users(id),
  is_shared   boolean NOT NULL DEFAULT false,
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE rubric_criteria (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  rubric_id    uuid NOT NULL REFERENCES rubrics(id) ON DELETE CASCADE,
  name         text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 100),
  description  text CHECK (char_length(description) <= 500),
  max_score    smallint NOT NULL CHECK (max_score BETWEEN 1 AND 100),
  weight       numeric(5,2) NOT NULL DEFAULT 1 CHECK (weight > 0),
  position     smallint NOT NULL DEFAULT 0
);

CREATE TABLE assignments (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  class_id     uuid NOT NULL REFERENCES classes(id),
  rubric_id    uuid REFERENCES rubrics(id),
  created_by   uuid NOT NULL REFERENCES users(id),
  title        text NOT NULL CHECK (char_length(title) BETWEEN 3 AND 150),
  instructions text CHECK (char_length(instructions) <= 5000),
  due_at       timestamptz,
  status       text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'open', 'closed')),
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_assignments_class ON assignments (class_id, status);
CREATE TRIGGER trg_assignments_updated BEFORE UPDATE ON assignments FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE assignment_submissions (
  id             uuid PRIMARY KEY DEFAULT uuidv7(),
  assignment_id  uuid NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
  student_id     uuid NOT NULL REFERENCES users(id),
  content_id     uuid NOT NULL REFERENCES contents(id),
  submitted_at   timestamptz NOT NULL DEFAULT now(),
  UNIQUE (assignment_id, student_id)
);

CREATE TABLE assessments (
  id             uuid PRIMARY KEY DEFAULT uuidv7(),
  submission_id  uuid NOT NULL UNIQUE REFERENCES assignment_submissions(id) ON DELETE CASCADE,
  assessor_id    uuid NOT NULL REFERENCES users(id),
  final_comment  text CHECK (char_length(final_comment) <= 2000),
  total_score    numeric(6,2),
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_assessments_updated BEFORE UPDATE ON assessments FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE assessment_scores (
  assessment_id  uuid NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
  criterion_id   uuid NOT NULL REFERENCES rubric_criteria(id),
  score          smallint NOT NULL CHECK (score >= 0),
  comment        text CHECK (char_length(comment) <= 1000),
  PRIMARY KEY (assessment_id, criterion_id)
);

-- =============================================================================
-- 15. KURASI (fase 2)
-- =============================================================================
CREATE TABLE curations (
  id              uuid PRIMARY KEY DEFAULT uuidv7(),
  slug            citext NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]{2,80}$'),
  title           text NOT NULL CHECK (char_length(title) BETWEEN 3 AND 120),
  description     text CHECK (char_length(description) <= 1000),
  cover_media_id  uuid REFERENCES media_assets(id),
  created_by      uuid NOT NULL REFERENCES users(id),
  is_published    boolean NOT NULL DEFAULT false,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_curations_updated BEFORE UPDATE ON curations FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE curation_items (
  curation_id  uuid NOT NULL REFERENCES curations(id) ON DELETE CASCADE,
  content_id   uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  note         text CHECK (char_length(note) <= 300),
  added_by     uuid NOT NULL REFERENCES users(id),
  position     smallint NOT NULL DEFAULT 0,
  added_at     timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (curation_id, content_id)
);

-- +goose Down
DROP TABLE IF EXISTS curation_items;
DROP TABLE IF EXISTS curations;
DROP TABLE IF EXISTS assessment_scores;
DROP TABLE IF EXISTS assessments;
DROP TABLE IF EXISTS assignment_submissions;
DROP TABLE IF EXISTS assignments;
DROP TABLE IF EXISTS rubric_criteria;
DROP TABLE IF EXISTS rubrics;
