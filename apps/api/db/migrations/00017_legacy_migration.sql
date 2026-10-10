-- +goose Up
-- Tabel mapping migrasi v1 -> v2 (idempotent, sesuai PRD §16):
-- setiap baris v1 hanya boleh menghasilkan satu baris v2.
CREATE TABLE IF NOT EXISTS legacy_id_map (
  source_table TEXT NOT NULL,
  source_id TEXT NOT NULL,
  target_table TEXT NOT NULL,
  target_id UUID NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (source_table, source_id)
);
CREATE INDEX IF NOT EXISTS idx_legacy_id_map_target ON legacy_id_map (target_table, target_id);

-- URL lama yang dipertahankan untuk redirect 301 selama 6 bulan (PRD §16.1):
-- /storage/* lama -> /api/media/by-key/* tanpa overwrite buta kolom baru.
CREATE TABLE IF NOT EXISTS legacy_urls (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  old_url TEXT NOT NULL UNIQUE,
  media_id UUID REFERENCES media_assets (id) ON DELETE SET NULL,
  object_key TEXT,
  hit_count BIGINT NOT NULL DEFAULT 0,
  last_hit_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_legacy_urls_media ON legacy_urls (media_id);

-- Block yang gagal validasi saat migrasi tidak di-drop diam-diam (PRD §16.2):
-- masuk quarantine_blocks + laporan, bukan hilang.
CREATE TABLE IF NOT EXISTS quarantine_blocks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  portfolio_id UUID REFERENCES portfolios (id) ON DELETE CASCADE,
  source_block_id TEXT,
  block_type TEXT NOT NULL,
  raw_payload JSONB NOT NULL DEFAULT '{}',
  reason TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_quarantine_blocks_portfolio ON quarantine_blocks (portfolio_id);

-- Status quarantine untuk media mencurigakan (PRD §7.2 lifecycle).
ALTER TABLE media_assets DROP CONSTRAINT IF EXISTS media_status_valid;
ALTER TABLE media_assets ADD CONSTRAINT media_status_valid
  CHECK (status IN ('pending', 'confirmed', 'deleted', 'quarantined'));

-- +goose Down
ALTER TABLE media_assets DROP CONSTRAINT IF EXISTS media_status_valid;
ALTER TABLE media_assets ADD CONSTRAINT media_status_valid
  CHECK (status IN ('pending', 'confirmed', 'deleted'));
DROP TABLE IF EXISTS quarantine_blocks;
DROP TABLE IF EXISTS legacy_urls;
DROP TABLE IF EXISTS legacy_id_map;
