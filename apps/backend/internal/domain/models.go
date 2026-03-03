package domain

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Enum types
type UserRole string

const (
	RoleStudent UserRole = "student"
	RoleAlumni  UserRole = "alumni"
	RoleAdmin   UserRole = "admin"
)

type PortfolioStatus string

const (
	StatusDraft         PortfolioStatus = "draft"
	StatusPendingReview PortfolioStatus = "pending_review"
	StatusRejected      PortfolioStatus = "rejected"
	StatusPublished     PortfolioStatus = "published"
	StatusArchived      PortfolioStatus = "archived"
)

type ContentBlockType string

const (
	BlockText    ContentBlockType = "text"
	BlockImage   ContentBlockType = "image"
	BlockTable   ContentBlockType = "table"
	BlockYoutube ContentBlockType = "youtube"
	BlockButton  ContentBlockType = "button"
	BlockEmbed   ContentBlockType = "embed"
	// New block types for rich embeds
	BlockFigma ContentBlockType = "figma"
	BlockCanva ContentBlockType = "canva"
	BlockPPT   ContentBlockType = "ppt"
	BlockPDF   ContentBlockType = "pdf"
	BlockDoc   ContentBlockType = "doc"
)

type SocialPlatform string

const (
	PlatformFacebook        SocialPlatform = "facebook"
	PlatformInstagram       SocialPlatform = "instagram"
	PlatformGithub          SocialPlatform = "github"
	PlatformLinkedin        SocialPlatform = "linkedin"
	PlatformTwitter         SocialPlatform = "twitter"
	PlatformPersonalWebsite SocialPlatform = "personal_website"
	PlatformTiktok          SocialPlatform = "tiktok"
	PlatformYoutube         SocialPlatform = "youtube"
	PlatformBehance         SocialPlatform = "behance"
	PlatformDribbble        SocialPlatform = "dribbble"
	PlatformThreads         SocialPlatform = "threads"
	PlatformBluesky         SocialPlatform = "bluesky"
	PlatformMedium          SocialPlatform = "medium"
	PlatformGitlab          SocialPlatform = "gitlab"
)

// JSONB type for GORM
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Base model with soft delete
type BaseModel struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt *time.Time `gorm:"index" json:"-"`
}

// Jurusan (Department/Major)
type Jurusan struct {
	BaseModel
	Nama string `gorm:"type:varchar(100);not null" json:"nama"`
	Kode string `gorm:"type:varchar(10);not null;uniqueIndex" json:"kode"`
}

func (Jurusan) TableName() string { return "jurusan" }

// TahunAjaran (Academic Year)
type TahunAjaran struct {
	BaseModel
	TahunMulai     int  `gorm:"not null;uniqueIndex" json:"tahun_mulai"`
	IsActive       bool `gorm:"not null;default:false" json:"is_active"`
	PromotionMonth int  `gorm:"type:smallint;not null;default:7" json:"promotion_month"`
	PromotionDay   int  `gorm:"type:smallint;not null;default:1" json:"promotion_day"`
}

func (TahunAjaran) TableName() string { return "tahun_ajaran" }

// Kelas (Class)
type Kelas struct {
	BaseModel
	TahunAjaranID uuid.UUID    `gorm:"type:uuid;not null" json:"tahun_ajaran_id"`
	JurusanID     uuid.UUID    `gorm:"type:uuid;not null" json:"jurusan_id"`
	Tingkat       int          `gorm:"type:smallint;not null" json:"tingkat"`
	Rombel        string       `gorm:"type:char(1);not null" json:"rombel"`
	Nama          string       `gorm:"type:varchar(20);not null" json:"nama"`
	TahunAjaran   *TahunAjaran `gorm:"foreignKey:TahunAjaranID" json:"tahun_ajaran,omitempty"`
	Jurusan       *Jurusan     `gorm:"foreignKey:JurusanID" json:"jurusan,omitempty"`
}

func (Kelas) TableName() string { return "kelas" }

// Tags
type Tag struct {
	BaseModel
	Nama string `gorm:"type:varchar(50);not null;uniqueIndex" json:"nama"`
}

func (Tag) TableName() string { return "tags" }

// Series - Template portofolio dengan block konten yang sudah ditentukan
type Series struct {
	BaseModel
	Nama      string        `gorm:"type:varchar(100);not null;uniqueIndex" json:"nama"`
	Deskripsi *string       `gorm:"type:text" json:"deskripsi,omitempty"`
	IsActive  bool          `gorm:"not null;default:true" json:"is_active"`
	Blocks    []SeriesBlock `gorm:"foreignKey:SeriesID" json:"blocks,omitempty"`
}

func (Series) TableName() string { return "series" }

// SeriesBlock - Template block konten untuk series
type SeriesBlock struct {
	ID         uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	SeriesID   uuid.UUID        `gorm:"type:uuid;not null" json:"series_id"`
	BlockType  ContentBlockType `gorm:"type:content_block_type;not null" json:"block_type"`
	BlockOrder int              `gorm:"not null" json:"block_order"`
	Instruksi  string           `gorm:"type:text;not null" json:"instruksi"`
	CreatedAt  time.Time        `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt  time.Time        `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (SeriesBlock) TableName() string { return "series_blocks" }

// User
type User struct {
	BaseModel
	Username     string           `gorm:"type:varchar(30);not null;uniqueIndex" json:"username"`
	Email        string           `gorm:"type:varchar(255);not null;uniqueIndex" json:"email"`
	PasswordHash string           `gorm:"type:varchar(255);not null" json:"-"`
	Nama         string           `gorm:"type:varchar(100);not null" json:"nama"`
	Bio          *string          `gorm:"type:text" json:"bio,omitempty"`
	AvatarURL    *string          `gorm:"type:text" json:"avatar_url,omitempty"`
	BannerURL    *string          `gorm:"type:text" json:"banner_url,omitempty"`
	Role         UserRole         `gorm:"type:user_role;not null;default:'student'" json:"role"`
	NISN         *string          `gorm:"type:varchar(20)" json:"nisn,omitempty"`
	NIS          *string          `gorm:"type:varchar(30)" json:"nis,omitempty"`
	KelasID      *uuid.UUID       `gorm:"type:uuid" json:"kelas_id,omitempty"`
	TahunMasuk   *int             `gorm:"type:integer" json:"tahun_masuk,omitempty"`
	TahunLulus   *int             `gorm:"type:integer" json:"tahun_lulus,omitempty"`
	IsActive     bool             `gorm:"not null;default:true" json:"is_active"`
	LastLoginAt  *time.Time       `json:"last_login_at,omitempty"`
	Kelas        *Kelas           `gorm:"foreignKey:KelasID" json:"kelas,omitempty"`
	SocialLinks  []UserSocialLink `gorm:"foreignKey:UserID" json:"social_links,omitempty"`
}

func (User) TableName() string { return "users" }

// UserSocialLink
type UserSocialLink struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	Platform  SocialPlatform `gorm:"type:social_platform;not null" json:"platform"`
	URL       string         `gorm:"type:text;not null" json:"url"`
	CreatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (UserSocialLink) TableName() string { return "user_social_links" }

// StudentClassHistory
type StudentClassHistory struct {
	ID            uuid.UUID    `gorm:"type:uuid;primaryKey" json:"id"`
	UserID        uuid.UUID    `gorm:"type:uuid;not null" json:"user_id"`
	KelasID       uuid.UUID    `gorm:"type:uuid;not null" json:"kelas_id"`
	TahunAjaranID uuid.UUID    `gorm:"type:uuid;not null" json:"tahun_ajaran_id"`
	IsCurrent     bool         `gorm:"not null;default:false" json:"is_current"`
	CreatedAt     time.Time    `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	Kelas         *Kelas       `gorm:"foreignKey:KelasID" json:"kelas,omitempty"`
	TahunAjaran   *TahunAjaran `gorm:"foreignKey:TahunAjaranID" json:"tahun_ajaran,omitempty"`
}

func (StudentClassHistory) TableName() string { return "student_class_history" }

// RefreshToken
type RefreshToken struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID        uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	TokenHash     string     `gorm:"type:varchar(64);not null;uniqueIndex" json:"-"`
	FamilyID      uuid.UUID  `gorm:"type:uuid;not null" json:"family_id"`
	DeviceInfo    JSONB      `gorm:"type:jsonb" json:"device_info,omitempty"`
	IPAddress     *string    `gorm:"type:inet" json:"ip_address,omitempty"`
	IsRevoked     bool       `gorm:"not null;default:false" json:"is_revoked"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	RevokedReason *string    `gorm:"type:varchar(100)" json:"revoked_reason,omitempty"`
	ExpiresAt     time.Time  `gorm:"not null" json:"expires_at"`
	CreatedAt     time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`
	User          *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

// TokenBlacklist
type TokenBlacklist struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	JTI           string     `gorm:"type:varchar(64);not null;uniqueIndex" json:"jti"`
	UserID        *uuid.UUID `gorm:"type:uuid" json:"user_id,omitempty"`
	ExpiresAt     time.Time  `gorm:"not null" json:"expires_at"`
	BlacklistedAt time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"blacklisted_at"`
	Reason        *string    `gorm:"type:varchar(100)" json:"reason,omitempty"`
}

func (TokenBlacklist) TableName() string { return "token_blacklist" }

// Follow
type Follow struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	FollowerID  uuid.UUID `gorm:"type:uuid;not null" json:"follower_id"`
	FollowingID uuid.UUID `gorm:"type:uuid;not null" json:"following_id"`
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	Follower    *User     `gorm:"foreignKey:FollowerID" json:"follower,omitempty"`
	Following   *User     `gorm:"foreignKey:FollowingID" json:"following,omitempty"`
}

func (Follow) TableName() string { return "follows" }

// Portfolio
type Portfolio struct {
	BaseModel
	UserID          uuid.UUID       `gorm:"type:uuid;not null" json:"user_id"`
	Judul           string          `gorm:"type:varchar(200);not null" json:"judul"`
	Slug            string          `gorm:"type:varchar(250);not null" json:"slug"`
	ThumbnailURL    *string         `gorm:"type:text" json:"thumbnail_url,omitempty"`
	Status          PortfolioStatus `gorm:"type:portfolio_status;not null;default:'draft'" json:"status"`
	AdminReviewNote *string         `gorm:"type:text" json:"admin_review_note,omitempty"`
	ReviewedBy      *uuid.UUID      `gorm:"type:uuid" json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time      `json:"reviewed_at,omitempty"`
	PublishedAt     *time.Time      `json:"published_at,omitempty"`
	SeriesID        *uuid.UUID      `gorm:"type:uuid" json:"series_id,omitempty"`
	User            *User           `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Reviewer        *User           `gorm:"foreignKey:ReviewedBy" json:"reviewer,omitempty"`
	Series          *Series         `gorm:"foreignKey:SeriesID" json:"series,omitempty"`
	Tags            []Tag           `gorm:"many2many:portfolio_tags" json:"tags,omitempty"`
	ContentBlocks   []ContentBlock  `gorm:"foreignKey:PortfolioID" json:"content_blocks,omitempty"`
}

func (Portfolio) TableName() string { return "portfolios" }

// PortfolioTag (junction table)
type PortfolioTag struct {
	PortfolioID uuid.UUID `gorm:"type:uuid;primaryKey" json:"portfolio_id"`
	TagID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"tag_id"`
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (PortfolioTag) TableName() string { return "portfolio_tags" }

// ContentBlock
type ContentBlock struct {
	ID          uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	PortfolioID uuid.UUID        `gorm:"type:uuid;not null" json:"portfolio_id"`
	BlockType   ContentBlockType `gorm:"type:content_block_type;not null" json:"block_type"`
	BlockOrder  int              `gorm:"not null" json:"block_order"`
	Payload     JSONB            `gorm:"type:jsonb;not null;default:'{}'" json:"payload"`
	CreatedAt   time.Time        `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt   time.Time        `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (ContentBlock) TableName() string { return "content_blocks" }

// PortfolioLike
type PortfolioLike struct {
	UserID      uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	PortfolioID uuid.UUID `gorm:"type:uuid;primaryKey" json:"portfolio_id"`
	CreatedAt   time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (PortfolioLike) TableName() string { return "portfolio_likes" }

// AppSetting
type AppSetting struct {
	Key         string     `gorm:"type:varchar(100);primaryKey" json:"key"`
	Value       JSONB      `gorm:"type:jsonb;not null" json:"value"`
	Description *string    `gorm:"type:text" json:"description,omitempty"`
	UpdatedAt   time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	UpdatedBy   *uuid.UUID `gorm:"type:uuid" json:"updated_by,omitempty"`
}

func (AppSetting) TableName() string { return "app_settings" }

// Feedback enums
type FeedbackKategori string

const (
	FeedbackKategoriBug     FeedbackKategori = "bug"
	FeedbackKategoriSaran   FeedbackKategori = "saran"
	FeedbackKategoriLainnya FeedbackKategori = "lainnya"
)

type FeedbackStatus string

const (
	FeedbackStatusPending  FeedbackStatus = "pending"
	FeedbackStatusRead     FeedbackStatus = "read"
	FeedbackStatusResolved FeedbackStatus = "resolved"
)

// Feedback
type Feedback struct {
	ID         uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	UserID     *uuid.UUID       `gorm:"type:uuid" json:"user_id,omitempty"`
	Kategori   FeedbackKategori `gorm:"type:feedback_kategori;not null" json:"kategori"`
	Pesan      string           `gorm:"type:text;not null" json:"pesan"`
	Status     FeedbackStatus   `gorm:"type:feedback_status;not null;default:'pending'" json:"status"`
	AdminNotes *string          `gorm:"type:text" json:"admin_notes,omitempty"`
	ResolvedBy *uuid.UUID       `gorm:"type:uuid" json:"resolved_by,omitempty"`
	ResolvedAt *time.Time       `json:"resolved_at,omitempty"`
	CreatedAt  time.Time        `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt  time.Time        `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	User       *User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Resolver   *User            `gorm:"foreignKey:ResolvedBy" json:"resolver,omitempty"`
}

func (Feedback) TableName() string { return "feedback" }

// ============================================================================
// ASSESSMENT MODELS
// ============================================================================

// AssessmentMetric - Master data metrik penilaian
type AssessmentMetric struct {
	BaseModel
	Nama      string  `gorm:"type:varchar(100);not null" json:"nama"`
	Deskripsi *string `gorm:"type:text" json:"deskripsi,omitempty"`
	Urutan    int     `gorm:"not null;default:0" json:"urutan"`
	IsActive  bool    `gorm:"not null;default:true" json:"is_active"`
}

func (AssessmentMetric) TableName() string { return "assessment_metrics" }

// PortfolioAssessment - Header penilaian portfolio
type PortfolioAssessment struct {
	ID           uuid.UUID                  `gorm:"type:uuid;primaryKey" json:"id"`
	PortfolioID  uuid.UUID                  `gorm:"type:uuid;not null;uniqueIndex" json:"portfolio_id"`
	AssessedBy   uuid.UUID                  `gorm:"type:uuid;not null" json:"assessed_by"`
	FinalComment *string                    `gorm:"type:text" json:"final_comment,omitempty"`
	TotalScore   *float64                   `gorm:"type:decimal(4,2)" json:"total_score,omitempty"`
	CreatedAt    time.Time                  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt    time.Time                  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	Portfolio    *Portfolio                 `gorm:"foreignKey:PortfolioID" json:"portfolio,omitempty"`
	Assessor     *User                      `gorm:"foreignKey:AssessedBy" json:"assessor,omitempty"`
	Scores       []PortfolioAssessmentScore `gorm:"foreignKey:AssessmentID" json:"scores,omitempty"`
}

func (PortfolioAssessment) TableName() string { return "portfolio_assessments" }

// PortfolioAssessmentScore - Detail nilai per metrik
type PortfolioAssessmentScore struct {
	ID           uuid.UUID         `gorm:"type:uuid;primaryKey" json:"id"`
	AssessmentID uuid.UUID         `gorm:"type:uuid;not null" json:"assessment_id"`
	MetricID     uuid.UUID         `gorm:"type:uuid;not null" json:"metric_id"`
	Score        int               `gorm:"type:smallint;not null" json:"score"`
	Comment      *string           `gorm:"type:text" json:"comment,omitempty"`
	CreatedAt    time.Time         `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt    time.Time         `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	Metric       *AssessmentMetric `gorm:"foreignKey:MetricID" json:"metric,omitempty"`
}

func (PortfolioAssessmentScore) TableName() string { return "portfolio_assessment_scores" }

// ============================================================================
// NOTIFICATION MODELS
// ============================================================================

// NotificationType enum
type NotificationType string

const (
	NotifNewFollower       NotificationType = "new_follower"
	NotifPortfolioLiked    NotificationType = "portfolio_liked"
	NotifPortfolioApproved NotificationType = "portfolio_approved"
	NotifPortfolioRejected NotificationType = "portfolio_rejected"
	NotifFeedbackUpdated   NotificationType = "feedback_updated"
	NotifNewComment        NotificationType = "new_comment"
	NotifReplyComment      NotificationType = "reply_comment"
)

// Comment
type Comment struct {
	BaseModel
	PortfolioID uuid.UUID  `gorm:"type:uuid;not null" json:"portfolio_id"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null" json:"user_id"`
	ParentID    *uuid.UUID `gorm:"type:uuid" json:"parent_id,omitempty"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	IsEdited    bool       `gorm:"default:false" json:"is_edited"`
	User        *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Portfolio   *Portfolio `gorm:"foreignKey:PortfolioID" json:"portfolio,omitempty"`
	Parent      *Comment   `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Children    []Comment  `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

func (Comment) TableName() string { return "comments" }

// Notification
type Notification struct {
	ID        uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	UserID    uuid.UUID        `gorm:"type:uuid;not null" json:"user_id"`
	Type      NotificationType `gorm:"type:notification_type;not null" json:"type"`
	Title     string           `gorm:"type:varchar(255);not null" json:"title"`
	Message   *string          `gorm:"type:text" json:"message,omitempty"`
	Data      JSONB            `gorm:"type:jsonb;default:'{}'" json:"data,omitempty"`
	IsRead    bool             `gorm:"default:false" json:"is_read"`
	ReadAt    *time.Time       `json:"read_at,omitempty"`
	CreatedAt time.Time        `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	User      *User            `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (Notification) TableName() string { return "notifications" }

// ============================================================================
// SPECIAL ROLE MODELS
// ============================================================================

// StringArray type for PostgreSQL text[] array
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return "{}", nil
	}
	return "{" + stringArrayJoin(a) + "}", nil
}

func stringArrayJoin(arr []string) string {
	result := ""
	for i, s := range arr {
		if i > 0 {
			result += ","
		}
		result += "\"" + s + "\""
	}
	return result
}

func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = []string{}
		return nil
	}

	var str string
	switch v := value.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		*a = []string{}
		return nil
	}

	// Parse PostgreSQL array format: {item1,item2,item3}
	if str == "{}" || str == "" {
		*a = []string{}
		return nil
	}
	str = str[1 : len(str)-1] // Remove { and }
	*a = parsePostgresArray(str)
	return nil
}

func parsePostgresArray(s string) []string {
	if s == "" {
		return []string{}
	}
	var result []string
	var current string
	inQuote := false
	for _, c := range s {
		switch c {
		case '"':
			inQuote = !inQuote
		case ',':
			if !inQuote {
				result = append(result, current)
				current = ""
			} else {
				current += string(c)
			}
		default:
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// SpecialRole - Custom admin role dengan capabilities tertentu
type SpecialRole struct {
	BaseModel
	Nama         string      `gorm:"type:varchar(100);not null;uniqueIndex" json:"nama"`
	Description  *string     `gorm:"type:text" json:"description,omitempty"`
	Color        string      `gorm:"type:varchar(7);not null;default:'#6366f1'" json:"color"`
	Capabilities StringArray `gorm:"type:text[]" json:"capabilities"`
	IsActive     bool        `gorm:"not null;default:true" json:"is_active"`
}

func (SpecialRole) TableName() string { return "special_roles" }

// UserSpecialRole - Junction table untuk user dan special roles
type UserSpecialRole struct {
	UserID        uuid.UUID    `gorm:"type:uuid;primaryKey" json:"user_id"`
	SpecialRoleID uuid.UUID    `gorm:"type:uuid;primaryKey" json:"special_role_id"`
	AssignedBy    *uuid.UUID   `gorm:"type:uuid" json:"assigned_by,omitempty"`
	AssignedAt    time.Time    `gorm:"not null;default:CURRENT_TIMESTAMP" json:"assigned_at"`
	User          *User        `gorm:"foreignKey:UserID" json:"user,omitempty"`
	SpecialRole   *SpecialRole `gorm:"foreignKey:SpecialRoleID" json:"special_role,omitempty"`
	Assigner      *User        `gorm:"foreignKey:AssignedBy" json:"assigner,omitempty"`
}

func (UserSpecialRole) TableName() string { return "user_special_roles" }

// ============================================================================
// SMART FEED ALGORITHM MODELS
// ============================================================================

// FeedAlgorithm type untuk pilihan algoritma feed
type FeedAlgorithm string

const (
	FeedAlgorithmSmart     FeedAlgorithm = "smart"
	FeedAlgorithmRecent    FeedAlgorithm = "recent"
	FeedAlgorithmFollowing FeedAlgorithm = "following"
)

// PortfolioView - Tracking view portfolio untuk feed algorithm
type PortfolioView struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	PortfolioID uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex:idx_pv_user;uniqueIndex:idx_pv_session" json:"portfolio_id"`
	UserID      *uuid.UUID `gorm:"type:uuid;uniqueIndex:idx_pv_user" json:"user_id,omitempty"`
	SessionID   *string    `gorm:"type:varchar(64);uniqueIndex:idx_pv_session" json:"session_id,omitempty"`
	ViewedAt    time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"viewed_at"`
	Portfolio   *Portfolio `gorm:"foreignKey:PortfolioID" json:"portfolio,omitempty"`
	User        *User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (PortfolioView) TableName() string { return "portfolio_views" }

// UserInterest - Profil interest user dari aktivitas like
type UserInterest struct {
	UserID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	LikedTags    JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"liked_tags"`
	LikedJurusan JSONB     `gorm:"type:jsonb;not null;default:'{}'" json:"liked_jurusan"`
	TotalLikes   int       `gorm:"not null;default:0" json:"total_likes"`
	UpdatedAt    time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	User         *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (UserInterest) TableName() string { return "user_interests" }

// UserFeedPreference - Preferensi algoritma feed per user
type UserFeedPreference struct {
	UserID    uuid.UUID     `gorm:"type:uuid;primaryKey" json:"user_id"`
	Algorithm FeedAlgorithm `gorm:"type:varchar(20);not null;default:'smart'" json:"algorithm"`
	UpdatedAt time.Time     `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	User      *User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (UserFeedPreference) TableName() string { return "user_feed_preferences" }

// ============================================================================
// FEED SERVICE TYPES (non-persisted)
// ============================================================================

// RankingWeights - Bobot untuk setiap signal dalam ranking
type RankingWeights struct {
	Following  float64 // 0.30
	Recency    float64 // 0.25
	Engagement float64 // 0.20
	Relevance  float64 // 0.15
	Quality    float64 // 0.10
}

// DefaultRankingWeights returns default weights for ranking calculation
func DefaultRankingWeights() RankingWeights {
	return RankingWeights{
		Following:  0.30,
		Recency:    0.25,
		Engagement: 0.20,
		Relevance:  0.15,
		Quality:    0.10,
	}
}

// SignalScores - Skor untuk setiap signal
type SignalScores struct {
	Following  float64 `json:"following"`
	Recency    float64 `json:"recency"`
	Engagement float64 `json:"engagement"`
	Relevance  float64 `json:"relevance"`
	Quality    float64 `json:"quality"`
}

// Calculate menghitung total ranking score dari signal scores
func (s SignalScores) Calculate(weights RankingWeights) float64 {
	return s.Following*weights.Following +
		s.Recency*weights.Recency +
		s.Engagement*weights.Engagement +
		s.Relevance*weights.Relevance +
		s.Quality*weights.Quality
}

// ============================================================================
// CHANGELOG MODELS
// ============================================================================

// ChangelogCategory enum
type ChangelogCategory string

const (
	ChangelogCategoryAdded   ChangelogCategory = "added"
	ChangelogCategoryUpdated ChangelogCategory = "updated"
	ChangelogCategoryRemoved ChangelogCategory = "removed"
	ChangelogCategoryFixed   ChangelogCategory = "fixed"
)

// Changelog - Main changelog entry
type Changelog struct {
	BaseModel
	Version      string                 `gorm:"type:varchar(50);not null" json:"version"`
	Title        string                 `gorm:"type:varchar(255);not null" json:"title"`
	Description  *string                `gorm:"type:text" json:"description,omitempty"`
	ReleaseDate  time.Time              `gorm:"type:date;not null;default:CURRENT_DATE" json:"release_date"`
	IsPublished  bool                   `gorm:"not null;default:false" json:"is_published"`
	CreatedBy    uuid.UUID              `gorm:"type:uuid;not null" json:"created_by"`
	Creator      *User                  `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Sections     []ChangelogSection     `gorm:"foreignKey:ChangelogID" json:"sections,omitempty"`
	Contributors []ChangelogContributor `gorm:"foreignKey:ChangelogID" json:"contributors,omitempty"`
}

func (Changelog) TableName() string { return "changelogs" }

// ChangelogSection - Section within a changelog (added, updated, removed, fixed)
type ChangelogSection struct {
	ID           uuid.UUID               `gorm:"type:uuid;primaryKey" json:"id"`
	ChangelogID  uuid.UUID               `gorm:"type:uuid;not null" json:"changelog_id"`
	Category     ChangelogCategory       `gorm:"type:varchar(20);not null" json:"category"`
	SectionOrder int                     `gorm:"not null;default:0" json:"section_order"`
	CreatedAt    time.Time               `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	Blocks       []ChangelogSectionBlock `gorm:"foreignKey:SectionID" json:"blocks,omitempty"`
}

func (ChangelogSection) TableName() string { return "changelog_sections" }

// ChangelogSectionBlock - Content block within a section
type ChangelogSectionBlock struct {
	ID         uuid.UUID        `gorm:"type:uuid;primaryKey" json:"id"`
	SectionID  uuid.UUID        `gorm:"type:uuid;not null" json:"section_id"`
	BlockType  ContentBlockType `gorm:"type:varchar(20);not null" json:"block_type"`
	BlockOrder int              `gorm:"not null;default:0" json:"block_order"`
	Payload    JSONB            `gorm:"type:jsonb;not null;default:'{}'" json:"payload"`
	CreatedAt  time.Time        `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt  time.Time        `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
}

func (ChangelogSectionBlock) TableName() string { return "changelog_section_blocks" }

// ChangelogContributor - User who contributed to a changelog
type ChangelogContributor struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	ChangelogID      uuid.UUID `gorm:"type:uuid;not null" json:"changelog_id"`
	UserID           uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Contribution     string    `gorm:"type:varchar(255);not null" json:"contribution"`
	ContributorOrder int       `gorm:"not null;default:0" json:"contributor_order"`
	CreatedAt        time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	User             *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (ChangelogContributor) TableName() string { return "changelog_contributors" }

// ChangelogRead - Track which changelogs have been read by users
type ChangelogRead struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	ChangelogID uuid.UUID `gorm:"type:uuid;not null" json:"changelog_id"`
	ReadAt      time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"read_at"`
}

func (ChangelogRead) TableName() string { return "changelog_reads" }

// ============================================================================
// DIRECT MESSAGING SYSTEM MODELS
// ============================================================================

// DM Privacy enum
type DMPrivacy string

const (
	DMPrivacyOpen      DMPrivacy = "open"
	DMPrivacyFollowers DMPrivacy = "followers"
	DMPrivacyMutual    DMPrivacy = "mutual"
	DMPrivacyClosed    DMPrivacy = "closed"
)

// Message Type enum
type MessageType string

const (
	MessageTypeText      MessageType = "text"
	MessageTypeImage     MessageType = "image"
	MessageTypePortfolio MessageType = "portfolio"
	MessageTypeSystem    MessageType = "system"
)

// Conversation - DM conversation between users
type Conversation struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	CreatedAt          time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt          time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	LastMessageAt      *time.Time `json:"last_message_at,omitempty"`
	LastMessagePreview *string    `gorm:"type:text" json:"last_message_preview,omitempty"`

	// Relationships
	Participants []ConversationParticipant `gorm:"foreignKey:ConversationID" json:"participants,omitempty"`
	Messages     []Message                 `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}

func (Conversation) TableName() string { return "conversations" }

func (m *Conversation) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// ConversationParticipant - User participation in a conversation
type ConversationParticipant struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	ConversationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"conversation_id"`
	UserID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	JoinedAt       time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"joined_at"`
	LastReadAt     *time.Time `json:"last_read_at,omitempty"`
	IsMuted        bool       `gorm:"not null;default:false" json:"is_muted"`
	IsArchived     bool       `gorm:"not null;default:false" json:"is_archived"`
	UnreadCount    int        `gorm:"not null;default:0" json:"unread_count"`

	// Relationships
	Conversation *Conversation `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
	User         *User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (ConversationParticipant) TableName() string { return "conversation_participants" }

func (m *ConversationParticipant) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// Message - A message in a conversation
type Message struct {
	ID             uuid.UUID   `gorm:"type:uuid;primaryKey" json:"id"`
	ConversationID uuid.UUID   `gorm:"type:uuid;not null;index" json:"conversation_id"`
	SenderID       uuid.UUID   `gorm:"type:uuid;not null;index" json:"sender_id"`
	MessageType    MessageType `gorm:"type:varchar(20);not null;default:'text'" json:"message_type"`
	Content        JSONB       `gorm:"type:jsonb;not null" json:"content"`
	ReplyToID      *uuid.UUID  `gorm:"type:uuid;index" json:"reply_to_id,omitempty"`
	CreatedAt      time.Time   `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time   `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt      *time.Time  `gorm:"index" json:"deleted_at,omitempty"`

	// Relationships
	Conversation *Conversation     `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
	Sender       *User             `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
	ReplyTo      *Message          `gorm:"foreignKey:ReplyToID" json:"reply_to,omitempty"`
	Reactions    []MessageReaction `gorm:"foreignKey:MessageID" json:"reactions,omitempty"`
}

func (Message) TableName() string { return "messages" }

func (m *Message) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// MessageReaction - Emoji reaction to a message
type MessageReaction struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	MessageID uuid.UUID `gorm:"type:uuid;not null;index" json:"message_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Emoji     string    `gorm:"type:varchar(10);not null" json:"emoji"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`

	// Relationships
	Message *Message `gorm:"foreignKey:MessageID" json:"message,omitempty"`
	User    *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (MessageReaction) TableName() string { return "message_reactions" }

func (m *MessageReaction) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// DMSettings - User DM privacy settings
type DMSettings struct {
	UserID              uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	DMPrivacy           DMPrivacy `gorm:"type:varchar(20);not null;default:'followers'" json:"dm_privacy"`
	ShowReadReceipts    bool      `gorm:"not null;default:true" json:"show_read_receipts"`
	ShowTypingIndicator bool      `gorm:"not null;default:true" json:"show_typing_indicator"`
	UpdatedAt           time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// Relationships
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

func (DMSettings) TableName() string { return "dm_settings" }

// ChatStreak - Track chat streaks between two users
type ChatStreak struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserAID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_a_id"`
	UserBID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_b_id"`
	CurrentStreak int        `gorm:"not null;default:0" json:"current_streak"`
	LongestStreak int        `gorm:"not null;default:0" json:"longest_streak"`
	LastChatDate  *time.Time `gorm:"type:date" json:"last_chat_date,omitempty"`
	StartedAt     time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"started_at"`
	UpdatedAt     time.Time  `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// Relationships
	UserA *User `gorm:"foreignKey:UserAID" json:"user_a,omitempty"`
	UserB *User `gorm:"foreignKey:UserBID" json:"user_b,omitempty"`
}

func (ChatStreak) TableName() string { return "chat_streaks" }

func (m *ChatStreak) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// UserBlock - Block another user from DM
type UserBlock struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	BlockerID uuid.UUID `gorm:"type:uuid;not null;index" json:"blocker_id"`
	BlockedID uuid.UUID `gorm:"type:uuid;not null;index" json:"blocked_id"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`

	// Relationships
	Blocker *User `gorm:"foreignKey:BlockerID" json:"blocker,omitempty"`
	Blocked *User `gorm:"foreignKey:BlockedID" json:"blocked,omitempty"`
}

func (UserBlock) TableName() string { return "user_blocks" }

func (m *UserBlock) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// ============================================================================
// HOOKS FOR UUID GENERATION
// ============================================================================

// setUUIDIfEmpty checks if ID is nil and sets it to a new UUID
func setUUIDIfEmpty(id *uuid.UUID) {
	if *id == uuid.Nil {
		*id = uuid.New()
	}
}

// BaseModel Hook
func (b *BaseModel) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&b.ID)
	return nil
}

// SeriesBlock Hook
func (m *SeriesBlock) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// UserSocialLink Hook
func (m *UserSocialLink) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// StudentClassHistory Hook
func (m *StudentClassHistory) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// RefreshToken Hook
func (m *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// TokenBlacklist Hook
func (m *TokenBlacklist) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// Follow Hook
func (m *Follow) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// ContentBlock Hook
func (m *ContentBlock) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// Feedback Hook
func (m *Feedback) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// PortfolioAssessment Hook
func (m *PortfolioAssessment) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// PortfolioAssessmentScore Hook
func (m *PortfolioAssessmentScore) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// Notification Hook
func (m *Notification) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// PortfolioView Hook
func (m *PortfolioView) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// ChangelogSection Hook
func (m *ChangelogSection) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// ChangelogSectionBlock Hook
func (m *ChangelogSectionBlock) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// ChangelogContributor Hook
func (m *ChangelogContributor) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}

// ChangelogRead Hook
func (m *ChangelogRead) BeforeCreate(tx *gorm.DB) error {
	setUUIDIfEmpty(&m.ID)
	return nil
}
