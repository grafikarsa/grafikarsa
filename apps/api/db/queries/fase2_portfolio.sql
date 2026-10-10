-- Helpers slug + portfolio CRUD.

-- name: CreatePortfolio :one
INSERT INTO portfolios (owner_id, title, slug, series_id)
VALUES ($1, $2, $3, $4) RETURNING id;

-- name: GetPortfolioByID :one
SELECT id, owner_id, title, slug, status, thumbnail_media_id, series_id,
  draft_payload, published_at, submitted_at, reviewed_by, reviewed_at, review_note, created_at
FROM portfolios WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetPortfolioBySlug :one
SELECT id, owner_id, title, slug, status, thumbnail_media_id, series_id,
  draft_payload, published_at, submitted_at, reviewed_by, reviewed_at, review_note, created_at
FROM portfolios WHERE owner_id = $1 AND slug = $2 AND deleted_at IS NULL LIMIT 1;

-- name: UpdatePortfolioTitle :exec
UPDATE portfolios SET title = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdatePortfolioThumbnail :exec
UPDATE portfolios SET thumbnail_media_id = $2, updated_at = NOW() WHERE id = $1;

-- name: UpdatePortfolioSeries :exec
UPDATE portfolios SET series_id = $2, updated_at = NOW() WHERE id = $1;

-- name: SubmitPortfolio :exec
UPDATE portfolios SET status = 'pending_review', submitted_at = NOW(),
  submitted_snapshot = $2, updated_at = NOW() WHERE id = $1;

-- name: CancelSubmit :exec
UPDATE portfolios SET status = 'draft', updated_at = NOW() WHERE id = $1;

-- name: ModeratePortfolio :exec
UPDATE portfolios SET status = $2, review_note = $3, reviewed_by = $4,
  reviewed_at = NOW(), published_at = CASE WHEN $2 = 'published' THEN NOW() ELSE published_at END,
  updated_at = NOW() WHERE id = $1;

-- name: ArchivePortfolio :exec
UPDATE portfolios SET status = 'archived', updated_at = NOW() WHERE id = $1;

-- name: UnarchivePortfolio :exec
UPDATE portfolios SET status = 'draft', updated_at = NOW() WHERE id = $1;

-- name: SoftDeletePortfolio :exec
UPDATE portfolios SET deleted_at = NOW() WHERE id = $1;

-- name: RestorePortfolio :exec
UPDATE portfolios SET deleted_at = NULL WHERE id = $1;

-- name: ListPublishedPortfolios :many
SELECT p.id, p.owner_id, p.title, p.slug, p.thumbnail_media_id, p.published_at,
  u.username, u.name AS owner_name
FROM portfolios p JOIN users u ON u.id = p.owner_id
WHERE p.status = 'published' AND p.deleted_at IS NULL
ORDER BY p.published_at DESC LIMIT $1 OFFSET $2;

-- name: CountPublishedPortfolios :one
SELECT COUNT(*) FROM portfolios WHERE status = 'published' AND deleted_at IS NULL;

-- name: ListMyPortfolios :many
SELECT id, title, slug, status, thumbnail_media_id, published_at, created_at
FROM portfolios WHERE owner_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: ListPendingPortfolios :many
SELECT p.id, p.owner_id, p.title, p.slug, p.submitted_at, u.username, u.name AS owner_name
FROM portfolios p JOIN users u ON u.id = p.owner_id
WHERE p.status = 'pending_review' AND p.deleted_at IS NULL
ORDER BY p.submitted_at ASC LIMIT $1 OFFSET $2;

-- name: CountPortfolioBlocks :one
SELECT COUNT(*) FROM portfolio_blocks WHERE portfolio_id = $1;

-- name: CountPortfolioTags :one
SELECT COUNT(*) FROM portfolio_tags WHERE portfolio_id = $1;

-- name: DeletePortfolioTags :exec
DELETE FROM portfolio_tags WHERE portfolio_id = $1;

-- name: AddPortfolioTag :exec
INSERT INTO portfolio_tags (portfolio_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING;

-- name: ListPortfolioTags :many
SELECT t.id, t.slug, t.name FROM tags t
JOIN portfolio_tags pt ON pt.tag_id = t.id WHERE pt.portfolio_id = $1 ORDER BY t.name;

-- name: EnsureTag :one
INSERT INTO tags (slug, name) VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name RETURNING id;

-- name: ListTags :many
SELECT id, slug, name FROM tags ORDER BY name ASC LIMIT $1 OFFSET $2;

-- name: CreateSeries :one
INSERT INTO series (slug, name, description) VALUES ($1, $2, $3) RETURNING id;

-- name: ListSeries :many
SELECT id, slug, name, description, is_active FROM series ORDER BY name ASC LIMIT $1 OFFSET $2;

-- name: GetSeries :one
SELECT id, slug, name, description, is_active FROM series WHERE id = $1 LIMIT 1;

-- name: ListSeriesBlocks :many
SELECT id, series_id, block_type, position, instruction FROM series_blocks
WHERE series_id = $1 ORDER BY position ASC;

-- name: CreateBlock :one
INSERT INTO portfolio_blocks (portfolio_id, block_type, position, payload, asset_media_id, is_from_series)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING id;

-- name: ListBlocks :many
SELECT id, portfolio_id, block_type, position, payload, asset_media_id, is_from_series
FROM portfolio_blocks WHERE portfolio_id = $1 ORDER BY position ASC;

-- name: GetBlock :one
SELECT id, portfolio_id, block_type, position, payload, asset_media_id, is_from_series
FROM portfolio_blocks WHERE id = $1 LIMIT 1;

-- name: UpdateBlockPayload :exec
UPDATE portfolio_blocks SET payload = $2 WHERE id = $1;

-- name: UpdateBlockPosition :exec
UPDATE portfolio_blocks SET position = $2 WHERE id = $1;

-- name: DeleteBlock :exec
DELETE FROM portfolio_blocks WHERE id = $1;

-- name: MaxBlockPosition :one
SELECT COALESCE(MAX(position), -1)::int FROM portfolio_blocks WHERE portfolio_id = $1;

-- name: FilterPublishedPortfolios :many
SELECT p.id, p.owner_id, p.title, p.slug, p.thumbnail_media_id, p.published_at,
  u.username, u.name AS owner_name
FROM portfolios p JOIN users u ON u.id = p.owner_id
WHERE p.status = 'published' AND p.deleted_at IS NULL
  AND ($1 = '' OR p.title ILIKE '%' || $1 || '%' OR u.name ILIKE '%' || $1 || '%')
ORDER BY p.published_at DESC LIMIT $2 OFFSET $3;

-- name: CountFilteredPortfolios :one
SELECT COUNT(*) FROM portfolios p
JOIN users u ON u.id = p.owner_id
WHERE p.status = 'published' AND p.deleted_at IS NULL
  AND ($1 = '' OR p.title ILIKE '%' || $1 || '%' OR u.name ILIKE '%' || $1 || '%');

-- name: FilterByTag :many
SELECT p.id, p.owner_id, p.title, p.slug, p.thumbnail_media_id, p.published_at,
  u.username, u.name AS owner_name
FROM portfolios p JOIN users u ON u.id = p.owner_id
JOIN portfolio_tags pt ON pt.portfolio_id = p.id AND pt.tag_id = $1
WHERE p.status = 'published' AND p.deleted_at IS NULL
ORDER BY p.published_at DESC LIMIT $2 OFFSET $3;

-- name: FilterByMajor :many
SELECT p.id, p.owner_id, p.title, p.slug, p.thumbnail_media_id, p.published_at,
  u.username, u.name AS owner_name
FROM portfolios p JOIN users u ON u.id = p.owner_id
WHERE p.status = 'published' AND p.deleted_at IS NULL AND u.major_id = $1
ORDER BY p.published_at DESC LIMIT $2 OFFSET $3;

-- name: FilterBySearch :many
SELECT p.id, p.owner_id, p.title, p.slug, p.thumbnail_media_id, p.published_at,
  u.username, u.name AS owner_name
FROM portfolios p JOIN users u ON u.id = p.owner_id
WHERE p.status = 'published' AND p.deleted_at IS NULL
  AND (p.title ILIKE '%' || $1 || '%' OR p.slug ILIKE '%' || $1 || '%')
ORDER BY p.published_at DESC LIMIT $2 OFFSET $3;

-- name: AddSeriesBlock :exec
INSERT INTO series_blocks (series_id, block_type, instruction, position)
VALUES ($1, $2, $3, $4) ON CONFLICT (series_id, position) DO UPDATE SET instruction = EXCLUDED.instruction;

-- name: ListPortfoliosForAdmin :many
SELECT p.id, p.title, p.slug, p.status, p.owner_id, p.submitted_at, p.published_at,
  u.username, u.name AS owner_name, u.major_id
FROM portfolios p JOIN users u ON u.id = p.owner_id
WHERE p.deleted_at IS NULL
  AND ($1::text = '' OR p.status = $1)
  AND ($2::text = '' OR u.major_id::text = $2)
ORDER BY COALESCE(p.submitted_at, p.created_at) DESC LIMIT $3 OFFSET $4;

-- name: CountPortfoliosForAdmin :one
SELECT COUNT(*) FROM portfolios p JOIN users u ON u.id = p.owner_id
WHERE p.deleted_at IS NULL
  AND ($1::text = '' OR p.status = $1)
  AND ($2::text = '' OR u.major_id::text = $2);
