-- +goose Up
-- =============================================================================
-- 17. LEGACY (migrasi v1)
-- =============================================================================
CREATE TABLE legacy_id_map (
  source_table  text NOT NULL,
  source_id     uuid NOT NULL,
  target_table  text NOT NULL,
  target_id     uuid NOT NULL,
  migrated_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (source_table, source_id)
);
CREATE INDEX ix_legacy_target ON legacy_id_map (target_table, target_id);

-- Redirect URL lama v1 (/{username}/{slug}, /storage/...) → URL baru.
CREATE TABLE legacy_redirects (
  path         text PRIMARY KEY,
  target_path  text NOT NULL,
  hit_count    bigint NOT NULL DEFAULT 0,
  last_hit_at  timestamptz,
  created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE legacy_quarantine (
  id            uuid PRIMARY KEY DEFAULT uuidv7(),
  source_table  text NOT NULL,
  source_id     uuid NOT NULL,
  reason        text NOT NULL,
  payload       jsonb,
  created_at    timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS legacy_quarantine;
DROP TABLE IF EXISTS legacy_redirects;
DROP TABLE IF EXISTS legacy_id_map;
