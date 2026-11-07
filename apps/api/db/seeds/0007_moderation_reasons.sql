-- Seed 0007: alasan moderasi standar (moderation_reasons).
-- Idempoten: di-upsert berdasarkan slug.
BEGIN;

INSERT INTO moderation_reasons (slug, label, applies_to, sort_order) VALUES
  ('tidak-pantas',
   'Konten tidak pantas (kekerasan, seksual, kata kasar)',
   ARRAY['content', 'comment', 'profile', 'group', 'report']::text[], 1),
  ('sara-ujaran-kebencian',
   'SARA / ujaran kebencian',
   ARRAY['content', 'comment', 'profile', 'group', 'report']::text[], 2),
  ('perundungan',
   'Perundungan / pelecehan',
   ARRAY['comment', 'profile', 'report']::text[], 3),
  ('data-pribadi',
   'Membuka data pribadi orang lain',
   ARRAY['content', 'comment', 'profile', 'group', 'report']::text[], 4),
  ('plagiarisme',
   'Plagiarisme / karya orang lain tanpa kredit',
   ARRAY['content', 'report']::text[], 5),
  ('hak-cipta',
   'Pelanggaran hak cipta (aset, musik, font berbayar)',
   ARRAY['content', 'report']::text[], 6),
  ('spam',
   'Spam / promosi',
   ARRAY['content', 'comment', 'report']::text[], 7),
  ('tipe-tidak-sesuai',
   'Tidak sesuai tipe (tulisan sebagai portfolio, dll)',
   ARRAY['content']::text[], 8),
  ('kualitas-kelengkapan',
   'Belum lengkap / kualitas belum layak tayang',
   ARRAY['content']::text[], 9),
  ('informasi-menyesatkan',
   'Informasi menyesatkan',
   ARRAY['content', 'report']::text[], 10),
  ('lainnya',
   'Lainnya (wajib catatan)',
   ARRAY['content', 'comment', 'profile', 'group', 'report']::text[], 11)
ON CONFLICT (slug) DO UPDATE
SET label      = EXCLUDED.label,
    applies_to = EXCLUDED.applies_to,
    sort_order = EXCLUDED.sort_order,
    is_active  = true;

COMMIT;
