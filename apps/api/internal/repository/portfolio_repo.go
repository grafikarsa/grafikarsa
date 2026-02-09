package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"grafikarsa/internal/domain"
	"grafikarsa/internal/utils"
)

// PortfolioRepository handles portfolio data access
type PortfolioRepository struct {
	db *pgxpool.Pool
}

// NewPortfolioRepository creates a new PortfolioRepository
func NewPortfolioRepository(db *pgxpool.Pool) *PortfolioRepository {
	return &PortfolioRepository{db: db}
}

// PortfolioFilter contains filtering options for portfolio listing
type PortfolioFilter struct {
	Search  string
	TagIDs  []uuid.UUID
	MajorID *uuid.UUID
	ClassID *uuid.UUID
	UserID  *uuid.UUID
	Status  string // For owner view
	Sort    string // -published_at, -like_count, title
	Page    int
	Limit   int
}

// Create creates a new portfolio with an initial version
func (r *PortfolioRepository) Create(ctx context.Context, portfolio *domain.Portfolio, version *domain.PortfolioVersion) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Create portfolio
	portfolioQuery := `
		INSERT INTO portfolios (id, user_id, slug, like_count, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(ctx, portfolioQuery,
		portfolio.ID, portfolio.UserID, portfolio.Slug, 0, portfolio.CreatedAt, portfolio.UpdatedAt,
	)
	if err != nil {
		return err
	}

	// Create initial version
	versionQuery := `
		INSERT INTO portfolio_versions (id, portfolio_id, version_number, title, thumbnail_url, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err = tx.Exec(ctx, versionQuery,
		version.ID, portfolio.ID, 1, version.Title, version.ThumbnailURL, domain.StatusDraft, version.CreatedAt, version.UpdatedAt,
	)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// GetByID retrieves a portfolio by ID with its current active version
func (r *PortfolioRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Portfolio, *domain.PortfolioVersion, error) {
	portfolioQuery := `
		SELECT id, user_id, slug, like_count, created_at, updated_at
		FROM portfolios
		WHERE id = $1 AND deleted_at IS NULL
	`

	var portfolio domain.Portfolio
	err := r.db.QueryRow(ctx, portfolioQuery, id).Scan(
		&portfolio.ID, &portfolio.UserID, &portfolio.Slug, &portfolio.LikeCount,
		&portfolio.CreatedAt, &portfolio.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	// Get latest version
	version, err := r.GetLatestVersion(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	return &portfolio, version, nil
}

// GetByUserAndSlug retrieves a portfolio by username and slug
func (r *PortfolioRepository) GetByUserAndSlug(ctx context.Context, username, slug string) (*domain.Portfolio, *domain.PortfolioVersion, error) {
	query := `
		SELECT p.id, p.user_id, p.slug, p.like_count, p.created_at, p.updated_at
		FROM portfolios p
		INNER JOIN users u ON p.user_id = u.id
		WHERE LOWER(u.username) = LOWER($1) AND p.slug = $2 
		  AND p.deleted_at IS NULL AND u.deleted_at IS NULL
	`

	var portfolio domain.Portfolio
	err := r.db.QueryRow(ctx, query, username, slug).Scan(
		&portfolio.ID, &portfolio.UserID, &portfolio.Slug, &portfolio.LikeCount,
		&portfolio.CreatedAt, &portfolio.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}

	// Get published version for public view
	version, err := r.GetPublishedVersion(ctx, portfolio.ID)
	if err != nil {
		return nil, nil, err
	}

	return &portfolio, version, nil
}

// GetLatestVersion gets the latest version of a portfolio
func (r *PortfolioRepository) GetLatestVersion(ctx context.Context, portfolioID uuid.UUID) (*domain.PortfolioVersion, error) {
	query := `
		SELECT id, portfolio_id, version_number, title, thumbnail_url, status, 
		       admin_review_note, reviewed_by, reviewed_at, published_at, created_at, updated_at
		FROM portfolio_versions
		WHERE portfolio_id = $1
		ORDER BY version_number DESC
		LIMIT 1
	`

	var v domain.PortfolioVersion
	err := r.db.QueryRow(ctx, query, portfolioID).Scan(
		&v.ID, &v.PortfolioID, &v.VersionNumber, &v.Title, &v.ThumbnailURL, &v.Status,
		&v.AdminReviewNote, &v.ReviewedBy, &v.ReviewedAt, &v.PublishedAt, &v.CreatedAt, &v.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &v, nil
}

// GetPublishedVersion gets the published version of a portfolio
func (r *PortfolioRepository) GetPublishedVersion(ctx context.Context, portfolioID uuid.UUID) (*domain.PortfolioVersion, error) {
	query := `
		SELECT id, portfolio_id, version_number, title, thumbnail_url, status, 
		       admin_review_note, reviewed_by, reviewed_at, published_at, created_at, updated_at
		FROM portfolio_versions
		WHERE portfolio_id = $1 AND status = 'published'
		ORDER BY version_number DESC
		LIMIT 1
	`

	var v domain.PortfolioVersion
	err := r.db.QueryRow(ctx, query, portfolioID).Scan(
		&v.ID, &v.PortfolioID, &v.VersionNumber, &v.Title, &v.ThumbnailURL, &v.Status,
		&v.AdminReviewNote, &v.ReviewedBy, &v.ReviewedAt, &v.PublishedAt, &v.CreatedAt, &v.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &v, nil
}

// GetVersionByID gets a specific portfolio version
func (r *PortfolioRepository) GetVersionByID(ctx context.Context, versionID uuid.UUID) (*domain.PortfolioVersion, error) {
	query := `
		SELECT id, portfolio_id, version_number, title, thumbnail_url, status, 
		       admin_review_note, reviewed_by, reviewed_at, published_at, created_at, updated_at
		FROM portfolio_versions
		WHERE id = $1
	`

	var v domain.PortfolioVersion
	err := r.db.QueryRow(ctx, query, versionID).Scan(
		&v.ID, &v.PortfolioID, &v.VersionNumber, &v.Title, &v.ThumbnailURL, &v.Status,
		&v.AdminReviewNote, &v.ReviewedBy, &v.ReviewedAt, &v.PublishedAt, &v.CreatedAt, &v.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &v, nil
}

// UpdateVersion updates a portfolio version
func (r *PortfolioRepository) UpdateVersion(ctx context.Context, version *domain.PortfolioVersion) error {
	query := `
		UPDATE portfolio_versions
		SET title = $2, thumbnail_url = $3, status = $4, admin_review_note = $5,
		    reviewed_by = $6, reviewed_at = $7, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		version.ID, version.Title, version.ThumbnailURL, version.Status,
		version.AdminReviewNote, version.ReviewedBy, version.ReviewedAt,
	)
	return err
}

// UpdateVersionStatus updates just the status of a version
func (r *PortfolioRepository) UpdateVersionStatus(ctx context.Context, versionID uuid.UUID, status domain.PortfolioStatus) error {
	query := `UPDATE portfolio_versions SET status = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, versionID, status)
	return err
}

// SetVersionPublished marks a version as published
func (r *PortfolioRepository) SetVersionPublished(ctx context.Context, versionID, reviewedBy uuid.UUID) error {
	query := `
		UPDATE portfolio_versions 
		SET status = 'published', published_at = NOW(), reviewed_by = $2, reviewed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, versionID, reviewedBy)
	return err
}

// SetVersionRejected marks a version as rejected
func (r *PortfolioRepository) SetVersionRejected(ctx context.Context, versionID, reviewedBy uuid.UUID, note string) error {
	query := `
		UPDATE portfolio_versions 
		SET status = 'rejected', admin_review_note = $2, reviewed_by = $3, reviewed_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, versionID, note, reviewedBy)
	return err
}

// UpdateSlug updates the portfolio slug
func (r *PortfolioRepository) UpdateSlug(ctx context.Context, portfolioID uuid.UUID, slug string) error {
	query := `UPDATE portfolios SET slug = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, portfolioID, slug)
	return err
}

// UpdateThumbnailURL updates the portfolio thumbnail
func (r *PortfolioRepository) UpdateThumbnailURL(ctx context.Context, versionID uuid.UUID, url string) error {
	query := `UPDATE portfolio_versions SET thumbnail_url = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, versionID, url)
	return err
}

// SoftDelete soft deletes a portfolio
func (r *PortfolioRepository) SoftDelete(ctx context.Context, portfolioID uuid.UUID) error {
	query := `UPDATE portfolios SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.Exec(ctx, query, portfolioID)
	return err
}

// SlugExists checks if a slug exists for a user
func (r *PortfolioRepository) SlugExists(ctx context.Context, userID uuid.UUID, slug string, excludePortfolioID *uuid.UUID) (bool, error) {
	var query string
	var args []interface{}

	if excludePortfolioID != nil {
		query = `SELECT EXISTS(SELECT 1 FROM portfolios WHERE user_id = $1 AND slug = $2 AND id != $3 AND deleted_at IS NULL)`
		args = []interface{}{userID, slug, *excludePortfolioID}
	} else {
		query = `SELECT EXISTS(SELECT 1 FROM portfolios WHERE user_id = $1 AND slug = $2 AND deleted_at IS NULL)`
		args = []interface{}{userID, slug}
	}

	var exists bool
	err := r.db.QueryRow(ctx, query, args...).Scan(&exists)
	return exists, err
}

// CountUserPortfoliosToday counts portfolios created by user today (for rate limiting)
func (r *PortfolioRepository) CountUserPortfoliosToday(ctx context.Context, userID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM portfolios
		WHERE user_id = $1 AND created_at >= CURRENT_DATE AND deleted_at IS NULL
	`

	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

// ListPublished lists published portfolios with filtering
func (r *PortfolioRepository) ListPublished(ctx context.Context, filter PortfolioFilter, currentUserID *uuid.UUID) ([]domain.PortfolioListItem, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, "p.deleted_at IS NULL")
	conditions = append(conditions, "pv.status = 'published'")

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(pv.title ILIKE $%d OR u.name ILIKE $%d)", argNum, argNum))
		args = append(args, "%"+filter.Search+"%")
		argNum++
	}

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("p.user_id = $%d", argNum))
		args = append(args, *filter.UserID)
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

	// Tag filtering with ALL tags matching
	if len(filter.TagIDs) > 0 {
		tagPlaceholders := make([]string, len(filter.TagIDs))
		for i, tagID := range filter.TagIDs {
			tagPlaceholders[i] = fmt.Sprintf("$%d", argNum)
			args = append(args, tagID)
			argNum++
		}
		conditions = append(conditions, fmt.Sprintf(`
			EXISTS (
				SELECT 1 FROM portfolio_tags pt2 
				WHERE pt2.portfolio_id = p.id AND pt2.tag_id IN (%s)
				GROUP BY pt2.portfolio_id 
				HAVING COUNT(DISTINCT pt2.tag_id) = %d
			)
		`, strings.Join(tagPlaceholders, ","), len(filter.TagIDs)))
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT p.id)
		FROM portfolios p
		INNER JOIN portfolio_versions pv ON p.id = pv.portfolio_id
		INNER JOIN users u ON p.user_id = u.id
		LEFT JOIN classes c ON u.current_class_id = c.id
		WHERE %s
	`, whereClause)

	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Determine sort order
	orderBy := "pv.published_at DESC"
	switch filter.Sort {
	case "-like_count":
		orderBy = "p.like_count DESC"
	case "title":
		orderBy = "pv.title ASC"
	case "-published_at":
		orderBy = "pv.published_at DESC"
	}

	pagination := utils.Pagination{Page: filter.Page, Limit: filter.Limit}
	pagination.Validate(50)

	query := fmt.Sprintf(`
		SELECT DISTINCT ON (p.id) p.id, pv.title, p.slug, pv.thumbnail_url, pv.published_at, p.like_count,
		       u.id, u.username, u.name, u.avatar_url, u.role, c.name
		FROM portfolios p
		INNER JOIN portfolio_versions pv ON p.id = pv.portfolio_id
		INNER JOIN users u ON p.user_id = u.id
		LEFT JOIN classes c ON u.current_class_id = c.id
		WHERE %s
		ORDER BY p.id, pv.version_number DESC
	`, whereClause)

	// Wrap for proper ordering and pagination
	query = fmt.Sprintf(`
		SELECT * FROM (%s) sub
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, query, orderBy, argNum, argNum+1)

	args = append(args, pagination.Limit, pagination.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var portfolios []domain.PortfolioListItem
	for rows.Next() {
		var p domain.PortfolioListItem
		var user domain.UserListItem
		var className *string

		if err := rows.Scan(
			&p.ID, &p.Title, &p.Slug, &p.ThumbnailURL, &p.PublishedAt, &p.LikeCount,
			&user.ID, &user.Username, &user.Name, &user.AvatarURL, &user.Role, &className,
		); err != nil {
			return nil, 0, err
		}

		if className != nil {
			user.ClassName = *className
		}
		p.User = &user

		// Get tags for this portfolio
		tags, _ := r.GetPortfolioTags(ctx, p.ID)
		p.Tags = tags

		portfolios = append(portfolios, p)
	}

	return portfolios, total, rows.Err()
}

// ListByUser lists all portfolios for a user (including non-published)
func (r *PortfolioRepository) ListByUser(ctx context.Context, userID uuid.UUID, status string, page, limit int) ([]domain.PortfolioListItem, int, error) {
	var conditions []string
	var args []interface{}
	argNum := 1

	conditions = append(conditions, "p.deleted_at IS NULL")
	conditions = append(conditions, fmt.Sprintf("p.user_id = $%d", argNum))
	args = append(args, userID)
	argNum++

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("pv.status = $%d", argNum))
		args = append(args, status)
		argNum++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT p.id)
		FROM portfolios p
		INNER JOIN portfolio_versions pv ON p.id = pv.portfolio_id
		WHERE %s
	`, whereClause)

	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	pagination := utils.Pagination{Page: page, Limit: limit}
	pagination.Validate(50)

	query := fmt.Sprintf(`
		SELECT DISTINCT ON (p.id) p.id, pv.title, p.slug, pv.thumbnail_url, pv.status, 
		       pv.admin_review_note, p.created_at, p.updated_at, p.like_count
		FROM portfolios p
		INNER JOIN portfolio_versions pv ON p.id = pv.portfolio_id
		WHERE %s
		ORDER BY p.id, pv.version_number DESC
	`, whereClause)

	query = fmt.Sprintf(`
		SELECT * FROM (%s) sub
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, query, argNum, argNum+1)

	args = append(args, pagination.Limit, pagination.Offset())

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var portfolios []domain.PortfolioListItem
	for rows.Next() {
		var p domain.PortfolioListItem

		if err := rows.Scan(
			&p.ID, &p.Title, &p.Slug, &p.ThumbnailURL, &p.Status,
			&p.AdminReviewNote, &p.CreatedAt, &p.UpdatedAt, &p.LikeCount,
		); err != nil {
			return nil, 0, err
		}

		portfolios = append(portfolios, p)
	}

	return portfolios, total, rows.Err()
}

// GetPortfolioTags gets tags for a portfolio
func (r *PortfolioRepository) GetPortfolioTags(ctx context.Context, portfolioID uuid.UUID) ([]domain.TagInfo, error) {
	query := `
		SELECT t.id, t.name
		FROM tags t
		INNER JOIN portfolio_tags pt ON t.id = pt.tag_id
		WHERE pt.portfolio_id = $1 AND t.deleted_at IS NULL
		ORDER BY t.name
	`

	rows, err := r.db.Query(ctx, query, portfolioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []domain.TagInfo
	for rows.Next() {
		var tag domain.TagInfo
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

// SetPortfolioTags sets the tags for a portfolio (replaces existing)
func (r *PortfolioRepository) SetPortfolioTags(ctx context.Context, portfolioID uuid.UUID, tagIDs []uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete existing tags
	_, err = tx.Exec(ctx, "DELETE FROM portfolio_tags WHERE portfolio_id = $1", portfolioID)
	if err != nil {
		return err
	}

	// Insert new tags
	for _, tagID := range tagIDs {
		_, err = tx.Exec(ctx, "INSERT INTO portfolio_tags (portfolio_id, tag_id) VALUES ($1, $2)", portfolioID, tagID)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetContentBlocks gets content blocks for a portfolio version
func (r *PortfolioRepository) GetContentBlocks(ctx context.Context, versionID uuid.UUID) ([]domain.ContentBlock, error) {
	query := `
		SELECT id, portfolio_version_id, block_type, block_order, payload, created_at, updated_at
		FROM content_blocks
		WHERE portfolio_version_id = $1
		ORDER BY block_order ASC
	`

	rows, err := r.db.Query(ctx, query, versionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var blocks []domain.ContentBlock
	for rows.Next() {
		var block domain.ContentBlock
		if err := rows.Scan(
			&block.ID, &block.PortfolioVersionID, &block.BlockType, &block.BlockOrder,
			&block.Payload, &block.CreatedAt, &block.UpdatedAt,
		); err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, rows.Err()
}

// CreateContentBlock creates a new content block
func (r *PortfolioRepository) CreateContentBlock(ctx context.Context, block *domain.ContentBlock) error {
	query := `
		INSERT INTO content_blocks (id, portfolio_version_id, block_type, block_order, payload, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(ctx, query,
		block.ID, block.PortfolioVersionID, block.BlockType, block.BlockOrder,
		block.Payload, block.CreatedAt, block.UpdatedAt,
	)
	return err
}

// UpdateContentBlock updates a content block
func (r *PortfolioRepository) UpdateContentBlock(ctx context.Context, blockID uuid.UUID, payload json.RawMessage) error {
	query := `UPDATE content_blocks SET payload = $2, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, blockID, payload)
	return err
}

// DeleteContentBlock deletes a content block
func (r *PortfolioRepository) DeleteContentBlock(ctx context.Context, blockID uuid.UUID) error {
	query := `DELETE FROM content_blocks WHERE id = $1`
	_, err := r.db.Exec(ctx, query, blockID)
	return err
}

// GetContentBlockByID gets a content block by ID
func (r *PortfolioRepository) GetContentBlockByID(ctx context.Context, blockID uuid.UUID) (*domain.ContentBlock, error) {
	query := `
		SELECT id, portfolio_version_id, block_type, block_order, payload, created_at, updated_at
		FROM content_blocks
		WHERE id = $1
	`

	var block domain.ContentBlock
	err := r.db.QueryRow(ctx, query, blockID).Scan(
		&block.ID, &block.PortfolioVersionID, &block.BlockType, &block.BlockOrder,
		&block.Payload, &block.CreatedAt, &block.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &block, nil
}

// ReorderContentBlocks updates the order of content blocks
func (r *PortfolioRepository) ReorderContentBlocks(ctx context.Context, orders []struct {
	ID    uuid.UUID
	Order int
}) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, o := range orders {
		_, err = tx.Exec(ctx, "UPDATE content_blocks SET block_order = $2 WHERE id = $1", o.ID, o.Order)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// CountContentBlocks counts blocks in a version
func (r *PortfolioRepository) CountContentBlocks(ctx context.Context, versionID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM content_blocks WHERE portfolio_version_id = $1", versionID).Scan(&count)
	return count, err
}

// GetPortfolioUser gets the owner of a portfolio
func (r *PortfolioRepository) GetPortfolioUser(ctx context.Context, portfolioID uuid.UUID) (*domain.UserListItem, error) {
	query := `
		SELECT u.id, u.username, u.name, u.avatar_url, u.role, c.name
		FROM portfolios p
		INNER JOIN users u ON p.user_id = u.id
		LEFT JOIN classes c ON u.current_class_id = c.id
		WHERE p.id = $1
	`

	var user domain.UserListItem
	var className *string

	err := r.db.QueryRow(ctx, query, portfolioID).Scan(
		&user.ID, &user.Username, &user.Name, &user.AvatarURL, &user.Role, &className,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if className != nil {
		user.ClassName = *className
	}

	return &user, nil
}

// ListPendingReview lists portfolios pending review (for admin moderation)
func (r *PortfolioRepository) ListPendingReview(ctx context.Context, page, limit int) ([]domain.PortfolioListItem, int, error) {
	pagination := utils.Pagination{Page: page, Limit: limit}
	pagination.Validate(50)

	countQuery := `
		SELECT COUNT(*)
		FROM portfolios p
		INNER JOIN portfolio_versions pv ON p.id = pv.portfolio_id
		WHERE pv.status = 'pending_review' AND p.deleted_at IS NULL
	`

	var total int
	if err := r.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT p.id, pv.title, p.slug, pv.thumbnail_url, pv.status, p.created_at, p.updated_at,
		       u.id, u.username, u.name, u.avatar_url, u.role, c.name
		FROM portfolios p
		INNER JOIN portfolio_versions pv ON p.id = pv.portfolio_id
		INNER JOIN users u ON p.user_id = u.id
		LEFT JOIN classes c ON u.current_class_id = c.id
		WHERE pv.status = 'pending_review' AND p.deleted_at IS NULL
		ORDER BY pv.updated_at ASC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, pagination.Limit, pagination.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var portfolios []domain.PortfolioListItem
	for rows.Next() {
		var p domain.PortfolioListItem
		var user domain.UserListItem
		var className *string

		if err := rows.Scan(
			&p.ID, &p.Title, &p.Slug, &p.ThumbnailURL, &p.Status, &p.CreatedAt, &p.UpdatedAt,
			&user.ID, &user.Username, &user.Name, &user.AvatarURL, &user.Role, &className,
		); err != nil {
			return nil, 0, err
		}

		if className != nil {
			user.ClassName = *className
		}
		p.User = &user

		portfolios = append(portfolios, p)
	}

	return portfolios, total, rows.Err()
}

// IncrementLikeCount increments the like count
func (r *PortfolioRepository) IncrementLikeCount(ctx context.Context, portfolioID uuid.UUID) error {
	query := `UPDATE portfolios SET like_count = like_count + 1 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, portfolioID)
	return err
}

// DecrementLikeCount decrements the like count
func (r *PortfolioRepository) DecrementLikeCount(ctx context.Context, portfolioID uuid.UUID) error {
	query := `UPDATE portfolios SET like_count = GREATEST(like_count - 1, 0) WHERE id = $1`
	_, err := r.db.Exec(ctx, query, portfolioID)
	return err
}

// CreateLike creates a like
func (r *PortfolioRepository) CreateLike(ctx context.Context, userID, portfolioID uuid.UUID) error {
	query := `INSERT INTO likes (id, user_id, portfolio_id, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(ctx, query, uuid.New(), userID, portfolioID, time.Now())
	return err
}

// DeleteLike removes a like
func (r *PortfolioRepository) DeleteLike(ctx context.Context, userID, portfolioID uuid.UUID) error {
	query := `DELETE FROM likes WHERE user_id = $1 AND portfolio_id = $2`
	_, err := r.db.Exec(ctx, query, userID, portfolioID)
	return err
}

// HasLiked checks if a user has liked a portfolio
func (r *PortfolioRepository) HasLiked(ctx context.Context, userID, portfolioID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM likes WHERE user_id = $1 AND portfolio_id = $2)`
	var exists bool
	err := r.db.QueryRow(ctx, query, userID, portfolioID).Scan(&exists)
	return exists, err
}

// GetFeed gets portfolios from followed users
func (r *PortfolioRepository) GetFeed(ctx context.Context, userID uuid.UUID, page, limit int) ([]domain.PortfolioListItem, int, error) {
	pagination := utils.Pagination{Page: page, Limit: limit}
	pagination.Validate(50)

	countQuery := `
		SELECT COUNT(*)
		FROM portfolios p
		INNER JOIN portfolio_versions pv ON p.id = pv.portfolio_id
		INNER JOIN follows f ON p.user_id = f.following_id
		WHERE f.follower_id = $1 AND pv.status = 'published' AND p.deleted_at IS NULL
	`

	var total int
	if err := r.db.QueryRow(ctx, countQuery, userID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT DISTINCT ON (p.id) p.id, pv.title, p.slug, pv.thumbnail_url, pv.published_at, p.like_count,
		       u.id, u.username, u.name, u.avatar_url, u.role, c.name
		FROM portfolios p
		INNER JOIN portfolio_versions pv ON p.id = pv.portfolio_id
		INNER JOIN follows f ON p.user_id = f.following_id
		INNER JOIN users u ON p.user_id = u.id
		LEFT JOIN classes c ON u.current_class_id = c.id
		WHERE f.follower_id = $1 AND pv.status = 'published' AND p.deleted_at IS NULL
		ORDER BY p.id, pv.version_number DESC
	`

	query = fmt.Sprintf(`
		SELECT * FROM (%s) sub
		ORDER BY published_at DESC
		LIMIT $2 OFFSET $3
	`, query)

	rows, err := r.db.Query(ctx, query, userID, pagination.Limit, pagination.Offset())
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var portfolios []domain.PortfolioListItem
	for rows.Next() {
		var p domain.PortfolioListItem
		var user domain.UserListItem
		var className *string

		if err := rows.Scan(
			&p.ID, &p.Title, &p.Slug, &p.ThumbnailURL, &p.PublishedAt, &p.LikeCount,
			&user.ID, &user.Username, &user.Name, &user.AvatarURL, &user.Role, &className,
		); err != nil {
			return nil, 0, err
		}

		if className != nil {
			user.ClassName = *className
		}
		p.User = &user

		// Get tags
		tags, _ := r.GetPortfolioTags(ctx, p.ID)
		p.Tags = tags

		portfolios = append(portfolios, p)
	}

	return portfolios, total, rows.Err()
}
