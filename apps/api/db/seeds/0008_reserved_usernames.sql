-- Seed 0008: username terlarang (reserved_usernames) untuk route dan nama resmi.
-- Idempoten: di-insert sekali per username (ON CONFLICT DO NOTHING).
BEGIN;

INSERT INTO reserved_usernames (username, reason) VALUES
  ('admin',        'route'),
  ('api',          'route'),
  ('studio',       'route'),
  ('karya',        'route'),
  ('blog',         'route'),
  ('jurusan',      'route'),
  ('kelas',        'route'),
  ('cari',         'route'),
  ('masuk',        'route'),
  ('aktivasi',     'route'),
  ('onboarding',   'route'),
  ('lupa-sandi',   'route'),
  ('ganti-sandi',  'route'),
  ('pengaturan',   'route'),
  ('moderasi',     'route'),
  ('notifikasi',   'route'),
  ('tersimpan',    'route'),
  ('kurasi',       'route'),
  ('penugasan',    'route'),
  ('pedoman',      'route'),
  ('tentang',      'route'),
  ('privasi',      'route'),
  ('syarat',       'route'),
  ('media',        'route'),
  ('internal',     'route'),
  ('static',       'route'),
  ('_next',        'route'),

  ('grafikarsa',   'official'),
  ('grafika',      'official'),
  ('smkn4',        'official'),
  ('smkn4malang',  'official'),
  ('smkn4mlg',     'official'),
  ('official',     'official'),
  ('resmi',        'official'),
  ('administrator', 'official'),
  ('moderator',    'official'),
  ('support',      'official'),
  ('bantuan',      'official'),
  ('sekolah',      'official'),
  ('osis',         'official'),
  ('humas',        'official')
ON CONFLICT (username) DO NOTHING;

COMMIT;
