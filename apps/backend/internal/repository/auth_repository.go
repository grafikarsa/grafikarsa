package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/grafikarsa/backend/internal/domain"
	"gorm.io/gorm"
)

type AuthRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) CreateRefreshToken(token *domain.RefreshToken) error {
	return r.db.Create(token).Error
}

func (r *AuthRepository) FindRefreshTokenByHash(hash string) (*domain.RefreshToken, error) {
	var token domain.RefreshToken
	err := r.db.Where("token_hash = ?", hash).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *AuthRepository) FindRefreshTokenByID(id uuid.UUID) (*domain.RefreshToken, error) {
	var token domain.RefreshToken
	err := r.db.Where("id = ?", id).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *AuthRepository) RevokeRefreshToken(id uuid.UUID, reason string) error {
	now := time.Now()
	return r.db.Model(&domain.RefreshToken{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"is_revoked":     true,
			"revoked_at":     now,
			"revoked_reason": reason,
		}).Error
}

func (r *AuthRepository) RevokeTokenFamily(familyID uuid.UUID, reason string) error {
	now := time.Now()
	return r.db.Model(&domain.RefreshToken{}).
		Where("family_id = ?", familyID).
		Updates(map[string]interface{}{
			"is_revoked":     true,
			"revoked_at":     now,
			"revoked_reason": reason,
		}).Error
}

func (r *AuthRepository) RevokeAllUserTokens(userID uuid.UUID, reason string) (int64, error) {
	now := time.Now()
	result := r.db.Model(&domain.RefreshToken{}).
		Where("user_id = ? AND is_revoked = false", userID).
		Updates(map[string]interface{}{
			"is_revoked":     true,
			"revoked_at":     now,
			"revoked_reason": reason,
		})
	return result.RowsAffected, result.Error
}

func (r *AuthRepository) UpdateLastUsed(id uuid.UUID) error {
	return r.db.Model(&domain.RefreshToken{}).
		Where("id = ?", id).
		Update("last_used_at", time.Now()).Error
}

func (r *AuthRepository) GetUserSessions(userID uuid.UUID) ([]domain.RefreshToken, error) {
	var tokens []domain.RefreshToken
	err := r.db.Where("user_id = ? AND is_revoked = false AND expires_at > ?", userID, time.Now()).
		Order("created_at DESC").
		Find(&tokens).Error
	return tokens, err
}

func (r *AuthRepository) BlacklistToken(jti string, userID *uuid.UUID, expiresAt time.Time, reason string) error {
	blacklist := domain.TokenBlacklist{
		JTI:       jti,
		UserID:    userID,
		ExpiresAt: expiresAt,
		Reason:    &reason,
	}
	return r.db.Create(&blacklist).Error
}

func (r *AuthRepository) IsTokenBlacklisted(jti string) (bool, error) {
	var count int64
	err := r.db.Model(&domain.TokenBlacklist{}).Where("jti = ?", jti).Count(&count).Error
	return count > 0, err
}

func (r *AuthRepository) CleanupExpiredTokens() error {
	now := time.Now()
	if err := r.db.Where("expires_at < ?", now).Delete(&domain.RefreshToken{}).Error; err != nil {
		return err
	}
	return r.db.Where("expires_at < ?", now).Delete(&domain.TokenBlacklist{}).Error
}
