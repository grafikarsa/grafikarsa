-- +goose Up
-- =============================================================================
-- 2. MEDIA (dibuat lebih awal karena direferensikan banyak tabel)
-- =============================================================================
CREATE TABLE media_assets (
  id                 uuid PRIMARY KEY DEFAULT uuidv7(),
  owner_id           uuid,                                   -- FK ditambahkan setelah users
  kind               text NOT NULL CHECK (kind IN ('image', 'pdf', 'model3d', 'document')),
  purpose            text NOT NULL CHECK (purpose IN (
                       'content', 'avatar', 'banner', 'group_avatar', 'group_banner',
                       'experience_attachment', 'badge_icon', 'curation_cover', 'import')),
  original_key       text NOT NULL UNIQUE,                   -- object key di gk-private
  original_filename  text CHECK (char_length(original_filename) <= 255), -- hanya untuk tampilan/unduh
  mime_type          text NOT NULL,
  size_bytes         bigint NOT NULL CHECK (size_bytes > 0),
  checksum_sha256    bytea,
  width              integer CHECK (width > 0),
  height             integer CHECK (height > 0),
  page_count         integer CHECK (page_count > 0),
  status             text NOT NULL DEFAULT 'uploaded'
                       CHECK (status IN ('uploaded', 'processing', 'ready', 'failed')),
  failure_reason     text,
  -- {"w320": {"key": "...", "format": "avif", "w": 320, "h": 180, "bytes": 12345}, ...}
  variants           jsonb NOT NULL DEFAULT '{}'::jsonb,
  -- true = varian sudah disalin ke gk-public (konten disetujui / avatar / banner).
  is_public          boolean NOT NULL DEFAULT false,
  public_prefix      text,                                    -- prefix key di gk-public
  blurhash           text,                                    -- placeholder saat loading
  dominant_color     text CHECK (dominant_color ~ '^#[0-9a-f]{6}$'),
  created_at         timestamptz NOT NULL DEFAULT now(),
  processed_at       timestamptz,
  deleted_at         timestamptz
);
CREATE INDEX ix_media_owner ON media_assets (owner_id, created_at DESC);
CREATE INDEX ix_media_pending ON media_assets (created_at) WHERE status IN ('uploaded', 'processing');

-- +goose Down
DROP TABLE IF EXISTS media_assets;
