-- +goose Up
-- =============================================================================
-- 11. SOSIAL
-- =============================================================================
CREATE TABLE follows (
  follower_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  followee_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  created_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (follower_id, followee_id),
  CHECK (follower_id <> followee_id)
);
CREATE INDEX ix_follows_followee ON follows (followee_id, created_at DESC);

CREATE TABLE appreciations (
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content_id  uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, content_id)
);
CREATE INDEX ix_appreciations_content ON appreciations (content_id, created_at DESC);

CREATE TABLE bookmarks (
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content_id  uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, content_id)
);
CREATE INDEX ix_bookmarks_user ON bookmarks (user_id, created_at DESC);

CREATE TABLE comments (
  id            uuid PRIMARY KEY DEFAULT uuidv7(),
  content_id    uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  author_id     uuid NOT NULL REFERENCES users(id),
  parent_id     uuid REFERENCES comments(id) ON DELETE CASCADE,   -- balasan 1 level
  body          text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 1000),
  status        text NOT NULL DEFAULT 'visible' CHECK (status IN (
                  'visible', 'hidden_by_moderator', 'deleted_by_author', 'deleted_by_content_owner')),
  reply_count   integer NOT NULL DEFAULT 0,
  edited_at     timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now(),
  CHECK (parent_id IS NULL OR parent_id <> id)
);
CREATE INDEX ix_comments_root ON comments (content_id, created_at, id) WHERE parent_id IS NULL;
CREATE INDEX ix_comments_replies ON comments (parent_id, created_at, id) WHERE parent_id IS NOT NULL;
CREATE INDEX ix_comments_author ON comments (author_id, created_at DESC);
CREATE TRIGGER trg_comments_updated BEFORE UPDATE ON comments FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE comment_mentions (
  comment_id  uuid NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  PRIMARY KEY (comment_id, user_id)
);

CREATE TABLE pinned_contents (
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  content_id  uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  position    smallint NOT NULL CHECK (position BETWEEN 1 AND 10),
  PRIMARY KEY (user_id, content_id),
  UNIQUE (user_id, position) DEFERRABLE INITIALLY DEFERRED
);

-- +goose Down
DROP TABLE IF EXISTS pinned_contents;
DROP TABLE IF EXISTS comment_mentions;
DROP TABLE IF EXISTS comments;
DROP TABLE IF EXISTS bookmarks;
DROP TABLE IF EXISTS appreciations;
DROP TABLE IF EXISTS follows;
