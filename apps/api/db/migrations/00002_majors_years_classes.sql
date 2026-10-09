-- +goose Up
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE majors (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code CITEXT NOT NULL UNIQUE,
  name VARCHAR(100) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_majors_updated_at
  BEFORE UPDATE ON majors FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE academic_years (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  start_year INTEGER NOT NULL UNIQUE,
  is_active BOOLEAN NOT NULL DEFAULT FALSE,
  promotion_month SMALLINT NOT NULL DEFAULT 7,
  promotion_day SMALLINT NOT NULL DEFAULT 1,
  starts_at DATE,
  ends_at DATE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT years_valid CHECK (start_year >= 2000 AND start_year <= 2100),
  CONSTRAINT promo_month_valid CHECK (promotion_month BETWEEN 1 AND 12),
  CONSTRAINT promo_day_valid CHECK (promotion_day BETWEEN 1 AND 31)
);
CREATE UNIQUE INDEX idx_years_active ON academic_years (is_active) WHERE is_active = TRUE;
CREATE TRIGGER trg_years_updated_at
  BEFORE UPDATE ON academic_years FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE classes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  academic_year_id UUID NOT NULL REFERENCES academic_years (id) ON DELETE RESTRICT,
  major_id UUID NOT NULL REFERENCES majors (id) ON DELETE RESTRICT,
  grade SMALLINT NOT NULL,
  class_group CHAR(1) NOT NULL,
  name VARCHAR(20) NOT NULL,
  -- TODO(Fase 3): FK ke users setelah tabel users ada di migrasi 00003.
  -- Sengaja tanpa REFERENCES agar urutan migrasi tetap linear.
  homeroom_teacher_id UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT class_grade_valid CHECK (grade IN (10, 11, 12)),
  CONSTRAINT class_group_valid CHECK (class_group ~ '^[A-Z]$'),
  CONSTRAINT class_unique_per_year UNIQUE (academic_year_id, major_id, grade, class_group)
);
CREATE INDEX idx_classes_year ON classes (academic_year_id);
CREATE INDEX idx_classes_major ON classes (major_id);
CREATE TRIGGER trg_classes_updated_at
  BEFORE UPDATE ON classes FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE majors IS 'Master jurusan/program keahlian';
COMMENT ON TABLE academic_years IS 'Tahun ajaran; hanya satu yang aktif';
COMMENT ON TABLE classes IS 'Rombel per tahun ajaran; homeroom FK menyusul Fase 3';

-- +goose Down
DROP TABLE IF EXISTS classes;
DROP TABLE IF EXISTS academic_years;
DROP TABLE IF EXISTS majors;
DROP FUNCTION IF EXISTS set_updated_at();
