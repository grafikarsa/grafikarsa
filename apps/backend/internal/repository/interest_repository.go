package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InterestRepository struct {
	db *gorm.DB
}

func NewInterestRepository(db *gorm.DB) *InterestRepository {
	return &InterestRepository{db: db}
}

// GetUserInterest fetches the interest profile for a user
// Returns nil if user has no interest profile yet
func (r *InterestRepository) GetUserInterest(userID uuid.UUID) (*domain.UserInterest, error) {
	var interest domain.UserInterest
	err := r.db.Where("user_id = ?", userID).First(&interest).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &interest, nil
}

// GetOrCreateUserInterest gets existing interest or creates a new empty one
func (r *InterestRepository) GetOrCreateUserInterest(userID uuid.UUID) (*domain.UserInterest, error) {
	interest, err := r.GetUserInterest(userID)
	if err != nil {
		return nil, err
	}
	if interest != nil {
		return interest, nil
	}

	// Create new empty interest profile
	newInterest := &domain.UserInterest{
		UserID:       userID,
		LikedTags:    domain.JSONB{},
		LikedJurusan: domain.JSONB{},
		TotalLikes:   0,
		UpdatedAt:    time.Now(),
	}

	err = r.db.Create(newInterest).Error
	if err != nil {
		return nil, err
	}
	return newInterest, nil
}

// UpdateTagInterest increments tag counters for the given tag IDs
func (r *InterestRepository) UpdateTagInterest(userID uuid.UUID, tagIDs []uuid.UUID) error {
	if len(tagIDs) == 0 {
		return nil
	}

	interest, err := r.GetOrCreateUserInterest(userID)
	if err != nil {
		return err
	}

	// Update tag counts
	likedTags := interest.LikedTags
	if likedTags == nil {
		likedTags = domain.JSONB{}
	}

	for _, tagID := range tagIDs {
		tagKey := tagID.String()
		currentCount := float64(0)
		if val, ok := likedTags[tagKey]; ok {
			if count, ok := val.(float64); ok {
				currentCount = count
			}
		}
		likedTags[tagKey] = currentCount + 1
	}

	// Update in database
	return r.db.Model(&domain.UserInterest{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"liked_tags": likedTags,
			"updated_at": time.Now(),
		}).Error
}

// UpdateJurusanInterest increments jurusan counter for the given jurusan ID
func (r *InterestRepository) UpdateJurusanInterest(userID uuid.UUID, jurusanID uuid.UUID) error {
	interest, err := r.GetOrCreateUserInterest(userID)
	if err != nil {
		return err
	}

	// Update jurusan count
	likedJurusan := interest.LikedJurusan
	if likedJurusan == nil {
		likedJurusan = domain.JSONB{}
	}

	jurusanKey := jurusanID.String()
	currentCount := float64(0)
	if val, ok := likedJurusan[jurusanKey]; ok {
		if count, ok := val.(float64); ok {
			currentCount = count
		}
	}
	likedJurusan[jurusanKey] = currentCount + 1

	// Update in database
	return r.db.Model(&domain.UserInterest{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"liked_jurusan": likedJurusan,
			"updated_at":    time.Now(),
		}).Error
}

// IncrementTotalLikes increments the total likes counter
func (r *InterestRepository) IncrementTotalLikes(userID uuid.UUID) error {
	return r.db.Model(&domain.UserInterest{}).
		Where("user_id = ?", userID).
		Update("total_likes", gorm.Expr("total_likes + 1")).Error
}

// DecrementTagInterest decrements tag counters (for unlike)
func (r *InterestRepository) DecrementTagInterest(userID uuid.UUID, tagIDs []uuid.UUID) error {
	if len(tagIDs) == 0 {
		return nil
	}

	interest, err := r.GetUserInterest(userID)
	if err != nil || interest == nil {
		return err
	}

	likedTags := interest.LikedTags
	if likedTags == nil {
		return nil
	}

	for _, tagID := range tagIDs {
		tagKey := tagID.String()
		if val, ok := likedTags[tagKey]; ok {
			if count, ok := val.(float64); ok && count > 0 {
				likedTags[tagKey] = count - 1
			}
		}
	}

	return r.db.Model(&domain.UserInterest{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"liked_tags": likedTags,
			"updated_at": time.Now(),
		}).Error
}

// DecrementJurusanInterest decrements jurusan counter (for unlike)
func (r *InterestRepository) DecrementJurusanInterest(userID uuid.UUID, jurusanID uuid.UUID) error {
	interest, err := r.GetUserInterest(userID)
	if err != nil || interest == nil {
		return err
	}

	likedJurusan := interest.LikedJurusan
	if likedJurusan == nil {
		return nil
	}

	jurusanKey := jurusanID.String()
	if val, ok := likedJurusan[jurusanKey]; ok {
		if count, ok := val.(float64); ok && count > 0 {
			likedJurusan[jurusanKey] = count - 1
		}
	}

	return r.db.Model(&domain.UserInterest{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"liked_jurusan": likedJurusan,
			"updated_at":    time.Now(),
		}).Error
}

// DecrementTotalLikes decrements the total likes counter (for unlike)
func (r *InterestRepository) DecrementTotalLikes(userID uuid.UUID) error {
	return r.db.Model(&domain.UserInterest{}).
		Where("user_id = ? AND total_likes > 0", userID).
		Update("total_likes", gorm.Expr("total_likes - 1")).Error
}

// GetTopTagInterests returns the top N tag IDs by interest count
func (r *InterestRepository) GetTopTagInterests(userID uuid.UUID, limit int) ([]uuid.UUID, error) {
	interest, err := r.GetUserInterest(userID)
	if err != nil || interest == nil {
		return nil, err
	}

	if interest.LikedTags == nil {
		return nil, nil
	}

	// Convert to sortable slice
	type tagCount struct {
		ID    uuid.UUID
		Count float64
	}

	var tags []tagCount
	for key, val := range interest.LikedTags {
		id, err := uuid.Parse(key)
		if err != nil {
			continue
		}
		count, ok := val.(float64)
		if !ok {
			continue
		}
		tags = append(tags, tagCount{ID: id, Count: count})
	}

	// Sort by count descending
	for i := 0; i < len(tags)-1; i++ {
		for j := i + 1; j < len(tags); j++ {
			if tags[j].Count > tags[i].Count {
				tags[i], tags[j] = tags[j], tags[i]
			}
		}
	}

	// Return top N
	result := make([]uuid.UUID, 0, limit)
	for i := 0; i < len(tags) && i < limit; i++ {
		result = append(result, tags[i].ID)
	}

	return result, nil
}

// SaveFeedPreference saves user's feed algorithm preference
func (r *InterestRepository) SaveFeedPreference(userID uuid.UUID, algorithm domain.FeedAlgorithm) error {
	pref := domain.UserFeedPreference{
		UserID:    userID,
		Algorithm: algorithm,
		UpdatedAt: time.Now(),
	}

	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"algorithm", "updated_at"}),
	}).Create(&pref).Error
}

// GetFeedPreference gets user's feed algorithm preference
func (r *InterestRepository) GetFeedPreference(userID uuid.UUID) (domain.FeedAlgorithm, error) {
	var pref domain.UserFeedPreference
	err := r.db.Where("user_id = ?", userID).First(&pref).Error
	if err == gorm.ErrRecordNotFound {
		return domain.FeedAlgorithmSmart, nil // Default to smart
	}
	if err != nil {
		return "", err
	}
	return pref.Algorithm, nil
}
