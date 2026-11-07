-- Seed 0006: katalog badge sistem (otomatis) dan badge sekolah.
-- Idempoten: di-upsert berdasarkan slug.
BEGIN;

INSERT INTO badges (slug, name, description, kind, icon_key, criteria, repeatable) VALUES
  ('karya-pertama',   'Karya Pertama',   'Menerbitkan portfolio pertama.',        'system', 'sparkles', '{"type":"published_count","content_type":"portfolio","min":1}'::jsonb, false),
  ('tulisan-pertama', 'Tulisan Pertama', 'Menerbitkan blog pertama.',             'system', 'pen-line', '{"type":"published_count","content_type":"blog","min":1}'::jsonb,      false),
  ('produktif',       'Produktif',       'Menerbitkan 10 konten.',                'system', 'flame',    '{"type":"published_count","content_type":null,"min":10}'::jsonb,         false),
  ('kolaborator',     'Kolaborator',     'Berkolaborasi di 5 konten tayang.',     'system', 'users',    '{"type":"collab_count","min":5}'::jsonb,                                 false),
  ('karya-pilihan',   'Karya Pilihan',   'Karyanya masuk kurasi Pilihan Guru.',   'system', 'award',    '{"type":"curated","curation_slug":"pilihan-guru"}'::jsonb,               false),
  ('diapresiasi',     'Diapresiasi',     'Mendapat 100 apresiasi.',               'system', 'heart',    '{"type":"appreciation_total","min":100}'::jsonb,                         false),

  ('juara-lks',       'Juara LKS',       'Menjadi juara Lomba Kompetensi Siswa.', 'school', 'trophy',          NULL, true),
  ('juara-lomba',     'Juara Lomba',     'Menjadi juara lomba di tingkat sekolah, kota, provinsi, atau nasional.', 'school', 'trophy', NULL, true),
  ('lolos-snbp',      'Lolos SNBP',      'Lolos Seleksi Nasional Berdasarkan Prestasi.', 'school', 'graduation-cap', NULL, false),
  ('lolos-snbt',      'Lolos SNBT',      'Lolos Seleksi Nasional Berdasarkan Tes.', 'school', 'graduation-cap', NULL, false),
  ('sertifikasi-lsp', 'Bersertifikat LSP', 'Mendapat sertifikasi dari Lembaga Sertifikasi Profesi.', 'school', 'badge-check', NULL, true),
  ('pengurus-osis',   'Pengurus OSIS/MPK', 'Menjabat sebagai pengurus OSIS/MPK.', 'school', 'flag',           NULL, true),
  ('karya-gkp',       'Peserta Gelar Karya Pembelajaran', 'Berpartisipasi dalam Gelar Karya Pembelajaran.', 'school', 'presentation', NULL, true)
ON CONFLICT (slug) DO UPDATE
SET name        = EXCLUDED.name,
    description = EXCLUDED.description,
    kind        = EXCLUDED.kind,
    icon_key    = EXCLUDED.icon_key,
    criteria    = EXCLUDED.criteria,
    repeatable  = EXCLUDED.repeatable,
    is_active   = true;

COMMIT;
