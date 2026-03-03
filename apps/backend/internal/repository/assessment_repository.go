package repository

import (
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"gorm.io/gorm"
)

type AssessmentRepository struct {
	db *gorm.DB
}

func NewAssessmentRepository(db *gorm.DB) *AssessmentRepository {
	return &AssessmentRepository{db: db}
}

// ============================================================================
// ASSESSMENT METRICS
// ============================================================================

func (r *AssessmentRepository) CreateMetric(metric *domain.AssessmentMetric) error {
	// Get max urutan
	var maxUrutan int
	r.db.Model(&domain.AssessmentMetric{}).
		Where("deleted_at IS NULL").
		Select("COALESCE(MAX(urutan), 0)").
		Scan(&maxUrutan)
	metric.Urutan = maxUrutan + 1
	return r.db.Create(metric).Error
}

func (r *AssessmentRepository) FindMetricByID(id uuid.UUID) (*domain.AssessmentMetric, error) {
	var metric domain.AssessmentMetric
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&metric).Error
	if err != nil {
		return nil, err
	}
	return &metric, nil
}

func (r *AssessmentRepository) UpdateMetric(metric *domain.AssessmentMetric) error {
	return r.db.Save(metric).Error
}

func (r *AssessmentRepository) DeleteMetric(id uuid.UUID) error {
	return r.db.Model(&domain.AssessmentMetric{}).
		Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *AssessmentRepository) ListMetrics(activeOnly bool) ([]domain.AssessmentMetric, error) {
	var metrics []domain.AssessmentMetric
	query := r.db.Where("deleted_at IS NULL")
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	err := query.Order("urutan ASC").Find(&metrics).Error
	return metrics, err
}

func (r *AssessmentRepository) ReorderMetrics(orders []struct {
	ID     uuid.UUID
	Urutan int
}) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, order := range orders {
			if err := tx.Model(&domain.AssessmentMetric{}).
				Where("id = ?", order.ID).
				Update("urutan", order.Urutan).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// ============================================================================
// PORTFOLIO ASSESSMENTS
// ============================================================================

func (r *AssessmentRepository) CreateAssessment(assessment *domain.PortfolioAssessment) error {
	return r.db.Create(assessment).Error
}

func (r *AssessmentRepository) FindAssessmentByPortfolioID(portfolioID uuid.UUID) (*domain.PortfolioAssessment, error) {
	var assessment domain.PortfolioAssessment
	err := r.db.
		Preload("Assessor").
		Preload("Scores").
		Preload("Scores.Metric").
		Where("portfolio_id = ?", portfolioID).
		First(&assessment).Error
	if err != nil {
		return nil, err
	}
	return &assessment, nil
}

func (r *AssessmentRepository) FindAssessmentByID(id uuid.UUID) (*domain.PortfolioAssessment, error) {
	var assessment domain.PortfolioAssessment
	err := r.db.
		Preload("Portfolio").
		Preload("Assessor").
		Preload("Scores").
		Preload("Scores.Metric").
		Where("id = ?", id).
		First(&assessment).Error
	if err != nil {
		return nil, err
	}
	return &assessment, nil
}

func (r *AssessmentRepository) UpdateAssessment(assessment *domain.PortfolioAssessment) error {
	return r.db.Save(assessment).Error
}

func (r *AssessmentRepository) DeleteAssessment(id uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete scores first
		if err := tx.Where("assessment_id = ?", id).Delete(&domain.PortfolioAssessmentScore{}).Error; err != nil {
			return err
		}
		// Delete assessment
		return tx.Where("id = ?", id).Delete(&domain.PortfolioAssessment{}).Error
	})
}

// ============================================================================
// ASSESSMENT SCORES
// ============================================================================

func (r *AssessmentRepository) CreateScores(scores []domain.PortfolioAssessmentScore) error {
	return r.db.Create(&scores).Error
}

func (r *AssessmentRepository) DeleteScoresByAssessmentID(assessmentID uuid.UUID) error {
	return r.db.Where("assessment_id = ?", assessmentID).Delete(&domain.PortfolioAssessmentScore{}).Error
}

func (r *AssessmentRepository) ReplaceScores(assessmentID uuid.UUID, scores []domain.PortfolioAssessmentScore) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Delete existing scores
		if err := tx.Where("assessment_id = ?", assessmentID).Delete(&domain.PortfolioAssessmentScore{}).Error; err != nil {
			return err
		}
		// Create new scores
		if len(scores) > 0 {
			return tx.Create(&scores).Error
		}
		return nil
	})
}

// ============================================================================
// ASSESSMENT STATS
// ============================================================================

type AssessmentStats struct {
	TotalPublished int64 `json:"total_published"`
	Assessed       int64 `json:"assessed"`
	Pending        int64 `json:"pending"`
}

func (r *AssessmentRepository) GetAssessmentStats() (*AssessmentStats, error) {
	var stats AssessmentStats

	// Count total published portfolios
	if err := r.db.Model(&domain.Portfolio{}).
		Where("status = ? AND deleted_at IS NULL", domain.StatusPublished).
		Count(&stats.TotalPublished).Error; err != nil {
		return nil, err
	}

	// Count assessed portfolios
	if err := r.db.Model(&domain.Portfolio{}).
		Where("status = ? AND deleted_at IS NULL", domain.StatusPublished).
		Where("EXISTS (SELECT 1 FROM portfolio_assessments pa WHERE pa.portfolio_id = portfolios.id)").
		Count(&stats.Assessed).Error; err != nil {
		return nil, err
	}

	stats.Pending = stats.TotalPublished - stats.Assessed

	return &stats, nil
}

// ============================================================================
// PORTFOLIO LIST FOR ASSESSMENT
// ============================================================================

type PortfolioWithAssessment struct {
	domain.Portfolio
	Assessment *domain.PortfolioAssessment
}

func (r *AssessmentRepository) ListPublishedPortfolios(filter string, search string, page, limit int) ([]PortfolioWithAssessment, int64, error) {
	var results []PortfolioWithAssessment
	var total int64

	query := r.db.Model(&domain.Portfolio{}).
		Where("portfolios.status = ? AND portfolios.deleted_at IS NULL", domain.StatusPublished)

	// Filter: pending (no assessment), assessed (has assessment), all
	switch filter {
	case "pending":
		query = query.Where("NOT EXISTS (SELECT 1 FROM portfolio_assessments pa WHERE pa.portfolio_id = portfolios.id)")
	case "assessed":
		query = query.Where("EXISTS (SELECT 1 FROM portfolio_assessments pa WHERE pa.portfolio_id = portfolios.id)")
	}

	// Search by title or user name
	if search != "" {
		query = query.Joins("LEFT JOIN users ON users.id = portfolios.user_id").
			Where("portfolios.judul ILIKE ? OR users.nama ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch portfolios
	var portfolios []domain.Portfolio
	offset := (page - 1) * limit
	err := query.
		Preload("User").
		Order("portfolios.published_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&portfolios).Error
	if err != nil {
		return nil, 0, err
	}

	// Fetch assessments for these portfolios
	portfolioIDs := make([]uuid.UUID, len(portfolios))
	for i, p := range portfolios {
		portfolioIDs[i] = p.ID
	}

	var assessments []domain.PortfolioAssessment
	if len(portfolioIDs) > 0 {
		r.db.Preload("Assessor").
			Where("portfolio_id IN ?", portfolioIDs).
			Find(&assessments)
	}

	// Map assessments to portfolios
	assessmentMap := make(map[uuid.UUID]*domain.PortfolioAssessment)
	for i := range assessments {
		assessmentMap[assessments[i].PortfolioID] = &assessments[i]
	}

	for _, p := range portfolios {
		results = append(results, PortfolioWithAssessment{
			Portfolio:  p,
			Assessment: assessmentMap[p.ID],
		})
	}

	return results, total, nil
}
