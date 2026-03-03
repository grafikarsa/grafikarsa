package dto

import (
	"time"

	"github.com/google/uuid"
)

// Jurusan
type JurusanDTO struct {
	ID   uuid.UUID `json:"id"`
	Nama string    `json:"nama"`
	Kode string    `json:"kode,omitempty"`
}

type JurusanDetailDTO struct {
	ID        uuid.UUID `json:"id"`
	Nama      string    `json:"nama"`
	Kode      string    `json:"kode"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateJurusanRequest struct {
	Nama string `json:"nama" validate:"required"`
	Kode string `json:"kode" validate:"required"`
}

type UpdateJurusanRequest struct {
	Nama *string `json:"nama,omitempty"`
	Kode *string `json:"kode,omitempty"`
}

// Tahun Ajaran
type TahunAjaranDTO struct {
	ID             uuid.UUID `json:"id"`
	TahunMulai     int       `json:"tahun_mulai"`
	IsActive       bool      `json:"is_active"`
	PromotionMonth int       `json:"promotion_month"`
	PromotionDay   int       `json:"promotion_day"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateTahunAjaranRequest struct {
	TahunMulai     int  `json:"tahun_mulai" validate:"required"`
	IsActive       bool `json:"is_active"`
	PromotionMonth int  `json:"promotion_month"`
	PromotionDay   int  `json:"promotion_day"`
}

type UpdateTahunAjaranRequest struct {
	IsActive       *bool `json:"is_active,omitempty"`
	PromotionMonth *int  `json:"promotion_month,omitempty"`
	PromotionDay   *int  `json:"promotion_day,omitempty"`
}

// Kelas
type KelasDTO struct {
	ID   uuid.UUID `json:"id"`
	Nama string    `json:"nama"`
}

type KelasDetailDTO struct {
	ID           uuid.UUID       `json:"id"`
	Nama         string          `json:"nama"`
	Tingkat      int             `json:"tingkat"`
	Rombel       string          `json:"rombel"`
	TahunAjaran  *TahunAjaranDTO `json:"tahun_ajaran,omitempty"`
	Jurusan      *JurusanDTO     `json:"jurusan,omitempty"`
	StudentCount int64           `json:"student_count,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type CreateKelasRequest struct {
	TahunAjaranID uuid.UUID `json:"tahun_ajaran_id" validate:"required"`
	JurusanID     uuid.UUID `json:"jurusan_id" validate:"required"`
	Tingkat       int       `json:"tingkat" validate:"required"`
	Rombel        string    `json:"rombel" validate:"required"`
}

type UpdateKelasRequest struct {
	Rombel *string `json:"rombel,omitempty"`
}

// Tags
type TagDTO struct {
	ID   uuid.UUID `json:"id"`
	Nama string    `json:"nama"`
}

type TagDetailDTO struct {
	ID             uuid.UUID `json:"id"`
	Nama           string    `json:"nama"`
	PortfolioCount int64     `json:"portfolio_count,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateTagRequest struct {
	Nama string `json:"nama" validate:"required"`
}

type UpdateTagRequest struct {
	Nama *string `json:"nama,omitempty"`
}

// Admin User Management
type AdminUserDTO struct {
	ID          uuid.UUID   `json:"id"`
	Username    string      `json:"username"`
	Email       string      `json:"email"`
	Nama        string      `json:"nama"`
	AvatarURL   *string     `json:"avatar_url,omitempty"`
	Role        string      `json:"role"`
	NISN        *string     `json:"nisn,omitempty"`
	NIS         *string     `json:"nis,omitempty"`
	Kelas       *KelasDTO   `json:"kelas,omitempty"`
	Jurusan     *JurusanDTO `json:"jurusan,omitempty"`
	TahunMasuk  *int        `json:"tahun_masuk,omitempty"`
	TahunLulus  *int        `json:"tahun_lulus,omitempty"`
	IsActive    bool        `json:"is_active"`
	LastLoginAt *time.Time  `json:"last_login_at,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

type AdminUserDetailDTO struct {
	AdminUserDTO
	Bio          *string           `json:"bio,omitempty"`
	BannerURL    *string           `json:"banner_url,omitempty"`
	ClassHistory []ClassHistoryDTO `json:"class_history,omitempty"`
	SocialLinks  []SocialLinkDTO   `json:"social_links,omitempty"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type CreateUserRequest struct {
	Username   string     `json:"username" validate:"required"`
	Email      string     `json:"email" validate:"required,email"`
	Password   string     `json:"password" validate:"required,min=8"`
	Nama       string     `json:"nama" validate:"required"`
	Role       string     `json:"role"`
	NISN       *string    `json:"nisn,omitempty"`
	NIS        *string    `json:"nis,omitempty"`
	KelasID    *uuid.UUID `json:"kelas_id,omitempty"`
	TahunMasuk *int       `json:"tahun_masuk,omitempty"`
}

type UpdateUserRequest struct {
	Nama       *string    `json:"nama,omitempty"`
	Username   *string    `json:"username,omitempty"`
	Email      *string    `json:"email,omitempty"`
	Role       *string    `json:"role,omitempty"`
	NISN       *string    `json:"nisn,omitempty"`
	NIS        *string    `json:"nis,omitempty"`
	KelasID    *uuid.UUID `json:"kelas_id,omitempty"`
	TahunMasuk *int       `json:"tahun_masuk,omitempty"`
	TahunLulus *int       `json:"tahun_lulus,omitempty"`
	IsActive   *bool      `json:"is_active,omitempty"`
	AvatarURL  *string    `json:"avatar_url,omitempty"`
	BannerURL  *string    `json:"banner_url,omitempty"`
}

type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// Admin Portfolio
type AdminPortfolioDTO struct {
	ID           uuid.UUID         `json:"id"`
	Judul        string            `json:"judul"`
	Slug         string            `json:"slug"`
	ThumbnailURL *string           `json:"thumbnail_url,omitempty"`
	Status       string            `json:"status"`
	CreatedAt    time.Time         `json:"created_at"`
	User         *PortfolioUserDTO `json:"user,omitempty"`
}

type AdminPortfolioDetailDTO struct {
	PortfolioDetailDTO
	ReviewedBy *PortfolioUserDTO `json:"reviewed_by,omitempty"`
}

type ModeratePortfolioRequest struct {
	Note string `json:"note,omitempty"`
}

type AdminUpdatePortfolioRequest struct {
	Judul  *string     `json:"judul,omitempty"`
	Status *string     `json:"status,omitempty"`
	TagIDs []uuid.UUID `json:"tag_ids,omitempty"`
}

// Dashboard Stats
type DashboardStatsDTO struct {
	Users                   UserStatsDTO                `json:"users"`
	Portfolios              PortfolioStatsDTO           `json:"portfolios"`
	Jurusan                 CountDTO                    `json:"jurusan"`
	Kelas                   KelasStatsDTO               `json:"kelas"`
	RecentUsers             []RecentUserDTO             `json:"recent_users"`
	RecentPendingPortfolios []RecentPendingPortfolioDTO `json:"recent_pending_portfolios"`
}

type UserStatsDTO struct {
	Total        int64 `json:"total"`
	Students     int64 `json:"students"`
	Alumni       int64 `json:"alumni"`
	Admins       int64 `json:"admins"`
	NewThisMonth int64 `json:"new_this_month"`
}

type PortfolioStatsDTO struct {
	Total         int64 `json:"total"`
	Published     int64 `json:"published"`
	PendingReview int64 `json:"pending_review"`
	Draft         int64 `json:"draft"`
	Rejected      int64 `json:"rejected"`
	Archived      int64 `json:"archived"`
	NewThisMonth  int64 `json:"new_this_month"`
}

type CountDTO struct {
	Total int64 `json:"total"`
}

type KelasStatsDTO struct {
	Total             int64 `json:"total"`
	ActiveTahunAjaran int64 `json:"active_tahun_ajaran"`
}

type RecentUserDTO struct {
	ID        uuid.UUID `json:"id"`
	Username  string    `json:"username"`
	Nama      string    `json:"nama"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	Role      string    `json:"role"`
	KelasNama *string   `json:"kelas_nama,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type RecentPendingPortfolioDTO struct {
	ID            uuid.UUID `json:"id"`
	Judul         string    `json:"judul"`
	Slug          string    `json:"slug"`
	ThumbnailURL  *string   `json:"thumbnail_url,omitempty"`
	UserNama      string    `json:"user_nama"`
	UserUsername  string    `json:"user_username"`
	UserAvatarURL *string   `json:"user_avatar_url,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
