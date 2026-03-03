package dto

import (
	"time"

	"github.com/google/uuid"
)

// FeedItemDTO represents a portfolio item in the feed
type FeedItemDTO struct {
	ID           uuid.UUID    `json:"id"`
	Judul        string       `json:"judul"`
	Slug         string       `json:"slug"`
	ThumbnailURL *string      `json:"thumbnail_url"`
	PreviewText  *string      `json:"preview_text,omitempty"`
	PublishedAt  *time.Time   `json:"published_at"`
	CreatedAt    time.Time    `json:"created_at"`
	User         *FeedUserDTO `json:"user"`
	Tags         []TagDTO     `json:"tags"`
	LikeCount    int64        `json:"like_count"`
	ViewCount    int64        `json:"view_count"`
	IsLiked      bool         `json:"is_liked"`
	RankingScore float64      `json:"ranking_score,omitempty"`
}

// FeedUserDTO represents user info in feed items
type FeedUserDTO struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Nama      string    `json:"nama"`
	AvatarURL *string   `json:"avatar_url"`
	Role      string    `json:"role"`
	KelasNama *string   `json:"kelas_nama,omitempty"`
}

// FeedPreferenceDTO represents user's feed algorithm preference
type FeedPreferenceDTO struct {
	Algorithm string `json:"algorithm"`
}

// UpdateFeedPreferenceRequest represents request to update feed preference
type UpdateFeedPreferenceRequest struct {
	Algorithm string `json:"algorithm" validate:"required,oneof=smart recent following"`
}

// FeedResponse represents the feed API response
type FeedResponse struct {
	Data []FeedItemDTO `json:"data"`
	Meta *Meta         `json:"meta"`
}
