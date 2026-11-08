-- name: InsertMediaAsset :one
INSERT INTO media_assets (
  owner_id, kind, purpose, original_key, original_filename, mime_type,
  size_bytes, checksum_sha256, status
) VALUES (
  @owner_id, @kind, @purpose, @original_key, @original_filename, @mime_type,
  @size_bytes, @checksum_sha256, 'uploaded'
)
RETURNING *;

-- name: GetOwnedMediaAsset :one
SELECT * FROM media_assets
WHERE id = @id AND owner_id = @owner_id AND deleted_at IS NULL;

-- name: SoftDeleteMediaAsset :execrows
UPDATE media_assets
SET deleted_at = now()
WHERE id = @id AND owner_id = @owner_id AND deleted_at IS NULL;

-- name: MediaAssetIsReferenced :one
SELECT count(*) > 0 AS referenced
FROM (
  SELECT 1 FROM content_draft_media WHERE content_draft_media.media_id = @asset_id
  UNION ALL
  SELECT 1 FROM content_version_media WHERE content_version_media.media_id = @asset_id
  UNION ALL
  SELECT 1 FROM profiles WHERE profiles.avatar_media_id = @asset_id OR profiles.banner_media_id = @asset_id
  UNION ALL
  SELECT 1 FROM groups WHERE groups.avatar_media_id = @asset_id OR groups.banner_media_id = @asset_id
  UNION ALL
  SELECT 1 FROM experience_attachments WHERE experience_attachments.media_id = @asset_id
) AS refs;
