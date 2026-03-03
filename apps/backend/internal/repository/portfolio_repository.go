package repository

import (
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"gorm.io/gorm"
)

type PortfolioRepository struct {
	db *gorm.DB
}

func NewPortfolioRepository(db *gorm.DB) *PortfolioRepository {
	return &PortfolioRepository{db: db}
}

func (r *PortfolioRepository) Create(portfolio *domain.Portfolio) error {
	return r.db.Create(portfolio).Error
}

func (r *PortfolioRepository) FindByID(id uuid.UUID) (*domain.Portfolio, error) {
	var portfolio domain.Portfolio
	err := r.db.Preload("User.Kelas.Jurusan").Preload("Tags").Preload("Series").Preload("ContentBlocks", func(db *gorm.DB) *gorm.DB {
		return db.Order("block_order ASC")
	}).Where("id = ? AND deleted_at IS NULL", id).First(&portfolio).Error
	if err != nil {
		return nil, err
	}
	return &portfolio, nil
}

func (r *PortfolioRepository) FindBySlugAndUserID(slug string, userID uuid.UUID) (*domain.Portfolio, error) {
	var portfolio domain.Portfolio
	err := r.db.Preload("User.Kelas.Jurusan").Preload("Tags").Preload("Series").Preload("ContentBlocks", func(db *gorm.DB) *gorm.DB {
		return db.Order("block_order ASC")
	}).Where("slug = ? AND user_id = ? AND deleted_at IS NULL", slug, userID).First(&portfolio).Error
	if err != nil {
		return nil, err
	}
	return &portfolio, nil
}

func (r *PortfolioRepository) FindPublishedBySlugAndUsername(slug string, username string) (*domain.Portfolio, error) {
	var portfolio domain.Portfolio
	query := r.db.Preload("User.Kelas.Jurusan").Preload("Tags").Preload("ContentBlocks", func(db *gorm.DB) *gorm.DB {
		return db.Order("block_order ASC")
	}).Joins("JOIN users ON users.id = portfolios.user_id").
		Where("portfolios.slug = ? AND portfolios.deleted_at IS NULL AND portfolios.status = ?", slug, domain.StatusPublished)

	if username != "" {
		query = query.Where("users.username = ? AND users.deleted_at IS NULL", username)
	}

	err := query.First(&portfolio).Error
	if err != nil {
		return nil, err
	}
	return &portfolio, nil
}

func (r *PortfolioRepository) Update(portfolio *domain.Portfolio) error {
	return r.db.Save(portfolio).Error
}

func (r *PortfolioRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&domain.Portfolio{}).Error
}

func (r *PortfolioRepository) ListPublished(search string, tagIDs []uuid.UUID, jurusanID, kelasID, userID *uuid.UUID, sort string, page, limit int) ([]domain.Portfolio, int64, error) {
	var portfolios []domain.Portfolio
	var total int64

	// Base condition
	baseCondition := "portfolios.status = 'published' AND portfolios.deleted_at IS NULL"

	// Build count query
	countQuery := r.db.Model(&domain.Portfolio{}).Where(baseCondition)

	// Build fetch query
	fetchQuery := r.db.Model(&domain.Portfolio{}).Where(baseCondition)

	if search != "" {
		searchCondition := "portfolios.judul ILIKE ? OR portfolios.user_id IN (SELECT id FROM users WHERE nama ILIKE ?)"
		countQuery = countQuery.Where(searchCondition, "%"+search+"%", "%"+search+"%")
		fetchQuery = fetchQuery.Where(searchCondition, "%"+search+"%", "%"+search+"%")
	}

	if len(tagIDs) > 0 {
		tagCondition := "portfolios.id IN (SELECT portfolio_id FROM portfolio_tags WHERE tag_id IN ?)"
		countQuery = countQuery.Where(tagCondition, tagIDs)
		fetchQuery = fetchQuery.Where(tagCondition, tagIDs)
	}

	if userID != nil {
		countQuery = countQuery.Where("portfolios.user_id = ?", *userID)
		fetchQuery = fetchQuery.Where("portfolios.user_id = ?", *userID)
	}

	if kelasID != nil {
		kelasCondition := "portfolios.user_id IN (SELECT id FROM users WHERE kelas_id = ?)"
		countQuery = countQuery.Where(kelasCondition, *kelasID)
		fetchQuery = fetchQuery.Where(kelasCondition, *kelasID)
	} else if jurusanID != nil {
		jurusanCondition := "portfolios.user_id IN (SELECT id FROM users WHERE kelas_id IN (SELECT id FROM kelas WHERE jurusan_id = ?))"
		countQuery = countQuery.Where(jurusanCondition, *jurusanID)
		fetchQuery = fetchQuery.Where(jurusanCondition, *jurusanID)
	}

	// Count total
	countQuery.Count(&total)

	// Determine order
	orderBy := "portfolios.published_at DESC"
	switch sort {
	case "-like_count":
		orderBy = "portfolios.like_count DESC"
	case "judul":
		orderBy = "portfolios.judul ASC"
	}

	// Fetch with pagination
	offset := (page - 1) * limit
	err := fetchQuery.Preload("User.Kelas").Preload("Tags").
		Offset(offset).Limit(limit).
		Order(orderBy).
		Find(&portfolios).Error

	return portfolios, total, err
}

func (r *PortfolioRepository) ListByUser(userID uuid.UUID, status *string, page, limit int) ([]domain.Portfolio, int64, error) {
	var portfolios []domain.Portfolio
	var total int64

	query := r.db.Model(&domain.Portfolio{}).
		Where("user_id = ? AND deleted_at IS NULL", userID)

	if status != nil {
		query = query.Where("status = ?", *status)
	}

	query.Count(&total)

	offset := (page - 1) * limit
	err := query.Preload("Tags").
		Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&portfolios).Error

	return portfolios, total, err
}

func (r *PortfolioRepository) GetLikeCount(portfolioID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&domain.PortfolioLike{}).Where("portfolio_id = ?", portfolioID).Count(&count).Error
	return count, err
}

func (r *PortfolioRepository) IsLiked(userID, portfolioID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.PortfolioLike{}).
		Where("user_id = ? AND portfolio_id = ?", userID, portfolioID).
		Count(&count).Error
	return count > 0, err
}

func (r *PortfolioRepository) Like(userID, portfolioID uuid.UUID) error {
	like := domain.PortfolioLike{
		UserID:      userID,
		PortfolioID: portfolioID,
	}
	return r.db.Create(&like).Error
}

func (r *PortfolioRepository) Unlike(userID, portfolioID uuid.UUID) error {
	return r.db.Where("user_id = ? AND portfolio_id = ?", userID, portfolioID).
		Delete(&domain.PortfolioLike{}).Error
}

func (r *PortfolioRepository) UpdateTags(portfolioID uuid.UUID, tagIDs []uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing tags
		if err := tx.Where("portfolio_id = ?", portfolioID).Delete(&domain.PortfolioTag{}).Error; err != nil {
			return err
		}
		// Insert new tags
		if len(tagIDs) > 0 {
			var tags []domain.PortfolioTag
			for _, tagID := range tagIDs {
				tags = append(tags, domain.PortfolioTag{
					PortfolioID: portfolioID,
					TagID:       tagID,
				})
			}
			if err := tx.Create(&tags).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PortfolioRepository) UpdateSeriesID(portfolioID uuid.UUID, seriesID *uuid.UUID) error {
	return r.db.Model(&domain.Portfolio{}).Where("id = ?", portfolioID).Update("series_id", seriesID).Error
}

// Content Blocks
func (r *PortfolioRepository) CreateContentBlock(block *domain.ContentBlock) error {
	return r.db.Create(block).Error
}

func (r *PortfolioRepository) FindContentBlockByID(id uuid.UUID) (*domain.ContentBlock, error) {
	var block domain.ContentBlock
	err := r.db.Where("id = ?", id).First(&block).Error
	if err != nil {
		return nil, err
	}
	return &block, nil
}

func (r *PortfolioRepository) UpdateContentBlock(block *domain.ContentBlock) error {
	return r.db.Save(block).Error
}

func (r *PortfolioRepository) DeleteContentBlock(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&domain.ContentBlock{}).Error
}

func (r *PortfolioRepository) ReorderContentBlocks(portfolioID uuid.UUID, orders map[uuid.UUID]int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for blockID, order := range orders {
			if err := tx.Model(&domain.ContentBlock{}).
				Where("id = ? AND portfolio_id = ?", blockID, portfolioID).
				Update("block_order", order).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *PortfolioRepository) GetMaxBlockOrder(portfolioID uuid.UUID) (int, error) {
	var maxOrder *int
	err := r.db.Model(&domain.ContentBlock{}).
		Where("portfolio_id = ?", portfolioID).
		Select("MAX(block_order)").
		Scan(&maxOrder).Error
	if err != nil || maxOrder == nil {
		return -1, err
	}
	return *maxOrder, nil
}

// GetFeed returns portfolios from users that the given user follows
func (r *PortfolioRepository) GetFeed(userID uuid.UUID, page, limit int) ([]domain.Portfolio, int64, error) {
	var portfolios []domain.Portfolio
	var total int64

	query := r.db.Model(&domain.Portfolio{}).
		Joins("JOIN follows ON portfolios.user_id = follows.following_id").
		Where("follows.follower_id = ? AND portfolios.status = 'published' AND portfolios.deleted_at IS NULL", userID)

	query.Count(&total)

	offset := (page - 1) * limit
	err := query.Preload("User.Kelas").Preload("Tags").
		Offset(offset).Limit(limit).
		Order("portfolios.published_at DESC").
		Find(&portfolios).Error

	return portfolios, total, err
}
