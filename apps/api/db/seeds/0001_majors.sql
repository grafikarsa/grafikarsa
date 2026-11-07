-- Seed 0001: jurusan (majors) dan grup jurusan (groups kind = 'major').
-- Idempoten: majors di-upsert berdasarkan code; grup jurusan hanya dibuat
-- sekali per major (ON CONFLICT major_id DO NOTHING).
BEGIN;

INSERT INTO majors (code, name, short_name, sort_order) VALUES
  ('pg',          'Teknik Grafika — Produksi Grafika',       'PG',  1),
  ('dg',          'Teknik Grafika — Desain Grafika',         'DG',  2),
  ('dkv',         'Desain Komunikasi Visual',                'DKV', 3),
  ('animasi',     'Animasi',                                 'ANI', 4),
  ('rpl',         'Pengembangan Perangkat Lunak dan Gim (RPL)', 'RPL', 5),
  ('tkj',         'Teknik Komputer dan Jaringan',            'TKJ', 6),
  ('mekatronika', 'Teknik Mekatronika',                      'MKT', 7),
  ('logistik',    'Teknik Logistik',                         'LOG', 8),
  ('perhotelan',  'Perhotelan',                              'PH',  9)
ON CONFLICT (code) DO UPDATE
SET name       = EXCLUDED.name,
    short_name = EXCLUDED.short_name,
    sort_order = EXCLUDED.sort_order,
    is_active  = true;

INSERT INTO groups (kind, major_id)
SELECT 'major', m.id
FROM majors m
ON CONFLICT (major_id) DO NOTHING;

COMMIT;
