-- name: ListContentCategories :many
SELECT
    c.id,
    c.slug::text AS slug,
    c.name,
    c.description,
    COALESCE(
        (
            SELECT array_agg(m.code::text ORDER BY m.code::text)
            FROM content_category_majors ccm
            JOIN majors m ON m.id = ccm.major_id
            WHERE ccm.category_id = c.id
        ),
        ARRAY[]::text[]
    )::text[] AS major_codes
FROM content_categories c
WHERE c.content_type = sqlc.arg(content_type)::text AND c.is_active
ORDER BY c.sort_order, c.name;

-- name: ListActiveTools :many
SELECT slug::text AS slug, name, category
FROM tools
WHERE is_active
ORDER BY name
LIMIT 20;

-- name: SearchTools :many
SELECT slug::text AS slug, name, category
FROM tools
WHERE is_active
  AND (
      name ILIKE '%' || sqlc.arg(q)::text || '%'
      OR slug::text ILIKE sqlc.arg(q)::text || '%'
  )
ORDER BY
    (lower(name) = lower(sqlc.arg(q)::text)) DESC,
    (name ILIKE sqlc.arg(q)::text || '%') DESC,
    similarity(name, sqlc.arg(q)::text) DESC,
    name
LIMIT 20;

-- name: ListPopularTags :many
SELECT t.slug::text AS slug, t.name
FROM tags t
LEFT JOIN content_tags ct ON ct.tag_id = t.id
WHERE NOT t.is_blocked
GROUP BY t.id
ORDER BY count(ct.tag_id) DESC, t.name
LIMIT 20;

-- name: SearchTags :many
SELECT slug::text AS slug, name
FROM tags
WHERE NOT is_blocked
  AND (
      name ILIKE '%' || sqlc.arg(q)::text || '%'
      OR slug::text ILIKE sqlc.arg(q)::text || '%'
  )
ORDER BY
    (lower(name) = lower(sqlc.arg(q)::text)) DESC,
    (name ILIKE sqlc.arg(q)::text || '%') DESC,
    similarity(name, sqlc.arg(q)::text) DESC,
    name
LIMIT 20;
