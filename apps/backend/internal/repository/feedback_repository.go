package repository

import (
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"gorm.io/gorm"
)

type FeedbackRepository struct {
	db *gorm.DB
}

func NewFeedbackRepository(db *gorm.DB) *FeedbackRepository {
	return &FeedbackRepository{db: db}
}

func (r *FeedbackRepository) Create(feedback *domain.Feedback) error {
	return r.db.Create(feedback).Error
}

func (r *FeedbackRepository) FindByID(id uuid.UUID) (*domain.Feedback, error) {
	var feedback domain.Feedback
	err := r.db.Preload("User").Where("id = ?", id).First(&feedback).Error
	if err != nil {
		return nil, err
	}
	return &feedback, nil
}

func (r *FeedbackRepository) Update(feedback *domain.Feedback) error {
	return r.db.Save(feedback).Error
}

func (r *FeedbackRepository) Delete(id uuid.UUID) error {
	return r.db.Where("id = ?", id).Delete(&domain.Feedback{}).Error
}

func (r *FeedbackRepository) List(status, kategori, search string, page, limit int) ([]domain.Feedback, int64, error) {
	var feedbacks []domain.Feedback
	var total int64

	query := r.db.Model(&domain.Feedback{})

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if kategori != "" {
		query = query.Where("kategori = ?", kategori)
	}

	if search != "" {
		query = query.Where("pesan ILIKE ?", "%"+search+"%")
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Fetch with pagination
	offset := (page - 1) * limit
	err := query.Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&feedbacks).Error

	if err != nil {
		return nil, 0, err
	}

	return feedbacks, total, nil
}

func (r *FeedbackRepository) GetStats() (total, pending, read, resolved int64, err error) {
	err = r.db.Model(&domain.Feedback{}).Count(&total).Error
	if err != nil {
		return
	}

	err = r.db.Model(&domain.Feedback{}).Where("status = ?", domain.FeedbackStatusPending).Count(&pending).Error
	if err != nil {
		return
	}

	err = r.db.Model(&domain.Feedback{}).Where("status = ?", domain.FeedbackStatusRead).Count(&read).Error
	if err != nil {
		return
	}

	err = r.db.Model(&domain.Feedback{}).Where("status = ?", domain.FeedbackStatusResolved).Count(&resolved).Error
	return
}
