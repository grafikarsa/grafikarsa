-- +goose Up
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

-- +goose Down
DROP TABLE IF EXISTS recovery_codes;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS username_history;
DROP TABLE IF EXISTS reserved_usernames;
ALTER TABLE media_assets DROP CONSTRAINT IF EXISTS fk_media_owner;
DROP TABLE IF EXISTS users;
