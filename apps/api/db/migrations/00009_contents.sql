-- +goose Up
-- =============================================================================
-- 8. KONTEN (portfolio & blog)
-- =============================================================================
-- contents          : identitas + status lifecycle + pointer ke versi tayang
-- content_drafts    : salinan kerja yang bisa diedit (satu per konten)
-- content_versions  : snapshot IMUTABEL, dibuat setiap kali dikirim ke moderasi
-- -----------------------------------------------------------------------------
CREATE TABLE contents (
  id                  uuid PRIMARY KEY DEFAULT uuidv7(),
  public_id           text NOT NULL UNIQUE CHECK (public_id ~ '^[0-9a-z]{8}$'),  -- dipakai di URL
  type                text NOT NULL CHECK (type IN ('portfolio', 'blog')),
  status              text NOT NULL DEFAULT 'draft' CHECK (status IN (
                        'draft', 'in_review', 'changes_requested', 'published',
                        'archived', 'taken_down', 'rejected', 'deleted')),
  -- Revisi atas konten yang sedang tayang (status = published).
  revision_status     text CHECK (revision_status IN ('editing', 'in_review', 'changes_requested')),
  status_before_delete text CHECK (status_before_delete IN (
                        'draft', 'changes_requested', 'published', 'archived', 'taken_down', 'rejected')),
  owner_id            uuid NOT NULL REFERENCES users(id),
  live_version_id     uuid,                                  -- FK ditambahkan setelah content_versions
  slug                text NOT NULL CHECK (slug ~ '^[a-z0-9-]{1,120}$'),   -- slug kanonik
  -- Denormalisasi dari versi tayang (untuk filter & kartu tanpa join ke snapshot).
  title               text NOT NULL CHECK (char_length(title) BETWEEN 1 AND 150),
  category_id         uuid REFERENCES content_categories(id),
  cover_media_id      uuid REFERENCES media_assets(id),
  excerpt             text CHECK (char_length(excerpt) <= 300),
  reading_minutes     smallint CHECK (reading_minutes > 0),
  search_vector       tsvector,
  first_published_at  timestamptz,
  last_published_at   timestamptz,
  archived_at         timestamptz,
  taken_down_at       timestamptz,
  rejected_at         timestamptz,
  deleted_at          timestamptz,
  purge_after         timestamptz,
  legacy_v1_id        uuid UNIQUE,
  created_at          timestamptz NOT NULL DEFAULT now(),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  CHECK (status <> 'published' OR live_version_id IS NOT NULL),
  CHECK (revision_status IS NULL OR status = 'published'),
  CHECK ((status = 'deleted') = (deleted_at IS NOT NULL)),
  CHECK (status <> 'deleted' OR (purge_after IS NOT NULL AND status_before_delete IS NOT NULL))
);
-- Katalog & feed: konten tayang terbaru (keyset pagination).
CREATE INDEX ix_contents_published ON contents (type, first_published_at DESC, id DESC) WHERE status = 'published';
CREATE INDEX ix_contents_published_cat ON contents (type, category_id, first_published_at DESC, id DESC) WHERE status = 'published';
CREATE INDEX ix_contents_owner ON contents (owner_id, status, updated_at DESC);
CREATE INDEX ix_contents_search ON contents USING gin (search_vector) WHERE status = 'published';
CREATE INDEX ix_contents_title_trgm ON contents USING gin (title gin_trgm_ops) WHERE status = 'published';
CREATE INDEX ix_contents_purge ON contents (purge_after) WHERE status = 'deleted';
CREATE TRIGGER trg_contents_updated BEFORE UPDATE ON contents FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE content_versions (
  id               uuid PRIMARY KEY DEFAULT uuidv7(),
  content_id       uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  number           integer NOT NULL CHECK (number > 0),
  kind             text NOT NULL CHECK (kind IN ('initial', 'resubmission', 'revision', 'legacy')),
  state            text NOT NULL DEFAULT 'pending' CHECK (state IN (
                     'pending',            -- menunggu / sedang ditinjau
                     'approved',           -- disetujui (live atau pernah live)
                     'changes_requested',  -- diminta revisi
                     'rejected',           -- ditolak
                     'withdrawn')),        -- ditarik author sebelum di-claim
  -- Isi snapshot (tidak pernah diubah setelah insert).
  title            text NOT NULL CHECK (char_length(title) BETWEEN 3 AND 150),
  slug             text NOT NULL CHECK (slug ~ '^[a-z0-9-]{1,120}$'),
  category_id      uuid REFERENCES content_categories(id),
  cover_media_id   uuid REFERENCES media_assets(id),
  excerpt          text CHECK (char_length(excerpt) <= 300),
  body             jsonb NOT NULL,          -- portfolio: {blocks:[...]}, blog: ProseMirror doc
  meta             jsonb NOT NULL DEFAULT '{}'::jsonb,  -- field khusus tipe (lihat database.md §5)
  tag_ids          uuid[] NOT NULL DEFAULT '{}' CHECK (cardinality(tag_ids) <= 10),
  tool_ids         uuid[] NOT NULL DEFAULT '{}' CHECK (cardinality(tool_ids) <= 20),
  skill_ids        uuid[] NOT NULL DEFAULT '{}' CHECK (cardinality(skill_ids) <= 20),
  body_schema      smallint NOT NULL DEFAULT 1,              -- versi skema body (migrasi format)
  content_hash     bytea NOT NULL,                           -- SHA-256 JSON kanonik snapshot
  submitted_by     uuid NOT NULL REFERENCES users(id),
  submitted_at     timestamptz NOT NULL DEFAULT now(),
  decided_at       timestamptz,
  UNIQUE (content_id, number)
);
CREATE INDEX ix_versions_content ON content_versions (content_id, number DESC);
-- Paling banyak satu versi pending per konten.
CREATE UNIQUE INDEX uq_versions_one_pending ON content_versions (content_id) WHERE state = 'pending';

ALTER TABLE contents ADD CONSTRAINT fk_contents_live_version
  FOREIGN KEY (live_version_id) REFERENCES content_versions(id) DEFERRABLE INITIALLY DEFERRED;

-- Mencegah perubahan isi snapshot setelah dibuat (hanya state & decided_at boleh berubah).
-- +goose StatementBegin
CREATE FUNCTION forbid_version_body_update() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  IF (NEW.title, NEW.slug, NEW.category_id, NEW.cover_media_id, NEW.excerpt, NEW.body, NEW.meta,
      NEW.tag_ids, NEW.tool_ids, NEW.skill_ids, NEW.content_hash, NEW.content_id, NEW.number)
     IS DISTINCT FROM
     (OLD.title, OLD.slug, OLD.category_id, OLD.cover_media_id, OLD.excerpt, OLD.body, OLD.meta,
      OLD.tag_ids, OLD.tool_ids, OLD.skill_ids, OLD.content_hash, OLD.content_id, OLD.number) THEN
    RAISE EXCEPTION 'content_versions bersifat imutabel (version %)', OLD.id;
  END IF;
  RETURN NEW;
END $$;
-- +goose StatementEnd
CREATE TRIGGER trg_versions_immutable BEFORE UPDATE ON content_versions
  FOR EACH ROW EXECUTE FUNCTION forbid_version_body_update();

CREATE TABLE content_drafts (
  content_id       uuid PRIMARY KEY REFERENCES contents(id) ON DELETE CASCADE,
  base_version_id  uuid REFERENCES content_versions(id),   -- versi tayang yang direvisi (NULL = konten baru)
  title            text NOT NULL DEFAULT '' CHECK (char_length(title) <= 150),
  category_id      uuid REFERENCES content_categories(id),
  cover_media_id   uuid REFERENCES media_assets(id),
  excerpt          text CHECK (char_length(excerpt) <= 300),
  body             jsonb NOT NULL,
  meta             jsonb NOT NULL DEFAULT '{}'::jsonb,
  tag_ids          uuid[] NOT NULL DEFAULT '{}' CHECK (cardinality(tag_ids) <= 10),
  tool_ids         uuid[] NOT NULL DEFAULT '{}' CHECK (cardinality(tool_ids) <= 20),
  skill_ids        uuid[] NOT NULL DEFAULT '{}' CHECK (cardinality(skill_ids) <= 20),
  body_schema      smallint NOT NULL DEFAULT 1,
  lock_version     integer NOT NULL DEFAULT 1,               -- optimistic locking antar-editor
  updated_by       uuid NOT NULL REFERENCES users(id),
  updated_at       timestamptz NOT NULL DEFAULT now()
);

-- Media yang dipakai sebuah versi (diekstrak dari body saat submit).
-- Dipakai untuk: promosi ke gk-public saat disetujui, pencabutan, dan deteksi media yatim.
CREATE TABLE content_version_media (
  version_id  uuid NOT NULL REFERENCES content_versions(id) ON DELETE CASCADE,
  media_id    uuid NOT NULL REFERENCES media_assets(id),
  PRIMARY KEY (version_id, media_id)
);
CREATE INDEX ix_cvm_media ON content_version_media (media_id);

CREATE TABLE content_draft_media (
  content_id  uuid NOT NULL REFERENCES content_drafts(content_id) ON DELETE CASCADE,
  media_id    uuid NOT NULL REFERENCES media_assets(id),
  PRIMARY KEY (content_id, media_id)
);
CREATE INDEX ix_cdm_media ON content_draft_media (media_id);

CREATE TABLE content_authors (
  content_id     uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  user_id        uuid NOT NULL REFERENCES users(id),
  level          text NOT NULL CHECK (level IN ('owner', 'editor', 'contributor')),
  role_label     text CHECK (char_length(role_label) <= 60),   -- "Animator", "Backend"
  position       smallint NOT NULL DEFAULT 0,
  invite_status  text NOT NULL CHECK (invite_status IN ('pending', 'accepted', 'declined', 'expired', 'left', 'removed')),
  invited_by     uuid REFERENCES users(id),
  invited_at     timestamptz NOT NULL DEFAULT now(),
  invite_expires_at timestamptz,
  responded_at   timestamptz,
  PRIMARY KEY (content_id, user_id),
  CHECK (level <> 'owner' OR invite_status = 'accepted'),
  CHECK (invite_status <> 'pending' OR invite_expires_at IS NOT NULL)
);
CREATE UNIQUE INDEX uq_content_one_owner ON content_authors (content_id) WHERE level = 'owner';
CREATE INDEX ix_content_authors_user ON content_authors (user_id, content_id) WHERE invite_status = 'accepted';
CREATE INDEX ix_content_authors_invites ON content_authors (user_id, invited_at DESC) WHERE invite_status = 'pending';
CREATE INDEX ix_content_authors_expiry ON content_authors (invite_expires_at) WHERE invite_status = 'pending';

-- Index filter katalog, dibangun ulang dari versi tayang saat disetujui.
CREATE TABLE content_tags (
  content_id  uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  tag_id      uuid NOT NULL REFERENCES tags(id),
  PRIMARY KEY (content_id, tag_id)
);
CREATE INDEX ix_content_tags_tag ON content_tags (tag_id, content_id);

CREATE TABLE content_tools (
  content_id  uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  tool_id     uuid NOT NULL REFERENCES tools(id),
  PRIMARY KEY (content_id, tool_id)
);
CREATE INDEX ix_content_tools_tool ON content_tools (tool_id, content_id);

CREATE TABLE content_skills (
  content_id  uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  skill_id    uuid NOT NULL REFERENCES skills(id),
  PRIMARY KEY (content_id, skill_id)
);
CREATE INDEX ix_content_skills_skill ON content_skills (skill_id, content_id);

-- Metadata portfolio tayang yang sering difilter (konteks).
CREATE TABLE portfolio_facets (
  content_id   uuid PRIMARY KEY REFERENCES contents(id) ON DELETE CASCADE,
  context      text NOT NULL CHECK (context IN ('assignment', 'internship', 'competition', 'ukk', 'personal', 'other')),
  period_start date,
  period_end   date
);
CREATE INDEX ix_portfolio_facets_context ON portfolio_facets (context);

-- Urutan artikel dalam seri (blog tayang).
CREATE TABLE blog_series_items (
  series_id   uuid NOT NULL REFERENCES blog_series(id) ON DELETE CASCADE,
  content_id  uuid NOT NULL UNIQUE REFERENCES contents(id) ON DELETE CASCADE,
  position    smallint NOT NULL,
  PRIMARY KEY (series_id, content_id),
  UNIQUE (series_id, position) DEFERRABLE INITIALLY DEFERRED
);

-- Statistik dipisah dari contents supaya update counter tidak membuat bloat tabel utama.
CREATE TABLE content_stats (
  content_id         uuid PRIMARY KEY REFERENCES contents(id) ON DELETE CASCADE,
  view_count         bigint  NOT NULL DEFAULT 0,
  unique_view_count  bigint  NOT NULL DEFAULT 0,
  appreciation_count integer NOT NULL DEFAULT 0,
  comment_count      integer NOT NULL DEFAULT 0,
  bookmark_count     integer NOT NULL DEFAULT 0,
  share_count        integer NOT NULL DEFAULT 0,
  popular_score      double precision NOT NULL DEFAULT 0,
  trending_score     double precision NOT NULL DEFAULT 0,
  updated_at         timestamptz NOT NULL DEFAULT now()
) WITH (fillfactor = 80);   -- ruang untuk HOT update
CREATE INDEX ix_stats_popular ON content_stats (popular_score DESC, content_id);
CREATE INDEX ix_stats_trending ON content_stats (trending_score DESC, content_id) WHERE trending_score > 0;

-- Agregat view harian (dari HyperLogLog Valkey, di-flush worker).
CREATE TABLE content_views_daily (
  content_id   uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  day          date NOT NULL,
  views        integer NOT NULL DEFAULT 0,
  unique_views integer NOT NULL DEFAULT 0,
  PRIMARY KEY (content_id, day)
);
CREATE INDEX ix_views_daily_day ON content_views_daily (day);

-- +goose Down
DROP TABLE IF EXISTS content_views_daily;
DROP TABLE IF EXISTS content_stats;
DROP TABLE IF EXISTS blog_series_items;
DROP TABLE IF EXISTS portfolio_facets;
DROP TABLE IF EXISTS content_skills;
DROP TABLE IF EXISTS content_tools;
DROP TABLE IF EXISTS content_tags;
DROP TABLE IF EXISTS content_authors;
DROP TABLE IF EXISTS content_draft_media;
DROP TABLE IF EXISTS content_version_media;
DROP TABLE IF EXISTS content_drafts;
ALTER TABLE contents DROP CONSTRAINT IF EXISTS fk_contents_live_version;
DROP TABLE IF EXISTS content_versions;
DROP FUNCTION IF EXISTS forbid_version_body_update();
DROP TABLE IF EXISTS contents;
