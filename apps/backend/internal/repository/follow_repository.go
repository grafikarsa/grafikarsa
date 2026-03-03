package repository

import (
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"gorm.io/gorm"
)

type FollowRepository struct {
	db *gorm.DB
}

func NewFollowRepository(db *gorm.DB) *FollowRepository {
	return &FollowRepository{db: db}
}

func (r *FollowRepository) Follow(followerID, followingID uuid.UUID) error {
	follow := domain.Follow{
		FollowerID:  followerID,
		FollowingID: followingID,
	}
	return r.db.Create(&follow).Error
}

func (r *FollowRepository) Unfollow(followerID, followingID uuid.UUID) error {
	return r.db.Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Delete(&domain.Follow{}).Error
}

func (r *FollowRepository) IsFollowing(followerID, followingID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.Follow{}).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Count(&count).Error
	return count > 0, err
}

func (r *FollowRepository) GetFollowers(userID uuid.UUID, search string, page, limit int) ([]domain.Follow, int64, error) {
	var follows []domain.Follow
	var total int64

	query := r.db.Model(&domain.Follow{}).Where("following_id = ?", userID)

	if search != "" {
		query = query.Joins("JOIN users ON follows.follower_id = users.id").
			Where("users.nama ILIKE ? OR users.username ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)

	offset := (page - 1) * limit
	err := query.Preload("Follower.Kelas").
		Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&follows).Error

	return follows, total, err
}

func (r *FollowRepository) GetFollowing(userID uuid.UUID, search string, page, limit int) ([]domain.Follow, int64, error) {
	var follows []domain.Follow
	var total int64

	query := r.db.Model(&domain.Follow{}).Where("follower_id = ?", userID)

	if search != "" {
		query = query.Joins("JOIN users ON follows.following_id = users.id").
			Where("users.nama ILIKE ? OR users.username ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	query.Count(&total)

	offset := (page - 1) * limit
	err := query.Preload("Following.Kelas").
		Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&follows).Error

	return follows, total, err
}

func (r *FollowRepository) GetFollowerCount(userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&domain.Follow{}).Where("following_id = ?", userID).Count(&count).Error
	return count, err
}

// IsMutualFollow checks if two users follow each other
func (r *FollowRepository) IsMutualFollow(userA, userB uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&domain.Follow{}).
		Where("(follower_id = ? AND following_id = ?) OR (follower_id = ? AND following_id = ?)",
			userA, userB, userB, userA).
		Count(&count).Error
	return count == 2, err
}
