package repository

import (
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"gorm.io/gorm"
)

type FeedRepository struct {
	db *gorm.DB
}

func NewFeedRepository(db *gorm.DB) *FeedRepository {
	return &FeedRepository{db: db}
}

// FeedPortfolio represents a portfolio with additional feed-related data
type FeedPortfolio struct {
	domain.Portfolio
	LikeCount       int64    `gorm:"column:like_count"`
	ViewCount       int64    `gorm:"column:view_count"`
	AssessmentScore *float64 `gorm:"column:assessment_score"`
}

// GetPublishedPortfolios returns all published portfolios for feed calculation
// This is the base query that will be used by the feed service to calculate rankings
func (r *FeedRepository) GetPublishedPortfolios(page, limit int) ([]FeedPortfolio, int64, error) {
	var portfolios []FeedPortfolio
	var total int64

	baseQuery := r.db.Model(&domain.Portfolio{}).
		Where("portfolios.status = ? AND portfolios.deleted_at IS NULL", domain.StatusPublished)

	baseQuery.Count(&total)

	offset := (page - 1) * limit
	err := r.db.Table("portfolios").
		Select(`
			portfolios.*,
			COALESCE((SELECT COUNT(*) FROM portfolio_likes WHERE portfolio_id = portfolios.id), 0) as like_count,
			COALESCE((SELECT COUNT(*) FROM portfolio_views WHERE portfolio_id = portfolios.id), 0) as view_count,
			(SELECT total_score FROM portfolio_assessments WHERE portfolio_id = portfolios.id) as assessment_score
		`).
		Where("portfolios.status = ? AND portfolios.deleted_at IS NULL", domain.StatusPublished).
		Preload("User.Kelas.Jurusan").
		Preload("ContentBlocks").
		Offset(offset).
		Limit(limit).
		Order("portfolios.published_at DESC").
		Find(&portfolios).Error

	if err != nil {
		return nil, 0, err
	}

	// Load tags separately to avoid GORM foreign key issue with embedded struct
	r.loadTagsForPortfolios(portfolios)

	return portfolios, total, nil
}

// loadTagsForPortfolios loads tags for a slice of FeedPortfolio
func (r *FeedRepository) loadTagsForPortfolios(portfolios []FeedPortfolio) {
	if len(portfolios) == 0 {
		return
	}

	// Collect portfolio IDs
	ids := make([]uuid.UUID, len(portfolios))
	idMap := make(map[uuid.UUID]int)
	for i, p := range portfolios {
		ids[i] = p.ID
		idMap[p.ID] = i
	}

	// Query portfolio_tags join table
	type PortfolioTag struct {
		PortfolioID uuid.UUID `gorm:"column:portfolio_id"`
		TagID       uuid.UUID `gorm:"column:tag_id"`
	}
	var ptags []PortfolioTag
	r.db.Table("portfolio_tags").Where("portfolio_id IN ?", ids).Find(&ptags)

	if len(ptags) == 0 {
		return
	}

	// Collect unique tag IDs
	tagIDs := make([]uuid.UUID, 0)
	tagIDSet := make(map[uuid.UUID]bool)
	for _, pt := range ptags {
		if !tagIDSet[pt.TagID] {
			tagIDSet[pt.TagID] = true
			tagIDs = append(tagIDs, pt.TagID)
		}
	}

	// Load tags
	var tags []domain.Tag
	r.db.Where("id IN ?", tagIDs).Find(&tags)

	// Create tag map
	tagMap := make(map[uuid.UUID]domain.Tag)
	for _, t := range tags {
		tagMap[t.ID] = t
	}

	// Assign tags to portfolios
	for _, pt := range ptags {
		if idx, ok := idMap[pt.PortfolioID]; ok {
			if tag, ok := tagMap[pt.TagID]; ok {
				portfolios[idx].Tags = append(portfolios[idx].Tags, tag)
			}
		}
	}
}

// GetRecentFeed returns portfolios sorted by published_at descending
func (r *FeedRepository) GetRecentFeed(page, limit int) ([]FeedPortfolio, int64, error) {
	var portfolios []FeedPortfolio
	var total int64

	baseQuery := r.db.Model(&domain.Portfolio{}).
		Where("portfolios.status = ? AND portfolios.deleted_at IS NULL", domain.StatusPublished)

	baseQuery.Count(&total)

	offset := (page - 1) * limit
	err := r.db.Table("portfolios").
		Select(`
			portfolios.*,
			COALESCE((SELECT COUNT(*) FROM portfolio_likes WHERE portfolio_id = portfolios.id), 0) as like_count,
			COALESCE((SELECT COUNT(*) FROM portfolio_views WHERE portfolio_id = portfolios.id), 0) as view_count,
			(SELECT total_score FROM portfolio_assessments WHERE portfolio_id = portfolios.id) as assessment_score
		`).
		Where("portfolios.status = ? AND portfolios.deleted_at IS NULL", domain.StatusPublished).
		Preload("User.Kelas.Jurusan").
		Offset(offset).
		Limit(limit).
		Order("portfolios.published_at DESC").
		Find(&portfolios).Error

	if err != nil {
		return nil, 0, err
	}

	// Load tags separately to avoid GORM foreign key issue with embedded struct
	r.loadTagsForPortfolios(portfolios)

	return portfolios, total, nil
}

// GetFollowingFeed returns portfolios from users that the given user follows
func (r *FeedRepository) GetFollowingFeed(userID uuid.UUID, page, limit int) ([]FeedPortfolio, int64, error) {
	var portfolios []FeedPortfolio
	var total int64

	baseQuery := r.db.Model(&domain.Portfolio{}).
		Joins("JOIN follows ON portfolios.user_id = follows.following_id").
		Where("follows.follower_id = ? AND portfolios.status = ? AND portfolios.deleted_at IS NULL",
			userID, domain.StatusPublished)

	baseQuery.Count(&total)

	offset := (page - 1) * limit
	err := r.db.Table("portfolios").
		Select(`
			portfolios.*,
			COALESCE((SELECT COUNT(*) FROM portfolio_likes WHERE portfolio_id = portfolios.id), 0) as like_count,
			COALESCE((SELECT COUNT(*) FROM portfolio_views WHERE portfolio_id = portfolios.id), 0) as view_count,
			(SELECT total_score FROM portfolio_assessments WHERE portfolio_id = portfolios.id) as assessment_score
		`).
		Joins("JOIN follows ON portfolios.user_id = follows.following_id").
		Where("follows.follower_id = ? AND portfolios.status = ? AND portfolios.deleted_at IS NULL",
			userID, domain.StatusPublished).
		Preload("User.Kelas.Jurusan").
		Offset(offset).
		Limit(limit).
		Order("portfolios.published_at DESC").
		Find(&portfolios).Error

	if err != nil {
		return nil, 0, err
	}

	// Load tags separately
	r.loadTagsForPortfolios(portfolios)

	return portfolios, total, nil
}

// GetPortfoliosForSmartFeed returns portfolios with all data needed for smart ranking
// Excludes user's own portfolios from the feed
func (r *FeedRepository) GetPortfoliosForSmartFeed(userID uuid.UUID, batchSize int) ([]FeedPortfolio, error) {
	var portfolios []FeedPortfolio

	err := r.db.Table("portfolios").
		Select(`
			portfolios.*,
			COALESCE((SELECT COUNT(*) FROM portfolio_likes WHERE portfolio_id = portfolios.id), 0) as like_count,
			COALESCE((SELECT COUNT(*) FROM portfolio_views WHERE portfolio_id = portfolios.id), 0) as view_count,
			(SELECT total_score FROM portfolio_assessments WHERE portfolio_id = portfolios.id) as assessment_score
		`).
		Where("portfolios.status = ? AND portfolios.deleted_at IS NULL AND portfolios.user_id != ?",
			domain.StatusPublished, userID).
		Preload("User.Kelas.Jurusan").
		Preload("ContentBlocks").
		Limit(batchSize).
		Find(&portfolios).Error

	if err != nil {
		return nil, err
	}

	// Load tags separately
	r.loadTagsForPortfolios(portfolios)

	return portfolios, nil
}

// GetMaxEngagementStats returns the maximum like and view counts for normalization
func (r *FeedRepository) GetMaxEngagementStats() (maxLikes int64, maxViews int64, err error) {
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
				COALESCE((SELECT COUNT(*) FROM portfolio_likes WHERE portfolio_id = p.id), 0) as like_count,
				COALESCE((SELECT COUNT(*) FROM portfolio_views WHERE portfolio_id = p.id), 0) as view_count
			FROM portfolios p
			WHERE p.status = 'published' AND p.deleted_at IS NULL
		) stats
	`).Scan(&stats).Error

	if stats.MaxLikes == 0 {
		stats.MaxLikes = 1
	}
	if stats.MaxViews == 0 {
		stats.MaxViews = 1
	}

	return stats.MaxLikes, stats.MaxViews, err
}

// IsLikedByUser checks if a portfolio is liked by the given user
func (r *FeedRepository) IsLikedByUser(userID, portfolioID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.PortfolioLike{}).
		Where("user_id = ? AND portfolio_id = ?", userID, portfolioID).
		Count(&count).Error
	return count > 0, err
}
