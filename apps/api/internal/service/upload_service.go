package service

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"grafikarsa/internal/config"
	"grafikarsa/internal/domain"
	"grafikarsa/internal/repository"
)

// UploadService handles file upload operations with MinIO
type UploadService struct {
	minioClient   *minio.Client
	uploadRepo    *repository.UploadRepository
	userRepo      *repository.UserRepository
	portfolioRepo *repository.PortfolioRepository
	config        *config.Config
}

// NewUploadService creates a new UploadService
func NewUploadService(
	cfg *config.Config,
	uploadRepo *repository.UploadRepository,
	userRepo *repository.UserRepository,
	portfolioRepo *repository.PortfolioRepository,
) (*UploadService, error) {
	// Initialize MinIO client
	minioClient, err := minio.New(cfg.MinIOEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinIOAccessKeyID, cfg.MinIOSecretAccessKey, ""),
		Secure: cfg.MinIOUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	return &UploadService{
		minioClient:   minioClient,
		uploadRepo:    uploadRepo,
		userRepo:      userRepo,
		portfolioRepo: portfolioRepo,
		config:        cfg,
	}, nil
}

// PresignInput contains input for generating a presigned URL
type PresignInput struct {
	UploadType  domain.UploadType
	Filename    string
	ContentType string
	FileSize    int64
	PortfolioID *uuid.UUID // Required for thumbnail and portfolio_image
}

// GeneratePresignedURL generates a presigned URL for uploading
func (s *UploadService) GeneratePresignedURL(ctx context.Context, userID uuid.UUID, input PresignInput) (*domain.PresignedUploadResponse, error) {
	// Validate upload type
	uploadConfig := domain.GetUploadConfig(input.UploadType)
	if uploadConfig.Path == "" {
		return nil, ErrInvalidUploadType
	}

	// Validate file size
	if !input.UploadType.IsFileSizeAllowed(input.FileSize) {
		return nil, ErrFileTooLarge
	}

	// Validate content type
	if !input.UploadType.IsAllowedContentType(input.ContentType) {
		return nil, ErrInvalidContentType
	}

	// For portfolio uploads, verify ownership
	if input.UploadType == domain.UploadTypeThumbnail || input.UploadType == domain.UploadTypePortfolioImage {
		if input.PortfolioID == nil {
			return nil, ErrPortfolioIDRequired
		}

		portfolio, _, err := s.portfolioRepo.GetByID(ctx, *input.PortfolioID)
		if err != nil {
			return nil, fmt.Errorf("database error: %w", err)
		}
		if portfolio == nil {
			return nil, ErrPortfolioNotFound
		}
		if portfolio.UserID != userID {
			return nil, ErrUploadForbidden
		}
	}

	// Generate unique filename
	ext := filepath.Ext(input.Filename)
	uniqueFilename := fmt.Sprintf("%s%s", uuid.New().String(), ext)

	// Generate object key
	var objectKey string
	switch input.UploadType {
	case domain.UploadTypeAvatar, domain.UploadTypeBanner:
		objectKey = fmt.Sprintf("%s/%s/%s", uploadConfig.Path, userID.String(), uniqueFilename)
	case domain.UploadTypeThumbnail, domain.UploadTypePortfolioImage:
		objectKey = fmt.Sprintf("%s/%s/%s", uploadConfig.Path, input.PortfolioID.String(), uniqueFilename)
	}

	// Generate presigned URL (15 minutes expiry)
	expiry := 15 * time.Minute
	presignedURL, err := s.minioClient.PresignedPutObject(ctx, s.config.MinIOBucketName, objectKey, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to generate presigned URL: %w", err)
	}

	// Store pending upload
	uploadID := uuid.New()
	pendingUpload := &domain.PendingUpload{
		ID:          uploadID,
		UserID:      userID,
		UploadType:  input.UploadType,
		ObjectKey:   objectKey,
		Filename:    input.Filename,
		ContentType: input.ContentType,
		FileSize:    input.FileSize,
		PortfolioID: input.PortfolioID,
		ExpiresAt:   time.Now().Add(expiry),
		CreatedAt:   time.Now(),
	}

	if err := s.uploadRepo.CreatePendingUpload(ctx, pendingUpload); err != nil {
		return nil, fmt.Errorf("failed to store pending upload: %w", err)
	}

	return &domain.PresignedUploadResponse{
		UploadID:     uploadID,
		PresignedURL: presignedURL.String(),
		ObjectKey:    objectKey,
		ExpiresIn:    int64(expiry.Seconds()),
		Method:       "PUT",
		Headers: map[string]string{
			"Content-Type": input.ContentType,
		},
	}, nil
}

// ConfirmUpload confirms an upload and updates the database
func (s *UploadService) ConfirmUpload(ctx context.Context, userID uuid.UUID, uploadID uuid.UUID, objectKey string) (*domain.UploadConfirmation, error) {
	// Get pending upload
	pendingUpload, err := s.uploadRepo.GetPendingUpload(ctx, uploadID)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if pendingUpload == nil {
		return nil, ErrUploadNotFound
	}

	// Verify ownership
	if pendingUpload.UserID != userID {
		return nil, ErrUploadForbidden
	}

	// Verify object key matches
	if pendingUpload.ObjectKey != objectKey {
		return nil, ErrObjectKeyMismatch
	}

	// Check if expired
	if pendingUpload.IsExpired() {
		_ = s.uploadRepo.DeletePendingUpload(ctx, uploadID)
		return nil, ErrUploadExpired
	}

	// Verify object exists in MinIO
	_, err = s.minioClient.StatObject(ctx, s.config.MinIOBucketName, objectKey, minio.StatObjectOptions{})
	if err != nil {
		return nil, ErrObjectNotFound
	}

	// Generate public URL
	publicURL := fmt.Sprintf("%s/%s", s.config.MinioCDNURL, objectKey)

	// Update database based on upload type
	switch pendingUpload.UploadType {
	case domain.UploadTypeAvatar:
		if err := s.userRepo.UpdateAvatarURL(ctx, userID, publicURL); err != nil {
			return nil, fmt.Errorf("failed to update avatar: %w", err)
		}

	case domain.UploadTypeBanner:
		if err := s.userRepo.UpdateBannerURL(ctx, userID, publicURL); err != nil {
			return nil, fmt.Errorf("failed to update banner: %w", err)
		}

	case domain.UploadTypeThumbnail:
		if pendingUpload.PortfolioID == nil {
			return nil, ErrPortfolioIDRequired
		}
		_, version, err := s.portfolioRepo.GetByID(ctx, *pendingUpload.PortfolioID)
		if err != nil || version == nil {
			return nil, ErrPortfolioNotFound
		}
		if err := s.portfolioRepo.UpdateThumbnailURL(ctx, version.ID, publicURL); err != nil {
			return nil, fmt.Errorf("failed to update thumbnail: %w", err)
		}

	case domain.UploadTypePortfolioImage:
		// For portfolio images, just return the URL - the client will use it in content blocks
		// No database update needed here
	}

	// Delete pending upload
	_ = s.uploadRepo.DeletePendingUpload(ctx, uploadID)

	return &domain.UploadConfirmation{
		Type:        pendingUpload.UploadType,
		URL:         publicURL,
		ObjectKey:   objectKey,
		PortfolioID: pendingUpload.PortfolioID,
	}, nil
}

// EnsureBucketExists creates the bucket if it doesn't exist
func (s *UploadService) EnsureBucketExists(ctx context.Context) error {
	exists, err := s.minioClient.BucketExists(ctx, s.config.MinIOBucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket: %w", err)
	}

	if !exists {
		if err := s.minioClient.MakeBucket(ctx, s.config.MinIOBucketName, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		// Set bucket policy for public read access
		policy := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [{
				"Effect": "Allow",
				"Principal": "*",
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}]
		}`, s.config.MinIOBucketName)

		if err := s.minioClient.SetBucketPolicy(ctx, s.config.MinIOBucketName, policy); err != nil {
			// Non-fatal, log warning
		}
	}

	return nil
}

// CleanupExpiredUploads removes expired pending uploads
func (s *UploadService) CleanupExpiredUploads(ctx context.Context) error {
	return s.uploadRepo.CleanupExpiredUploads(ctx)
}

// Upload service errors
var (
	ErrInvalidUploadType   = fmt.Errorf("invalid upload type")
	ErrFileTooLarge        = fmt.Errorf("file too large")
	ErrInvalidContentType  = fmt.Errorf("invalid content type")
	ErrPortfolioIDRequired = fmt.Errorf("portfolio ID required")
	ErrUploadForbidden     = fmt.Errorf("upload forbidden")
	ErrUploadNotFound      = fmt.Errorf("upload not found")
	ErrObjectKeyMismatch   = fmt.Errorf("object key mismatch")
	ErrUploadExpired       = fmt.Errorf("upload expired")
	ErrObjectNotFound      = fmt.Errorf("object not found in storage")
)
