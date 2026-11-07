-- name: ListMajors :many
SELECT
  m.id,
  m.code,
  m.name,
  m.short_name,
  m.is_active,
  m.sort_order,
  g.id AS group_id
FROM majors m
JOIN groups g ON g.major_id = m.id
ORDER BY m.sort_order, m.name;

-- name: GetMajorByID :one
SELECT
  m.id,
  m.code,
  m.name,
  m.short_name,
  m.is_active,
  m.sort_order,
  g.id AS group_id
FROM majors m
JOIN groups g ON g.major_id = m.id
WHERE m.id = @id::uuid;

-- name: GetMajorDetailByCode :one
SELECT
  m.id,
  m.code,
  m.name,
  m.short_name,
  m.is_active,
  m.sort_order,
  g.id AS group_id
FROM majors m
JOIN groups g ON g.major_id = m.id
WHERE m.code = @code::citext;

-- name: InsertMajor :one
INSERT INTO majors (code, name, short_name, sort_order)
VALUES (@code::citext, @name::text, @short_name::text, @sort_order::smallint)
RETURNING id;

-- name: UpdateMajor :one
UPDATE majors
SET name       = COALESCE(sqlc.narg('name')::text, name),
    short_name = COALESCE(sqlc.narg('short_name')::text, short_name),
    sort_order = COALESCE(sqlc.narg('sort_order')::smallint, sort_order),
    is_active  = COALESCE(sqlc.narg('is_active')::boolean, is_active)
WHERE id = @id::uuid
RETURNING id;

-- name: ListAcademicYears :many
SELECT id, start_year, starts_on, ends_on, is_current
FROM academic_years
ORDER BY start_year DESC;

-- name: GetAcademicYearByID :one
SELECT id, start_year, starts_on, ends_on, is_current
FROM academic_years
WHERE id = @id::uuid;

-- name: GetCurrentAcademicYear :one
SELECT id, start_year, starts_on, ends_on, is_current
FROM academic_years
WHERE is_current;

-- name: InsertAcademicYear :one
INSERT INTO academic_years (start_year, starts_on, ends_on)
VALUES (@start_year::smallint, @starts_on::date, @ends_on::date)
RETURNING id, start_year, starts_on, ends_on, is_current;

-- name: UpdateAcademicYearDates :one
UPDATE academic_years
SET starts_on = @starts_on::date,
    ends_on   = @ends_on::date
WHERE id = @id::uuid
RETURNING id, start_year, starts_on, ends_on, is_current;

-- name: ClearCurrentAcademicYear :exec
UPDATE academic_years
SET is_current = false
WHERE is_current;

-- name: SetCurrentAcademicYear :exec
UPDATE academic_years
SET is_current = true
WHERE id = @id::uuid;

-- name: ListClasses :many
SELECT
  c.id,
  c.academic_year_id,
  c.major_id,
  m.code AS major_code,
  m.short_name AS major_short_name,
  c.grade,
  c.section,
  c.name,
  c.slug,
  g.id AS group_id,
  (SELECT count(*) FROM enrollments e WHERE e.class_id = c.id)::bigint AS enrollment_count
FROM classes c
JOIN majors m ON m.id = c.major_id
JOIN groups g ON g.class_id = c.id
WHERE (sqlc.narg('academic_year_id')::uuid IS NULL OR c.academic_year_id = sqlc.narg('academic_year_id')::uuid)
  AND (sqlc.narg('major_id')::uuid IS NULL OR c.major_id = sqlc.narg('major_id')::uuid)
ORDER BY c.grade, c.section;

-- name: GetClassByID :one
SELECT
  c.id,
  c.academic_year_id,
  c.major_id,
  m.code AS major_code,
  m.short_name AS major_short_name,
  c.grade,
  c.section,
  c.name,
  c.slug,
  g.id AS group_id,
  (SELECT count(*) FROM enrollments e WHERE e.class_id = c.id)::bigint AS enrollment_count
FROM classes c
JOIN majors m ON m.id = c.major_id
JOIN groups g ON g.class_id = c.id
WHERE c.id = @id::uuid;

-- name: InsertClass :one
INSERT INTO classes (academic_year_id, major_id, grade, section, name, slug)
VALUES (@academic_year_id::uuid, @major_id::uuid, @grade::smallint, @section::text, @name::text, @slug::text)
RETURNING id;

-- name: UpdateClassSection :one
UPDATE classes
SET section = @section::text,
    name    = @name::text,
    slug    = @slug::text
WHERE id = @id::uuid
RETURNING id;

-- name: DeleteClass :exec
DELETE FROM classes
WHERE id = @id::uuid;

-- name: CountClassEnrollments :one
SELECT count(*) FROM enrollments WHERE class_id = @class_id::uuid;

-- name: InsertMajorGroup :one
INSERT INTO groups (kind, major_id)
VALUES ('major', @major_id::uuid)
RETURNING id;

-- name: InsertClassGroup :one
INSERT INTO groups (kind, class_id, status, history_since)
VALUES ('class', @class_id::uuid, @status::text, sqlc.narg('history_since')::timestamptz)
RETURNING id;

-- name: GetMajorGroupByMajorID :one
SELECT id, status, history_since
FROM groups
WHERE major_id = @major_id::uuid;

-- name: GetClassGroupByClassID :one
SELECT id, status, history_since
FROM groups
WHERE class_id = @class_id::uuid;

-- name: SetMajorGroupStatus :exec
UPDATE groups
SET status        = @status::text,
    history_since = sqlc.narg('history_since')::timestamptz
WHERE major_id = @major_id::uuid;

-- name: SetClassGroupStatus :exec
UPDATE groups
SET status        = @status::text,
    history_since = sqlc.narg('history_since')::timestamptz
WHERE class_id = @class_id::uuid;

-- name: DeleteClassGroupByClassID :exec
DELETE FROM groups
WHERE class_id = @class_id::uuid;
