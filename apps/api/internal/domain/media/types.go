// Package media implements media uploads: content detection, size and quota
// enforcement, storage in the private bucket, and the processing job.
package media

import (
	"time"

	"github.com/google/uuid"
)

// Purpose is why a media asset was uploaded.
type Purpose string

// Supported upload purposes.
const (
	PurposeContent              Purpose = "content"
	PurposeAvatar               Purpose = "avatar"
	PurposeBanner               Purpose = "banner"
	PurposeGroupAvatar          Purpose = "group_avatar"
	PurposeGroupBanner          Purpose = "group_banner"
	PurposeExperienceAttachment Purpose = "experience_attachment"
	PurposeBadgeIcon            Purpose = "badge_icon"
	PurposeCurationCover        Purpose = "curation_cover"
	PurposeImport               Purpose = "import"
)

// Kind is the media type detected from the file content.
type Kind string

// Supported media kinds.
const (
	KindImage    Kind = "image"
	KindPDF      Kind = "pdf"
	KindModel3D  Kind = "model3d"
	KindDocument Kind = "document"
)

// Asset is one row of media_assets as seen by the domain.
type Asset struct {
	ID               uuid.UUID
	OwnerID          uuid.UUID
	Kind             Kind
	Purpose          Purpose
	MimeType         string
	SizeBytes        int64
	Width            *int32
	Height           *int32
	PageCount        *int32
	Status           string
	Variants         map[string]Variant
	IsPublic         bool
	Blurhash         *string
	DominantColor    *string
	OriginalFilename *string
	CreatedAt        time.Time
}

// Variant describes one processed variant. Object keys are deliberately not
// part of this type: they never leave the server.
type Variant struct {
	Key    string `json:"key"`
	Format string `json:"format"`
	W      int    `json:"w"`
	H      int    `json:"h"`
	Bytes  int64  `json:"bytes"`
}
