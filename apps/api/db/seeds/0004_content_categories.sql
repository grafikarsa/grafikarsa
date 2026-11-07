-- Seed 0004: kategori konten portfolio & blog, beserta saran jurusan.
-- Idempoten: kategori di-upsert berdasarkan (content_type, slug); pemetaan
-- jurusan hanya ditambah jika belum ada (ON CONFLICT DO NOTHING).
BEGIN;

INSERT INTO content_categories (content_type, slug, name, sort_order) VALUES
  ('portfolio', 'cetak-kemasan',          'Cetak & Kemasan',                              1),
  ('portfolio', 'pracetak-layout',        'Pracetak & Layout',                            2),
  ('portfolio', 'branding-identitas',     'Branding & Identitas Visual',                  3),
  ('portfolio', 'poster-publikasi',       'Poster & Publikasi',                           4),
  ('portfolio', 'ilustrasi',              'Ilustrasi',                                    5),
  ('portfolio', 'fotografi',              'Fotografi',                                    6),
  ('portfolio', 'ui-ux',                  'UI/UX Design',                                 7),
  ('portfolio', 'video-film',             'Video & Film',                                 8),
  ('portfolio', 'animasi-2d',             'Animasi 2D',                                   9),
  ('portfolio', 'animasi-3d',             'Animasi & Modeling 3D',                       10),
  ('portfolio', 'motion-graphic',         'Motion Graphic',                               11),
  ('portfolio', 'web-app',                'Aplikasi Web',                                 12),
  ('portfolio', 'mobile-app',             'Aplikasi Mobile',                              13),
  ('portfolio', 'game',                   'Game',                                         14),
  ('portfolio', 'jaringan-infrastruktur', 'Jaringan & Infrastruktur',                     15),
  ('portfolio', 'keamanan-siber',         'Keamanan Siber',                               16),
  ('portfolio', 'iot-embedded',           'IoT & Embedded',                               17),
  ('portfolio', 'otomasi-robotik',        'Otomasi & Robotik',                            18),
  ('portfolio', 'logistik-rantai-pasok',  'Logistik & Rantai Pasok',                      19),
  ('portfolio', 'layanan-perhotelan',     'Layanan Perhotelan',                           20),
  ('portfolio', 'tata-boga-minuman',      'Tata Hidang & Minuman',                        21),
  ('portfolio', 'lainnya',                'Lainnya',                                      22),
  ('blog',      'tutorial',               'Tutorial',                                      1),
  ('blog',      'cerita-pengalaman',      'Cerita & Pengalaman (PKL, lomba, kuliah)',      2),
  ('blog',      'opini',                  'Opini',                                         3),
  ('blog',      'review',                 'Review',                                        4),
  ('blog',      'tips',                   'Tips',                                          5),
  ('blog',      'lainnya',                'Lainnya',                                       6)
ON CONFLICT (content_type, slug) DO UPDATE
SET name       = EXCLUDED.name,
    sort_order = EXCLUDED.sort_order,
    is_active  = true;

INSERT INTO content_category_majors (category_id, major_id)
SELECT c.id, m.id
FROM (VALUES
  ('cetak-kemasan',          'pg'),
  ('cetak-kemasan',          'dg'),
  ('pracetak-layout',        'dg'),
  ('pracetak-layout',        'pg'),
  ('branding-identitas',     'dg'),
  ('branding-identitas',     'dkv'),
  ('poster-publikasi',       'dg'),
  ('poster-publikasi',       'dkv'),
  ('ilustrasi',              'dkv'),
  ('ilustrasi',              'animasi'),
  ('ilustrasi',              'dg'),
  ('fotografi',              'dkv'),
  ('fotografi',              'animasi'),
  ('ui-ux',                  'dkv'),
  ('ui-ux',                  'rpl'),
  ('video-film',             'dkv'),
  ('video-film',             'animasi'),
  ('animasi-2d',             'animasi'),
  ('animasi-3d',             'animasi'),
  ('motion-graphic',         'animasi'),
  ('motion-graphic',         'dkv'),
  ('web-app',                'rpl'),
  ('mobile-app',             'rpl'),
  ('game',                   'rpl'),
  ('game',                   'animasi'),
  ('jaringan-infrastruktur', 'tkj'),
  ('keamanan-siber',         'tkj'),
  ('keamanan-siber',         'rpl'),
  ('iot-embedded',           'mekatronika'),
  ('iot-embedded',           'tkj'),
  ('otomasi-robotik',        'mekatronika'),
  ('logistik-rantai-pasok',  'logistik'),
  ('layanan-perhotelan',     'perhotelan'),
  ('tata-boga-minuman',      'perhotelan')
) AS v(category_slug, major_code)
JOIN content_categories c
  ON c.content_type = 'portfolio' AND c.slug = v.category_slug
JOIN majors m ON m.code = v.major_code
ON CONFLICT DO NOTHING;

-- Kategori "Lainnya" relevan untuk semua jurusan.
INSERT INTO content_category_majors (category_id, major_id)
SELECT c.id, m.id
FROM content_categories c
CROSS JOIN majors m
WHERE c.content_type = 'portfolio' AND c.slug = 'lainnya'
ON CONFLICT DO NOTHING;

COMMIT;
