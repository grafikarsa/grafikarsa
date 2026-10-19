-- +goose Up
CREATE TABLE media_assets (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  purpose TEXT NOT NULL,
  object_key TEXT NOT NULL UNIQUE,
  mime TEXT NOT NULL,
  size_bytes BIGINT NOT NULL,
  width INTEGER,
  height INTEGER,
  checksum_sha256 CHAR(64) NOT NULL,
  status TEXT NOT NULL DEFAULT 'pending',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  confirmed_at TIMESTAMPTZ,
  CONSTRAINT media_purpose_valid CHECK (purpose IN ('avatar', 'banner', 'thumbnail', 'block_image', 'document', 'feedback', 'changelog_cover')),
  CONSTRAINT media_status_valid CHECK (status IN ('pending', 'confirmed', 'quarantined', 'deleted')),
  CONSTRAINT media_size_valid CHECK (size_bytes > 0)
);
CREATE INDEX idx_media_owner ON media_assets (owner_id) WHERE status = 'confirmed';
CREATE INDEX idx_media_pending ON media_assets (created_at) WHERE status = 'pending';

-- Kolom profil menunjuk ke media (bukan URL absolut).
ALTER TABLE users ADD COLUMN avatar_media_id UUID REFERENCES media_assets (id) ON DELETE SET NULL;
ALTER TABLE users ADD COLUMN banner_media_id UUID REFERENCES media_assets (id) ON DELETE SET NULL;

COMMENT ON TABLE media_assets IS 'File upload; DB simpan object_key, bukan URL absolut';

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS banner_media_id;
ALTER TABLE users DROP COLUMN IF EXISTS avatar_media_id;
DROP TABLE IF EXISTS media_assets;
