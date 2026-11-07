-- +goose Up
-- =============================================================================
-- 9. GRUP (jurusan & kelas)
-- =============================================================================
CREATE TABLE groups (
  id               uuid PRIMARY KEY DEFAULT uuidv7(),
  kind             text NOT NULL CHECK (kind IN ('major', 'class')),
  major_id         uuid UNIQUE REFERENCES majors(id),
  class_id         uuid UNIQUE REFERENCES classes(id),
  status           text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'history')),
  avatar_media_id  uuid REFERENCES media_assets(id) ON DELETE SET NULL,
  banner_media_id  uuid REFERENCES media_assets(id) ON DELETE SET NULL,
  description      text CHECK (char_length(description) <= 1000),
  history_since    timestamptz,
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),
  CHECK ((kind = 'major' AND major_id IS NOT NULL AND class_id IS NULL)
      OR (kind = 'class' AND class_id IS NOT NULL AND major_id IS NULL)),
  CHECK ((status = 'history') = (history_since IS NOT NULL))
);
CREATE TRIGGER trg_groups_updated BEFORE UPDATE ON groups FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE group_managers (
  group_id     uuid NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  assigned_by  uuid NOT NULL REFERENCES users(id),
  assigned_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (group_id, user_id)
);
CREATE INDEX ix_group_managers_user ON group_managers (user_id);

-- Index isi grup (mirror). Dipelihara worker saat konten terbit/turun, author berubah,
-- atau enrollment berubah. Feed grup = scan index ini, tanpa join berat.
CREATE TABLE group_contents (
  group_id            uuid NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
  content_id          uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  content_type        text NOT NULL CHECK (content_type IN ('portfolio', 'blog')),
  first_published_at  timestamptz NOT NULL,
  PRIMARY KEY (group_id, content_id)
);
CREATE INDEX ix_group_contents_feed ON group_contents (group_id, first_published_at DESC, content_id DESC);
CREATE INDEX ix_group_contents_type ON group_contents (group_id, content_type, first_published_at DESC, content_id DESC);
CREATE INDEX ix_group_contents_content ON group_contents (content_id);

-- +goose Down
DROP TABLE IF EXISTS group_contents;
DROP TABLE IF EXISTS group_managers;
DROP TABLE IF EXISTS groups;
