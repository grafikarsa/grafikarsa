package dto

import (
	"time"

	"github.com/google/uuid"
)

// Portfolio List Item
type PortfolioListDTO struct {
	ID           uuid.UUID           `json:"id"`
	Judul        string              `json:"judul"`
	Slug         string              `json:"slug"`
	ThumbnailURL *string             `json:"thumbnail_url,omitempty"`
	PublishedAt  *time.Time          `json:"published_at,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	LikeCount    int64               `json:"like_count"`
	IsLiked      bool                `json:"is_liked,omitempty"`
	User         *PortfolioUserDTO   `json:"user,omitempty"`
	Tags         []TagDTO            `json:"tags,omitempty"`
	Series       *PortfolioSeriesDTO `json:"series,omitempty"`
}

// PortfolioSeriesDTO - series info dalam portfolio
type PortfolioSeriesDTO struct {
	ID   uuid.UUID `json:"id"`
	Nama string    `json:"nama"`
}

type PortfolioUserDTO struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	Nama        string    `json:"nama"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	Role        string    `json:"role"`
	KelasNama   *string   `json:"kelas_nama,omitempty"`
	JurusanNama *string   `json:"jurusan_nama,omitempty"`
}

// Portfolio Detail
type PortfolioDetailDTO struct {
	ID              uuid.UUID           `json:"id"`
	Judul           string              `json:"judul"`
	Slug            string              `json:"slug"`
	ThumbnailURL    *string             `json:"thumbnail_url,omitempty"`
	Status          string              `json:"status"`
	AdminReviewNote *string             `json:"admin_review_note,omitempty"`
	ReviewedAt      *time.Time          `json:"reviewed_at,omitempty"`
	PublishedAt     *time.Time          `json:"published_at,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	LikeCount       int64               `json:"like_count"`
	IsLiked         bool                `json:"is_liked"`
	User            *PortfolioUserDTO   `json:"user,omitempty"`
	Tags            []TagDTO            `json:"tags,omitempty"`
	Series          *PortfolioSeriesDTO `json:"series,omitempty"`
	ContentBlocks   []ContentBlockDTO   `json:"content_blocks,omitempty"`
}

// My Portfolio List Item
type MyPortfolioDTO struct {
	ID              uuid.UUID  `json:"id"`
	Judul           string     `json:"judul"`
	Slug            string     `json:"slug"`
	ThumbnailURL    *string    `json:"thumbnail_url,omitempty"`
	Status          string     `json:"status"`
	AdminReviewNote *string    `json:"admin_review_note,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	LikeCount       int64      `json:"like_count"`
}

// Create/Update Portfolio
type CreatePortfolioRequest struct {
	Judul    string      `json:"judul" validate:"required"`
	UserID   *uuid.UUID  `json:"user_id,omitempty"` // Admin can assign to another user
	TagIDs   []uuid.UUID `json:"tag_ids,omitempty"`
	SeriesID *uuid.UUID  `json:"series_id,omitempty"`
}

type UpdatePortfolioRequest struct {
	Judul        *string     `json:"judul,omitempty"`
	ThumbnailURL *string     `json:"thumbnail_url,omitempty"`
	TagIDs       []uuid.UUID `json:"tag_ids,omitempty"`
	SeriesID     *uuid.UUID  `json:"series_id,omitempty"`
}

// Content Block
type ContentBlockDTO struct {
	ID         uuid.UUID              `json:"id"`
	BlockType  string                 `json:"block_type"`
	BlockOrder int                    `json:"block_order"`
	Payload    map[string]interface{} `json:"payload"`
	CreatedAt  time.Time              `json:"created_at,omitempty"`
	UpdatedAt  time.Time              `json:"updated_at,omitempty"`
}

type CreateContentBlockRequest struct {
	BlockType  string                 `json:"block_type" validate:"required"`
	BlockOrder int                    `json:"block_order"`
	Payload    map[string]interface{} `json:"payload" validate:"required"`
}

type UpdateContentBlockRequest struct {
	Payload map[string]interface{} `json:"payload"`
}

type ReorderBlocksRequest struct {
	BlockOrders []BlockOrderItem `json:"block_orders" validate:"required"`
}

type BlockOrderItem struct {
	ID    uuid.UUID `json:"id"`
	Order int       `json:"order"`
}

// Portfolio Status Response
type PortfolioStatusResponse struct {
	ID     uuid.UUID `json:"id"`
	Status string    `json:"status"`
}

// Like Response
type LikeResponse struct {
	IsLiked   bool  `json:"is_liked"`
	LikeCount int64 `json:"like_count"`
}
