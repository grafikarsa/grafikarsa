package dto

import (
	"time"

	"github.com/google/uuid"
)

// User List Item
type UserListDTO struct {
	ID         uuid.UUID   `json:"id"`
	Username   string      `json:"username"`
	Nama       string      `json:"nama"`
	AvatarURL  *string     `json:"avatar_url,omitempty"`
	BannerURL  *string     `json:"banner_url,omitempty"`
	Role       string      `json:"role"`
	TahunMasuk *int        `json:"tahun_masuk,omitempty"`
	TahunLulus *int        `json:"tahun_lulus,omitempty"`
	Kelas      *KelasDTO   `json:"kelas,omitempty"`
	Jurusan    *JurusanDTO `json:"jurusan,omitempty"`
}

// User Detail
type UserDetailDTO struct {
	ID             uuid.UUID            `json:"id"`
	Username       string               `json:"username"`
	Email          string               `json:"email"`
	Nama           string               `json:"nama"`
	Bio            *string              `json:"bio,omitempty"`
	AvatarURL      *string              `json:"avatar_url,omitempty"`
	BannerURL      *string              `json:"banner_url,omitempty"`
	Role           string               `json:"role"`
	IsActive       bool                 `json:"is_active"`
	TahunMasuk     *int                 `json:"tahun_masuk,omitempty"`
	TahunLulus     *int                 `json:"tahun_lulus,omitempty"`
	Kelas          *KelasDTO            `json:"kelas,omitempty"`
	Jurusan        *JurusanDTO          `json:"jurusan,omitempty"`
	ClassHistory   []ClassHistoryDTO    `json:"class_history,omitempty"`
	SocialLinks    []SocialLinkDTO      `json:"social_links,omitempty"`
	SpecialRoles   []UserSpecialRoleDTO `json:"special_roles,omitempty"`
	FollowerCount  int64                `json:"follower_count"`
	FollowingCount int64                `json:"following_count"`
	PortfolioCount int64                `json:"portfolio_count"`
	IsFollowing    bool                 `json:"is_following"`
	CreatedAt      time.Time            `json:"created_at"`
}

// UserSpecialRoleDTO for public profile special roles
type UserSpecialRoleDTO struct {
	ID    uuid.UUID `json:"id"`
	Nama  string    `json:"nama"`
	Color string    `json:"color"`
}

// Profile (Me)
type ProfileDTO struct {
	ID             uuid.UUID            `json:"id"`
	Username       string               `json:"username"`
	Email          string               `json:"email"`
	Nama           string               `json:"nama"`
	Bio            *string              `json:"bio,omitempty"`
	AvatarURL      *string              `json:"avatar_url,omitempty"`
	BannerURL      *string              `json:"banner_url,omitempty"`
	Role           string               `json:"role"`
	NISN           *string              `json:"nisn,omitempty"`
	NIS            *string              `json:"nis,omitempty"`
	TahunMasuk     *int                 `json:"tahun_masuk,omitempty"`
	TahunLulus     *int                 `json:"tahun_lulus,omitempty"`
	Kelas          *KelasDTO            `json:"kelas,omitempty"`
	Jurusan        *JurusanDTO          `json:"jurusan,omitempty"`
	SocialLinks    []SocialLinkDTO      `json:"social_links,omitempty"`
	FollowerCount  int64                `json:"follower_count"`
	FollowingCount int64                `json:"following_count"`
	SpecialRoles   []ProfileSpecialRole `json:"special_roles,omitempty"`
	Capabilities   []string             `json:"capabilities,omitempty"`
	CreatedAt      time.Time            `json:"created_at"`
}

// ProfileSpecialRole for user's special roles in profile
type ProfileSpecialRole struct {
	ID           uuid.UUID `json:"id"`
	Nama         string    `json:"nama"`
	Color        string    `json:"color"`
	Capabilities []string  `json:"capabilities"`
	IsActive     bool      `json:"is_active"`
}

type ClassHistoryDTO struct {
	KelasNama   string `json:"kelas_nama"`
	TahunAjaran int    `json:"tahun_ajaran"`
}

type SocialLinkDTO struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

// Update Profile
type UpdateProfileRequest struct {
	Nama     *string `json:"nama,omitempty"`
	Username *string `json:"username,omitempty"`
	Bio      *string `json:"bio,omitempty"`
	Email    *string `json:"email,omitempty"`
}

type UpdatePasswordRequest struct {
	CurrentPassword         string `json:"current_password" validate:"required"`
	NewPassword             string `json:"new_password" validate:"required,min=8"`
	NewPasswordConfirmation string `json:"new_password_confirmation" validate:"required"`
}

type UpdateSocialLinksRequest struct {
	SocialLinks []SocialLinkDTO `json:"social_links"`
}

type CheckUsernameResponse struct {
	Username  string `json:"username"`
	Available bool   `json:"available"`
}

// Follower/Following
type FollowerDTO struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	Nama        string    `json:"nama"`
	AvatarURL   *string   `json:"avatar_url,omitempty"`
	Role        string    `json:"role"`
	KelasNama   *string   `json:"kelas_nama,omitempty"`
	IsFollowing bool      `json:"is_following"`
	FollowedAt  time.Time `json:"followed_at"`
}

// Follow Response
type FollowResponse struct {
	IsFollowing   bool  `json:"is_following"`
	FollowerCount int64 `json:"follower_count"`
}
