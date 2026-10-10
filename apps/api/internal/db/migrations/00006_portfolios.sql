-- +goose Up
CREATE TABLE tags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  name VARCHAR(100) NOT NULL UNIQUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE series (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  name VARCHAR(150) NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TRIGGER trg_series_updated_at
  BEFORE UPDATE ON series FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE series_blocks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  series_id UUID NOT NULL REFERENCES series (id) ON DELETE CASCADE,
  block_type TEXT NOT NULL,
  position INTEGER NOT NULL,
  instruction TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT series_blocks_unique UNIQUE (series_id, position)
);

CREATE TABLE portfolios (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  title VARCHAR(200) NOT NULL,
  slug TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  thumbnail_media_id UUID REFERENCES media_assets (id) ON DELETE SET NULL,
  series_id UUID REFERENCES series (id) ON DELETE SET NULL,
  draft_payload JSONB,
  submitted_snapshot JSONB,
  published_at TIMESTAMPTZ,
  submitted_at TIMESTAMPTZ,
  reviewed_by UUID REFERENCES users (id) ON DELETE SET NULL,
  reviewed_at TIMESTAMPTZ,
  review_note TEXT NOT NULL DEFAULT '',
  parent_version_id UUID REFERENCES portfolios (id) ON DELETE SET NULL,
  search_tsv TSVECTOR GENERATED ALWAYS AS (
    to_tsvector('simple', coalesce(title, '') || ' ' || coalesce(review_note, ''))
  ) STORED,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT portfolios_status_valid CHECK (status IN ('draft', 'pending_review', 'changes_requested', 'published', 'rejected', 'archived')),
  CONSTRAINT portfolios_slug_unique UNIQUE (owner_id, slug)
);
CREATE TRIGGER trg_portfolios_updated_at
  BEFORE UPDATE ON portfolios FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE portfolio_tags (
  portfolio_id UUID NOT NULL REFERENCES portfolios (id) ON DELETE CASCADE,
  tag_id UUID NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
  PRIMARY KEY (portfolio_id, tag_id)
);

CREATE TABLE portfolio_blocks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  portfolio_id UUID NOT NULL REFERENCES portfolios (id) ON DELETE CASCADE,
  block_type TEXT NOT NULL,
  position INTEGER NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}',
  asset_media_id UUID REFERENCES media_assets (id) ON DELETE SET NULL,
  is_from_series BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT blocks_type_valid CHECK (block_type IN ('text', 'image', 'table', 'youtube', 'button', 'embed', 'figma', 'canva', 'ppt', 'pdf', 'doc', 'link')),
  CONSTRAINT blocks_unique UNIQUE (portfolio_id, position) DEFERRABLE INITIALLY DEFERRED
);

COMMENT ON TABLE portfolios IS 'Katalog; slug unik per owner; visibility via status';
COMMENT ON TABLE portfolio_blocks IS 'Block modular; payload tervalidasi JSON Schema per type';

-- +goose Down
DROP TABLE IF EXISTS portfolio_blocks;
DROP TABLE IF EXISTS portfolio_tags;
DROP TABLE IF EXISTS portfolios;
DROP TABLE IF EXISTS series_blocks;
DROP TABLE IF EXISTS series;
DROP TABLE IF EXISTS tags;
