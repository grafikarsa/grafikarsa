package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"grafikarsa/internal/domain"
)

// TagRepository handles tag data access
type TagRepository struct {
	db *pgxpool.Pool
}

// NewTagRepository creates a new TagRepository
func NewTagRepository(db *pgxpool.Pool) *TagRepository {
	return &TagRepository{db: db}
}

// GetAll retrieves all tags
func (r *TagRepository) GetAll(ctx context.Context, search string) ([]domain.Tag, error) {
	var query string
	var args []interface{}

	if search != "" {
		query = `
			SELECT id, name, created_at, updated_at
			FROM tags
			WHERE deleted_at IS NULL AND name ILIKE $1
			ORDER BY name ASC
		`
		args = []interface{}{"%" + search + "%"}
	} else {
		query = `
			SELECT id, name, created_at, updated_at
			FROM tags
			WHERE deleted_at IS NULL
			ORDER BY name ASC
		`
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}

	return tags, rows.Err()
}

// GetAllWithCount retrieves all tags with portfolio count (for admin)
func (r *TagRepository) GetAllWithCount(ctx context.Context) ([]domain.Tag, error) {
	query := `
		SELECT t.id, t.name, t.created_at, t.updated_at,
		       (SELECT COUNT(*) FROM portfolio_tags pt WHERE pt.tag_id = t.id) as portfolio_count
		FROM tags t
		WHERE t.deleted_at IS NULL
		ORDER BY t.name ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt, &t.PortfolioCount); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}

	return tags, rows.Err()
}

// GetByID retrieves a tag by ID
func (r *TagRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tag, error) {
	query := `
		SELECT id, name, created_at, updated_at
		FROM tags
		WHERE id = $1 AND deleted_at IS NULL
	`

	var t domain.Tag
	err := r.db.QueryRow(ctx, query, id).Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &t, nil
}

// GetByIDs retrieves multiple tags by IDs
func (r *TagRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Tag, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query := `
		SELECT id, name, created_at, updated_at
		FROM tags
		WHERE id = ANY($1) AND deleted_at IS NULL
		ORDER BY name ASC
	`

	rows, err := r.db.Query(ctx, query, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}

	return tags, rows.Err()
}

// Create creates a new tag
func (r *TagRepository) Create(ctx context.Context, tag *domain.Tag) error {
	query := `
		INSERT INTO tags (id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.Exec(ctx, query, tag.ID, tag.Name, tag.CreatedAt, tag.UpdatedAt)
	return err
}

// Update updates a tag
func (r *TagRepository) Update(ctx context.Context, tag *domain.Tag) error {
	query := `UPDATE tags SET name = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.Exec(ctx, query, tag.ID, tag.Name)
	return err
}

// SoftDelete soft deletes a tag
func (r *TagRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE tags SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// NameExists checks if a tag name exists
func (r *TagRepository) NameExists(ctx context.Context, name string, excludeID *uuid.UUID) (bool, error) {
	var query string
	var args []interface{}

	if excludeID != nil {
		query = `SELECT EXISTS(SELECT 1 FROM tags WHERE LOWER(name) = LOWER($1) AND id != $2 AND deleted_at IS NULL)`
		args = []interface{}{name, *excludeID}
	} else {
		query = `SELECT EXISTS(SELECT 1 FROM tags WHERE LOWER(name) = LOWER($1) AND deleted_at IS NULL)`
		args = []interface{}{name}
	}

	var exists bool
	err := r.db.QueryRow(ctx, query, args...).Scan(&exists)
	return exists, err
}

// FollowRepository handles follow data access
type FollowRepository struct {
	db *pgxpool.Pool
}

// NewFollowRepository creates a new FollowRepository
func NewFollowRepository(db *pgxpool.Pool) *FollowRepository {
	return &FollowRepository{db: db}
}

// Create creates a follow relationship
func (r *FollowRepository) Create(ctx context.Context, followerID, followingID uuid.UUID) error {
	query := `
		INSERT INTO follows (id, follower_id, following_id, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.Exec(ctx, query, uuid.New(), followerID, followingID, time.Now())
	return err
}

// Delete removes a follow relationship
func (r *FollowRepository) Delete(ctx context.Context, followerID, followingID uuid.UUID) error {
	query := `DELETE FROM follows WHERE follower_id = $1 AND following_id = $2`
	_, err := r.db.Exec(ctx, query, followerID, followingID)
	return err
}

// Exists checks if a follow relationship exists
func (r *FollowRepository) Exists(ctx context.Context, followerID, followingID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM follows WHERE follower_id = $1 AND following_id = $2)`
	var exists bool
	err := r.db.QueryRow(ctx, query, followerID, followingID).Scan(&exists)
	return exists, err
}

// GetFollowerCount gets the follower count for a user
func (r *FollowRepository) GetFollowerCount(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM follows WHERE following_id = $1`
	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

// GetFollowingCount gets the following count for a user
func (r *FollowRepository) GetFollowingCount(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM follows WHERE follower_id = $1`
	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}
