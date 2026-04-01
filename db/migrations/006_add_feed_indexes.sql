-- Migration: Add indexes for feed optimization
-- Created: 2026-04-01
-- Description: Adds database indexes to optimize feed queries for production

-- Index for published portfolios sorted by date
CREATE INDEX IF NOT EXISTS idx_portfolios_published_status 
ON portfolios(published_at DESC, status) 
WHERE status = 'published' AND deleted_at IS NULL;

-- Index for user's published portfolios
CREATE INDEX IF NOT EXISTS idx_portfolios_user_published 
ON portfolios(user_id, published_at DESC) 
WHERE status = 'published' AND deleted_at IS NULL;

-- Index for portfolio likes lookup
CREATE INDEX IF NOT EXISTS idx_portfolio_likes_user_portfolio 
ON portfolio_likes(user_id, portfolio_id);

-- Index for portfolio views count
CREATE INDEX IF NOT EXISTS idx_portfolio_views_portfolio 
ON portfolio_views(portfolio_id);

-- Index for follow relationships
CREATE INDEX IF NOT EXISTS idx_follows_follower_following 
ON follows(follower_id, following_id);

-- Composite index for smart feed queries
CREATE INDEX IF NOT EXISTS idx_portfolios_smart_feed 
ON portfolios(status, published_at DESC, user_id) 
INCLUDE (id, judul, slug, thumbnail_url, created_at)
WHERE status = 'published' AND deleted_at IS NULL;

-- Index for portfolio tags join
CREATE INDEX IF NOT EXISTS idx_portfolio_tags_portfolio 
ON portfolio_tags(portfolio_id, tag_id);

-- Index for portfolio tags reverse lookup
CREATE INDEX IF NOT EXISTS idx_portfolio_tags_tag 
ON portfolio_tags(tag_id, portfolio_id);

-- Add comments for documentation
COMMENT ON INDEX idx_portfolios_published_status IS 'Optimizes queries for published portfolios sorted by date';
COMMENT ON INDEX idx_portfolios_user_published IS 'Optimizes queries for user portfolios';
COMMENT ON INDEX idx_portfolio_likes_user_portfolio IS 'Optimizes batch like queries (N+1 prevention)';
COMMENT ON INDEX idx_portfolio_views_portfolio IS 'Optimizes view count queries';
COMMENT ON INDEX idx_follows_follower_following IS 'Optimizes follow relationship queries';
COMMENT ON INDEX idx_portfolios_smart_feed IS 'Optimizes smart feed algorithm queries';
COMMENT ON INDEX idx_portfolio_tags_portfolio IS 'Optimizes tag lookup for portfolios';
COMMENT ON INDEX idx_portfolio_tags_tag IS 'Optimizes portfolio lookup by tag';
