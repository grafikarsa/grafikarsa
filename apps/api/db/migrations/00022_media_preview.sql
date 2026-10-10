-- +goose Up
-- Preview halaman-1 untuk PDF (PRD §7.6): disimpan sebagai media terpisah
-- supaya bisa disajikan/di-cache seperti gambar biasa.
ALTER TABLE media_assets ADD COLUMN IF NOT EXISTS preview_media_id UUID REFERENCES media_assets (id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_media_preview ON media_assets (preview_media_id);

-- +goose Down
DROP INDEX IF EXISTS idx_media_preview;
ALTER TABLE media_assets DROP COLUMN IF EXISTS preview_media_id;
