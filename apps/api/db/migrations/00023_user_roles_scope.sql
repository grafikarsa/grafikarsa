-- +goose Up
-- +goose StatementBegin
-- user_roles: primary key (user_id, role_id) -> id surrogate.
--
-- PK lama membuat satu user hanya bisa punya SATU baris per role, sehingga
-- role yang sama dengan scope berbeda (mis. admin_jurusan di dua jurusan)
-- tidak bisa direpresentasikan, dan AssignUserRole diam-diam menimpa
-- scope lama lewat ON CONFLICT DO UPDATE.
--
-- Scope yang nullable butuh NULLS NOT DISTINCT agar scope NULL (role global)
-- ikut dianggap satu nilai yang sama oleh unique index. Butuh PostgreSQL 15+
-- (target v2 memakai PostgreSQL 16).
ALTER TABLE user_roles DROP CONSTRAINT user_roles_pkey;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE user_roles ADD COLUMN id UUID PRIMARY KEY DEFAULT gen_random_uuid();
-- +goose StatementEnd
-- +goose StatementBegin
CREATE UNIQUE INDEX user_roles_user_role_scope_uniq
  ON user_roles (user_id, role_id, scope_major_id) NULLS NOT DISTINCT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS user_roles_user_role_scope_uniq;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE user_roles DROP CONSTRAINT user_roles_pkey;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE user_roles DROP COLUMN id;
-- +goose StatementEnd
-- +goose StatementBegin
ALTER TABLE user_roles ADD PRIMARY KEY (user_id, role_id);
-- +goose StatementEnd
