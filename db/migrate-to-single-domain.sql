-- ============================================================
-- GRAFIKARSA — Migrasi URL storage subdomain -> single-domain
-- ============================================================
-- Dipakai HANYA jika DB berisi URL lama berbentuk:
--   https://storage.grafikarsa.com/grafikarsa/...
-- dan deployment baru memakai:
--   https://grafikarsa.com/storage/grafikarsa/...
--
-- CARA PAKAI (ganti 2 nilai di bawah, backup dulu!):
--   docker exec -i grafikarsa-db psql -U grafikarsa -d grafikarsa \
--     < db/migrate-to-single-domain.sql
--
-- Atau dengan variabel:
--   docker exec -i grafikarsa-db psql -U grafikarsa -d grafikarsa \
--     -v old_host='https://storage.grafikarsa.com/grafikarsa' \
--     -v new_base='https://grafikarsa.com/storage/grafikarsa' \
--     -f db/migrate-to-single-domain.sql
--
-- Aman diulang: hanya baris yang masih mengandung old_host yang diubah.
-- ============================================================

\if :{?old_host}
-- dipanggil dengan -v, pakai variabel
\else
-- nilai default: EDIT DI SINI jika tidak pakai -v
\set old_host 'https://storage.grafikarsa.com/grafikarsa'
\set new_base 'https://grafikarsa.com/storage/grafikarsa'
\endif

-- 1. users: avatar & banner
UPDATE users
SET avatar_url = replace(avatar_url, :'old_host', :'new_base')
WHERE avatar_url LIKE '%' || :'old_host' || '%';

UPDATE users
SET banner_url = replace(banner_url, :'old_host', :'new_base')
WHERE banner_url LIKE '%' || :'old_host' || '%';

-- 2. portfolios: thumbnail
UPDATE portfolios
SET thumbnail_url = replace(thumbnail_url, :'old_host', :'new_base')
WHERE thumbnail_url LIKE '%' || :'old_host' || '%';

-- 3. feedback: attachment
UPDATE feedback
SET attachment_url = replace(attachment_url, :'old_host', :'new_base')
WHERE attachment_url LIKE '%' || :'old_host' || '%';

-- 4. content_blocks payload (JSONB berisi URL image/file per block)
--    Hati-hati: rewrite teks JSON. Backup dulu, verifikasi count sesudahnya.
UPDATE content_blocks
SET payload = replace(payload::text, :'old_host', :'new_base')::jsonb
WHERE payload::text LIKE '%' || :'old_host' || '%';

-- 5. Verifikasi: harusnya 0 tersisa
SELECT 'users.avatar_url' AS kolom, count(*) AS sisa_url_lama FROM users WHERE avatar_url LIKE '%' || :'old_host' || '%'
UNION ALL
SELECT 'users.banner_url', count(*) FROM users WHERE banner_url LIKE '%' || :'old_host' || '%'
UNION ALL
SELECT 'portfolios.thumbnail_url', count(*) FROM portfolios WHERE thumbnail_url LIKE '%' || :'old_host' || '%'
UNION ALL
SELECT 'feedback.attachment_url', count(*) FROM feedback WHERE attachment_url LIKE '%' || :'old_host' || '%'
UNION ALL
SELECT 'content_blocks.payload', count(*) FROM content_blocks WHERE payload::text LIKE '%' || :'old_host' || '%';
