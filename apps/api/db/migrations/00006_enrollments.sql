-- +goose Up
-- =============================================================================
-- 5. ENROLLMENT (riwayat kelas, imutabel per tahun ajaran)
-- =============================================================================
CREATE TABLE enrollments (
  id                uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id           uuid NOT NULL REFERENCES users(id),
  class_id          uuid NOT NULL REFERENCES classes(id),
  academic_year_id  uuid NOT NULL REFERENCES academic_years(id),
  -- Hasil di akhir tahun ajaran; NULL selama tahun berjalan.
  outcome           text CHECK (outcome IN ('promoted', 'retained', 'graduated', 'graduation_deferred',
                                            'transferred_out', 'moved')),
  promotion_job_id  uuid,                          -- FK ditambahkan setelah promotion_jobs
  source            text NOT NULL DEFAULT 'import' CHECK (source IN ('import', 'promotion', 'admin', 'legacy')),
  created_at        timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, academic_year_id)
);
CREATE INDEX ix_enrollments_class ON enrollments (class_id);
CREATE INDEX ix_enrollments_user ON enrollments (user_id, academic_year_id);

-- +goose Down
DROP TABLE IF EXISTS enrollments;
