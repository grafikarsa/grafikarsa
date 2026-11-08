-- Queries for public profiles, own profile settings, experiences, follows, and
-- skill endorsements.

-- name: GetProfileUserByUsername :one
SELECT
  u.id,
  u.username::text AS username,
  u.display_name,
  u.account_type,
  u.status,
  (u.activated_at IS NOT NULL)::boolean AS activated,
  u.username_changed_at,
  p.headline,
  p.bio,
  p.city,
  p.avatar_media_id,
  p.banner_media_id,
  p.teaching_subjects
FROM users u
LEFT JOIN profiles p ON p.user_id = u.id
WHERE u.username = @username::citext;

-- name: GetProfileUserByID :one
SELECT
  u.id,
  u.username::text AS username,
  u.display_name,
  u.account_type,
  u.status,
  (u.activated_at IS NOT NULL)::boolean AS activated,
  u.username_changed_at,
  p.headline,
  p.bio,
  p.city,
  p.avatar_media_id,
  p.banner_media_id,
  p.teaching_subjects
FROM users u
LEFT JOIN profiles p ON p.user_id = u.id
WHERE u.id = @user_id::uuid;

-- name: GetUsernameRedirect :one
SELECT u.username::text AS username
FROM username_history h
JOIN users u ON u.id = h.user_id
WHERE h.old_username = @old_username::citext
  AND h.released_at IS NULL
  AND h.redirect_until > now()
  AND u.username IS NOT NULL
ORDER BY h.changed_at DESC
LIMIT 1;

-- name: ListProfileLinks :many
SELECT
  platform,
  url
FROM profile_links
WHERE user_id = @user_id::uuid
ORDER BY position, id;

-- name: ListProfileSkills :many
SELECT
  s.slug::text AS slug,
  s.name,
  s.category,
  (
    SELECT count(*)::bigint
    FROM skill_endorsements se
    WHERE se.user_id = us.user_id
      AND se.skill_id = us.skill_id
  ) AS endorsement_count,
  EXISTS (
    SELECT 1
    FROM skill_endorsements se2
    WHERE se2.user_id = us.user_id
      AND se2.skill_id = us.skill_id
      AND se2.endorser_id = sqlc.narg('viewer_id')::uuid
  ) AS endorsed_by_viewer
FROM user_skills us
JOIN skills s ON s.id = us.skill_id
WHERE us.user_id = @user_id::uuid
ORDER BY us.position, us.created_at;

-- name: ListProfileBadges :many
SELECT
  b.slug::text AS slug,
  b.name,
  b.description,
  b.icon_key,
  ub.note,
  ub.awarded_at
FROM user_badges ub
JOIN badges b ON b.id = ub.badge_id
WHERE ub.user_id = @user_id::uuid
  AND ub.revoked_at IS NULL
  AND (ub.expires_at IS NULL OR ub.expires_at > now())
  AND b.is_active
ORDER BY ub.awarded_at DESC, ub.id;

-- name: GetProfileLatestClass :one
SELECT
  c.name AS class_name,
  m.code AS major_code
FROM enrollments e
JOIN classes c ON c.id = e.class_id
JOIN majors m ON m.id = c.major_id
JOIN academic_years ay ON ay.id = e.academic_year_id
WHERE e.user_id = @user_id::uuid
ORDER BY ay.start_year DESC, e.created_at DESC
LIMIT 1;

-- name: ListProfileTeacherMajors :many
SELECT m.code, m.name
FROM teacher_majors tm
JOIN majors m ON m.id = tm.major_id
WHERE tm.user_id = @user_id::uuid
ORDER BY m.code;

-- name: GetProfileRelations :one
SELECT
  (SELECT count(*)::bigint FROM follows WHERE followee_id = @user_id::uuid) AS followers,
  (SELECT count(*)::bigint FROM follows WHERE follower_id = @user_id::uuid) AS following,
  EXISTS (
    SELECT 1
    FROM follows
    WHERE followee_id = @user_id::uuid
      AND follower_id = sqlc.narg('viewer_id')::uuid
  ) AS viewer_following;

-- name: ListProfileFollowers :many
SELECT
  u.id,
  u.username::text AS username,
  u.display_name,
  u.account_type,
  p.avatar_media_id,
  f.created_at
FROM follows f
JOIN users u ON u.id = f.follower_id
LEFT JOIN profiles p ON p.user_id = u.id
WHERE f.followee_id = @user_id::uuid
  AND u.username IS NOT NULL
  AND u.activated_at IS NOT NULL
  AND u.status IN ('active', 'alumni')
  AND (
    sqlc.narg('cursor_time')::timestamptz IS NULL
    OR (f.created_at, f.follower_id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::uuid)
  )
ORDER BY f.created_at DESC, f.follower_id DESC
LIMIT @limit_count::int;

-- name: ListProfileFollowing :many
SELECT
  u.id,
  u.username::text AS username,
  u.display_name,
  u.account_type,
  p.avatar_media_id,
  f.created_at
FROM follows f
JOIN users u ON u.id = f.followee_id
LEFT JOIN profiles p ON p.user_id = u.id
WHERE f.follower_id = @user_id::uuid
  AND u.username IS NOT NULL
  AND u.activated_at IS NOT NULL
  AND u.status IN ('active', 'alumni')
  AND (
    sqlc.narg('cursor_time')::timestamptz IS NULL
    OR (f.created_at, f.followee_id) < (sqlc.narg('cursor_time')::timestamptz, sqlc.narg('cursor_id')::uuid)
  )
ORDER BY f.created_at DESC, f.followee_id DESC
LIMIT @limit_count::int;

-- name: InsertFollow :exec
INSERT INTO follows (follower_id, followee_id)
VALUES (@follower_id::uuid, @followee_id::uuid)
ON CONFLICT DO NOTHING;

-- name: DeleteFollow :exec
DELETE FROM follows
WHERE follower_id = @follower_id::uuid
  AND followee_id = @followee_id::uuid;

-- name: EnsureProfile :exec
INSERT INTO profiles (user_id)
VALUES (@user_id::uuid)
ON CONFLICT (user_id) DO NOTHING;

-- name: UpdateUserDisplayName :exec
UPDATE users
SET display_name = @display_name::text
WHERE id = @user_id::uuid;

-- name: UpdateProfileFields :exec
UPDATE profiles
SET headline = CASE WHEN @set_headline::boolean THEN sqlc.narg('headline')::text ELSE headline END,
    bio = CASE WHEN @set_bio::boolean THEN sqlc.narg('bio')::text ELSE bio END,
    city = CASE WHEN @set_city::boolean THEN sqlc.narg('city')::text ELSE city END,
    teaching_subjects = CASE
      WHEN @set_teaching_subjects::boolean THEN @teaching_subjects::text[]
      ELSE teaching_subjects
    END
WHERE user_id = @user_id::uuid;

-- name: GetOwnedMedia :one
SELECT id
FROM media_assets
WHERE id = @media_id::uuid
  AND owner_id = @user_id::uuid
  AND kind = 'image'
  AND purpose = @purpose::text
  AND deleted_at IS NULL;

-- name: SetProfileAvatar :exec
INSERT INTO profiles (user_id, avatar_media_id)
VALUES (@user_id::uuid, sqlc.narg('media_id')::uuid)
ON CONFLICT (user_id) DO UPDATE
SET avatar_media_id = EXCLUDED.avatar_media_id,
    updated_at = now();

-- name: SetProfileBanner :exec
INSERT INTO profiles (user_id, banner_media_id)
VALUES (@user_id::uuid, sqlc.narg('media_id')::uuid)
ON CONFLICT (user_id) DO UPDATE
SET banner_media_id = EXCLUDED.banner_media_id,
    updated_at = now();

-- name: DeleteProfileLinks :exec
DELETE FROM profile_links
WHERE user_id = @user_id::uuid;

-- name: InsertProfileLink :exec
INSERT INTO profile_links (user_id, platform, url, position)
VALUES (@user_id::uuid, @platform::text, @url::text, @position::smallint);

-- name: DeleteUserSkills :exec
DELETE FROM user_skills
WHERE user_id = @user_id::uuid;

-- name: InsertUserSkill :exec
INSERT INTO user_skills (user_id, skill_id, position)
SELECT @user_id::uuid, s.id, @position::smallint
FROM skills s
WHERE s.slug = @slug::citext
  AND s.is_active;

-- name: ListSkillsBySlugs :many
SELECT id, slug::text AS slug, name, category
FROM skills
WHERE slug::text = ANY(@slug_list::text[])
  AND is_active;

-- name: GetUserSkill :one
SELECT us.user_id
FROM user_skills us
JOIN skills s ON s.id = us.skill_id
WHERE us.user_id = @user_id::uuid
  AND s.slug = @slug::citext
  AND s.is_active;

-- name: InsertSkillEndorsement :exec
INSERT INTO skill_endorsements (endorser_id, user_id, skill_id)
SELECT @endorser_id::uuid, us.user_id, us.skill_id
FROM user_skills us
JOIN skills s ON s.id = us.skill_id
WHERE us.user_id = @user_id::uuid
  AND s.slug = @slug::citext
ON CONFLICT DO NOTHING;

-- name: DeleteSkillEndorsement :exec
DELETE FROM skill_endorsements se
USING skills s
WHERE se.endorser_id = @endorser_id::uuid
  AND se.user_id = @user_id::uuid
  AND se.skill_id = s.id
  AND s.slug = @slug::citext;

-- name: ListExperiencesByUser :many
SELECT
  id,
  kind,
  title,
  organization,
  location,
  employment_type,
  start_month::text AS start_month,
  (CASE WHEN end_month IS NULL THEN '' ELSE end_month::text END) AS end_month,
  is_current,
  description,
  credential_url,
  is_system
FROM experiences
WHERE user_id = @user_id::uuid
ORDER BY is_current DESC, start_month DESC, created_at DESC;

-- name: GetExperienceByOwner :one
SELECT
  id,
  kind,
  title,
  organization,
  location,
  employment_type,
  start_month::text AS start_month,
  (CASE WHEN end_month IS NULL THEN '' ELSE end_month::text END) AS end_month,
  is_current,
  description,
  credential_url,
  is_system
FROM experiences
WHERE id = @id::uuid
  AND user_id = @user_id::uuid;

-- name: ListExperienceSkills :many
SELECT es.experience_id, s.slug::text AS slug, s.name
FROM experience_skills es
JOIN skills s ON s.id = es.skill_id
WHERE es.experience_id = ANY(@experience_ids::uuid[])
ORDER BY s.name;

-- name: InsertExperience :one
INSERT INTO experiences (
  user_id,
  kind,
  title,
  organization,
  location,
  employment_type,
  start_month,
  end_month,
  is_current,
  description,
  credential_url
)
VALUES (
  @user_id::uuid,
  @kind::text,
  @title::text,
  @organization::text,
  sqlc.narg('location')::text,
  sqlc.narg('employment_type')::text,
  @start_month::date,
  sqlc.narg('end_month')::date,
  @is_current::boolean,
  sqlc.narg('description')::text,
  sqlc.narg('credential_url')::text
)
RETURNING id;

-- name: UpdateExperience :exec
UPDATE experiences
SET kind = @kind::text,
    title = @title::text,
    organization = @organization::text,
    location = sqlc.narg('location')::text,
    employment_type = sqlc.narg('employment_type')::text,
    start_month = @start_month::date,
    end_month = sqlc.narg('end_month')::date,
    is_current = @is_current::boolean,
    description = sqlc.narg('description')::text,
    credential_url = sqlc.narg('credential_url')::text
WHERE id = @id::uuid
  AND user_id = @user_id::uuid;

-- name: DeleteExperience :exec
DELETE FROM experiences
WHERE id = @id::uuid
  AND user_id = @user_id::uuid;

-- name: DeleteExperienceSkills :exec
DELETE FROM experience_skills
WHERE experience_id = @experience_id::uuid;

-- name: InsertExperienceSkill :exec
INSERT INTO experience_skills (experience_id, skill_id)
SELECT @experience_id::uuid, s.id
FROM skills s
WHERE s.slug = @slug::citext
  AND s.is_active;

-- name: GetUsernameState :one
SELECT username::text AS username, username_changed_at
FROM users
WHERE id = @user_id::uuid;

-- name: SearchSkills :many
SELECT slug::text AS slug, name, category
FROM skills
WHERE is_active
  AND (
    sqlc.narg('query')::text IS NULL
    OR name ILIKE '%' || sqlc.narg('query')::text || '%'
  )
ORDER BY
  CASE WHEN sqlc.narg('query')::text IS NULL THEN 0 ELSE -similarity(name, sqlc.narg('query')::text) END,
  name
LIMIT 20;
