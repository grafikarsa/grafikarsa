-- name: ListActiveMajors :many
SELECT id, code, name, short_name, sort_order FROM majors WHERE is_active ORDER BY sort_order, name;

-- name: GetMajorByCode :one
SELECT id, code, name, short_name, is_active, sort_order FROM majors WHERE code = $1;
