-- name: ListUsers :many
SELECT
  u.id,
  u.account_type,
  u.status,
  u.display_name,
  u.username,
  u.nisn,
  u.staff_number,
  (u.activated_at IS NOT NULL)::boolean AS activated,
  (u.content_blocked_at IS NOT NULL)::boolean AS content_blocked,
  u.last_login_at,
  u.created_at,
  c.id AS class_id,
  c.name AS class_name,
  m.code AS class_major_code,
  y.start_year AS class_start_year
FROM users u
LEFT JOIN LATERAL (
  SELECT e.class_id
  FROM enrollments e
  JOIN academic_years ay ON ay.id = e.academic_year_id
  WHERE e.user_id = u.id
  ORDER BY ay.start_year DESC
  LIMIT 1
) le ON true
LEFT JOIN classes c ON c.id = le.class_id
LEFT JOIN majors m ON m.id = c.major_id
LEFT JOIN academic_years y ON y.id = c.academic_year_id
WHERE (sqlc.narg('account_type')::text IS NULL OR u.account_type = sqlc.narg('account_type')::text)
  AND (sqlc.narg('status')::text IS NULL OR u.status = sqlc.narg('status')::text)
  AND (
    sqlc.narg('major_id')::uuid IS NULL
    OR (
      (u.account_type = 'student' AND EXISTS (
        SELECT 1
        FROM enrollments e2
        JOIN classes c2 ON c2.id = e2.class_id
        JOIN academic_years y2 ON y2.id = e2.academic_year_id
        WHERE e2.user_id = u.id
          AND c2.major_id = sqlc.narg('major_id')::uuid
          AND y2.start_year = (
            SELECT max(y3.start_year)
            FROM enrollments e3
            JOIN academic_years y3 ON y3.id = e3.academic_year_id
            WHERE e3.user_id = u.id
          )
      ))
      OR (u.account_type = 'teacher' AND EXISTS (
        SELECT 1
        FROM teacher_majors tm
        WHERE tm.user_id = u.id
          AND tm.major_id = sqlc.narg('major_id')::uuid
      ))
    )
  )
  AND (
    sqlc.narg('class_id')::uuid IS NULL
    OR EXISTS (
      SELECT 1
      FROM enrollments e4
      WHERE e4.user_id = u.id
        AND e4.class_id = sqlc.narg('class_id')::uuid
    )
  )
  AND (
    sqlc.narg('query')::text IS NULL
    OR u.display_name ILIKE '%' || sqlc.narg('query')::text || '%'
    OR u.username::text ILIKE '%' || sqlc.narg('query')::text || '%'
    OR u.nisn = sqlc.narg('query')::text
    OR u.staff_number = sqlc.narg('query')::text
  )
  AND (
    @scope_global::boolean
    OR (
      cardinality(@scope_major_ids::uuid[]) > 0
      AND (
        (u.account_type = 'student' AND EXISTS (
          SELECT 1
          FROM enrollments e5
          JOIN classes c5 ON c5.id = e5.class_id
          JOIN academic_years y5 ON y5.id = e5.academic_year_id
          WHERE e5.user_id = u.id
            AND c5.major_id = ANY(@scope_major_ids::uuid[])
            AND y5.start_year = (
              SELECT max(y6.start_year)
              FROM enrollments e6
              JOIN academic_years y6 ON y6.id = e6.academic_year_id
              WHERE e6.user_id = u.id
            )
        ))
        OR (u.account_type = 'teacher' AND EXISTS (
          SELECT 1
          FROM teacher_majors tm2
          WHERE tm2.user_id = u.id
            AND tm2.major_id = ANY(@scope_major_ids::uuid[])
        ))
      )
    )
  )
  AND (
    sqlc.narg('cursor_time')::timestamptz IS NULL
    OR (u.created_at, u.id) < (
      sqlc.narg('cursor_time')::timestamptz,
      sqlc.narg('cursor_id')::uuid
    )
  )
ORDER BY u.created_at DESC, u.id DESC
LIMIT @limit_count;

-- name: GetUserAdminDetail :one
SELECT
  u.id,
  u.account_type,
  u.status,
  u.display_name,
  u.username,
  u.nisn,
  u.staff_number,
  u.entry_year,
  u.graduation_year,
  (u.activated_at IS NOT NULL)::boolean AS activated,
  u.password_kind,
  u.password_expires_at,
  (u.content_blocked_at IS NOT NULL)::boolean AS content_blocked,
  u.last_login_at,
  u.deactivated_at,
  u.created_at,
  c.id AS class_id,
  c.name AS class_name,
  c.major_id AS class_major_id,
  m.code AS class_major_code,
  y.start_year AS class_start_year
FROM users u
LEFT JOIN LATERAL (
  SELECT e.class_id
  FROM enrollments e
  JOIN academic_years ay ON ay.id = e.academic_year_id
  WHERE e.user_id = u.id
  ORDER BY ay.start_year DESC
  LIMIT 1
) le ON true
LEFT JOIN classes c ON c.id = le.class_id
LEFT JOIN majors m ON m.id = c.major_id
LEFT JOIN academic_years y ON y.id = c.academic_year_id
WHERE u.id = @id::uuid;

-- name: ListTeacherMajorIDs :many
SELECT major_id
FROM teacher_majors
WHERE user_id = @user_id::uuid
ORDER BY major_id;

-- name: ListUserRoleAssignments :many
SELECT
  ur.id,
  r.slug::text AS role_slug,
  r.name AS role_name,
  ur.major_id,
  ur.expires_at
FROM user_roles ur
JOIN roles r ON r.id = ur.role_id
WHERE ur.user_id = @user_id::uuid
  AND ur.revoked_at IS NULL
  AND (ur.expires_at IS NULL OR ur.expires_at > now())
ORDER BY ur.assigned_at, ur.id;

-- name: ListUserEnrollments :many
SELECT
  e.class_id,
  c.name AS class_name,
  m.code AS major_code,
  y.start_year,
  e.outcome
FROM enrollments e
JOIN classes c ON c.id = e.class_id
JOIN majors m ON m.id = c.major_id
JOIN academic_years y ON y.id = e.academic_year_id
WHERE e.user_id = @user_id::uuid
ORDER BY y.start_year DESC, e.created_at DESC;

-- name: CountUserActiveSessions :one
SELECT count(*)::bigint
FROM sessions
WHERE user_id = @user_id::uuid
  AND revoked_at IS NULL
  AND expires_at > now()
  AND idle_expires_at > now();

-- name: InsertUser :one
INSERT INTO users (
  account_type,
  nisn,
  staff_number,
  display_name,
  password_hash,
  password_kind,
  password_expires_at,
  entry_year
)
VALUES (
  @account_type::text,
  sqlc.narg('nisn')::text,
  sqlc.narg('staff_number')::text,
  @display_name::text,
  @password_hash::text,
  @password_kind::text,
  @password_expires_at::timestamptz,
  sqlc.narg('entry_year')::smallint
)
RETURNING id;

-- name: InsertEnrollment :exec
INSERT INTO enrollments (user_id, class_id, academic_year_id, source)
VALUES (
  @user_id::uuid,
  @class_id::uuid,
  @academic_year_id::uuid,
  @source::text
);

-- name: InsertTeacherMajor :exec
INSERT INTO teacher_majors (user_id, major_id)
VALUES (@user_id::uuid, @major_id::uuid);

-- name: DeleteTeacherMajors :exec
DELETE FROM teacher_majors
WHERE user_id = @user_id::uuid;

-- name: InsertSystemExperience :exec
INSERT INTO experiences (
  user_id,
  kind,
  title,
  organization,
  start_month,
  end_month,
  is_current,
  is_system
)
VALUES (
  @user_id::uuid,
  'education',
  @title::text,
  @organization::text,
  @start_month::date,
  NULL,
  true,
  true
);

-- name: UpdateUserProfile :exec
UPDATE users
SET account_type = @account_type::text,
    display_name = @display_name::text,
    nisn = sqlc.narg('nisn')::text,
    staff_number = sqlc.narg('staff_number')::text
WHERE id = @id::uuid;

-- name: SetUserPassword :exec
UPDATE users
SET password_hash = @password_hash::text,
    password_kind = @password_kind::text,
    password_expires_at = @password_expires_at::timestamptz
WHERE id = @id::uuid;

-- name: DeactivateUser :exec
UPDATE users
SET status = 'inactive',
    deactivated_at = now()
WHERE id = @id::uuid;

-- name: ReactivateUser :exec
UPDATE users
SET status = CASE WHEN graduation_year IS NOT NULL THEN 'alumni' ELSE 'active' END,
    deactivated_at = NULL
WHERE id = @id::uuid;

-- name: BlockUserContent :exec
UPDATE users
SET content_blocked_at = now(),
    content_blocked_reason = @reason::text
WHERE id = @id::uuid;

-- name: UnblockUserContent :exec
UPDATE users
SET content_blocked_at = NULL,
    content_blocked_reason = NULL
WHERE id = @id::uuid;
