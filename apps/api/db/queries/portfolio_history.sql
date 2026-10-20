-- name: ListPortfolioStatusHistory :many
-- Riwayat transisi status sebuah karya.
--
-- PRD §8.1: transisi status dicatat "implisit via audit_logs". Action
-- transisi memakai pola `portfolio.<status>` dan diff berisi from/to, jadi
-- riwayatnya bisa dibaca ulang tanpa tabel terpisah.
--
-- $1 = id portfolio (entity_id), $2 = jumlah baris maksimum.
SELECT
  al.action,
  COALESCE(al.diff->>'from', '')::text AS from_status,
  COALESCE(al.diff->>'to', '')::text AS to_status,
  COALESCE(al.diff->>'note', '')::text AS note,
  al.actor_id,
  al.created_at
FROM audit_logs al
WHERE al.entity = 'portfolios'
  AND al.entity_id = $1
  AND al.action LIKE 'portfolio.%'
ORDER BY al.created_at DESC
LIMIT $2;
