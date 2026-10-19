-- +goose Up
-- Pengaturan global yang bisa diubah tanpa deploy (PRD §9.1):
-- feed_weights (bobot algoritma) + batas upload. Diubah via /admin/settings.
CREATE TABLE IF NOT EXISTS system_settings (
  key TEXT PRIMARY KEY,
  value JSONB NOT NULL DEFAULT '{}',
  updated_by UUID REFERENCES users (id) ON DELETE SET NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO system_settings (key, value) VALUES
  ('feed_weights', '{"quality": 0.30, "recency": 0.25, "engagement": 0.25, "following": 0.10, "interest": 0.10}'),
  ('upload_limits', '{"avatar_mb": 2, "banner_mb": 10, "thumbnail_mb": 5, "block_image_mb": 10, "document_mb": 20, "feedback_mb": 5}')
ON CONFLICT (key) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS system_settings;
