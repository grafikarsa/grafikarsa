package dto

import "time"

// Feedback kategori enum
type FeedbackKategori string

const (
	FeedbackKategoriBug     FeedbackKategori = "bug"
	FeedbackKategoriSaran   FeedbackKategori = "saran"
	FeedbackKategoriLainnya FeedbackKategori = "lainnya"
)

// Feedback status enum
type FeedbackStatus string

const (
	FeedbackStatusPending  FeedbackStatus = "pending"
	FeedbackStatusRead     FeedbackStatus = "read"
	FeedbackStatusResolved FeedbackStatus = "resolved"
)

// UserBriefResponse - brief user info for embedding
type UserBriefResponse struct {
	ID        string  `json:"id"`
	Username  string  `json:"username"`
	Nama      string  `json:"nama"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

// CreateFeedbackRequest - request untuk membuat feedback
type CreateFeedbackRequest struct {
	Kategori FeedbackKategori `json:"kategori" validate:"required,oneof=bug saran lainnya"`
	Pesan    string           `json:"pesan" validate:"required,min=10,max=2000"`
}

// UpdateFeedbackRequest - request untuk update feedback (admin)
type UpdateFeedbackRequest struct {
	Status     *FeedbackStatus `json:"status,omitempty" validate:"omitempty,oneof=pending read resolved"`
	AdminNotes *string         `json:"admin_notes,omitempty" validate:"omitempty,max=1000"`
}

// FeedbackResponse - response untuk feedback
type FeedbackResponse struct {
	ID         string             `json:"id"`
	UserID     *string            `json:"user_id,omitempty"`
	User       *UserBriefResponse `json:"user,omitempty"`
	Kategori   FeedbackKategori   `json:"kategori"`
	Pesan      string             `json:"pesan"`
	Status     FeedbackStatus     `json:"status"`
	AdminNotes *string            `json:"admin_notes,omitempty"`
	ResolvedBy *string            `json:"resolved_by,omitempty"`
	ResolvedAt *time.Time         `json:"resolved_at,omitempty"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
}

// PaginationMeta - pagination metadata
type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// PaginatedResponse - helper untuk paginated response
func PaginatedResponse(data interface{}, meta PaginationMeta) Response {
	return Response{
		Success: true,
		Data:    data,
		Meta: &Meta{
			CurrentPage: meta.Page,
			PerPage:     meta.Limit,
			TotalPages:  meta.TotalPages,
			TotalCount:  meta.Total,
		},
	}
}

// FeedbackStats - statistik feedback untuk dashboard
type FeedbackStats struct {
	Total    int64 `json:"total"`
	Pending  int64 `json:"pending"`
	Read     int64 `json:"read"`
	Resolved int64 `json:"resolved"`
}
