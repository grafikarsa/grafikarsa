-- +goose Up
-- Tipe notifikasi untuk kenaikan peringkat leaderboard (PRD §10).
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notif_type_valid;
ALTER TABLE notifications ADD CONSTRAINT notif_type_valid CHECK (type IN (
  'new_follower', 'portfolio_liked', 'portfolio_approved',
  'portfolio_changes_requested', 'portfolio_rejected', 'new_comment',
  'reply_comment', 'feedback_updated', 'announcement', 'digest',
  'leaderboard_ranked'
));

-- +goose Down
ALTER TABLE notifications DROP CONSTRAINT IF EXISTS notif_type_valid;
ALTER TABLE notifications ADD CONSTRAINT notif_type_valid CHECK (type IN (
  'new_follower', 'portfolio_liked', 'portfolio_approved',
  'portfolio_changes_requested', 'portfolio_rejected', 'new_comment',
  'reply_comment', 'feedback_updated', 'announcement', 'digest'
));
