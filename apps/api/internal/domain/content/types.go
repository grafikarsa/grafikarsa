package content

import (
	"time"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
)

// Content types.
const (
	TypePortfolio = "portfolio"
	TypeBlog      = "blog"
)

// Lifecycle statuses.
const (
	StatusDraft            = "draft"
	StatusInReview         = "in_review"
	StatusChangesRequested = "changes_requested"
	StatusPublished        = "published"
	StatusArchived         = "archived"
	StatusTakenDown        = "taken_down"
	StatusRejected         = "rejected"
	StatusDeleted          = "deleted"
)

// Revision statuses, only present while a published content is being revised.
const (
	RevisionEditing          = "editing"
	RevisionInReview         = "in_review"
	RevisionChangesRequested = "changes_requested"
)

// Author levels.
const (
	LevelOwner       = "owner"
	LevelEditor      = "editor"
	LevelContributor = "contributor"
)

const (
	permissionContentCreate = "content.create"
	maxActiveDraftsSetting  = "content.max_active_drafts"
	defaultMaxActiveDrafts  = 20
	maxInReviewSetting      = "content.max_in_review"
	defaultMaxInReview      = 5
	trashRetentionSetting   = "content.trash_retention_days"
	defaultTrashRetention   = 30

	maxPublicIDAttempts = 5
)

// Actor is the authenticated principal performing an action, plus the request
// IP recorded in the audit log.
type Actor struct {
	Principal *auth.Principal
	IP        string
}

// CreateInput is the payload for Create.
type CreateInput struct {
	Type string
}

// DraftInput is the payload for SaveDraft.
type DraftInput struct {
	LockVersion  int
	Title        string
	Excerpt      *string
	CategoryID   *uuid.UUID
	CoverMediaID *uuid.UUID
	Body         map[string]any
	Meta         map[string]any
	Tags         []string
	Tools        []string
	Skills       []string
}

// TaxonomyTerm is a tag, tool, or skill shown in the editor.
type TaxonomyTerm struct {
	Slug     string
	Name     string
	Category *string
}

// ImageRef is a reference to an image asset.
type ImageRef struct {
	MediaID       uuid.UUID
	URL           *string
	Width         *int
	Height        *int
	Blurhash      *string
	DominantColor *string
}

// AuthorMini is the compact author representation.
type AuthorMini struct {
	UserID      uuid.UUID
	Username    *string
	DisplayName string
	AvatarURL   *string
}

// ModerationAnnotation is one block/node note left by a moderator.
type ModerationAnnotation struct {
	TargetRef string
	Note      string
}

// ModerationFeedback is the latest moderation decision for a content.
type ModerationFeedback struct {
	Decision     string
	ReasonLabel  *string
	Note         *string
	Annotations  []ModerationAnnotation
	DecidedAt    time.Time
	CanAppeal    bool
	AppealStatus *string
}

// Draft is the editable working copy of a content.
type Draft struct {
	LockVersion int
	Title       string
	Excerpt     *string
	CategoryID  *uuid.UUID
	Cover       *ImageRef
	Body        map[string]any
	Meta        map[string]any
	Tags        []TaxonomyTerm
	Tools       []TaxonomyTerm
	Skills      []TaxonomyTerm
	UpdatedAt   time.Time
	UpdatedBy   *AuthorMini
}

// StudioContent is the editor view of one content.
type StudioContent struct {
	ID               uuid.UUID
	PublicID         string
	Type             string
	Status           string
	RevisionStatus   *string
	Slug             string
	Title            string
	MyLevel          string
	Draft            Draft
	LastModeration   *ModerationFeedback
	HasLiveVersion   bool
	InReviewClaimed  bool
	AuthorsCount     int
	FirstPublishedAt *time.Time
	ArchivedAt       *time.Time
	DeletedAt        *time.Time
	PurgeAfter       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CategoryRef is the compact category representation used in version
// snapshots.
type CategoryRef struct {
	Slug string
	Name string
}

// ContentVersion is one entry of a content's version history.
type ContentVersion struct {
	ID          uuid.UUID
	Number      int
	Kind        string
	State       string
	Title       string
	SubmittedAt time.Time
	DecidedAt   *time.Time
	SubmittedBy AuthorMini
	Decision    *ModerationFeedback
}

// ContentVersionDetail is a version snapshot with taxonomy and media expanded.
type ContentVersionDetail struct {
	Version  ContentVersion
	Title    string
	Excerpt  *string
	Category *CategoryRef
	Cover    *ImageRef
	Body     map[string]any
	Meta     map[string]any
	Tags     []TaxonomyTerm
	Tools    []TaxonomyTerm
	Skills   []TaxonomyTerm
}

// StudioContentItem is one entry of the Studio content list.
type StudioContentItem struct {
	ID             uuid.UUID
	PublicID       string
	Type           string
	Status         string
	RevisionStatus *string
	Title          string
	Cover          *ImageRef
	MyLevel        string
	AuthorsCount   int
	UpdatedAt      time.Time
}

// StudioPage is one page of Studio contents.
type StudioPage struct {
	Items      []StudioContentItem
	NextCursor string
}

// StudioSummary is the badge summary of the Studio menu.
type StudioSummary struct {
	Counts             map[string]int
	PendingInvitations int
}

// ListFilter narrows ListStudio. Zero values mean "no filter".
type ListFilter struct {
	Status string
	Type   string
	Cursor string
	Limit  int
}

// statuses lists every lifecycle status in display order.
var statuses = []string{
	StatusDraft,
	StatusInReview,
	StatusChangesRequested,
	StatusPublished,
	StatusArchived,
	StatusTakenDown,
	StatusRejected,
	StatusDeleted,
}
