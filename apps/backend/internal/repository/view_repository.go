package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ViewRepository struct {
	db *gorm.DB
}

func NewViewRepository(db *gorm.DB) *ViewRepository {
	return &ViewRepository{db: db}
}

// RecordView records a portfolio view with upsert logic
// If user is authenticated, use userID; otherwise use sessionID for guests
func (r *ViewRepository) RecordView(portfolioID uuid.UUID, userID *uuid.UUID, sessionID *string) error {
	view := domain.PortfolioView{
		PortfolioID: portfolioID,
		UserID:      userID,
		SessionID:   sessionID,
		ViewedAt:    time.Now(),
	}

	// Use different conflict columns based on whether it's a user or session view
	if userID != nil {
		// Upsert for authenticated user
		return r.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "portfolio_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"viewed_at"}),
		}).Create(&view).Error
	}

	// Upsert for guest (session-based)
	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "portfolio_id"}, {Name: "session_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"viewed_at"}),
	}).Create(&view).Error
}

// GetViewCount returns the unique viewer count for a portfolio
func (r *ViewRepository) GetViewCount(portfolioID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&domain.PortfolioView{}).
		Where("portfolio_id = ?", portfolioID).
		Count(&count).Error
	return count, err
}

// GetViewsByUser returns all portfolios viewed by a user
func (r *ViewRepository) GetViewsByUser(userID uuid.UUID, page, limit int) ([]domain.PortfolioView, int64, error) {
	var views []domain.PortfolioView
	var total int64

	query := r.db.Model(&domain.PortfolioView{}).Where("user_id = ?", userID)
	query.Count(&total)

	offset := (page - 1) * limit
	err := query.Preload("Portfolio").
		Offset(offset).Limit(limit).
		Order("viewed_at DESC").
		Find(&views).Error

	return views, total, err
}

// HasUserViewed checks if a user has viewed a specific portfolio
func (r *ViewRepository) HasUserViewed(userID, portfolioID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.PortfolioView{}).
		Where("user_id = ? AND portfolio_id = ?", userID, portfolioID).
		Count(&count).Error
	return count > 0, err
}

// GetRecentViewers returns recent viewers of a portfolio
func (r *ViewRepository) GetRecentViewers(portfolioID uuid.UUID, limit int) ([]domain.PortfolioView, error) {
	var views []domain.PortfolioView
	err := r.db.Model(&domain.PortfolioView{}).
		Where("portfolio_id = ? AND user_id IS NOT NULL", portfolioID).
		Preload("User").
		Order("viewed_at DESC").
		Limit(limit).
		Find(&views).Error
	return views, err
}

// GetMaxEngagementStats returns max likes and views for normalization
func (r *ViewRepository) GetMaxEngagementStats() (maxLikes int64, maxViews int64, err error) {
	type Stats struct {
		MaxLikes int64
		MaxViews int64
	}
	var stats Stats

	err = r.db.Raw(`
		SELECT 
			COALESCE(MAX(like_count), 1) as max_likes,
			COALESCE(MAX(view_count), 1) as max_views
		FROM (
			SELECT 
				p.id,
				(SELECT COUNT(*) FROM portfolio_likes WHERE portfolio_id = p.id) as like_count,
				(SELECT COUNT(*) FROM portfolio_views WHERE portfolio_id = p.id) as view_count
			FROM portfolios p
			WHERE p.status = 'published' AND p.deleted_at IS NULL
		) stats
	`).Scan(&stats).Error

	return stats.MaxLikes, stats.MaxViews, err
}
