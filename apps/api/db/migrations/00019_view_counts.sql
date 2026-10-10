-- +goose Up
-- View counter berbasis HyperLogLog (Dragonfly) yang di-flush berkala ke tabel
-- ini. Menghindari satu INSERT per request (PRD §7.3 / §9.1).
CREATE TABLE IF NOT EXISTS portfolio_view_counts (
  portfolio_id UUID NOT NULL REFERENCES portfolios (id) ON DELETE CASCADE,
  viewed_date DATE NOT NULL,
  unique_views INTEGER NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (portfolio_id, viewed_date)
);
CREATE INDEX IF NOT EXISTS idx_view_counts_date ON portfolio_view_counts (viewed_date DESC);

-- +goose Down
DROP TABLE IF EXISTS portfolio_view_counts;
