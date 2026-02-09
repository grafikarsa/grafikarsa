package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// UserRole represents the user's role
type UserRole string

const (
	RoleAdmin   UserRole = "admin"
	RoleStudent UserRole = "student"
	RoleAlumni  UserRole = "alumni"
)

// UserStatus represents the user's account status
type UserStatus string

const (
	StatusActive     UserStatus = "active"
	StatusGraduated  UserStatus = "graduated"
	StatusDroppedOut UserStatus = "dropped_out"
	StatusInactive   UserStatus = "inactive"
)

// SocialLinks represents user's social media links
type SocialLinks map[string]string

// User represents a user in the system
type User struct {
	ID             uuid.UUID   `json:"id"`
	Username       string      `json:"username"`
	Email          string      `json:"email,omitempty"` // Only visible to owner/admin
	PasswordHash   string      `json:"-"`               // Never expose
	Name           string      `json:"name"`
	Bio            string      `json:"bio,omitempty"`
	AvatarURL      string      `json:"avatar_url,omitempty"`
	BannerURL      string      `json:"banner_url,omitempty"`
	Role           UserRole    `json:"role"`
	Status         UserStatus  `json:"status"`
	NISN           string      `json:"nisn,omitempty"` // Only visible to owner/admin
	NIS            string      `json:"nis,omitempty"`  // Only visible to owner/admin
	CurrentClassID *uuid.UUID  `json:"current_class_id,omitempty"`
	EntryYear      *int        `json:"entry_year,omitempty"`
	GraduationYear *int        `json:"graduation_year,omitempty"`
	SocialLinks    SocialLinks `json:"social_links,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	DeletedAt      *time.Time  `json:"-"`

	// Related data (populated by joins)
	Class        *Class         `json:"class,omitempty"`
	Major        *Major         `json:"major,omitempty"`
	ClassHistory []ClassHistory `json:"class_history,omitempty"`

	// Computed fields
	FollowerCount  int  `json:"follower_count,omitempty"`
	FollowingCount int  `json:"following_count,omitempty"`
	PortfolioCount int  `json:"portfolio_count,omitempty"`
	IsFollowing    bool `json:"is_following,omitempty"` // If current user follows this user
}

// ClassHistory represents a user's class assignment history
type ClassHistory struct {
	ClassName    string    `json:"class_name"`
	AcademicYear int       `json:"academic_year"`
	AssignedAt   time.Time `json:"assigned_at"`
}

// IsAdmin checks if the user is an admin
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsActive checks if the user account is active
func (u *User) IsActive() bool {
	return u.Status == StatusActive
}

// CanLogin checks if the user can log in
func (u *User) CanLogin() bool {
	return u.Status == StatusActive || u.Status == StatusGraduated
}

// UserPublicProfile returns a user profile suitable for public view
type UserPublicProfile struct {
	ID             uuid.UUID      `json:"id"`
	Username       string         `json:"username"`
	Name           string         `json:"name"`
	Bio            string         `json:"bio,omitempty"`
	AvatarURL      string         `json:"avatar_url,omitempty"`
	BannerURL      string         `json:"banner_url,omitempty"`
	Role           UserRole       `json:"role"`
	Status         UserStatus     `json:"status"`
	EntryYear      *int           `json:"entry_year,omitempty"`
	GraduationYear *int           `json:"graduation_year,omitempty"`
	Class          *ClassInfo     `json:"class,omitempty"`
	Major          *MajorInfo     `json:"major,omitempty"`
	ClassHistory   []ClassHistory `json:"class_history,omitempty"`
	SocialLinks    SocialLinks    `json:"social_links,omitempty"`
	FollowerCount  int            `json:"follower_count"`
	FollowingCount int            `json:"following_count"`
	PortfolioCount int            `json:"portfolio_count"`
	IsFollowing    bool           `json:"is_following"`
	CreatedAt      time.Time      `json:"created_at"`
}

// ToPublicProfile converts a User to UserPublicProfile
func (u *User) ToPublicProfile() UserPublicProfile {
	profile := UserPublicProfile{
		ID:             u.ID,
		Username:       u.Username,
		Name:           u.Name,
		Bio:            u.Bio,
		AvatarURL:      u.AvatarURL,
		BannerURL:      u.BannerURL,
		Role:           u.Role,
		Status:         u.Status,
		EntryYear:      u.EntryYear,
		GraduationYear: u.GraduationYear,
		SocialLinks:    u.SocialLinks,
		FollowerCount:  u.FollowerCount,
		FollowingCount: u.FollowingCount,
		PortfolioCount: u.PortfolioCount,
		IsFollowing:    u.IsFollowing,
		ClassHistory:   u.ClassHistory,
		CreatedAt:      u.CreatedAt,
	}

	if u.Class != nil {
		profile.Class = &ClassInfo{
			ID:   u.Class.ID,
			Name: u.Class.Name,
		}
	}

	if u.Major != nil {
		profile.Major = &MajorInfo{
			ID:   u.Major.ID,
			Name: u.Major.Name,
		}
	}

	return profile
}

// UserListItem represents a user in list views
type UserListItem struct {
	ID        uuid.UUID  `json:"id"`
	Username  string     `json:"username"`
	Name      string     `json:"name"`
	AvatarURL string     `json:"avatar_url,omitempty"`
	Role      UserRole   `json:"role"`
	ClassName string     `json:"class_name,omitempty"`
	MajorName string     `json:"major_name,omitempty"`
	Class     *ClassInfo `json:"class,omitempty"`
	Major     *MajorInfo `json:"major,omitempty"`
}

// ToListItem converts User to UserListItem
func (u *User) ToListItem() UserListItem {
	item := UserListItem{
		ID:        u.ID,
		Username:  u.Username,
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
		Role:      u.Role,
	}
	if u.Class != nil {
		item.ClassName = u.Class.Name
		item.Class = &ClassInfo{ID: u.Class.ID, Name: u.Class.Name}
	}
	if u.Major != nil {
		item.MajorName = u.Major.Name
		item.Major = &MajorInfo{ID: u.Major.ID, Name: u.Major.Name}
	}
	return item
}

// FollowerItem represents a follower/following in list views
type FollowerItem struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	Name        string    `json:"name"`
	AvatarURL   string    `json:"avatar_url,omitempty"`
	Role        UserRole  `json:"role"`
	ClassName   string    `json:"class_name,omitempty"`
	IsFollowing bool      `json:"is_following"`
	FollowedAt  time.Time `json:"followed_at"`
}

// MarshalJSON for SocialLinks to handle empty map
func (sl SocialLinks) MarshalJSON() ([]byte, error) {
	if len(sl) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(map[string]string(sl))
}

// UserMeProfile returns the full profile for the current user
type UserMeProfile struct {
	ID             uuid.UUID   `json:"id"`
	Username       string      `json:"username"`
	Email          string      `json:"email"`
	Name           string      `json:"name"`
	Bio            string      `json:"bio,omitempty"`
	AvatarURL      string      `json:"avatar_url,omitempty"`
	BannerURL      string      `json:"banner_url,omitempty"`
	Role           UserRole    `json:"role"`
	Status         UserStatus  `json:"status"`
	NISN           string      `json:"nisn,omitempty"`
	NIS            string      `json:"nis,omitempty"`
	EntryYear      *int        `json:"entry_year,omitempty"`
	GraduationYear *int        `json:"graduation_year,omitempty"`
	Class          *ClassInfo  `json:"class,omitempty"`
	Major          *MajorInfo  `json:"major,omitempty"`
	SocialLinks    SocialLinks `json:"social_links,omitempty"`
	FollowerCount  int         `json:"follower_count"`
	FollowingCount int         `json:"following_count"`
	CreatedAt      time.Time   `json:"created_at"`
}

// ToMeProfile converts a User to UserMeProfile
func (u *User) ToMeProfile() UserMeProfile {
	profile := UserMeProfile{
		ID:             u.ID,
		Username:       u.Username,
		Email:          u.Email,
		Name:           u.Name,
		Bio:            u.Bio,
		AvatarURL:      u.AvatarURL,
		BannerURL:      u.BannerURL,
		Role:           u.Role,
		Status:         u.Status,
		NISN:           u.NISN,
		NIS:            u.NIS,
		EntryYear:      u.EntryYear,
		GraduationYear: u.GraduationYear,
		SocialLinks:    u.SocialLinks,
		FollowerCount:  u.FollowerCount,
		FollowingCount: u.FollowingCount,
		CreatedAt:      u.CreatedAt,
	}

	if u.Class != nil {
		profile.Class = &ClassInfo{
			ID:   u.Class.ID,
			Name: u.Class.Name,
		}
	}

	if u.Major != nil {
		profile.Major = &MajorInfo{
			ID:   u.Major.ID,
			Name: u.Major.Name,
		}
	}

	return profile
}
