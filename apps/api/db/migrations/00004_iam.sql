-- +goose Up
-- Katalog permission DIKUNCI di migrasi (bagian dari kode, bukan data).
-- Assignment role->permission + user admin ada di db/seeds (data).
CREATE TABLE roles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  slug TEXT NOT NULL UNIQUE,
  name VARCHAR(100) NOT NULL,
  scope_type TEXT NOT NULL DEFAULT 'global',
  is_system BOOLEAN NOT NULL DEFAULT FALSE,
  created_by UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT roles_scope_valid CHECK (scope_type IN ('global', 'jurusan'))
);

CREATE TABLE permissions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  key TEXT NOT NULL UNIQUE,
  permission_group TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE role_permissions (
  role_id UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
  permission_id UUID NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
  PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE user_roles (
  user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
  role_id UUID NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
  scope_major_id UUID REFERENCES majors (id) ON DELETE CASCADE,
  assigned_by UUID REFERENCES users (id) ON DELETE SET NULL,
  expires_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (user_id, role_id, scope_major_id)
);

-- 4 role sistem
INSERT INTO roles (slug, name, scope_type, is_system) VALUES
  ('siswa', 'Siswa', 'global', TRUE),
  ('guru', 'Guru', 'global', TRUE),
  ('admin_jurusan', 'Admin Jurusan', 'jurusan', TRUE),
  ('admin', 'Admin', 'global', TRUE)
ON CONFLICT (slug) DO NOTHING;

-- 53 permissions (PRD §1.2, final)
INSERT INTO permissions (key, permission_group, description) VALUES
  ('dashboard.view_global', 'dashboard', 'Lihat dashboard global'),
  ('dashboard.view_jurusan', 'dashboard', 'Lihat dashboard jurusan'),
  ('portfolios.create', 'portfolios', 'Buat portfolio'),
  ('portfolios.edit_own', 'portfolios', 'Edit portfolio sendiri'),
  ('portfolios.submit', 'portfolios', 'Kirim portfolio ke review'),
  ('portfolios.delete_own', 'portfolios', 'Hapus portfolio sendiri'),
  ('portfolios.view_all', 'portfolios', 'Lihat semua portfolio'),
  ('portfolios.moderate', 'portfolios', 'Moderasi semua portfolio'),
  ('portfolios.moderate_jurusan', 'portfolios', 'Moderasi portfolio satu jurusan'),
  ('portfolios.archive_any', 'portfolios', 'Arsipkan portfolio siapa pun'),
  ('blocks.manage_own', 'blocks', 'Kelola block portfolio sendiri'),
  ('media.upload', 'media', 'Upload file'),
  ('media.delete_own', 'media', 'Hapus file sendiri'),
  ('media.delete_any', 'media', 'Hapus file siapa pun'),
  ('social.follow', 'social', 'Follow/unfollow'),
  ('social.comment', 'social', 'Komentar'),
  ('social.like', 'social', 'Like'),
  ('users.view', 'users', 'Lihat user'),
  ('users.create', 'users', 'Buat user'),
  ('users.edit', 'users', 'Edit user'),
  ('users.reset_password', 'users', 'Reset password user'),
  ('users.activate', 'users', 'Aktif/nonaktif user'),
  ('users.delete', 'users', 'Hapus user'),
  ('users.view_jurusan', 'users', 'Lihat user satu jurusan'),
  ('users.create_jurusan', 'users', 'Buat user satu jurusan'),
  ('users.edit_jurusan', 'users', 'Edit user satu jurusan'),
  ('roles.view', 'roles', 'Lihat roles'),
  ('roles.create', 'roles', 'Buat custom role'),
  ('roles.assign', 'roles', 'Assign role global'),
  ('roles.assign_jurusan', 'roles', 'Assign role dalam jurusan'),
  ('academic.view', 'academic', 'Lihat data akademik'),
  ('academic.manage_classes', 'academic', 'Kelola kelas'),
  ('academic.manage_majors', 'academic', 'Kelola jurusan'),
  ('academic.manage_years', 'academic', 'Kelola tahun ajaran'),
  ('academic.promote_dryrun', 'academic', 'Preview kenaikan kelas'),
  ('academic.promote_execute', 'academic', 'Eksekusi kenaikan kelas'),
  ('academic.import_students', 'academic', 'Import siswa'),
  ('assessments.view', 'assessments', 'Lihat penilaian'),
  ('assessments.score', 'assessments', 'Menilai portfolio'),
  ('assessments.manage_metrics', 'assessments', 'Kelola metrik penilaian'),
  ('series.view', 'series', 'Lihat series'),
  ('series.manage', 'series', 'Kelola series'),
  ('tags.manage', 'tags', 'Kelola tags'),
  ('feedback.create', 'feedback', 'Buat feedback'),
  ('feedback.view_all', 'feedback', 'Lihat semua feedback'),
  ('feedback.triage', 'feedback', 'Triase feedback'),
  ('feedback.reply', 'feedback', 'Balas feedback'),
  ('changelog.view', 'changelog', 'Lihat changelog'),
  ('changelog.manage', 'changelog', 'Kelola changelog'),
  ('changelog.publish', 'changelog', 'Publish changelog'),
  ('system.settings', 'system', 'Pengaturan sistem'),
  ('system.backup', 'system', 'Backup'),
  ('system.jobs', 'system', 'Kelola jobs')
ON CONFLICT (key) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
