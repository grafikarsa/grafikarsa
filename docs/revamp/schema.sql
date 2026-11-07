-- =============================================================================
-- Grafikarsa Revamp — Skema Database (PostgreSQL 18)
-- =============================================================================
-- Sumber kebenaran skema. Penjelasan desain ada di docs/revamp/database.md.
-- Saat implementasi, file ini dipecah menjadi migrasi goose berurutan
-- (lihat database.md §9). Tabel River (antrian job) dibuat oleh migrasi River
-- sendiri dan tidak ditulis di sini.
--
-- Konvensi:
--   * Identifier DB berbahasa Inggris, label UI berbahasa Indonesia.
--   * PK uuid v7 (uuidv7() bawaan PG18): urut waktu, ramah index B-tree.
--   * Semua waktu timestamptz (UTC). Zona tampilan Asia/Jakarta di aplikasi.
--   * Nilai enumerasi memakai text + CHECK (mudah ditambah lewat migrasi).
--   * Panjang teks dibatasi CHECK di DB, bukan hanya di aplikasi.
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- -----------------------------------------------------------------------------
-- Util
-- -----------------------------------------------------------------------------
CREATE FUNCTION set_updated_at() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  NEW.updated_at := now();
  RETURN NEW;
END $$;

-- =============================================================================
-- 1. AKADEMIK
-- =============================================================================
CREATE TABLE majors (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  code        citext NOT NULL UNIQUE CHECK (code ~ '^[a-z0-9-]{2,20}$'),   -- slug URL: rpl, tkj, dkv
  name        text   NOT NULL CHECK (char_length(name) BETWEEN 2 AND 100),
  short_name  text   NOT NULL CHECK (char_length(short_name) BETWEEN 2 AND 20), -- "RPL" untuk nama kelas
  is_active   boolean NOT NULL DEFAULT true,
  sort_order  smallint NOT NULL DEFAULT 0,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_majors_updated BEFORE UPDATE ON majors FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE academic_years (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  start_year  smallint NOT NULL UNIQUE CHECK (start_year BETWEEN 2000 AND 2100),
  starts_on   date NOT NULL,
  ends_on     date NOT NULL,
  is_current  boolean NOT NULL DEFAULT false,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  CHECK (ends_on > starts_on)
);
-- Tepat satu tahun ajaran berjalan.
CREATE UNIQUE INDEX uq_academic_years_current ON academic_years ((true)) WHERE is_current;
CREATE TRIGGER trg_academic_years_updated BEFORE UPDATE ON academic_years FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE classes (
  id                uuid PRIMARY KEY DEFAULT uuidv7(),
  academic_year_id  uuid NOT NULL REFERENCES academic_years(id),
  major_id          uuid NOT NULL REFERENCES majors(id),
  grade             smallint NOT NULL CHECK (grade IN (10, 11, 12)),
  section           text NOT NULL CHECK (section ~ '^[A-Z]{1,2}$'),           -- rombel: A, B, C
  name              text NOT NULL CHECK (char_length(name) BETWEEN 3 AND 40),  -- "XI RPL B"
  slug              text NOT NULL CHECK (slug ~ '^[a-z0-9-]{3,40}$'),           -- "xi-rpl-b"
  created_at        timestamptz NOT NULL DEFAULT now(),
  updated_at        timestamptz NOT NULL DEFAULT now(),
  UNIQUE (academic_year_id, major_id, grade, section),
  UNIQUE (academic_year_id, slug)
);
CREATE INDEX ix_classes_major ON classes (major_id, academic_year_id);
CREATE TRIGGER trg_classes_updated BEFORE UPDATE ON classes FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- =============================================================================
-- 2. MEDIA (dibuat lebih awal karena direferensikan banyak tabel)
-- =============================================================================
CREATE TABLE media_assets (
  id                 uuid PRIMARY KEY DEFAULT uuidv7(),
  owner_id           uuid,                                   -- FK ditambahkan setelah users
  kind               text NOT NULL CHECK (kind IN ('image', 'pdf', 'model3d', 'document')),
  purpose            text NOT NULL CHECK (purpose IN (
                       'content', 'avatar', 'banner', 'group_avatar', 'group_banner',
                       'experience_attachment', 'badge_icon', 'curation_cover', 'import')),
  original_key       text NOT NULL UNIQUE,                   -- object key di gk-private
  original_filename  text CHECK (char_length(original_filename) <= 255), -- hanya untuk tampilan/unduh
  mime_type          text NOT NULL,
  size_bytes         bigint NOT NULL CHECK (size_bytes > 0),
  checksum_sha256    bytea,
  width              integer CHECK (width > 0),
  height             integer CHECK (height > 0),
  page_count         integer CHECK (page_count > 0),
  status             text NOT NULL DEFAULT 'uploaded'
                       CHECK (status IN ('uploaded', 'processing', 'ready', 'failed')),
  failure_reason     text,
  -- {"w320": {"key": "...", "format": "avif", "w": 320, "h": 180, "bytes": 12345}, ...}
  variants           jsonb NOT NULL DEFAULT '{}'::jsonb,
  -- true = varian sudah disalin ke gk-public (konten disetujui / avatar / banner).
  is_public          boolean NOT NULL DEFAULT false,
  public_prefix      text,                                    -- prefix key di gk-public
  blurhash           text,                                    -- placeholder saat loading
  dominant_color     text CHECK (dominant_color ~ '^#[0-9a-f]{6}$'),
  created_at         timestamptz NOT NULL DEFAULT now(),
  processed_at       timestamptz,
  deleted_at         timestamptz
);
CREATE INDEX ix_media_owner ON media_assets (owner_id, created_at DESC);
CREATE INDEX ix_media_pending ON media_assets (created_at) WHERE status IN ('uploaded', 'processing');

-- =============================================================================
-- 3. PENGGUNA & AUTENTIKASI
-- =============================================================================
CREATE TABLE users (
  id                     uuid PRIMARY KEY DEFAULT uuidv7(),
  account_type           text NOT NULL CHECK (account_type IN ('student', 'teacher', 'staff')),
  status                 text NOT NULL DEFAULT 'active'
                           CHECK (status IN ('active', 'alumni', 'transferred', 'inactive')),
  nisn                   text UNIQUE CHECK (nisn ~ '^[0-9]{10}$'),
  staff_number           text UNIQUE CHECK (char_length(staff_number) BETWEEN 3 AND 30), -- NIP / ID staf
  username               citext UNIQUE CHECK (username ~ '^[a-z0-9](?:[a-z0-9._]{1,28})[a-z0-9_]$'),
  display_name           text NOT NULL CHECK (char_length(display_name) BETWEEN 1 AND 100),
  -- Hash format PHC. Argon2id untuk akun baru; bcrypt hasil migrasi v1 di-rehash saat login.
  password_hash          text NOT NULL,
  -- initial   = sandi awal dari sekolah (hanya untuk aktivasi)
  -- temporary = sandi sementara hasil reset Admin
  -- user      = sandi yang dibuat user sendiri
  password_kind          text NOT NULL DEFAULT 'initial' CHECK (password_kind IN ('initial', 'temporary', 'user')),
  password_expires_at    timestamptz,
  password_changed_at    timestamptz,
  entry_year             smallint CHECK (entry_year BETWEEN 1950 AND 2100),     -- angkatan (siswa)
  graduation_year        smallint CHECK (graduation_year BETWEEN 1950 AND 2100),
  activated_at           timestamptz,
  recovery_codes_confirmed_at timestamptz,
  username_changed_at    timestamptz,
  content_blocked_at     timestamptz,           -- capability content.create dicabut
  content_blocked_reason text,
  last_login_at          timestamptz,
  deactivated_at         timestamptz,
  legacy_v1_id           uuid UNIQUE,
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),
  CHECK (account_type <> 'student' OR nisn IS NOT NULL OR legacy_v1_id IS NOT NULL),
  CHECK (account_type = 'student' OR status <> 'alumni'),
  CHECK (activated_at IS NULL OR username IS NOT NULL),
  CHECK (password_kind = 'user' OR password_expires_at IS NOT NULL)
);
CREATE INDEX ix_users_name_trgm ON users USING gin (display_name gin_trgm_ops);
CREATE INDEX ix_users_username_trgm ON users USING gin ((username::text) gin_trgm_ops);
CREATE INDEX ix_users_type_status ON users (account_type, status);
CREATE TRIGGER trg_users_updated BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE media_assets ADD CONSTRAINT fk_media_owner FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL;

CREATE TABLE reserved_usernames (
  username    citext PRIMARY KEY,
  reason      text NOT NULL CHECK (reason IN ('route', 'official', 'offensive', 'other')),
  created_by  uuid REFERENCES users(id),
  created_at  timestamptz NOT NULL DEFAULT now()
);

-- Username lama di-redirect 90 hari, lalu dilepas (released_at diisi worker).
CREATE TABLE username_history (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  old_username citext NOT NULL,
  changed_at   timestamptz NOT NULL DEFAULT now(),
  redirect_until timestamptz NOT NULL,
  released_at  timestamptz
);
CREATE INDEX ix_username_history_old ON username_history (old_username) WHERE released_at IS NULL;

CREATE TABLE sessions (
  id              uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id         uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash      bytea NOT NULL UNIQUE,         -- SHA-256 dari token opaque di cookie
  ip              inet,
  user_agent      text CHECK (char_length(user_agent) <= 512),
  created_at      timestamptz NOT NULL DEFAULT now(),
  last_seen_at    timestamptz NOT NULL DEFAULT now(),
  idle_expires_at timestamptz NOT NULL,          -- digeser saat aktif (sliding)
  expires_at      timestamptz NOT NULL,          -- batas absolut
  revoked_at      timestamptz,
  revoked_reason  text CHECK (revoked_reason IN (
                    'logout', 'logout_others', 'password_changed', 'password_reset',
                    'admin_reset', 'deactivated', 'expired', 'security'))
);
CREATE INDEX ix_sessions_user_active ON sessions (user_id) WHERE revoked_at IS NULL;
CREATE INDEX ix_sessions_expiry ON sessions (expires_at) WHERE revoked_at IS NULL;

CREATE TABLE recovery_codes (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  batch_id    uuid NOT NULL,                     -- satu kali generate = satu batch
  code_hash   text NOT NULL,                     -- Argon2id
  created_at  timestamptz NOT NULL DEFAULT now(),
  used_at     timestamptz,
  revoked_at  timestamptz                        -- batch lama hangus saat generate ulang
);
CREATE INDEX ix_recovery_codes_user ON recovery_codes (user_id) WHERE used_at IS NULL AND revoked_at IS NULL;

-- =============================================================================
-- 4. IAM (role & izin)
-- =============================================================================
CREATE TABLE permissions (
  key          text PRIMARY KEY CHECK (key ~ '^[a-z_]+\.[a-z_]+$'),
  description  text NOT NULL,
  scopable     boolean NOT NULL DEFAULT false     -- bisa dibatasi ke jurusan
);

CREATE TABLE roles (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  slug         citext NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9_]{2,50}$'),
  name         text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 60),
  description  text,
  is_system    boolean NOT NULL DEFAULT false,   -- member, moderator, major_admin, admin
  created_by   uuid REFERENCES users(id),
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_roles_updated BEFORE UPDATE ON roles FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE role_permissions (
  role_id         uuid NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  permission_key  text NOT NULL REFERENCES permissions(key) ON DELETE CASCADE,
  PRIMARY KEY (role_id, permission_key)
);

CREATE TABLE user_roles (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id      uuid NOT NULL REFERENCES roles(id),
  major_id     uuid REFERENCES majors(id),       -- NULL = scope global
  assigned_by  uuid REFERENCES users(id),
  assigned_at  timestamptz NOT NULL DEFAULT now(),
  expires_at   timestamptz,
  revoked_at   timestamptz,
  revoked_by   uuid REFERENCES users(id)
);
CREATE UNIQUE INDEX uq_user_roles_active ON user_roles (user_id, role_id, major_id) NULLS NOT DISTINCT
  WHERE revoked_at IS NULL;
CREATE INDEX ix_user_roles_user ON user_roles (user_id) WHERE revoked_at IS NULL;

-- Jurusan yang diampu guru (tampil di profil "Mengampu").
CREATE TABLE teacher_majors (
  user_id   uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  major_id  uuid NOT NULL REFERENCES majors(id),
  PRIMARY KEY (user_id, major_id)
);

-- =============================================================================
-- 5. ENROLLMENT (riwayat kelas, imutabel per tahun ajaran)
-- =============================================================================
CREATE TABLE enrollments (
  id                uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id           uuid NOT NULL REFERENCES users(id),
  class_id          uuid NOT NULL REFERENCES classes(id),
  academic_year_id  uuid NOT NULL REFERENCES academic_years(id),
  -- Hasil di akhir tahun ajaran; NULL selama tahun berjalan.
  outcome           text CHECK (outcome IN ('promoted', 'retained', 'graduated', 'graduation_deferred',
                                            'transferred_out', 'moved')),
  promotion_job_id  uuid,                          -- FK ditambahkan setelah promotion_jobs
  source            text NOT NULL DEFAULT 'import' CHECK (source IN ('import', 'promotion', 'admin', 'legacy')),
  created_at        timestamptz NOT NULL DEFAULT now(),
  UNIQUE (user_id, academic_year_id)
);
CREATE INDEX ix_enrollments_class ON enrollments (class_id);
CREATE INDEX ix_enrollments_user ON enrollments (user_id, academic_year_id);

-- =============================================================================
-- 6. PROFIL, EXPERIENCE, SKILL, BADGE
-- =============================================================================
CREATE TABLE profiles (
  user_id          uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  headline         text CHECK (char_length(headline) <= 120),
  bio              text CHECK (char_length(bio) <= 1000),
  city             text CHECK (char_length(city) <= 80),
  avatar_media_id  uuid REFERENCES media_assets(id) ON DELETE SET NULL,
  banner_media_id  uuid REFERENCES media_assets(id) ON DELETE SET NULL,
  teaching_subjects text[] NOT NULL DEFAULT '{}' CHECK (cardinality(teaching_subjects) <= 10),
  updated_at       timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_profiles_updated BEFORE UPDATE ON profiles FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE profile_links (
  id        uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id   uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  platform  text NOT NULL CHECK (platform IN (
              'github', 'gitlab', 'behance', 'dribbble', 'artstation', 'youtube', 'instagram',
              'tiktok', 'linkedin', 'x', 'figma', 'website')),
  url       text NOT NULL CHECK (url ~ '^https://' AND char_length(url) <= 500),
  position  smallint NOT NULL DEFAULT 0,
  UNIQUE (user_id, platform, url)
);
CREATE INDEX ix_profile_links_user ON profile_links (user_id, position);

CREATE TABLE skills (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  slug        citext NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]{1,60}$'),
  name        text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 60),
  category    text NOT NULL CHECK (category IN ('design', 'media', 'software', 'network', 'engineering',
                                                'logistics', 'hospitality', 'printing', 'soft_skill', 'other')),
  is_active   boolean NOT NULL DEFAULT true,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_skills_name_trgm ON skills USING gin (name gin_trgm_ops);

CREATE TABLE user_skills (
  user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  skill_id    uuid NOT NULL REFERENCES skills(id),
  position    smallint NOT NULL DEFAULT 0,
  created_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (user_id, skill_id)
);
CREATE INDEX ix_user_skills_skill ON user_skills (skill_id);

CREATE TABLE skill_endorsements (
  endorser_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  user_id      uuid NOT NULL,
  skill_id     uuid NOT NULL,
  created_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (endorser_id, user_id, skill_id),
  FOREIGN KEY (user_id, skill_id) REFERENCES user_skills(user_id, skill_id) ON DELETE CASCADE,
  CHECK (endorser_id <> user_id)
);
CREATE INDEX ix_endorsements_target ON skill_endorsements (user_id, skill_id);

CREATE TABLE tools (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  slug        citext NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]{1,60}$'),
  name        text NOT NULL CHECK (char_length(name) BETWEEN 1 AND 60),
  category    text NOT NULL CHECK (category IN ('design_software', 'media_software', 'language', 'framework',
                                                'platform', 'hardware', 'machine', 'other')),
  is_active   boolean NOT NULL DEFAULT true,
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_tools_name_trgm ON tools USING gin (name gin_trgm_ops);

CREATE TABLE experiences (
  id               uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id          uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind             text NOT NULL CHECK (kind IN ('education', 'internship', 'job', 'freelance', 'organization',
                                                 'committee', 'competition', 'certification', 'volunteer')),
  title            text NOT NULL CHECK (char_length(title) BETWEEN 2 AND 120),
  organization     text NOT NULL CHECK (char_length(organization) BETWEEN 2 AND 120),
  location         text CHECK (char_length(location) <= 120),
  employment_type  text CHECK (employment_type IN ('full_time', 'part_time', 'contract', 'internship', 'freelance')),
  start_month      date NOT NULL CHECK (extract(day FROM start_month) = 1),
  end_month        date CHECK (end_month IS NULL OR extract(day FROM end_month) = 1),
  is_current       boolean NOT NULL DEFAULT false,
  description      text CHECK (char_length(description) <= 2000),
  credential_url   text CHECK (credential_url ~ '^https://' AND char_length(credential_url) <= 500),
  is_system        boolean NOT NULL DEFAULT false,   -- entri "SMKN 4 Malang" otomatis
  created_at       timestamptz NOT NULL DEFAULT now(),
  updated_at       timestamptz NOT NULL DEFAULT now(),
  CHECK (is_current = (end_month IS NULL)),
  CHECK (end_month IS NULL OR end_month >= start_month)
);
CREATE INDEX ix_experiences_user ON experiences (user_id, start_month DESC);
CREATE UNIQUE INDEX uq_experiences_system ON experiences (user_id) WHERE is_system;
CREATE TRIGGER trg_experiences_updated BEFORE UPDATE ON experiences FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE experience_skills (
  experience_id uuid NOT NULL REFERENCES experiences(id) ON DELETE CASCADE,
  skill_id      uuid NOT NULL REFERENCES skills(id),
  PRIMARY KEY (experience_id, skill_id)
);

CREATE TABLE experience_attachments (
  experience_id uuid NOT NULL REFERENCES experiences(id) ON DELETE CASCADE,
  media_id      uuid NOT NULL REFERENCES media_assets(id),
  position      smallint NOT NULL DEFAULT 0,
  PRIMARY KEY (experience_id, media_id)
);

CREATE TABLE badges (
  id             uuid PRIMARY KEY DEFAULT uuidv7(),
  slug           citext NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]{2,60}$'),
  name           text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 60),
  description    text NOT NULL CHECK (char_length(description) <= 300),
  kind           text NOT NULL CHECK (kind IN ('system', 'school')),
  icon_key       text,                                    -- ikon bawaan
  icon_media_id  uuid REFERENCES media_assets(id),        -- ikon unggahan (badge sekolah)
  criteria       jsonb,                                   -- badge sistem: aturan otomatis
  repeatable     boolean NOT NULL DEFAULT false,          -- mis. Juara LKS bisa berkali-kali
  is_active      boolean NOT NULL DEFAULT true,
  created_at     timestamptz NOT NULL DEFAULT now(),
  CHECK (kind <> 'system' OR criteria IS NOT NULL)
);

CREATE TABLE user_badges (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  badge_id     uuid NOT NULL REFERENCES badges(id),
  dedupe_key   text NOT NULL DEFAULT '',                  -- badge repeatable: mis. "lks-2026"
  note         text CHECK (char_length(note) <= 200),     -- "Juara 1 IT Software Solutions 2026"
  awarded_by   uuid REFERENCES users(id),                 -- NULL = otomatis sistem
  awarded_at   timestamptz NOT NULL DEFAULT now(),
  expires_at   timestamptz,
  revoked_at   timestamptz,
  revoked_by   uuid REFERENCES users(id)
);
CREATE UNIQUE INDEX uq_user_badges_active ON user_badges (user_id, badge_id, dedupe_key) WHERE revoked_at IS NULL;
CREATE INDEX ix_user_badges_user ON user_badges (user_id, awarded_at DESC) WHERE revoked_at IS NULL;

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

-- =============================================================================
-- 10. MODERASI & LAPORAN
-- =============================================================================
CREATE TABLE moderation_reasons (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  slug        citext NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]{2,60}$'),
  label       text NOT NULL CHECK (char_length(label) BETWEEN 2 AND 80),
  description text CHECK (char_length(description) <= 500),
  applies_to  text[] NOT NULL CHECK (applies_to <@ ARRAY['content', 'comment', 'profile', 'group', 'report']::text[]),
  sort_order  smallint NOT NULL DEFAULT 0,
  is_active   boolean NOT NULL DEFAULT true
);

CREATE TABLE moderation_cases (
  id                uuid PRIMARY KEY DEFAULT uuidv7(),
  content_id        uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  version_id        uuid NOT NULL UNIQUE REFERENCES content_versions(id) ON DELETE CASCADE,
  kind              text NOT NULL CHECK (kind IN ('initial', 'resubmission', 'revision')),
  queue_major_id    uuid REFERENCES majors(id),        -- NULL = antrian global
  status            text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'claimed', 'decided', 'withdrawn')),
  claimed_by        uuid REFERENCES users(id),
  claimed_at        timestamptz,
  claim_expires_at  timestamptz,
  decision          text CHECK (decision IN ('approved', 'changes_requested', 'rejected')),
  reason_id         uuid REFERENCES moderation_reasons(id),
  decision_note     text CHECK (char_length(decision_note) <= 2000),
  decided_by        uuid REFERENCES users(id),
  decided_at        timestamptz,
  created_at        timestamptz NOT NULL DEFAULT now(),
  CHECK ((status = 'claimed') = (claimed_by IS NOT NULL AND claim_expires_at IS NOT NULL) OR status = 'decided'),
  CHECK ((status = 'decided') = (decision IS NOT NULL)),
  CHECK (decision IS NULL OR decided_by IS NOT NULL),
  CHECK (decision NOT IN ('changes_requested', 'rejected') OR decision_note IS NOT NULL),
  CHECK (decision <> 'rejected' OR reason_id IS NOT NULL)
);
CREATE INDEX ix_mod_cases_queue ON moderation_cases (queue_major_id, created_at) WHERE status IN ('open', 'claimed');
CREATE INDEX ix_mod_cases_claim_expiry ON moderation_cases (claim_expires_at) WHERE status = 'claimed';
CREATE INDEX ix_mod_cases_decided ON moderation_cases (decided_by, decided_at DESC) WHERE status = 'decided';

-- Catatan per blok / node saat minta revisi.
CREATE TABLE moderation_annotations (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  case_id     uuid NOT NULL REFERENCES moderation_cases(id) ON DELETE CASCADE,
  target_ref  text NOT NULL CHECK (char_length(target_ref) <= 100),   -- id blok / id node / "field:title"
  note        text NOT NULL CHECK (char_length(note) BETWEEN 1 AND 1000),
  created_by  uuid NOT NULL REFERENCES users(id),
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_mod_annotations_case ON moderation_annotations (case_id);

-- Banding: maksimal satu kali per penolakan.
CREATE TABLE appeals (
  id            uuid PRIMARY KEY DEFAULT uuidv7(),
  case_id       uuid NOT NULL UNIQUE REFERENCES moderation_cases(id) ON DELETE CASCADE,
  content_id    uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  submitted_by  uuid NOT NULL REFERENCES users(id),
  message       text NOT NULL CHECK (char_length(message) BETWEEN 10 AND 2000),
  status        text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'upheld', 'overturned')),
  decided_by    uuid REFERENCES users(id),
  decided_at    timestamptz,
  decision_note text CHECK (char_length(decision_note) <= 2000),
  created_at    timestamptz NOT NULL DEFAULT now(),
  CHECK ((status = 'open') = (decided_at IS NULL))
);
CREATE INDEX ix_appeals_open ON appeals (created_at) WHERE status = 'open';

CREATE TABLE reports (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  reporter_id  uuid NOT NULL REFERENCES users(id),
  target_type  text NOT NULL CHECK (target_type IN ('content', 'comment', 'profile', 'group')),
  target_id    uuid NOT NULL,
  reason_id    uuid NOT NULL REFERENCES moderation_reasons(id),
  note         text CHECK (char_length(note) <= 500),
  status       text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'actioned', 'dismissed')),
  handled_by   uuid REFERENCES users(id),
  handled_at   timestamptz,
  created_at   timestamptz NOT NULL DEFAULT now(),
  CHECK ((status = 'open') = (handled_at IS NULL))
);
CREATE UNIQUE INDEX uq_reports_open ON reports (reporter_id, target_type, target_id) WHERE status = 'open';
CREATE INDEX ix_reports_queue ON reports (created_at) WHERE status = 'open';
CREATE INDEX ix_reports_target ON reports (target_type, target_id);

-- Tindakan moderator di luar keputusan moderation_cases (takedown, sembunyikan komentar, reset profil).
CREATE TABLE moderation_actions (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  target_type  text NOT NULL CHECK (target_type IN ('content', 'comment', 'profile', 'experience', 'group')),
  target_id    uuid NOT NULL,
  action       text NOT NULL CHECK (action IN (
                 'take_down', 'hide_comment', 'restore_comment', 'reset_avatar', 'reset_banner',
                 'reset_bio', 'reset_headline', 'remove_experience', 'reset_group_avatar',
                 'reset_group_banner', 'reset_group_description', 'dismiss')),
  reason_id    uuid REFERENCES moderation_reasons(id),
  note         text CHECK (char_length(note) <= 2000),
  report_id    uuid REFERENCES reports(id),
  actor_id     uuid NOT NULL REFERENCES users(id),
  created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_mod_actions_target ON moderation_actions (target_type, target_id, created_at DESC);

CREATE TABLE word_filters (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  pattern     text NOT NULL UNIQUE CHECK (char_length(pattern) BETWEEN 2 AND 100),
  match_kind  text NOT NULL DEFAULT 'word' CHECK (match_kind IN ('word', 'substring')),
  action      text NOT NULL CHECK (action IN ('block', 'flag')),   -- block = tolak simpan, flag = laporan otomatis
  created_by  uuid REFERENCES users(id),
  created_at  timestamptz NOT NULL DEFAULT now()
);

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

-- =============================================================================
-- 12. NOTIFIKASI
-- =============================================================================
CREATE TABLE notifications (
  id            uuid PRIMARY KEY DEFAULT uuidv7(),
  recipient_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type          text NOT NULL CHECK (type IN (
                  'collab_invited', 'collab_accepted', 'collab_declined', 'collab_removed',
                  'ownership_transferred', 'moderation_approved', 'moderation_changes_requested',
                  'moderation_rejected', 'revision_approved', 'revision_changes_requested',
                  'revision_rejected', 'content_taken_down', 'appeal_decided', 'comment_on_content',
                  'comment_reply', 'comment_mention', 'appreciation', 'new_follower', 'badge_awarded',
                  'report_resolved', 'profile_reset', 'group_manager_assigned', 'class_changed',
                  'password_changed', 'recovery_codes_used', 'password_reset_by_admin')),
  -- Kunci agregasi: notifikasi belum dibaca dengan kunci sama digabung
  -- (mis. "appreciation:<content_id>" → "Rafa dan 12 lainnya mengapresiasi karyamu").
  group_key     text NOT NULL,
  actor_id      uuid REFERENCES users(id) ON DELETE SET NULL,   -- aktor terakhir
  actor_count   integer NOT NULL DEFAULT 1 CHECK (actor_count >= 1),
  subject_type  text CHECK (subject_type IN ('content', 'comment', 'user', 'group', 'badge', 'appeal', 'report')),
  subject_id    uuid,
  data          jsonb NOT NULL DEFAULT '{}'::jsonb,
  read_at       timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_notifications_unread_group ON notifications (recipient_id, group_key) WHERE read_at IS NULL;
CREATE INDEX ix_notifications_recipient ON notifications (recipient_id, updated_at DESC, id DESC);
CREATE INDEX ix_notifications_unread ON notifications (recipient_id) WHERE read_at IS NULL;
CREATE INDEX ix_notifications_retention ON notifications (created_at);

CREATE TABLE notification_preferences (
  user_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  type     text NOT NULL,
  enabled  boolean NOT NULL,
  PRIMARY KEY (user_id, type)
);

-- =============================================================================
-- 13. KENAIKAN KELAS
-- =============================================================================
CREATE TABLE promotion_jobs (
  id                     uuid PRIMARY KEY DEFAULT uuidv7(),
  from_academic_year_id  uuid NOT NULL REFERENCES academic_years(id),
  to_academic_year_id    uuid NOT NULL REFERENCES academic_years(id),
  status                 text NOT NULL DEFAULT 'draft' CHECK (status IN (
                           'draft', 'approved', 'executing', 'executed', 'failed', 'rolled_back', 'cancelled')),
  scheduled_for          date NOT NULL,
  dry_run_summary        jsonb,
  dry_run_at             timestamptz,
  conflict_count         integer NOT NULL DEFAULT 0,
  created_by             uuid NOT NULL REFERENCES users(id),
  approved_by            uuid REFERENCES users(id),
  approved_at            timestamptz,
  executed_at            timestamptz,
  rollback_deadline      timestamptz,
  rolled_back_by         uuid REFERENCES users(id),
  rolled_back_at         timestamptz,
  failure_reason         text,
  created_at             timestamptz NOT NULL DEFAULT now(),
  updated_at             timestamptz NOT NULL DEFAULT now(),
  CHECK (from_academic_year_id <> to_academic_year_id),
  CHECK (status NOT IN ('approved', 'executing', 'executed') OR (approved_by IS NOT NULL AND conflict_count = 0)),
  CHECK (status <> 'executed' OR (executed_at IS NOT NULL AND rollback_deadline IS NOT NULL))
);
CREATE UNIQUE INDEX uq_promotion_jobs_target ON promotion_jobs (to_academic_year_id)
  WHERE status NOT IN ('cancelled', 'rolled_back', 'failed');
CREATE TRIGGER trg_promotion_jobs_updated BEFORE UPDATE ON promotion_jobs FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE enrollments ADD CONSTRAINT fk_enrollments_promotion_job
  FOREIGN KEY (promotion_job_id) REFERENCES promotion_jobs(id);

CREATE TABLE promotion_class_mappings (
  job_id         uuid NOT NULL REFERENCES promotion_jobs(id) ON DELETE CASCADE,
  from_class_id  uuid NOT NULL REFERENCES classes(id),
  to_class_id    uuid REFERENCES classes(id),           -- NULL untuk kelas XII (lulus)
  PRIMARY KEY (job_id, from_class_id)
);

CREATE TABLE promotion_items (
  id                  uuid PRIMARY KEY DEFAULT uuidv7(),
  job_id              uuid NOT NULL REFERENCES promotion_jobs(id) ON DELETE CASCADE,
  student_id          uuid NOT NULL REFERENCES users(id),
  from_enrollment_id  uuid NOT NULL REFERENCES enrollments(id),
  action              text NOT NULL CHECK (action IN (
                        'promote', 'retain', 'move', 'transfer_out', 'graduate', 'defer_graduation')),
  target_class_id     uuid REFERENCES classes(id),
  is_exception        boolean NOT NULL DEFAULT false,   -- diubah dari default pemetaan
  note                text CHECK (char_length(note) <= 300),
  updated_by          uuid REFERENCES users(id),
  updated_at          timestamptz NOT NULL DEFAULT now(),
  UNIQUE (job_id, student_id),
  CHECK ((action IN ('transfer_out', 'graduate')) = (target_class_id IS NULL))
);
CREATE INDEX ix_promotion_items_job ON promotion_items (job_id, action);

-- =============================================================================
-- 14. PENUGASAN & PENILAIAN (fase 2)
-- =============================================================================
CREATE TABLE rubrics (
  id          uuid PRIMARY KEY DEFAULT uuidv7(),
  name        text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 100),
  created_by  uuid NOT NULL REFERENCES users(id),
  is_shared   boolean NOT NULL DEFAULT false,
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE rubric_criteria (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  rubric_id    uuid NOT NULL REFERENCES rubrics(id) ON DELETE CASCADE,
  name         text NOT NULL CHECK (char_length(name) BETWEEN 2 AND 100),
  description  text CHECK (char_length(description) <= 500),
  max_score    smallint NOT NULL CHECK (max_score BETWEEN 1 AND 100),
  weight       numeric(5,2) NOT NULL DEFAULT 1 CHECK (weight > 0),
  position     smallint NOT NULL DEFAULT 0
);

CREATE TABLE assignments (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  class_id     uuid NOT NULL REFERENCES classes(id),
  rubric_id    uuid REFERENCES rubrics(id),
  created_by   uuid NOT NULL REFERENCES users(id),
  title        text NOT NULL CHECK (char_length(title) BETWEEN 3 AND 150),
  instructions text CHECK (char_length(instructions) <= 5000),
  due_at       timestamptz,
  status       text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'open', 'closed')),
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ix_assignments_class ON assignments (class_id, status);
CREATE TRIGGER trg_assignments_updated BEFORE UPDATE ON assignments FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE assignment_submissions (
  id             uuid PRIMARY KEY DEFAULT uuidv7(),
  assignment_id  uuid NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
  student_id     uuid NOT NULL REFERENCES users(id),
  content_id     uuid NOT NULL REFERENCES contents(id),
  submitted_at   timestamptz NOT NULL DEFAULT now(),
  UNIQUE (assignment_id, student_id)
);

CREATE TABLE assessments (
  id             uuid PRIMARY KEY DEFAULT uuidv7(),
  submission_id  uuid NOT NULL UNIQUE REFERENCES assignment_submissions(id) ON DELETE CASCADE,
  assessor_id    uuid NOT NULL REFERENCES users(id),
  final_comment  text CHECK (char_length(final_comment) <= 2000),
  total_score    numeric(6,2),
  created_at     timestamptz NOT NULL DEFAULT now(),
  updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_assessments_updated BEFORE UPDATE ON assessments FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE assessment_scores (
  assessment_id  uuid NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
  criterion_id   uuid NOT NULL REFERENCES rubric_criteria(id),
  score          smallint NOT NULL CHECK (score >= 0),
  comment        text CHECK (char_length(comment) <= 1000),
  PRIMARY KEY (assessment_id, criterion_id)
);

-- =============================================================================
-- 15. KURASI (fase 2)
-- =============================================================================
CREATE TABLE curations (
  id              uuid PRIMARY KEY DEFAULT uuidv7(),
  slug            citext NOT NULL UNIQUE CHECK (slug ~ '^[a-z0-9-]{2,80}$'),
  title           text NOT NULL CHECK (char_length(title) BETWEEN 3 AND 120),
  description     text CHECK (char_length(description) <= 1000),
  cover_media_id  uuid REFERENCES media_assets(id),
  created_by      uuid NOT NULL REFERENCES users(id),
  is_published    boolean NOT NULL DEFAULT false,
  created_at      timestamptz NOT NULL DEFAULT now(),
  updated_at      timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER trg_curations_updated BEFORE UPDATE ON curations FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE curation_items (
  curation_id  uuid NOT NULL REFERENCES curations(id) ON DELETE CASCADE,
  content_id   uuid NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
  note         text CHECK (char_length(note) <= 300),
  added_by     uuid NOT NULL REFERENCES users(id),
  position     smallint NOT NULL DEFAULT 0,
  added_at     timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (curation_id, content_id)
);

-- =============================================================================
-- 16. SISTEM
-- =============================================================================
CREATE TABLE settings (
  key         text PRIMARY KEY CHECK (key ~ '^[a-z_]+(\.[a-z_]+)*$'),
  value       jsonb NOT NULL,
  updated_by  uuid REFERENCES users(id),
  updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE import_jobs (
  id           uuid PRIMARY KEY DEFAULT uuidv7(),
  kind         text NOT NULL CHECK (kind IN ('students', 'teachers', 'staff', 'promotion_exceptions')),
  media_id     uuid REFERENCES media_assets(id),
  status       text NOT NULL DEFAULT 'validating' CHECK (status IN (
                 'validating', 'ready', 'invalid', 'committing', 'committed', 'failed', 'cancelled')),
  total_rows   integer,
  valid_rows   integer,
  errors       jsonb NOT NULL DEFAULT '[]'::jsonb,     -- [{row, field, message}]
  summary      jsonb,
  created_by   uuid NOT NULL REFERENCES users(id),
  created_at   timestamptz NOT NULL DEFAULT now(),
  committed_at timestamptz
);

CREATE TABLE audit_logs (
  id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  occurred_at     timestamptz NOT NULL DEFAULT now(),
  actor_id        uuid REFERENCES users(id) ON DELETE SET NULL,
  actor_ip        inet,
  action          text NOT NULL CHECK (action ~ '^[a-z_]+\.[a-z_.]+$'),   -- mis. content.approve
  target_type     text,
  target_id       uuid,
  scope_major_id  uuid REFERENCES majors(id),
  data            jsonb NOT NULL DEFAULT '{}'::jsonb
);
CREATE INDEX ix_audit_time ON audit_logs USING brin (occurred_at);
CREATE INDEX ix_audit_target ON audit_logs (target_type, target_id, occurred_at DESC);
CREATE INDEX ix_audit_actor ON audit_logs (actor_id, occurred_at DESC);

-- =============================================================================
-- 17. LEGACY (migrasi v1)
-- =============================================================================
CREATE TABLE legacy_id_map (
  source_table  text NOT NULL,
  source_id     uuid NOT NULL,
  target_table  text NOT NULL,
  target_id     uuid NOT NULL,
  migrated_at   timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (source_table, source_id)
);
CREATE INDEX ix_legacy_target ON legacy_id_map (target_table, target_id);

-- Redirect URL lama v1 (/{username}/{slug}, /storage/...) → URL baru.
CREATE TABLE legacy_redirects (
  path         text PRIMARY KEY,
  target_path  text NOT NULL,
  hit_count    bigint NOT NULL DEFAULT 0,
  last_hit_at  timestamptz,
  created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE legacy_quarantine (
  id            uuid PRIMARY KEY DEFAULT uuidv7(),
  source_table  text NOT NULL,
  source_id     uuid NOT NULL,
  reason        text NOT NULL,
  payload       jsonb,
  created_at    timestamptz NOT NULL DEFAULT now()
);
