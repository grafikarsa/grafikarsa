-- +goose Up
-- =============================================================================
-- 13. KENAIKAN KELAS
-- =============================================================================
CREATE TABLE promotion_jobs (
  id                     uuid PRIMARY KEY DEFAULT uuidv7(),
  from_academic_year_id  uuid NOT NULL REFERENCES academic_years(id),
  to_academic_year_id    uuid NOT NULL REFERENCES academic_years(id),
  status                 text NOT NULL DEFAULT 'draft' CHECK (status IN (
                           'draft', 'approved', 'executing', 'executed', 'failed', 'rolled_back', 'cancelled')),
  scheduled_for          date NOT NULL,
  dry_run_summary        jsonb,
  dry_run_at             timestamptz,
  conflict_count         integer NOT NULL DEFAULT 0,
  created_by             uuid NOT NULL REFERENCES users(id),
  approved_by            uuid REFERENCES users(id),
  approved_at            timestamptz,
  executed_at            timestamptz,
  rollback_deadline      timestamptz,
  rolled_back_by         uuid REFERENCES users(id),
  rolled_back_at         timestamptz,
  failure_reason         text,
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),
  CHECK (from_academic_year_id <> to_academic_year_id),
  CHECK (status NOT IN ('approved', 'executing', 'executed') OR (approved_by IS NOT NULL AND conflict_count = 0)),
  CHECK (status <> 'executed' OR (executed_at IS NOT NULL AND rollback_deadline IS NOT NULL))
);
CREATE UNIQUE INDEX uq_promotion_jobs_target ON promotion_jobs (to_academic_year_id)
  WHERE status NOT IN ('cancelled', 'rolled_back', 'failed');
CREATE TRIGGER trg_promotion_jobs_updated BEFORE UPDATE ON promotion_jobs FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE enrollments ADD CONSTRAINT fk_enrollments_promotion_job
  FOREIGN KEY (promotion_job_id) REFERENCES promotion_jobs(id);

CREATE TABLE promotion_class_mappings (
  job_id         uuid NOT NULL REFERENCES promotion_jobs(id) ON DELETE CASCADE,
  from_class_id  uuid NOT NULL REFERENCES classes(id),
  to_class_id    uuid REFERENCES classes(id),           -- NULL untuk kelas XII (lulus)
  PRIMARY KEY (job_id, from_class_id)
);

CREATE TABLE promotion_items (
  id                  uuid PRIMARY KEY DEFAULT uuidv7(),
  job_id              uuid NOT NULL REFERENCES promotion_jobs(id) ON DELETE CASCADE,
  student_id          uuid NOT NULL REFERENCES users(id),
  from_enrollment_id  uuid NOT NULL REFERENCES enrollments(id),
  action              text NOT NULL CHECK (action IN (
                        'promote', 'retain', 'move', 'transfer_out', 'graduate', 'defer_graduation')),
  target_class_id     uuid REFERENCES classes(id),
  is_exception        boolean NOT NULL DEFAULT false,   -- diubah dari default pemetaan
  note                text CHECK (char_length(note) <= 300),
  updated_by          uuid REFERENCES users(id),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  UNIQUE (job_id, student_id),
  CHECK ((action IN ('transfer_out', 'graduate')) = (target_class_id IS NULL))
);
CREATE INDEX ix_promotion_items_job ON promotion_items (job_id, action);

-- +goose Down
DROP TABLE IF EXISTS promotion_items;
DROP TABLE IF EXISTS promotion_class_mappings;
ALTER TABLE enrollments DROP CONSTRAINT IF EXISTS fk_enrollments_promotion_job;
DROP TABLE IF EXISTS promotion_jobs;
