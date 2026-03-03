package repository

import (
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"gorm.io/gorm"
)

type CommentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

func (r *CommentRepository) Create(comment *domain.Comment) error {
	return r.db.Create(comment).Error
}

func (r *CommentRepository) FindByID(id uuid.UUID) (*domain.Comment, error) {
	var comment domain.Comment
	err := r.db.Preload("User").Preload("User.Kelas").Preload("Portfolio").First(&comment, "id = ?", id).Error
	return &comment, err
}

func (r *CommentRepository) Update(comment *domain.Comment) error {
	return r.db.Save(comment).Error
}

func (r *CommentRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&domain.Comment{}, "id = ?", id).Error
}

func (r *CommentRepository) GetByPortfolioID(portfolioID uuid.UUID) ([]domain.Comment, error) {
	var comments []domain.Comment
	// Fetch all comments for the portfolio, ordered by creation time
	// We will reconstruct the tree structure in the service or frontend
	err := r.db.Preload("User").Preload("User.Kelas").
		Where("portfolio_id = ?", portfolioID).
		Order("created_at ASC").
		Find(&comments).Error
	return comments, err
}

func (r *CommentRepository) CountByPortfolioID(portfolioID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&domain.Comment{}).Where("portfolio_id = ?", portfolioID).Count(&count).Error
	return count, err
}
