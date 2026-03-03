package dto

import "time"

// ============================================================================
// ASSESSMENT METRICS DTOs
// ============================================================================

// CreateMetricRequest - request untuk membuat metric baru
type CreateMetricRequest struct {
	Nama      string  `json:"nama" validate:"required,min=2,max=100"`
	Deskripsi *string `json:"deskripsi,omitempty" validate:"omitempty,max=1000"`
}

// UpdateMetricRequest - request untuk update metric
type UpdateMetricRequest struct {
	Nama      *string `json:"nama,omitempty" validate:"omitempty,min=2,max=100"`
	Deskripsi *string `json:"deskripsi,omitempty" validate:"omitempty,max=1000"`
	IsActive  *bool   `json:"is_active,omitempty"`
}

// ReorderMetricsRequest - request untuk reorder metrics
type ReorderMetricsRequest struct {
	Orders []MetricOrder `json:"orders" validate:"required,min=1"`
}

// MetricOrder - urutan metric
type MetricOrder struct {
	ID     string `json:"id" validate:"required,uuid"`
	Urutan int    `json:"urutan" validate:"min=0"`
}

// MetricResponse - response untuk metric
type MetricResponse struct {
	ID        string    `json:"id"`
	Nama      string    `json:"nama"`
	Deskripsi *string   `json:"deskripsi,omitempty"`
	Urutan    int       `json:"urutan"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ============================================================================
// PORTFOLIO ASSESSMENT DTOs
// ============================================================================

// ScoreInput - input nilai per metric
type ScoreInput struct {
	MetricID string  `json:"metric_id" validate:"required,uuid"`
	Score    int     `json:"score" validate:"required,min=1,max=10"`
	Comment  *string `json:"comment,omitempty" validate:"omitempty,max=500"`
}

// CreateAssessmentRequest - request untuk membuat/update assessment
type CreateAssessmentRequest struct {
	Scores       []ScoreInput `json:"scores" validate:"required,min=1"`
	FinalComment *string      `json:"final_comment,omitempty" validate:"omitempty,max=2000"`
}

// UpdateAssessmentRequest - request untuk update assessment (sama dengan create)
type UpdateAssessmentRequest = CreateAssessmentRequest

// ScoreResponse - response nilai per metric
type ScoreResponse struct {
	ID        string          `json:"id"`
	MetricID  string          `json:"metric_id"`
	Metric    *MetricResponse `json:"metric,omitempty"`
	Score     int             `json:"score"`
	Comment   *string         `json:"comment,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// AssessmentResponse - response untuk assessment
type AssessmentResponse struct {
	ID           string             `json:"id"`
	PortfolioID  string             `json:"portfolio_id"`
	Portfolio    *PortfolioBrief    `json:"portfolio,omitempty"`
	AssessedBy   string             `json:"assessed_by"`
	Assessor     *UserBriefResponse `json:"assessor,omitempty"`
	Scores       []ScoreResponse    `json:"scores"`
	FinalComment *string            `json:"final_comment,omitempty"`
	TotalScore   *float64           `json:"total_score,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// PortfolioBrief - brief portfolio info for embedding
type PortfolioBrief struct {
	ID           string  `json:"id"`
	Judul        string  `json:"judul"`
	Slug         string  `json:"slug"`
	ThumbnailURL *string `json:"thumbnail_url,omitempty"`
}

// PortfolioForAssessment - portfolio item untuk list assessment
type PortfolioForAssessment struct {
	ID           string             `json:"id"`
	Judul        string             `json:"judul"`
	Slug         string             `json:"slug"`
	ThumbnailURL *string            `json:"thumbnail_url,omitempty"`
	PublishedAt  *time.Time         `json:"published_at,omitempty"`
	User         *UserBriefResponse `json:"user,omitempty"`
	Assessment   *AssessmentBrief   `json:"assessment,omitempty"`
}

// AssessmentBrief - brief assessment info
type AssessmentBrief struct {
	ID         string             `json:"id"`
	TotalScore *float64           `json:"total_score,omitempty"`
	Assessor   *UserBriefResponse `json:"assessor,omitempty"`
	AssessedAt time.Time          `json:"assessed_at"`
}
