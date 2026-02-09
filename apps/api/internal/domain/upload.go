package domain

import (
	"time"

	"github.com/google/uuid"
)

// UploadType represents the type of file upload
type UploadType string

const (
	UploadTypeAvatar         UploadType = "avatar"
	UploadTypeBanner         UploadType = "banner"
	UploadTypeThumbnail      UploadType = "thumbnail"
	UploadTypePortfolioImage UploadType = "portfolio_image"
)

// UploadConfig contains configuration for an upload type
type UploadConfig struct {
	MaxSize      int64    // Max size in bytes
	AllowedTypes []string // Allowed MIME types
	Path         string   // Storage path prefix
}

// GetUploadConfig returns the configuration for an upload type
func GetUploadConfig(uploadType UploadType) UploadConfig {
	configs := map[UploadType]UploadConfig{
		UploadTypeAvatar: {
			MaxSize:      2 * 1024 * 1024, // 2MB
			AllowedTypes: []string{"image/jpeg", "image/png", "image/webp"},
			Path:         "avatars",
		},
		UploadTypeBanner: {
			MaxSize:      2 * 1024 * 1024, // 2MB
			AllowedTypes: []string{"image/jpeg", "image/png", "image/webp"},
			Path:         "banners",
		},
		UploadTypeThumbnail: {
			MaxSize:      5 * 1024 * 1024, // 5MB
			AllowedTypes: []string{"image/jpeg", "image/png", "image/webp"},
			Path:         "thumbnails",
		},
		UploadTypePortfolioImage: {
			MaxSize:      5 * 1024 * 1024, // 5MB
			AllowedTypes: []string{"image/jpeg", "image/png", "image/webp", "image/gif"},
			Path:         "portfolio-images",
		},
	}

	return configs[uploadType]
}

// PendingUpload represents a pending upload waiting for confirmation
type PendingUpload struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	UploadType  UploadType `json:"upload_type"`
	ObjectKey   string     `json:"object_key"`
	Filename    string     `json:"filename"`
	ContentType string     `json:"content_type"`
	FileSize    int64      `json:"file_size"`
	PortfolioID *uuid.UUID `json:"portfolio_id,omitempty"`
	ExpiresAt   time.Time  `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// IsExpired checks if the upload has expired
func (p *PendingUpload) IsExpired() bool {
	return time.Now().After(p.ExpiresAt)
}

// PresignedUploadResponse is the response for a presigned URL request
type PresignedUploadResponse struct {
	UploadID     uuid.UUID         `json:"upload_id"`
	PresignedURL string            `json:"presigned_url"`
	ObjectKey    string            `json:"object_key"`
	ExpiresIn    int64             `json:"expires_in"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers"`
}

// UploadConfirmation contains the result of confirming an upload
type UploadConfirmation struct {
	Type        UploadType `json:"type"`
	URL         string     `json:"url"`
	ObjectKey   string     `json:"object_key"`
	PortfolioID *uuid.UUID `json:"portfolio_id,omitempty"`
}

// IsAllowedContentType checks if a content type is allowed for an upload type
func (ut UploadType) IsAllowedContentType(contentType string) bool {
	config := GetUploadConfig(ut)
	for _, allowed := range config.AllowedTypes {
		if allowed == contentType {
			return true
		}
	}
	return false
}

// IsFileSizeAllowed checks if a file size is within limits
func (ut UploadType) IsFileSizeAllowed(size int64) bool {
	config := GetUploadConfig(ut)
	return size <= config.MaxSize
}
