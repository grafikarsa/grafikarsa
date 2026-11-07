-- +goose Up
-- =============================================================================
-- 16. SISTEM
-- =============================================================================
CREATE TABLE settings (
  key         text PRIMARY KEY CHECK (key ~ '^[a-z_]+(\.[a-z_]+)*$'),
  value       jsonb NOT NULL,
  updated_by  uuid REFERENCES users(id),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE import_jobs (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  kind         text NOT NULL CHECK (kind IN ('students', 'teachers', 'staff', 'promotion_exceptions')),
  media_id     uuid REFERENCES media_assets(id),
  status       text NOT NULL DEFAULT 'validating' CHECK (status IN (
                 'validating', 'ready', 'invalid', 'committing', 'committed', 'failed', 'cancelled')),
  total_rows   integer,
  valid_rows   integer,
  errors       jsonb NOT NULL DEFAULT '[]'::jsonb,     -- [{row, field, message}]
  summary      jsonb,
  created_by   uuid NOT NULL REFERENCES users(id),
  created_at   timestamptz NOT NULL DEFAULT now(),
  committed_at timestamptz
);

CREATE TABLE audit_logs (
  id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  occurred_at     timestamptz NOT NULL DEFAULT now(),
  actor_id        uuid REFERENCES users(id) ON DELETE SET NULL,
  actor_ip        inet,
  action          text NOT NULL CHECK (action ~ '^[a-z_]+\.[a-z_.]+$'),   -- mis. content.approve
  target_type     text,
  target_id       uuid,
  scope_major_id  uuid REFERENCES majors(id),
  data            jsonb NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX ix_audit_time ON audit_logs USING brin (occurred_at);
CREATE INDEX ix_audit_target ON audit_logs (target_type, target_id, occurred_at DESC);
CREATE INDEX ix_audit_actor ON audit_logs (actor_id, occurred_at DESC);

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS import_jobs;
DROP TABLE IF EXISTS settings;
