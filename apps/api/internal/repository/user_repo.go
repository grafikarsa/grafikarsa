package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"grafikarsa/internal/domain"
	"grafikarsa/internal/utils"
)

// UserRepository handles user data access
type UserRepository struct {
	db *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository
func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.password_hash, u.name, u.bio, 
		       u.avatar_url, u.banner_url, u.role, u.status, u.nisn, u.nis,
		       u.current_class_id, u.entry_year, u.graduation_year, u.social_links,
		       u.created_at, u.updated_at,
		       c.id, c.name,
		       m.id, m.name
		FROM users u
		LEFT JOIN classes c ON u.current_class_id = c.id
		LEFT JOIN majors m ON c.major_id = m.id
		WHERE u.id = $1 AND u.deleted_at IS NULL
	`

	return r.scanUser(r.db.QueryRow(ctx, query, id))
}

// GetByUsername retrieves a user by username
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.password_hash, u.name, u.bio, 
		       u.avatar_url, u.banner_url, u.role, u.status, u.nisn, u.nis,
		       u.current_class_id, u.entry_year, u.graduation_year, u.social_links,
		       u.created_at, u.updated_at,
		       c.id, c.name,
		       m.id, m.name
		FROM users u
		LEFT JOIN classes c ON u.current_class_id = c.id
		LEFT JOIN majors m ON c.major_id = m.id
		WHERE LOWER(u.username) = LOWER($1) AND u.deleted_at IS NULL
	`

	return r.scanUser(r.db.QueryRow(ctx, query, username))
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.password_hash, u.name, u.bio, 
		       u.avatar_url, u.banner_url, u.role, u.status, u.nisn, u.nis,
		       u.current_class_id, u.entry_year, u.graduation_year, u.social_links,
		       u.created_at, u.updated_at,
		       c.id, c.name,
		       m.id, m.name
		FROM users u
		LEFT JOIN classes c ON u.current_class_id = c.id
		LEFT JOIN majors m ON c.major_id = m.id
		WHERE LOWER(u.email) = LOWER($1) AND u.deleted_at IS NULL
	`

	return r.scanUser(r.db.QueryRow(ctx, query, email))
}

func (r *UserRepository) scanUser(row pgx.Row) (*domain.User, error) {
	var user domain.User
	var classID, className, majorID, majorName *string
	var socialLinksJSON []byte

	err := row.Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.Name, &user.Bio,
		&user.AvatarURL, &user.BannerURL, &user.Role, &user.Status, &user.NISN, &user.NIS,
		&user.CurrentClassID, &user.EntryYear, &user.GraduationYear, &socialLinksJSON,
		&user.CreatedAt, &user.UpdatedAt,
		&classID, &className,
		&majorID, &majorName,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Parse social links
	if len(socialLinksJSON) > 0 {
		if err := json.Unmarshal(socialLinksJSON, &user.SocialLinks); err != nil {
			user.SocialLinks = make(domain.SocialLinks)
		}
	} else {
		user.SocialLinks = make(domain.SocialLinks)
	}

	// Set class info
	if classID != nil && className != nil {
		cid, _ := uuid.Parse(*classID)
		user.Class = &domain.Class{
			ID:   cid,
			Name: *className,
		}
	}

	// Set major info
	if majorID != nil && majorName != nil {
		mid, _ := uuid.Parse(*majorID)
		user.Major = &domain.Major{
			ID:   mid,
			Name: *majorName,
		}
	}

	return &user, nil
}

// GetUserCounts retrieves follower, following, and portfolio counts
func (r *UserRepository) GetUserCounts(ctx context.Context, userID uuid.UUID) (followers, following, portfolios int, err error) {
	query := `
		SELECT 
			(SELECT COUNT(*) FROM follows WHERE following_id = $1),
			(SELECT COUNT(*) FROM follows WHERE follower_id = $1),
			(SELECT COUNT(*) FROM portfolios p 
			 INNER JOIN portfolio_versions pv ON p.id = pv.portfolio_id 
			 WHERE p.user_id = $1 AND pv.status = 'published' AND p.deleted_at IS NULL)
	`

	err = r.db.QueryRow(ctx, query, userID).Scan(&followers, &following, &portfolios)
	return
}

// IsFollowing checks if a user is following another user
func (r *UserRepository) IsFollowing(ctx context.Context, followerID, followingID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND following_id = $2)`

	var exists bool
	err := r.db.QueryRow(ctx, query, followerID, followingID).Scan(&exists)
	return exists, err
}

// UserFilter contains filtering options for user listing
type UserFilter struct {
	Search  string
	MajorID *uuid.UUID
	ClassID *uuid.UUID
	Role    string
	Status  string
	Page    int
	Limit   int
}

// List retrieves users with filtering and pagination
func (r *UserRepository) List(ctx context.Context, filter UserFilter) ([]domain.UserListItem, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, "u.deleted_at IS NULL")

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(u.name ILIKE $%d OR u.username ILIKE $%d OR u.bio ILIKE $%d)", argNum, argNum, argNum))
		args = append(args, "%"+filter.Search+"%")
		argNum++
	}

	if filter.MajorID != nil {
		conditions = append(conditions, fmt.Sprintf("c.major_id = $%d", argNum))
		args = append(args, *filter.MajorID)
		argNum++
	}

	if filter.ClassID != nil {
		conditions = append(conditions, fmt.Sprintf("u.current_class_id = $%d", argNum))
		args = append(args, *filter.ClassID)
		argNum++
	}

	if filter.Role != "" {
		conditions = append(conditions, fmt.Sprintf("u.role = $%d", argNum))
		args = append(args, filter.Role)
		argNum++
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("u.status = $%d", argNum))
		args = append(args, filter.Status)
		argNum++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM users u
		LEFT JOIN classes c ON u.current_class_id = c.id
		WHERE %s
	`, whereClause)

	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Data query
	pagination := utils.Pagination{Page: filter.Page, Limit: filter.Limit}
	pagination.Validate(50)

	query := fmt.Sprintf(`
		SELECT u.id, u.username, u.name, u.avatar_url, u.role,
		       c.id, c.name,
		       m.id, m.name
		FROM users u
		LEFT JOIN classes c ON u.current_class_id = c.id
		LEFT JOIN majors m ON c.major_id = m.id
		WHERE %s
		ORDER BY u.name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)

	args = append(args, pagination.Limit, pagination.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []domain.UserListItem
	for rows.Next() {
		var user domain.UserListItem
		var classID, className, majorID, majorName *string

		if err := rows.Scan(
			&user.ID, &user.Username, &user.Name, &user.AvatarURL, &user.Role,
			&classID, &className,
			&majorID, &majorName,
		); err != nil {
			return nil, 0, err
		}

		if classID != nil && className != nil {
			cid, _ := uuid.Parse(*classID)
			user.Class = &domain.ClassInfo{ID: cid, Name: *className}
			user.ClassName = *className
		}

		if majorID != nil && majorName != nil {
			mid, _ := uuid.Parse(*majorID)
			user.Major = &domain.MajorInfo{ID: mid, Name: *majorName}
			user.MajorName = *majorName
		}

		users = append(users, user)
	}

	return users, total, rows.Err()
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	socialLinksJSON, err := json.Marshal(user.SocialLinks)
	if err != nil {
		socialLinksJSON = []byte("{}")
	}

	query := `
		INSERT INTO users (id, username, email, password_hash, name, bio, avatar_url, banner_url,
		                   role, status, nisn, nis, current_class_id, entry_year, graduation_year,
		                   social_links, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	_, err = r.db.Exec(ctx, query,
		user.ID, user.Username, user.Email, user.PasswordHash, user.Name, user.Bio,
		user.AvatarURL, user.BannerURL, user.Role, user.Status, user.NISN, user.NIS,
		user.CurrentClassID, user.EntryYear, user.GraduationYear, socialLinksJSON,
		user.CreatedAt, user.UpdatedAt,
	)

	return err
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	socialLinksJSON, err := json.Marshal(user.SocialLinks)
	if err != nil {
		socialLinksJSON = []byte("{}")
	}

	query := `
		UPDATE users
		SET username = $2, email = $3, password_hash = $4, name = $5, bio = $6,
		    avatar_url = $7, banner_url = $8, role = $9, status = $10, nisn = $11, nis = $12,
		    current_class_id = $13, entry_year = $14, graduation_year = $15, social_links = $16,
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err = r.db.Exec(ctx, query,
		user.ID, user.Username, user.Email, user.PasswordHash, user.Name, user.Bio,
		user.AvatarURL, user.BannerURL, user.Role, user.Status, user.NISN, user.NIS,
		user.CurrentClassID, user.EntryYear, user.GraduationYear, socialLinksJSON,
	)

	return err
}

// UpdateProfile updates only profile fields (name, username, email, bio)
func (r *UserRepository) UpdateProfile(ctx context.Context, userID uuid.UUID, name, username, email, bio string) error {
	query := `
		UPDATE users
		SET name = $2, username = $3, email = $4, bio = $5, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`

	_, err := r.db.Exec(ctx, query, userID, name, username, email, bio)
	return err
}

// UpdatePassword updates user password
func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	query := `UPDATE users SET password_hash = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID, passwordHash)
	return err
}

// UpdateSocialLinks updates user social links
func (r *UserRepository) UpdateSocialLinks(ctx context.Context, userID uuid.UUID, links domain.SocialLinks) error {
	linksJSON, err := json.Marshal(links)
	if err != nil {
		return err
	}

	query := `UPDATE users SET social_links = $2, updated_at = NOW() WHERE id = $1`
	_, err = r.db.Exec(ctx, query, userID, linksJSON)
	return err
}

// UpdateAvatarURL updates user avatar
func (r *UserRepository) UpdateAvatarURL(ctx context.Context, userID uuid.UUID, url string) error {
	query := `UPDATE users SET avatar_url = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID, url)
	return err
}

// UpdateBannerURL updates user banner
func (r *UserRepository) UpdateBannerURL(ctx context.Context, userID uuid.UUID, url string) error {
	query := `UPDATE users SET banner_url = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID, url)
	return err
}

// SoftDelete soft deletes a user
func (r *UserRepository) SoftDelete(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

// UsernameExists checks if a username exists (excluding a specific user)
func (r *UserRepository) UsernameExists(ctx context.Context, username string, excludeUserID *uuid.UUID) (bool, error) {
	var query string
	var args []interface{}

	if excludeUserID != nil {
		query = `SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(username) = LOWER($1) AND id != $2 AND deleted_at IS NULL)`
		args = []interface{}{username, *excludeUserID}
	} else {
		query = `SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(username) = LOWER($1) AND deleted_at IS NULL)`
		args = []interface{}{username}
	}

	var exists bool
	err := r.db.QueryRow(ctx, query, args...).Scan(&exists)
	return exists, err
}

// EmailExists checks if an email exists (excluding a specific user)
func (r *UserRepository) EmailExists(ctx context.Context, email string, excludeUserID *uuid.UUID) (bool, error) {
	var query string
	var args []interface{}

	if excludeUserID != nil {
		query = `SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email) = LOWER($1) AND id != $2 AND deleted_at IS NULL)`
		args = []interface{}{email, *excludeUserID}
	} else {
		query = `SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL)`
		args = []interface{}{email}
	}

	var exists bool
	err := r.db.QueryRow(ctx, query, args...).Scan(&exists)
	return exists, err
}

// GetClassHistory retrieves the class assignment history for a user
func (r *UserRepository) GetClassHistory(ctx context.Context, userID uuid.UUID) ([]domain.ClassHistory, error) {
	query := `
		SELECT c.name, ay.year_start, uch.assigned_at
		FROM user_class_history uch
		INNER JOIN classes c ON uch.class_id = c.id
		INNER JOIN academic_years ay ON uch.academic_year_id = ay.id
		WHERE uch.user_id = $1
		ORDER BY uch.assigned_at ASC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []domain.ClassHistory
	for rows.Next() {
		var h domain.ClassHistory
		if err := rows.Scan(&h.ClassName, &h.AcademicYear, &h.AssignedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}

	return history, rows.Err()
}

// GetFollowers retrieves followers of a user
func (r *UserRepository) GetFollowers(ctx context.Context, userID uuid.UUID, currentUserID *uuid.UUID, search string, page, limit int) ([]domain.FollowerItem, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("f.following_id = $%d", argNum))
	args = append(args, userID)
	argNum++

	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(u.name ILIKE $%d OR u.username ILIKE $%d)", argNum, argNum))
		args = append(args, "%"+search+"%")
		argNum++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM follows f
		INNER JOIN users u ON f.follower_id = u.id
		WHERE %s AND u.deleted_at IS NULL
	`, whereClause)

	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Check if current user follows each follower
	isFollowingCase := "false"
	if currentUserID != nil {
		isFollowingCase = fmt.Sprintf("EXISTS(SELECT 1 FROM follows WHERE follower_id = $%d AND following_id = u.id)", argNum)
		args = append(args, *currentUserID)
		argNum++
	}

	pagination := utils.Pagination{Page: page, Limit: limit}
	pagination.Validate(50)

	query := fmt.Sprintf(`
		SELECT u.id, u.username, u.name, u.avatar_url, u.role, c.name, f.created_at,
		       %s as is_following
		FROM follows f
		INNER JOIN users u ON f.follower_id = u.id
		LEFT JOIN classes c ON u.current_class_id = c.id
		WHERE %s AND u.deleted_at IS NULL
		ORDER BY f.created_at DESC
		LIMIT $%d OFFSET $%d
	`, isFollowingCase, whereClause, argNum, argNum+1)

	args = append(args, pagination.Limit, pagination.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var followers []domain.FollowerItem
	for rows.Next() {
		var f domain.FollowerItem
		var className *string

		if err := rows.Scan(&f.ID, &f.Username, &f.Name, &f.AvatarURL, &f.Role, &className, &f.FollowedAt, &f.IsFollowing); err != nil {
			return nil, 0, err
		}

		if className != nil {
			f.ClassName = *className
		}

		followers = append(followers, f)
	}

	return followers, total, rows.Err()
}

// GetFollowing retrieves users that a user is following
func (r *UserRepository) GetFollowing(ctx context.Context, userID uuid.UUID, currentUserID *uuid.UUID, search string, page, limit int) ([]domain.FollowerItem, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("f.follower_id = $%d", argNum))
	args = append(args, userID)
	argNum++

	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(u.name ILIKE $%d OR u.username ILIKE $%d)", argNum, argNum))
		args = append(args, "%"+search+"%")
		argNum++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM follows f
		INNER JOIN users u ON f.following_id = u.id
		WHERE %s AND u.deleted_at IS NULL
	`, whereClause)

	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Check if current user follows each user
	isFollowingCase := "false"
	if currentUserID != nil {
		isFollowingCase = fmt.Sprintf("EXISTS(SELECT 1 FROM follows WHERE follower_id = $%d AND following_id = u.id)", argNum)
		args = append(args, *currentUserID)
		argNum++
	}

	pagination := utils.Pagination{Page: page, Limit: limit}
	pagination.Validate(50)

	query := fmt.Sprintf(`
		SELECT u.id, u.username, u.name, u.avatar_url, u.role, c.name, f.created_at,
		       %s as is_following
		FROM follows f
		INNER JOIN users u ON f.following_id = u.id
		LEFT JOIN classes c ON u.current_class_id = c.id
		WHERE %s AND u.deleted_at IS NULL
		ORDER BY f.created_at DESC
		LIMIT $%d OFFSET $%d
	`, isFollowingCase, whereClause, argNum, argNum+1)

	args = append(args, pagination.Limit, pagination.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var following []domain.FollowerItem
	for rows.Next() {
		var f domain.FollowerItem
		var className *string

		if err := rows.Scan(&f.ID, &f.Username, &f.Name, &f.AvatarURL, &f.Role, &className, &f.FollowedAt, &f.IsFollowing); err != nil {
			return nil, 0, err
		}

		if className != nil {
			f.ClassName = *className
		}

		following = append(following, f)
	}

	return following, total, rows.Err()
}
