-- Mapping role sistem -> permission default (PRD §1.2).
-- Dijalankan SETELAH migrasi (permissions sudah ada) dan SETELAH admin dibuat.
-- Idempoten (ON CONFLICT DO NOTHING).
-- siswa+guru+admin_jurusan: permission dasar + spesifik per role.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'siswa' AND p.key IN (
  'portfolios.create','portfolios.edit_own','portfolios.submit','portfolios.delete_own',
  'blocks.manage_own','media.upload','media.delete_own',
  'social.follow','social.comment','social.like',
  'feedback.create','changelog.view')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'guru' AND p.key IN (
  'portfolios.create','portfolios.edit_own','portfolios.submit','portfolios.delete_own',
  'blocks.manage_own','media.upload','media.delete_own',
  'social.follow','social.comment','social.like',
  'feedback.create','changelog.view',
  'assessments.view','assessments.score','series.view',
  'dashboard.view_jurusan','academic.view')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'admin_jurusan' AND p.key IN (
  'portfolios.create','portfolios.edit_own','portfolios.submit','portfolios.delete_own',
  'blocks.manage_own','media.upload','media.delete_own',
  'social.follow','social.comment','social.like',
  'feedback.create','changelog.view',
  'assessments.view','assessments.score','series.view',
  'dashboard.view_jurusan','academic.view',
  'portfolios.moderate_jurusan',
  'users.view_jurusan','users.create_jurusan','users.edit_jurusan',
  'roles.assign_jurusan',
  'academic.manage_classes',
  'feedback.triage','feedback.reply')
ON CONFLICT DO NOTHING;

-- admin: semua permission.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.slug = 'admin'
ON CONFLICT DO NOTHING;
