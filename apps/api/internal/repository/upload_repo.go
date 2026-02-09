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

// UploadRepository handles upload tracking data access
type UploadRepository struct {
	db *pgxpool.Pool
}

// NewUploadRepository creates a new UploadRepository
func NewUploadRepository(db *pgxpool.Pool) *UploadRepository {
	return &UploadRepository{db: db}
}

// CreatePendingUpload creates a pending upload record
func (r *UploadRepository) CreatePendingUpload(ctx context.Context, upload *domain.PendingUpload) error {
	query := `
		INSERT INTO pending_uploads (id, user_id, upload_type, object_key, filename, content_type, file_size, portfolio_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.Exec(ctx, query,
		upload.ID, upload.UserID, upload.UploadType, upload.ObjectKey, upload.Filename,
		upload.ContentType, upload.FileSize, upload.PortfolioID, upload.ExpiresAt, upload.CreatedAt,
	)
	return err
}

// GetPendingUpload retrieves a pending upload by ID
func (r *UploadRepository) GetPendingUpload(ctx context.Context, id uuid.UUID) (*domain.PendingUpload, error) {
	query := `
		SELECT id, user_id, upload_type, object_key, filename, content_type, file_size, portfolio_id, expires_at, created_at
		FROM pending_uploads
		WHERE id = $1
	`

	var upload domain.PendingUpload
	err := r.db.QueryRow(ctx, query, id).Scan(
		&upload.ID, &upload.UserID, &upload.UploadType, &upload.ObjectKey, &upload.Filename,
		&upload.ContentType, &upload.FileSize, &upload.PortfolioID, &upload.ExpiresAt, &upload.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &upload, nil
}

// DeletePendingUpload deletes a pending upload
func (r *UploadRepository) DeletePendingUpload(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM pending_uploads WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// CleanupExpiredUploads deletes expired pending uploads
func (r *UploadRepository) CleanupExpiredUploads(ctx context.Context) error {
	query := `DELETE FROM pending_uploads WHERE expires_at < $1`
	_, err := r.db.Exec(ctx, query, time.Now())
	return err
}
