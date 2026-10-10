-- +goose Up
-- Refresh token: batas absolut (tidak bisa diperpanjang selamanya) terpisah
-- dari idle expiry yang di-refresh tiap rotasi.
ALTER TABLE sessions ADD COLUMN absolute_expires_at TIMESTAMPTZ;
UPDATE sessions SET absolute_expires_at = created_at + INTERVAL '720 hours' WHERE absolute_expires_at IS NULL;
ALTER TABLE sessions ALTER COLUMN absolute_expires_at SET NOT NULL;
CREATE INDEX idx_sessions_absolute ON sessions (absolute_expires_at) WHERE revoked_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_sessions_absolute;
ALTER TABLE sessions DROP COLUMN IF EXISTS absolute_expires_at;
