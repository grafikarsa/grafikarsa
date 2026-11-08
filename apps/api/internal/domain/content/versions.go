package content

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

// ListVersions returns the version history of a content, newest first, with
// the moderation decision of each version.
func (s *Service) ListVersions(ctx context.Context, actor Actor, contentID uuid.UUID) ([]ContentVersion, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return nil, err
	}
	author, err := s.authorAccess(ctx, s.st.Queries, contentID, principal.UserID)
	if err != nil {
		return nil, err
	}

	rows, err := s.st.ListContentVersions(ctx, contentID)
	if err != nil {
		return nil, fmt.Errorf("content: list versions: %w", err)
	}

	versions := make([]ContentVersion, 0, len(rows))
	for _, row := range rows {
		decision, err := s.versionDecision(ctx, s.st.Queries, author.Level, versionDecisionParts{
			CaseID:       row.CaseID,
			Decision:     row.Decision,
			Note:         row.DecisionNote,
			ReasonLabel:  row.ReasonLabel,
			DecidedAt:    row.CaseDecidedAt,
			AppealID:     row.AppealID,
			AppealStatus: row.AppealStatus,
		})
		if err != nil {
			return nil, err
		}
		versions = append(versions, ContentVersion{
			ID:          row.ID,
			Number:      int(row.Number),
			Kind:        row.Kind,
			State:       row.State,
			Title:       row.Title,
			SubmittedAt: row.SubmittedAt,
			DecidedAt:   row.DecidedAt,
			SubmittedBy: AuthorMini{
				UserID:      row.SubmittedBy,
				Username:    row.SubmitterUsername,
				DisplayName: row.SubmitterDisplayName,
			},
			Decision: decision,
		})
	}
	return versions, nil
}

// GetVersion returns one immutable snapshot with its taxonomy and media
// expanded for the version detail screen.
func (s *Service) GetVersion(ctx context.Context, actor Actor, contentID, versionID uuid.UUID) (ContentVersionDetail, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return ContentVersionDetail{}, err
	}
	author, err := s.authorAccess(ctx, s.st.Queries, contentID, principal.UserID)
	if err != nil {
		return ContentVersionDetail{}, err
	}

	version, err := s.st.GetContentVersionByID(ctx, store.GetContentVersionByIDParams{
		ID:        versionID,
		ContentID: contentID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ContentVersionDetail{}, apperr.NotFound()
	}
	if err != nil {
		return ContentVersionDetail{}, fmt.Errorf("content: load version: %w", err)
	}

	submitter, err := s.st.GetAuthorMini(ctx, version.SubmittedBy)
	if err != nil {
		return ContentVersionDetail{}, fmt.Errorf("content: load version submitter: %w", err)
	}

	parts := versionDecisionParts{}
	decisionRow, err := s.st.GetVersionModerationDecision(ctx, versionID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
	case err != nil:
		return ContentVersionDetail{}, fmt.Errorf("content: load version decision: %w", err)
	default:
		parts = versionDecisionParts{
			CaseID:       &decisionRow.CaseID,
			Decision:     decisionRow.Decision,
			Note:         decisionRow.DecisionNote,
			ReasonLabel:  decisionRow.ReasonLabel,
			DecidedAt:    decisionRow.DecidedAt,
			AppealID:     decisionRow.AppealID,
			AppealStatus: decisionRow.AppealStatus,
		}
	}
	decision, err := s.versionDecision(ctx, s.st.Queries, author.Level, parts)
	if err != nil {
		return ContentVersionDetail{}, err
	}

	detail := ContentVersionDetail{
		Version: ContentVersion{
			ID:          version.ID,
			Number:      int(version.Number),
			Kind:        version.Kind,
			State:       version.State,
			Title:       version.Title,
			SubmittedAt: version.SubmittedAt,
			DecidedAt:   version.DecidedAt,
			SubmittedBy: AuthorMini{
				UserID:      submitter.ID,
				Username:    submitter.Username,
				DisplayName: submitter.DisplayName,
			},
			Decision: decision,
		},
		Title:   version.Title,
		Excerpt: version.Excerpt,
		Body:    decodeObject(version.Body),
		Meta:    decodeObject(version.Meta),
		Tags:    []TaxonomyTerm{},
		Tools:   []TaxonomyTerm{},
		Skills:  []TaxonomyTerm{},
	}

	if version.CategoryID != nil {
		category, err := s.st.GetContentCategoryRef(ctx, *version.CategoryID)
		if errors.Is(err, pgx.ErrNoRows) {
			// The FK guarantees the category exists; a missing row means it was
			// removed outside the normal lifecycle. Omit the reference.
		} else if err != nil {
			return ContentVersionDetail{}, fmt.Errorf("content: load version category: %w", err)
		} else {
			detail.Category = &CategoryRef{Slug: category.Slug, Name: category.Name}
		}
	}
	if version.CoverMediaID != nil {
		cover, err := loadImageRef(ctx, s.st.Queries, *version.CoverMediaID)
		if err != nil {
			return ContentVersionDetail{}, err
		}
		detail.Cover = cover
	}

	if len(version.TagIds) > 0 {
		rows, err := s.st.GetTagsByIDs(ctx, version.TagIds)
		if err != nil {
			return ContentVersionDetail{}, fmt.Errorf("content: load version tags: %w", err)
		}
		for _, row := range rows {
			detail.Tags = append(detail.Tags, TaxonomyTerm{Slug: row.Slug, Name: row.Name})
		}
	}
	if len(version.ToolIds) > 0 {
		rows, err := s.st.GetToolsByIDs(ctx, version.ToolIds)
		if err != nil {
			return ContentVersionDetail{}, fmt.Errorf("content: load version tools: %w", err)
		}
		for _, row := range rows {
			category := row.Category
			detail.Tools = append(detail.Tools, TaxonomyTerm{Slug: row.Slug, Name: row.Name, Category: &category})
		}
	}
	if len(version.SkillIds) > 0 {
		rows, err := s.st.GetSkillsByIDs(ctx, version.SkillIds)
		if err != nil {
			return ContentVersionDetail{}, fmt.Errorf("content: load version skills: %w", err)
		}
		for _, row := range rows {
			category := row.Category
			detail.Skills = append(detail.Skills, TaxonomyTerm{Slug: row.Slug, Name: row.Name, Category: &category})
		}
	}

	return detail, nil
}

type versionDecisionParts struct {
	CaseID       *uuid.UUID
	Decision     *string
	Note         *string
	ReasonLabel  *string
	DecidedAt    *time.Time
	AppealID     *uuid.UUID
	AppealStatus *string
}

func (s *Service) versionDecision(
	ctx context.Context,
	q *store.Queries,
	callerLevel string,
	parts versionDecisionParts,
) (*ModerationFeedback, error) {
	if parts.CaseID == nil || parts.Decision == nil {
		return nil, nil
	}

	annotationRows, err := q.ListModerationAnnotations(ctx, *parts.CaseID)
	if err != nil {
		return nil, fmt.Errorf("content: load version moderation annotations: %w", err)
	}
	annotations := make([]ModerationAnnotation, 0, len(annotationRows))
	for _, annotation := range annotationRows {
		annotations = append(annotations, ModerationAnnotation{
			TargetRef: annotation.TargetRef,
			Note:      annotation.Note,
		})
	}

	decidedAt := time.Time{}
	if parts.DecidedAt != nil {
		decidedAt = *parts.DecidedAt
	}
	return &ModerationFeedback{
		Decision:     *parts.Decision,
		ReasonLabel:  parts.ReasonLabel,
		Note:         parts.Note,
		Annotations:  annotations,
		DecidedAt:    decidedAt,
		CanAppeal:    *parts.Decision == "rejected" && parts.AppealID == nil && callerLevel == LevelOwner,
		AppealStatus: parts.AppealStatus,
	}, nil
}
