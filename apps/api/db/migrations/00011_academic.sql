-- +goose Up
-- Akademik lengkap: enrollments imutabel + promotion jobs + homeroom FK.
ALTER TABLE classes ADD CONSTRAINT classes_homeroom_fk
  FOREIGN KEY (homeroom_teacher_id) REFERENCES users (id) ON DELETE SET NULL;

CREATE TABLE enrollments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  class_id UUID NOT NULL REFERENCES classes (id) ON DELETE RESTRICT,
  academic_year_id UUID NOT NULL REFERENCES academic_years (id) ON DELETE RESTRICT,
  status TEXT NOT NULL DEFAULT 'enrolled',
  is_current BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT enroll_status_valid CHECK (status IN ('enrolled', 'promoted', 'graduated', 'repeated', 'transferred')),
  CONSTRAINT enroll_unique_year UNIQUE (user_id, academic_year_id)
);
CREATE INDEX idx_enrollments_current ON enrollments (user_id) WHERE is_current = TRUE;
CREATE INDEX idx_enrollments_class ON enrollments (class_id) WHERE is_current = TRUE;

CREATE TABLE promotion_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  from_year_id UUID NOT NULL REFERENCES academic_years (id) ON DELETE RESTRICT,
  to_year_id UUID NOT NULL REFERENCES academic_years (id) ON DELETE RESTRICT,
  status TEXT NOT NULL DEFAULT 'draft',
  dryrun_result JSONB NOT NULL DEFAULT '{}',
  exceptions JSONB NOT NULL DEFAULT '[]',
  executed_by UUID REFERENCES users (id) ON DELETE SET NULL,
  executed_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT promo_status_valid CHECK (status IN ('draft', 'previewed', 'approved', 'executing', 'done', 'failed')),
  CONSTRAINT promo_unique UNIQUE (from_year_id, to_year_id)
);

COMMENT ON TABLE enrollments IS 'Histori kelas imutabel; is_current satu per user+tahun';
COMMENT ON TABLE promotion_jobs IS 'Kenaikan kelas dry-run -> approve -> execute transaksional';

-- +goose Down
DROP TABLE IF EXISTS promotion_jobs;
DROP TABLE IF EXISTS enrollments;
ALTER TABLE classes DROP CONSTRAINT IF EXISTS classes_homeroom_fk;
