-- name: InsertImportJob :one
INSERT INTO import_jobs (kind, status, total_rows, valid_rows, errors, summary, created_by)
VALUES (
  @kind::text,
  @status::text,
  sqlc.narg('total_rows')::integer,
  sqlc.narg('valid_rows')::integer,
  @errors::jsonb,
  @summary::jsonb,
  @created_by::uuid
)
RETURNING id, kind, status, total_rows, valid_rows, errors, summary, created_by, created_at, committed_at;

-- name: GetImportJob :one
SELECT id, kind, status, total_rows, valid_rows, errors, summary, created_by, created_at, committed_at
FROM import_jobs
WHERE id = @id::uuid;

-- name: ListImportJobs :many
SELECT id, kind, status, total_rows, valid_rows, errors, summary, created_by, created_at, committed_at
FROM import_jobs
WHERE (sqlc.narg('created_by')::uuid IS NULL OR created_by = sqlc.narg('created_by')::uuid)
ORDER BY created_at DESC, id DESC
LIMIT @limit_count;

-- name: MarkImportJobCommitting :execrows
UPDATE import_jobs
SET status = 'committing'
WHERE id = @id::uuid
  AND status = 'ready';

-- name: CompleteImportJob :execrows
UPDATE import_jobs
SET status = 'committed',
    committed_at = now(),
    summary = @summary::jsonb
WHERE id = @id::uuid
  AND status = 'committing';

-- name: FailImportJob :execrows
UPDATE import_jobs
SET status = 'failed',
    summary = @summary::jsonb
WHERE id = @id::uuid
  AND status = 'committing';

-- name: ListExistingUserIdentifiers :many
SELECT nisn, staff_number
FROM users
WHERE nisn = ANY(@nisns::text[])
   OR staff_number = ANY(@staff_numbers::text[]);

-- name: GetImportExistingUser :one
SELECT id, account_type, display_name
FROM users
WHERE (sqlc.narg('nisn')::text IS NOT NULL AND nisn = sqlc.narg('nisn')::text)
   OR (sqlc.narg('staff_number')::text IS NOT NULL AND staff_number = sqlc.narg('staff_number')::text)
LIMIT 1;

-- name: ImportUserHasEnrollment :one
SELECT EXISTS (
  SELECT 1
  FROM enrollments
  WHERE user_id = @user_id::uuid
    AND class_id = @class_id::uuid
);
