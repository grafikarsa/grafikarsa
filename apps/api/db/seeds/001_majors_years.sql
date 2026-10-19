-- Seed master jurusan SMKN 4 Malang + tahun ajaran.
-- Idempoten (ON CONFLICT DO NOTHING). Jalankan via `make seed`.
INSERT INTO majors (code, name) VALUES
  ('rpl', 'Rekayasa Perangkat Lunak'),
  ('tkj', 'Teknik Komputer dan Jaringan'),
  ('mm', 'Multimedia'),
  ('dkv', 'Desain Komunikasi Visual'),
  ('animasi', 'Animasi')
ON CONFLICT (code) DO NOTHING;

INSERT INTO academic_years (start_year, is_active, starts_at, ends_at) VALUES
  (2024, FALSE, '2024-07-01', '2025-06-30'),
  (2025, TRUE, '2025-07-01', '2026-06-30')
ON CONFLICT (start_year) DO NOTHING;
