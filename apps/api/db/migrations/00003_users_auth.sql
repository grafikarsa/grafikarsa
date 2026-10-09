-- +goose Up
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username CITEXT NOT NULL UNIQUE,
  email CITEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  name VARCHAR(100) NOT NULL,
  role_primary TEXT NOT NULL DEFAULT 'siswa',
  status TEXT NOT NULL DEFAULT 'aktif',
  class_id UUID REFERENCES classes (id) ON DELETE SET NULL,
  major_id UUID REFERENCES majors (id) ON DELETE SET NULL,
  graduated_year INTEGER,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  must_change_password BOOLEAN NOT NULL DEFAULT FALSE,
  last_login_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  deleted_at TIMESTAMPTZ,
  CONSTRAINT users_role_valid CHECK (role_primary IN ('siswa', 'guru', 'admin_jurusan', 'admin')),
  CONSTRAINT users_status_valid CHECK (status IN ('aktif', 'lulus', 'nonaktif')),
  CONSTRAINT users_username_format CHECK (username ~ '^[a-z0-9._-]{3,30}$')
);
CREATE INDEX idx_users_class ON users (class_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_name_trgm ON users USING gin (name gin_trgm_ops) WHERE deleted_at IS NULL;
CREATE TRIGGER trg_users_updated_at
  BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  refresh_hash TEXT NOT NULL UNIQUE,
  family_id UUID NOT NULL,
  device_label VARCHAR(100),
  ip INET,
  user_agent TEXT,
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  rotated_to UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sessions_user ON sessions (user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_sessions_family ON sessions (family_id);

CREATE TABLE auth_audit (
  id BIGSERIAL PRIMARY KEY,
  user_id UUID REFERENCES users (id) ON DELETE SET NULL,
  event TEXT NOT NULL,
  ip INET,
  user_agent TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_auth_audit_created ON auth_audit USING brin (created_at);

COMMENT ON TABLE users IS 'Akun; alumni = status lulus, bukan role';
COMMENT ON TABLE sessions IS 'Refresh token opaque (hash sha256), rotation + reuse-detection';
COMMENT ON TABLE auth_audit IS 'Audit login/refresh/logout, retensi 90 hari';

-- +goose Down
DROP TABLE IF EXISTS auth_audit;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
