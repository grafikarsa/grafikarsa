-- +goose Up
-- =============================================================================
-- 7. TAKSONOMI KONTEN
-- =============================================================================
CREATE TABLE content_categories (
  id            uuid PRIMARY KEY DEFAULT uuidv7(),
  content_type  text NOT NULL CHECK (content_type IN ('portfolio', 'blog')),
  slug          citext NOT NULL CHECK (slug ~ '^[a-z0-9-]{2,60}$'),
  name          text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 60),
  description   text CHECK (char_length(description) <= 300),
  sort_order    smallint NOT NULL DEFAULT 0,
  is_active     boolean NOT NULL DEFAULT true,
  created_at    timestamptz NOT NULL DEFAULT now(),
  UNIQUE (content_type, slug)
);

-- Jurusan yang relevan untuk sebuah kategori (dipakai untuk saran kategori di editor).
CREATE TABLE content_category_majors (
  category_id  uuid NOT NULL REFERENCES content_categories(id) ON DELETE CASCADE,
  major_id     uuid NOT NULL REFERENCES majors(id) ON DELETE CASCADE,
  PRIMARY KEY (category_id, major_id)
);

CREATE TABLE tags (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  slug        citext NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]{1,40}$'),
  name        text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 40),
  is_blocked  boolean NOT NULL DEFAULT false,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_tags_name_trgm ON tags USING gin (name gin_trgm_ops);

CREATE TABLE blog_series (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  owner_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title        text NOT NULL CHECK (char_length(title) BETWEEN 3 AND 120),
  slug         text NOT NULL CHECK (slug ~ '^[a-z0-9-]{1,140}$'),
  description  text CHECK (char_length(description) <= 500),
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now(),
  UNIQUE (owner_id, slug)
);
CREATE TRIGGER trg_blog_series_updated BEFORE UPDATE ON blog_series FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS blog_series;
DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS content_category_majors;
DROP TABLE IF EXISTS content_categories;
