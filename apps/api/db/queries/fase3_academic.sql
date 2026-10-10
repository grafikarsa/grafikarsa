-- Akademik: majors, years, classes, enrollments, promotion.

-- name: ListMajorsFull :many
SELECT id, code, name, require_series FROM majors ORDER BY name ASC;

-- name: CreateMajor :one
INSERT INTO majors (code, name) VALUES ($1, $2) RETURNING id;

-- name: UpdateMajor :exec
UPDATE majors SET name = $2 WHERE id = $1;

-- name: DeleteMajor :exec
DELETE FROM majors WHERE id = $1;

-- name: ListYears :many
SELECT id, start_year, is_active, promotion_month, promotion_day, starts_at, ends_at
FROM academic_years ORDER BY start_year DESC;

-- name: CreateYear :one
INSERT INTO academic_years (start_year, starts_at, ends_at) VALUES ($1, $2, $3) RETURNING id;

-- name: SetActiveYear :exec
UPDATE academic_years SET is_active = (id = $1);

-- name: GetActiveYear :one
SELECT id, start_year FROM academic_years WHERE is_active = TRUE LIMIT 1;

-- name: ListClasses :many
SELECT c.id, c.academic_year_id, c.major_id, c.grade, c.class_group, c.name,
  m.code AS major_code, y.start_year
FROM classes c JOIN majors m ON m.id = c.major_id
JOIN academic_years y ON y.id = c.academic_year_id
WHERE ($1 = '' OR c.academic_year_id::text = $1)
  AND ($2 = '' OR c.major_id::text = $2)
ORDER BY y.start_year DESC, m.code, c.grade, c.class_group
LIMIT $3 OFFSET $4;

-- name: CreateClass :one
INSERT INTO classes (academic_year_id, major_id, grade, class_group, name)
VALUES ($1, $2, $3, $4, $5) RETURNING id;

-- name: DeleteClass :exec
DELETE FROM classes WHERE id = $1;

-- name: ListClassStudents :many
SELECT u.id, u.username, u.name FROM users u
JOIN enrollments e ON e.user_id = u.id AND e.is_current = TRUE
WHERE e.class_id = $1 AND u.deleted_at IS NULL ORDER BY u.name ASC;

-- name: UpsertEnrollment :exec
INSERT INTO enrollments (user_id, class_id, academic_year_id, status, is_current)
VALUES ($1, $2, $3, $4, TRUE)
ON CONFLICT (user_id, academic_year_id) DO UPDATE
SET class_id = EXCLUDED.class_id, status = EXCLUDED.status, is_current = TRUE;

-- name: DeactivateEnrollments :exec
UPDATE enrollments SET is_current = FALSE WHERE user_id = $1 AND academic_year_id = $2;

-- name: GetCurrentEnrollment :one
SELECT id, user_id, class_id, academic_year_id, status FROM enrollments
WHERE user_id = $1 AND is_current = TRUE LIMIT 1;

-- name: CreatePromotionJob :one
INSERT INTO promotion_jobs (from_year_id, to_year_id, status, dryrun_result)
VALUES ($1, $2, 'previewed', $3) RETURNING id;

-- name: GetPromotionJob :one
SELECT id, from_year_id, to_year_id, status, dryrun_result, exceptions, executed_by, executed_at
FROM promotion_jobs WHERE id = $1 LIMIT 1;

-- name: ListPromotionJobs :many
SELECT id, from_year_id, to_year_id, status, executed_at, created_at
FROM promotion_jobs ORDER BY created_at DESC LIMIT 20;

-- name: ApprovePromotionJob :exec
UPDATE promotion_jobs SET status = 'approved', exceptions = $2 WHERE id = $1;

-- name: MarkPromotionExecuting :exec
UPDATE promotion_jobs SET status = 'executing' WHERE id = $1 AND status = 'approved';

-- name: MarkPromotionDone :exec
UPDATE promotion_jobs SET status = 'done', executed_by = $2, executed_at = NOW() WHERE id = $1;

-- name: CreateClassForYear :one
INSERT INTO classes (academic_year_id, major_id, grade, class_group, name)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (academic_year_id, major_id, grade, class_group) DO UPDATE SET name = EXCLUDED.name
RETURNING id;

-- name: ListEnrollmentsByYear :many
SELECT e.user_id, e.class_id, c.grade, c.class_group, c.name AS class_name, c.major_id, u.role_primary
FROM enrollments e JOIN classes c ON c.id = e.class_id
JOIN users u ON u.id = e.user_id
WHERE e.academic_year_id = $1 AND e.is_current = TRUE AND u.deleted_at IS NULL;

-- name: SetUserStatus :exec
UPDATE users SET status = $2 WHERE id = $1;

-- name: CountClassStudents :one
SELECT COUNT(*) FROM enrollments WHERE class_id = $1 AND is_current = TRUE;

-- name: ListAdminUserIDs :many
SELECT id FROM users WHERE role_primary = 'admin' AND is_active = TRUE AND deleted_at IS NULL;

-- name: GetActiveYearFull :one
SELECT id, start_year, promotion_month, promotion_day FROM academic_years WHERE is_active = TRUE LIMIT 1;

-- name: FindYearByStartYear :one
SELECT id, start_year FROM academic_years WHERE start_year = $1 LIMIT 1;

-- name: GetPromotionJobByYears :one
SELECT id, status FROM promotion_jobs WHERE from_year_id = $1 AND to_year_id = $2 LIMIT 1;

-- name: ListPendingPortfoliosScoped :many
SELECT p.id, p.owner_id, p.title, p.slug, p.submitted_at, u.username, u.name AS owner_name
FROM portfolios p JOIN users u ON u.id = p.owner_id
WHERE p.status = 'pending_review' AND p.deleted_at IS NULL
  AND ($1 = '' OR u.major_id::text = $1)
ORDER BY p.submitted_at ASC LIMIT $2 OFFSET $3;

-- name: GraduateEnrollment :exec
UPDATE enrollments SET status = 'graduated', is_current = FALSE
WHERE user_id = $1 AND academic_year_id = $2;

-- name: MarkPromotionFailed :exec
UPDATE promotion_jobs SET status = 'failed' WHERE id = $1;

-- name: CountClassStudentsInYear :one
SELECT COUNT(*) FROM enrollments e JOIN classes c ON c.id = e.class_id
WHERE c.academic_year_id = $1 AND c.major_id = $2 AND c.grade = $3 AND c.class_group = $4 AND e.is_current = TRUE;

-- name: GetYearByID :one
SELECT id, start_year FROM academic_years WHERE id = $1 LIMIT 1;
