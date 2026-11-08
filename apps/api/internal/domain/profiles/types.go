// Package profiles implements public profiles, own profile settings, follows,
// skill endorsements, experiences, and username changes. HTTP handlers live
// elsewhere; this package only exposes the domain service.
package profiles

import (
	"time"

	"github.com/google/uuid"
)

const (
	accountTypeStudent = "student"
	accountTypeTeacher = "teacher"

	statusActive      = "active"
	statusAlumni      = "alumni"
	statusTransferred = "transferred"
	statusInactive    = "inactive"

	avatarPurpose = "avatar"
	bannerPurpose = "banner"

	targetTypeUser = "user"

	defaultMaxLinks                = 10
	defaultMaxSkills               = 20
	defaultUsernameChangeDays      = 30
	defaultUsernameRedirectDays    = 90
	maxTeachingSubjects            = 10
	maxTeachingSubjectLength       = 60
	maxExperienceSkills            = 10
	maxExperienceTitleLength       = 120
	maxExperienceOrganizationLen   = 120
	maxExperienceLocationLength    = 120
	maxExperienceDescriptionLength = 2000
	maxCredentialURLLength         = 500
	maxLinkURLLength               = 500
	defaultPageLimit               = 20
	maxPageLimit                   = 50
	maxSkillQueryLength            = 100
)

// Link is one social or website link on a profile.
type Link struct {
	Platform string
	URL      string
}

// Skill is one profile skill with its endorsement state.
type Skill struct {
	Slug             string
	Name             string
	Category         string
	EndorsementCount int64
	EndorsedByViewer bool
}

// Badge is one active badge awarded to a profile.
type Badge struct {
	Slug        string
	Name        string
	Description string
	IconKey     *string
	Note        *string
	AwardedAt   time.Time
}

// ClassRef is the latest class of a student or alumnus.
type ClassRef struct {
	Name      string
	MajorCode string
}

// MajorRef is one major taught by a teacher.
type MajorRef struct {
	Code string
	Name string
}

// Stats counts a profile's followers and following.
type Stats struct {
	Followers int64
	Following int64
}

// Viewer holds the state that depends on who is looking at the profile.
type Viewer struct {
	Following bool
	IsSelf    bool
}

// Profile is a full public profile.
type Profile struct {
	ID               uuid.UUID
	Username         string
	DisplayName      string
	AccountType      string
	Status           string
	Headline         *string
	Bio              *string
	City             *string
	AvatarMediaID    *uuid.UUID
	BannerMediaID    *uuid.UUID
	Links            []Link
	Skills           []Skill
	Badges           []Badge
	CurrentClass     *ClassRef
	TeacherMajors    []MajorRef
	TeachingSubjects []string
	Stats            Stats
	Viewer           Viewer
}

// MyProfile is the owner's profile plus account-level settings.
type MyProfile struct {
	Profile
	UsernameChangeAvailableAt *time.Time
}

// PublicProfileResult is the outcome of a public profile lookup. When
// RedirectUsername is set, the caller should answer with a redirect to that
// username instead of a profile body.
type PublicProfileResult struct {
	Profile          *Profile
	RedirectUsername string
}

// UserMini is a compact user card used in follower lists.
type UserMini struct {
	Username      string
	DisplayName   string
	AccountType   string
	AvatarMediaID *uuid.UUID
}

// UserMiniPage is one presentation-ordered page of user cards.
type UserMiniPage struct {
	Items      []UserMini
	NextCursor string
}

// SkillOption is one skill catalog entry returned by autocomplete.
type SkillOption struct {
	Slug     string
	Name     string
	Category string
}

// ExperienceSkill is a skill attached to an experience.
type ExperienceSkill struct {
	Slug string
	Name string
}

// Experience is one self-declared experience entry.
type Experience struct {
	ID             uuid.UUID
	Kind           string
	Title          string
	Organization   string
	Location       *string
	EmploymentType *string
	StartMonth     string
	EndMonth       *string
	IsCurrent      bool
	Description    *string
	CredentialURL  *string
	IsSystem       bool
	Skills         []ExperienceSkill
}

// ExperienceInput is the full payload for creating or updating an experience.
type ExperienceInput struct {
	Kind           string
	Title          string
	Organization   string
	Location       *string
	EmploymentType *string
	StartMonth     string
	EndMonth       *string
	IsCurrent      bool
	Description    *string
	CredentialURL  *string
	Skills         []string
}

// UpdateInput carries optional profile changes. A nil pointer means unchanged.
type UpdateInput struct {
	DisplayName      *string
	Headline         *string
	Bio              *string
	City             *string
	TeachingSubjects *[]string
}
