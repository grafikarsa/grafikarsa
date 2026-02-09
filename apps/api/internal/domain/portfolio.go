package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// PortfolioStatus represents the status of a portfolio version
type PortfolioStatus string

const (
	StatusDraft         PortfolioStatus = "draft"
	StatusPendingReview PortfolioStatus = "pending_review"
	StatusRejected      PortfolioStatus = "rejected"
	StatusPublished     PortfolioStatus = "published"
	StatusArchived      PortfolioStatus = "archived"
)

// Portfolio represents a portfolio
type Portfolio struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	Slug      string     `json:"slug"`
	LikeCount int        `json:"like_count"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"-"`

	// Current active version (joined)
	CurrentVersion *PortfolioVersion `json:"current_version,omitempty"`
	User           *User             `json:"user,omitempty"`
	Tags           []Tag             `json:"tags,omitempty"`
}

// PortfolioVersion represents a version of a portfolio
type PortfolioVersion struct {
	ID              uuid.UUID       `json:"id"`
	PortfolioID     uuid.UUID       `json:"portfolio_id"`
	VersionNumber   int             `json:"version_number"`
	Title           string          `json:"title"`
	ThumbnailURL    string          `json:"thumbnail_url,omitempty"`
	Status          PortfolioStatus `json:"status"`
	AdminReviewNote string          `json:"admin_review_note,omitempty"`
	ReviewedBy      *uuid.UUID      `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time      `json:"reviewed_at,omitempty"`
	PublishedAt     *time.Time      `json:"published_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`

	// Related data
	ContentBlocks []ContentBlock `json:"content_blocks,omitempty"`
}

// PortfolioListItem represents a portfolio in list views
type PortfolioListItem struct {
	ID           uuid.UUID       `json:"id"`
	Title        string          `json:"title"`
	Slug         string          `json:"slug"`
	ThumbnailURL string          `json:"thumbnail_url,omitempty"`
	Status       PortfolioStatus `json:"status,omitempty"`
	PublishedAt  *time.Time      `json:"published_at,omitempty"`
	LikeCount    int             `json:"like_count"`
	User         *UserListItem   `json:"user,omitempty"`
	Tags         []TagInfo       `json:"tags,omitempty"`
	CreatedAt    time.Time       `json:"created_at,omitempty"`
	UpdatedAt    time.Time       `json:"updated_at,omitempty"`

	// For owner view
	AdminReviewNote string `json:"admin_review_note,omitempty"`
}

// PortfolioDetail represents the full portfolio detail
type PortfolioDetail struct {
	ID              uuid.UUID       `json:"id"`
	Title           string          `json:"title"`
	Slug            string          `json:"slug"`
	ThumbnailURL    string          `json:"thumbnail_url,omitempty"`
	Status          PortfolioStatus `json:"status"`
	PublishedAt     *time.Time      `json:"published_at,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	LikeCount       int             `json:"like_count"`
	IsLiked         bool            `json:"is_liked"`
	User            *UserListItem   `json:"user,omitempty"`
	Tags            []TagInfo       `json:"tags,omitempty"`
	ContentBlocks   []ContentBlock  `json:"content_blocks,omitempty"`
	AdminReviewNote string          `json:"admin_review_note,omitempty"` // Only for owner/admin
}

// BlockType represents the type of content block
type BlockType string

const (
	BlockTypeText    BlockType = "text"
	BlockTypeImage   BlockType = "image"
	BlockTypeTable   BlockType = "table"
	BlockTypeYoutube BlockType = "youtube"
	BlockTypeButton  BlockType = "button"
)

// ContentBlock represents a content block in a portfolio
type ContentBlock struct {
	ID                 uuid.UUID       `json:"id"`
	PortfolioVersionID uuid.UUID       `json:"portfolio_version_id,omitempty"`
	BlockType          BlockType       `json:"block_type"`
	BlockOrder         int             `json:"block_order"`
	Payload            json.RawMessage `json:"payload"`
	CreatedAt          time.Time       `json:"created_at,omitempty"`
	UpdatedAt          time.Time       `json:"updated_at,omitempty"`
}

// TextPayload represents the payload for a text block
type TextPayload struct {
	Content string `json:"content"`
}

// ImagePayload represents the payload for an image block
type ImagePayload struct {
	URL     string `json:"url"`
	Caption string `json:"caption,omitempty"`
}

// TablePayload represents the payload for a table block
type TablePayload struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// YoutubePayload represents the payload for a YouTube block
type YoutubePayload struct {
	VideoID string `json:"video_id"`
}

// ButtonPayload represents the payload for a button block
type ButtonPayload struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// Tag represents a tag for portfolios
type Tag struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at,omitempty"`
	UpdatedAt time.Time  `json:"updated_at,omitempty"`
	DeletedAt *time.Time `json:"-"`

	// For admin views
	PortfolioCount int `json:"portfolio_count,omitempty"`
}

// TagInfo is a lightweight tag reference
type TagInfo struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// ToInfo converts Tag to TagInfo
func (t *Tag) ToInfo() TagInfo {
	return TagInfo{
		ID:   t.ID,
		Name: t.Name,
	}
}

// PortfolioTag represents the many-to-many relationship
type PortfolioTag struct {
	PortfolioID uuid.UUID `json:"portfolio_id"`
	TagID       uuid.UUID `json:"tag_id"`
}

// Follow represents a follow relationship
type Follow struct {
	ID          uuid.UUID `json:"id"`
	FollowerID  uuid.UUID `json:"follower_id"`
	FollowingID uuid.UUID `json:"following_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// Like represents a portfolio like
type Like struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	PortfolioID uuid.UUID `json:"portfolio_id"`
	CreatedAt   time.Time `json:"created_at"`
}

// CanTransitionTo checks if a portfolio can transition to the target status
func (s PortfolioStatus) CanTransitionTo(target PortfolioStatus) bool {
	transitions := map[PortfolioStatus][]PortfolioStatus{
		StatusDraft:         {StatusPendingReview, StatusArchived},
		StatusPendingReview: {StatusPublished, StatusRejected, StatusDraft},
		StatusRejected:      {StatusDraft, StatusPendingReview, StatusArchived},
		StatusPublished:     {StatusArchived},
		StatusArchived:      {StatusDraft},
	}

	allowed, ok := transitions[s]
	if !ok {
		return false
	}

	for _, a := range allowed {
		if a == target {
			return true
		}
	}
	return false
}

// IsPublic checks if the portfolio is publicly visible
func (s PortfolioStatus) IsPublic() bool {
	return s == StatusPublished
}

// CanEdit checks if the portfolio version can be edited
func (s PortfolioStatus) CanEdit() bool {
	return s == StatusDraft || s == StatusRejected
}
