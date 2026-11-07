-- +goose Up
-- =============================================================================
-- 1. AKADEMIK
-- =============================================================================
CREATE TABLE majors (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  code        citext NOT NULL UNIQUE CHECK (code ~ '^[a-z0-9-]{2,20}$'),   -- slug URL: rpl, tkj, dkv
  name        text   NOT NULL CHECK (char_length(name) BETWEEN 2 AND 100),
  short_name  text   NOT NULL CHECK (char_length(short_name) BETWEEN 2 AND 20), -- "RPL" untuk nama kelas
  is_active   boolean NOT NULL DEFAULT true,
  sort_order  smallint NOT NULL DEFAULT 0,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_majors_updated BEFORE UPDATE ON majors FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE academic_years (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  start_year  smallint NOT NULL UNIQUE CHECK (start_year BETWEEN 2000 AND 2100),
  starts_on   date NOT NULL,
  ends_on     date NOT NULL,
  is_current  boolean NOT NULL DEFAULT false,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  CHECK (ends_on > starts_on)
);
-- Tepat satu tahun ajaran berjalan.
CREATE UNIQUE INDEX uq_academic_years_current ON academic_years ((true)) WHERE is_current;
CREATE TRIGGER trg_academic_years_updated BEFORE UPDATE ON academic_years FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE classes (
  id                uuid PRIMARY KEY DEFAULT uuidv7(),
  academic_year_id  uuid NOT NULL REFERENCES academic_years(id),
  major_id          uuid NOT NULL REFERENCES majors(id),
  grade             smallint NOT NULL CHECK (grade IN (10, 11, 12)),
  section           text NOT NULL CHECK (section ~ '^[A-Z]{1,2}$'),           -- rombel: A, B, C
  name              text NOT NULL CHECK (char_length(name) BETWEEN 3 AND 40),  -- "XI RPL B"
  slug              text NOT NULL CHECK (slug ~ '^[a-z0-9-]{3,40}$'),           -- "xi-rpl-b"
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now(),
  UNIQUE (academic_year_id, major_id, grade, section),
  UNIQUE (academic_year_id, slug)
);
CREATE INDEX ix_classes_major ON classes (major_id, academic_year_id);
CREATE TRIGGER trg_classes_updated BEFORE UPDATE ON classes FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS classes;
DROP TABLE IF EXISTS academic_years;
DROP TABLE IF EXISTS majors;
