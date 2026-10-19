-- +goose Up
CREATE INDEX idx_portfolios_title_trgm ON portfolios USING gin (title gin_trgm_ops) WHERE status = 'published' AND deleted_at IS NULL;
CREATE INDEX idx_portfolios_tsv ON portfolios USING gin (search_tsv) WHERE status = 'published' AND deleted_at IS NULL;
CREATE INDEX idx_portfolios_feed ON portfolios (status, published_at DESC, owner_id)
  INCLUDE (id, title, slug) WHERE status = 'published' AND deleted_at IS NULL;
CREATE INDEX idx_portfolios_pending ON portfolios (submitted_at) WHERE status = 'pending_review';
CREATE INDEX idx_portfolios_owner ON portfolios (owner_id, status) WHERE deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_portfolios_owner;
DROP INDEX IF EXISTS idx_portfolios_pending;
DROP INDEX IF EXISTS idx_portfolios_feed;
DROP INDEX IF EXISTS idx_portfolios_tsv;
DROP INDEX IF EXISTS idx_portfolios_title_trgm;
