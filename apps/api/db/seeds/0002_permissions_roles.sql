-- Seed 0002: katalog izin (permissions), role sistem (roles), dan pemetaan
-- izin role (role_permissions).
-- Idempoten: permissions di-upsert per key, roles per slug. Sebelum insert,
-- role_permissions milik role sistem dibersihkan dari izin yang tidak lagi
-- termasuk daftar sehingga hasilnya selalu konvergen.
BEGIN;

INSERT INTO permissions (key, description, scopable) VALUES
  ('content.create',            'Membuat portfolio/blog & diundang kolaborasi',                     false),
  ('content.moderate',          'Meninjau antrian moderasi',                                        true),
  ('content.take_down',         'Menurunkan konten tayang',                                         true),
  ('comment.moderate',          'Menyembunyikan/memulihkan komentar',                               true),
  ('profile.moderate',          'Reset avatar/banner/bio/headline, hapus experience',               true),
  ('report.handle',             'Memproses laporan',                                                true),
  ('moderation.stats',          'Melihat statistik moderasi',                                       true),
  ('appeal.decide',             'Memutus banding',                                                  false),
  ('group.manage',              'Mengubah tampilan grup (di luar pengelola)',                        true),
  ('group.assign_manager',      'Menunjuk/mencabut pengelola grup',                                 true),
  ('user.view',                 'Melihat data akun di admin',                                       true),
  ('user.create',               'Membuat akun',                                                     true),
  ('user.update',               'Mengubah data akun',                                               true),
  ('user.reset_password',       'Sandi sementara, terbitkan ulang sandi awal, kartu akun',          true),
  ('user.deactivate',           'Nonaktif / aktifkan akun',                                         true),
  ('user.block_content',        'Mencabut content.create',                                          false),
  ('user.anonymize',            'Menghapus (menganonimkan) akun',                                   false),
  ('role.manage',               'Membuat/mengubah role',                                            false),
  ('role.assign',               'Memberi role (hanya subset izin milik sendiri)',                   true),
  ('academic.manage_majors',    'Mengelola data jurusan',                                           false),
  ('academic.manage_years',     'Mengelola tahun ajaran',                                           false),
  ('academic.manage_classes',   'Mengelola kelas',                                                  true),
  ('academic.import',           'Impor CSV',                                                        true),
  ('promotion.prepare',         'Mengisi pengecualian, dry-run',                                    true),
  ('promotion.approve',         'Menyetujui, eksekusi manual, batalkan',                            false),
  ('promotion.rollback',        'Membatalkan kenaikan kelas',                                       false),
  ('badge.manage',              'Katalog badge',                                                    false),
  ('badge.award',               'Memberi/mencabut badge sekolah',                                   false),
  ('taxonomy.manage',           'Kategori, skill, tools, tag, alasan moderasi, filter kata, username terlarang', false),
  ('settings.manage',           'Mengelola setting aplikasi',                                       false),
  ('audit.view',                'Melihat audit log',                                                true),
  ('dashboard.view',            'Melihat dashboard',                                                true),
  ('curation.manage',           'Mengelola kurasi (Fase 2)',                                        false),
  ('assignment.manage',         'Mengelola penugasan (Fase 2)',                                     false),
  ('assessment.score',          'Menilai asesmen (Fase 2)',                                         false)
ON CONFLICT (key) DO UPDATE
SET description = EXCLUDED.description,
    scopable    = EXCLUDED.scopable;

INSERT INTO roles (slug, name, description, is_system) VALUES
  ('member',      'Member',         'Semua akun aktif (otomatis)',              true),
  ('teacher',     'Guru',           'Semua akun guru (otomatis)',               true),
  ('moderator',   'Moderator',      'Ditunjuk dengan scope jurusan atau global', true),
  ('major_admin', 'Admin Jurusan',  'Pengelola jurusan / Kaprog',               true),
  ('admin',       'Admin',          'Admin sekolah dengan scope global',        true)
ON CONFLICT (slug) DO UPDATE
SET name        = EXCLUDED.name,
    description = EXCLUDED.description,
    is_system   = EXCLUDED.is_system;

-- Konvergensi: buang izin role sistem yang tidak ada di daftar yang diinginkan.
-- Untuk admin, daftar yang diinginkan adalah seluruh katalog permissions.
DELETE FROM role_permissions rp
USING roles r
WHERE rp.role_id = r.id
  AND (
    (r.slug = 'member' AND rp.permission_key <> 'content.create')
    OR (r.slug = 'teacher' AND rp.permission_key NOT IN (
          'curation.manage', 'assignment.manage', 'assessment.score'))
    OR (r.slug = 'moderator' AND rp.permission_key NOT IN (
          'content.moderate', 'content.take_down', 'comment.moderate',
          'profile.moderate', 'report.handle', 'moderation.stats'))
    OR (r.slug = 'major_admin' AND rp.permission_key NOT IN (
          'content.moderate', 'content.take_down', 'comment.moderate',
          'profile.moderate', 'report.handle', 'moderation.stats',
          'group.manage', 'group.assign_manager', 'user.view', 'user.create',
          'user.update', 'user.reset_password', 'user.deactivate',
          'role.assign', 'academic.manage_classes', 'academic.import',
          'promotion.prepare', 'audit.view', 'dashboard.view'))
    OR (r.slug = 'admin' AND NOT EXISTS (
          SELECT 1 FROM permissions p WHERE p.key = rp.permission_key))
  );

INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, v.permission_key
FROM (VALUES
  ('member',      'content.create'),
  ('teacher',     'curation.manage'),
  ('teacher',     'assignment.manage'),
  ('teacher',     'assessment.score'),
  ('moderator',   'content.moderate'),
  ('moderator',   'content.take_down'),
  ('moderator',   'comment.moderate'),
  ('moderator',   'profile.moderate'),
  ('moderator',   'report.handle'),
  ('moderator',   'moderation.stats'),
  ('major_admin', 'content.moderate'),
  ('major_admin', 'content.take_down'),
  ('major_admin', 'comment.moderate'),
  ('major_admin', 'profile.moderate'),
  ('major_admin', 'report.handle'),
  ('major_admin', 'moderation.stats'),
  ('major_admin', 'group.manage'),
  ('major_admin', 'group.assign_manager'),
  ('major_admin', 'user.view'),
  ('major_admin', 'user.create'),
  ('major_admin', 'user.update'),
  ('major_admin', 'user.reset_password'),
  ('major_admin', 'user.deactivate'),
  ('major_admin', 'role.assign'),
  ('major_admin', 'academic.manage_classes'),
  ('major_admin', 'academic.import'),
  ('major_admin', 'promotion.prepare'),
  ('major_admin', 'audit.view'),
  ('major_admin', 'dashboard.view')
) AS v(role_slug, permission_key)
JOIN roles r ON r.slug = v.role_slug
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, p.key
FROM roles r
CROSS JOIN permissions p
WHERE r.slug = 'admin'
ON CONFLICT DO NOTHING;

COMMIT;
