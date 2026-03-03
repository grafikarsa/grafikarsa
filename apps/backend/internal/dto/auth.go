package dto

import "github.com/google/uuid"

// Login
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
	User        UserBriefDTO `json:"user"`
}

type UserBriefDTO struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Nama      string    `json:"nama"`
	Role      string    `json:"role"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
}

// Refresh Token
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// Session
type SessionDTO struct {
	ID         uuid.UUID              `json:"id"`
	DeviceInfo map[string]interface{} `json:"device_info,omitempty"`
	IPAddress  *string                `json:"ip_address,omitempty"`
	CreatedAt  string                 `json:"created_at"`
	LastUsedAt *string                `json:"last_used_at,omitempty"`
	IsCurrent  bool                   `json:"is_current"`
}

type LogoutAllResponse struct {
	SessionsTerminated int `json:"sessions_terminated"`
}
