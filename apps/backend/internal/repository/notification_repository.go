package repository

import (
	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"gorm.io/gorm"
)

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create creates a new notification
func (r *NotificationRepository) Create(notification *domain.Notification) error {
	return r.db.Create(notification).Error
}

// FindByID finds a notification by ID
func (r *NotificationRepository) FindByID(id uuid.UUID) (*domain.Notification, error) {
	var notification domain.Notification
	err := r.db.Where("id = ?", id).First(&notification).Error
	if err != nil {
		return nil, err
	}
	return &notification, nil
}

// FindByUserID finds notifications by user ID with pagination
func (r *NotificationRepository) FindByUserID(userID uuid.UUID, unreadOnly bool, page, limit int) ([]domain.Notification, int64, error) {
	var notifications []domain.Notification
	var total int64

	query := r.db.Model(&domain.Notification{}).Where("user_id = ?", userID)

	if unreadOnly {
		query = query.Where("is_read = ?", false)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	offset := (page - 1) * limit
	err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&notifications).Error
	if err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

// CountUnread counts unread notifications for a user
func (r *NotificationRepository) CountUnread(userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&domain.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

// MarkAsRead marks a notification as read
func (r *NotificationRepository) MarkAsRead(id uuid.UUID) error {
	return r.db.Model(&domain.Notification{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": gorm.Expr("NOW()"),
		}).Error
}

// MarkAllAsRead marks all notifications as read for a user
func (r *NotificationRepository) MarkAllAsRead(userID uuid.UUID) error {
	return r.db.Model(&domain.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{
			"is_read": true,
			"read_at": gorm.Expr("NOW()"),
		}).Error
}

// Delete deletes a notification
func (r *NotificationRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&domain.Notification{}, "id = ?", id).Error
}

// DeleteOld deletes notifications older than specified days
func (r *NotificationRepository) DeleteOld(days int) error {
	return r.db.Exec("DELETE FROM notifications WHERE created_at < NOW() - INTERVAL '? days'", days).Error
}
