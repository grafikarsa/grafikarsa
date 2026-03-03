package dto

import "github.com/google/uuid"

type PresignRequest struct {
	UploadType  string     `json:"upload_type" validate:"required"`
	Filename    string     `json:"filename" validate:"required"`
	ContentType string     `json:"content_type" validate:"required"`
	FileSize    int64      `json:"file_size" validate:"required"`
	PortfolioID *uuid.UUID `json:"portfolio_id,omitempty"`
	BlockID     *uuid.UUID `json:"block_id,omitempty"`
}

type PresignResponse struct {
	UploadID     string            `json:"upload_id"`
	PresignedURL string            `json:"presigned_url"`
	ObjectKey    string            `json:"object_key"`
	ExpiresIn    int               `json:"expires_in"`
	Method       string            `json:"method"`
	Headers      map[string]string `json:"headers"`
}

type ConfirmUploadRequest struct {
	UploadID  string `json:"upload_id" validate:"required"`
	ObjectKey string `json:"object_key" validate:"required"`
}

type ConfirmUploadResponse struct {
	Type        string     `json:"type"`
	URL         string     `json:"url"`
	ObjectKey   string     `json:"object_key"`
	PortfolioID *uuid.UUID `json:"portfolio_id,omitempty"`
	BlockID     *uuid.UUID `json:"block_id,omitempty"`
}

type PresignViewResponse struct {
	URL       string `json:"url"`
	ExpiresIn int    `json:"expires_in"`
}
