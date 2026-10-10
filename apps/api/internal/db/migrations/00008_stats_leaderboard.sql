-- +goose Up
CREATE TABLE portfolio_stats (
  portfolio_id UUID PRIMARY KEY REFERENCES portfolios (id) ON DELETE CASCADE,
  likes INTEGER NOT NULL DEFAULT 0,
  views_7d INTEGER NOT NULL DEFAULT 0,
  views_30d INTEGER NOT NULL DEFAULT 0,
  comments INTEGER NOT NULL DEFAULT 0,
  assessment_avg NUMERIC(4, 2) NOT NULL DEFAULT 0,
  quality NUMERIC(5, 4) NOT NULL DEFAULT 0,
  engagement NUMERIC(5, 4) NOT NULL DEFAULT 0,
  trending NUMERIC(5, 4) NOT NULL DEFAULT 0,
  computed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_portfolio_stats_trending ON portfolio_stats (trending DESC);

CREATE TABLE leaderboards (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  period TEXT NOT NULL,
  scope TEXT NOT NULL,
  kind TEXT NOT NULL,
  computed_for DATE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT lb_period_valid CHECK (period IN ('daily', 'weekly', 'monthly')),
  CONSTRAINT lb_kind_valid CHECK (kind IN ('best_students', 'best_projects')),
  UNIQUE (period, scope, kind, computed_for)
);

CREATE TABLE leaderboard_entries (
  leaderboard_id UUID NOT NULL REFERENCES leaderboards (id) ON DELETE CASCADE,
  rank INTEGER NOT NULL,
  subject_user_id UUID REFERENCES users (id) ON DELETE CASCADE,
  subject_portfolio_id UUID REFERENCES portfolios (id) ON DELETE CASCADE,
  score NUMERIC(7, 3) NOT NULL,
  breakdown JSONB NOT NULL DEFAULT '{}',
  delta_rank INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (leaderboard_id, rank)
);

CREATE TABLE outbox (
  id BIGSERIAL PRIMARY KEY,
  topic TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'pending',
  attempts INTEGER NOT NULL DEFAULT 0,
  next_retry_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT outbox_status_valid CHECK (status IN ('pending', 'sent', 'failed'))
);
CREATE INDEX idx_outbox_pending ON outbox (next_retry_at) WHERE status = 'pending';

CREATE TABLE audit_logs (
  id BIGSERIAL PRIMARY KEY,
  actor_id UUID REFERENCES users (id) ON DELETE SET NULL,
  action TEXT NOT NULL,
  entity TEXT NOT NULL,
  entity_id TEXT NOT NULL DEFAULT '',
  diff JSONB NOT NULL DEFAULT '{}',
  ip INET,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_entity ON audit_logs (entity, created_at DESC);

COMMENT ON TABLE portfolio_stats IS 'Materialized oleh worker; sumber feed lapis 1';
COMMENT ON TABLE outbox IS 'Transactional outbox untuk worker (notif, stats, leaderboard)';

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS outbox;
DROP TABLE IF EXISTS leaderboard_entries;
DROP TABLE IF EXISTS leaderboards;
DROP TABLE IF EXISTS portfolio_stats;
