package domain

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a refresh token stored in the database
type RefreshToken struct {
	ID            uuid.UUID  `json:"id"`
	UserID        uuid.UUID  `json:"user_id"`
	TokenHash     string     `json:"-"` // Never expose hash
	FamilyID      uuid.UUID  `json:"family_id"`
	IsRevoked     bool       `json:"is_revoked"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	RevokedReason string     `json:"revoked_reason,omitempty"`
	ExpiresAt     time.Time  `json:"expires_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// RevokedReason constants
const (
	RevokedReasonLogout         = "logout"
	RevokedReasonLogoutAll      = "logout_all"
	RevokedReasonRotated        = "rotated"
	RevokedReasonSecurity       = "security"
	RevokedReasonPasswordChange = "password_change"
	RevokedReasonTokenReuse     = "token_reuse"
	RevokedReasonExpired        = "expired"
)

// IsExpired checks if the refresh token is expired
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}

// IsValid checks if the refresh token is valid (not revoked and not expired)
func (rt *RefreshToken) IsValid() bool {
	return !rt.IsRevoked && !rt.IsExpired()
}

// AuthSession represents an active user session
type AuthSession struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	RefreshTokenID uuid.UUID  `json:"refresh_token_id"`
	UserAgent      string     `json:"user_agent,omitempty"`
	IPAddress      string     `json:"ip_address,omitempty"`
	IsRevoked      bool       `json:"is_revoked"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt     time.Time  `json:"last_used_at"`
	CreatedAt      time.Time  `json:"created_at"`
	IsCurrent      bool       `json:"is_current,omitempty"` // Set dynamically
}

// SessionInfo represents session information for the API response
type SessionInfo struct {
	ID         uuid.UUID `json:"id"`
	UserAgent  string    `json:"user_agent"`
	IPAddress  string    `json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
	IsCurrent  bool      `json:"is_current"`
}

// ToSessionInfo converts AuthSession to SessionInfo for API response
func (s *AuthSession) ToSessionInfo() SessionInfo {
	return SessionInfo{
		ID:         s.ID,
		UserAgent:  s.UserAgent,
		IPAddress:  s.IPAddress,
		CreatedAt:  s.CreatedAt,
		LastUsedAt: s.LastUsedAt,
		IsCurrent:  s.IsCurrent,
	}
}

// LoginInfo contains information needed for login
type LoginInfo struct {
	Username  string
	Password  string
	UserAgent string
	IPAddress string
}

// TokenRotationResult contains the result of a token rotation
type TokenRotationResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64
	User         *User
}
