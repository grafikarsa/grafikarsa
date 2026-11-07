-- +goose Up
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

-- +goose Down
DROP TABLE IF EXISTS teacher_majors;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS permissions;
