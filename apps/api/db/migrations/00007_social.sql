-- +goose Up
CREATE TABLE follows (
  follower_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  following_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT follows_no_self CHECK (follower_id != following_id),
  PRIMARY KEY (follower_id, following_id)
);

CREATE TABLE portfolio_likes (
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  portfolio_id UUID NOT NULL REFERENCES portfolios (id) ON DELETE CASCADE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, portfolio_id)
);

CREATE TABLE comments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  portfolio_id UUID NOT NULL REFERENCES portfolios (id) ON DELETE CASCADE,
  author_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  parent_id UUID REFERENCES comments (id) ON DELETE CASCADE,
  body_md TEXT NOT NULL,
  body_html TEXT NOT NULL DEFAULT '',
  is_edited BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT comments_body_len CHECK (char_length(body_md) BETWEEN 1 AND 2000)
);
CREATE INDEX idx_comments_portfolio ON comments (portfolio_id, created_at) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_comments_updated_at
  BEFORE UPDATE ON comments FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE portfolio_views (
  id BIGSERIAL PRIMARY KEY,
  portfolio_id UUID NOT NULL REFERENCES portfolios (id) ON DELETE CASCADE,
  viewer_id UUID REFERENCES users (id) ON DELETE SET NULL,
  session_id TEXT,
  viewed_date DATE NOT NULL DEFAULT CURRENT_DATE,
  CONSTRAINT views_owner_or_session CHECK (viewer_id IS NOT NULL OR session_id IS NOT NULL),
  UNIQUE (portfolio_id, viewer_id, viewed_date),
  UNIQUE (portfolio_id, session_id, viewed_date)
);

CREATE TABLE notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  type TEXT NOT NULL,
  title VARCHAR(200) NOT NULL,
  body TEXT NOT NULL DEFAULT '',
  data JSONB NOT NULL DEFAULT '{}',
  is_read BOOLEAN NOT NULL DEFAULT FALSE,
  read_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT notif_type_valid CHECK (type IN ('new_follower', 'portfolio_liked', 'portfolio_approved', 'portfolio_changes_requested', 'portfolio_rejected', 'new_comment', 'reply_comment', 'feedback_updated', 'announcement'))
);
CREATE INDEX idx_notifications_unread ON notifications (user_id, created_at DESC) WHERE is_read = FALSE;

CREATE TABLE user_interests (
  user_id UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
  tag_weights JSONB NOT NULL DEFAULT '{}',
  major_weights JSONB NOT NULL DEFAULT '{}',
  total_likes INTEGER NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE feed_preferences (
  user_id UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
  algorithm TEXT NOT NULL DEFAULT 'smart',
  CONSTRAINT feed_algo_valid CHECK (algorithm IN ('smart', 'recent', 'following'))
);

COMMENT ON TABLE portfolio_views IS 'Agregat harian unik; bukan row per-view abadi';

-- +goose Down
DROP TABLE IF EXISTS feed_preferences;
DROP TABLE IF EXISTS user_interests;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS portfolio_views;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS portfolio_likes;
DROP TABLE IF EXISTS follows;
