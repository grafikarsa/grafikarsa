package dto

import "time"

// ChangelogCategory enum
type ChangelogCategory string

const (
	ChangelogCategoryAdded   ChangelogCategory = "added"
	ChangelogCategoryUpdated ChangelogCategory = "updated"
	ChangelogCategoryRemoved ChangelogCategory = "removed"
	ChangelogCategoryFixed   ChangelogCategory = "fixed"
)

// ============================================================================
// REQUEST DTOs
// ============================================================================

// CreateChangelogRequest - request untuk membuat changelog baru
type CreateChangelogRequest struct {
	Version      string                        `json:"version" validate:"required,max=50"`
	Title        string                        `json:"title" validate:"required,max=255"`
	Description  *string                       `json:"description,omitempty"`
	ReleaseDate  *string                       `json:"release_date,omitempty"` // format: YYYY-MM-DD
	Sections     []ChangelogSectionRequest     `json:"sections" validate:"required,min=1"`
	Contributors []ChangelogContributorRequest `json:"contributors" validate:"required,min=1"`
}

// UpdateChangelogRequest - request untuk update changelog
type UpdateChangelogRequest struct {
	Version      *string                       `json:"version,omitempty" validate:"omitempty,max=50"`
	Title        *string                       `json:"title,omitempty" validate:"omitempty,max=255"`
	Description  *string                       `json:"description,omitempty"`
	ReleaseDate  *string                       `json:"release_date,omitempty"`
	Sections     []ChangelogSectionRequest     `json:"sections,omitempty"`
	Contributors []ChangelogContributorRequest `json:"contributors,omitempty"`
}

// ChangelogSectionRequest - section dalam request
type ChangelogSectionRequest struct {
	Category string                  `json:"category" validate:"required,oneof=added updated removed fixed"`
	Blocks   []ChangelogBlockRequest `json:"blocks" validate:"required,min=1"`
}

// ChangelogBlockRequest - content block dalam request
type ChangelogBlockRequest struct {
	BlockType string                 `json:"block_type" validate:"required"`
	Payload   map[string]interface{} `json:"payload" validate:"required"`
}

// ChangelogContributorRequest - contributor dalam request
type ChangelogContributorRequest struct {
	UserID       string `json:"user_id" validate:"required,uuid"`
	Contribution string `json:"contribution" validate:"required,max=255"`
}

// ============================================================================
// RESPONSE DTOs
// ============================================================================

// ChangelogResponse - response untuk single changelog
type ChangelogResponse struct {
	ID           string                         `json:"id"`
	Version      string                         `json:"version"`
	Title        string                         `json:"title"`
	Description  *string                        `json:"description,omitempty"`
	ReleaseDate  string                         `json:"release_date"`
	IsPublished  bool                           `json:"is_published"`
	Sections     []ChangelogSectionResponse     `json:"sections"`
	Contributors []ChangelogContributorResponse `json:"contributors"`
	CreatedBy    *UserBriefResponse             `json:"created_by,omitempty"`
	CreatedAt    time.Time                      `json:"created_at"`
	UpdatedAt    time.Time                      `json:"updated_at"`
}

// ChangelogListResponse - response untuk list (tanpa full sections)
type ChangelogListResponse struct {
	ID          string    `json:"id"`
	Version     string    `json:"version"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	ReleaseDate string    `json:"release_date"`
	IsPublished bool      `json:"is_published"`
	Categories  []string  `json:"categories"` // list of categories that have content
	CreatedAt   time.Time `json:"created_at"`
}

// ChangelogSectionResponse - section dalam response
type ChangelogSectionResponse struct {
	ID       string                   `json:"id"`
	Category ChangelogCategory        `json:"category"`
	Blocks   []ChangelogBlockResponse `json:"blocks"`
}

// ChangelogBlockResponse - content block dalam response
type ChangelogBlockResponse struct {
	ID        string                 `json:"id"`
	BlockType string                 `json:"block_type"`
	Payload   map[string]interface{} `json:"payload"`
}

// ChangelogContributorResponse - contributor dalam response
type ChangelogContributorResponse struct {
	ID           string             `json:"id"`
	User         *UserBriefResponse `json:"user"`
	Contribution string             `json:"contribution"`
}

// ChangelogUnreadCountResponse - response untuk unread count
type ChangelogUnreadCountResponse struct {
	Count int64 `json:"count"`
}
