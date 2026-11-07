-- +goose Up
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

-- +goose Down
DROP TABLE IF EXISTS user_badges;
DROP TABLE IF EXISTS badges;
DROP TABLE IF EXISTS experience_attachments;
DROP TABLE IF EXISTS experience_skills;
DROP TABLE IF EXISTS experiences;
DROP TABLE IF EXISTS tools;
DROP TABLE IF EXISTS skill_endorsements;
DROP TABLE IF EXISTS user_skills;
DROP TABLE IF EXISTS skills;
DROP TABLE IF EXISTS profile_links;
DROP TABLE IF EXISTS profiles;
