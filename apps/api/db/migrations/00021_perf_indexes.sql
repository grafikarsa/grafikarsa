-- +goose Up
-- Index arah balik yang hilang: CountLikes/IsLiked per portfolio dan
-- CountFollowers/ListFollowers sebelumnya seq scan (PK komposit berawalan owner).
CREATE INDEX IF NOT EXISTS idx_portfolio_likes_portfolio ON portfolio_likes (portfolio_id);
CREATE INDEX IF NOT EXISTS idx_follows_following ON follows (following_id);
-- Daftar notifikasi tanpa filter is_read tidak memakai partial index unread.
CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications (user_id, created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_notifications_user_created;
DROP INDEX IF EXISTS idx_follows_following;
DROP INDEX IF EXISTS idx_portfolio_likes_portfolio;
