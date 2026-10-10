-- +goose Up
-- Kolom profil + jurusan wajib series + preferensi notifikasi + social links.
ALTER TABLE users ADD COLUMN bio TEXT NOT NULL DEFAULT '';
ALTER TABLE majors ADD COLUMN require_series BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN show_email BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE notification_preferences (
  user_id UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
  portfolio_liked BOOLEAN NOT NULL DEFAULT TRUE,
  new_comment BOOLEAN NOT NULL DEFAULT TRUE,
  new_follower BOOLEAN NOT NULL DEFAULT TRUE,
  digest_daily BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE user_social_links (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  platform TEXT NOT NULL,
  url TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT social_unique UNIQUE (user_id, platform)
);

-- +goose Down
DROP TABLE IF EXISTS user_social_links;
DROP TABLE IF EXISTS notification_preferences;
ALTER TABLE users DROP COLUMN IF EXISTS show_email;
ALTER TABLE majors DROP COLUMN IF EXISTS require_series;
ALTER TABLE users DROP COLUMN IF EXISTS bio;
