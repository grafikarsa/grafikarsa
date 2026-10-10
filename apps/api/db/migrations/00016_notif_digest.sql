-- +goose Up
-- Tipe notifikasi baru untuk ringkasan harian (digest).
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notif_type_valid;
ALTER TABLE notifications ADD CONSTRAINT notif_type_valid CHECK (type IN (
  'new_follower', 'portfolio_liked', 'portfolio_approved',
  'portfolio_changes_requested', 'portfolio_rejected', 'new_comment',
  'reply_comment', 'feedback_updated', 'announcement', 'digest'
));

-- Index untuk query leaderboard terbaru (dipakai GetLeaderboardLatest).
CREATE INDEX IF NOT EXISTS idx_leaderboards_lookup ON leaderboards (period, scope, kind, computed_for DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_leaderboards_lookup;
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notif_type_valid;
ALTER TABLE notifications ADD CONSTRAINT notif_type_valid CHECK (type IN (
  'new_follower', 'portfolio_liked', 'portfolio_approved',
  'portfolio_changes_requested', 'portfolio_rejected', 'new_comment',
  'reply_comment', 'feedback_updated', 'announcement'
));
