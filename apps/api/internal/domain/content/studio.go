package content

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/store"
)

const (
	defaultStudioLimit = 20
	maxStudioLimit     = 50
)

// GetStudio returns the editor view of one content for an accepted author.
func (s *Service) GetStudio(ctx context.Context, actor Actor, id uuid.UUID) (StudioContent, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return StudioContent{}, err
	}
	return s.getStudioQ(ctx, s.st.Queries, id, principal.UserID)
}

// ListStudio returns a page of contents where the actor is an accepted author.
func (s *Service) ListStudio(ctx context.Context, actor Actor, filter ListFilter) (StudioPage, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return StudioPage{}, err
	}

	if filter.Status != "" && !knownStatus(filter.Status) {
		return StudioPage{}, apperr.BadRequest("status tidak valid")
	}
	if filter.Type != "" && filter.Type != TypePortfolio && filter.Type != TypeBlog {
		return StudioPage{}, apperr.BadRequest("type tidak valid")
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = defaultStudioLimit
	}
	if limit > maxStudioLimit {
		limit = maxStudioLimit
	}

	params := store.ListStudioContentsParams{
		UserID:     principal.UserID,
		LimitCount: int32(limit + 1),
	}
	if filter.Status != "" {
		status := filter.Status
		params.Status = &status
	}
	if filter.Type != "" {
		contentType := filter.Type
		params.Type = &contentType
	}
	if filter.Cursor != "" {
		cursorTime, cursorID, err := decodeStudioCursor(filter.Cursor)
		if err != nil {
			return StudioPage{}, apperr.BadRequest("cursor tidak valid")
		}
		params.CursorTime = &cursorTime
		params.CursorID = &cursorID
	}

	rows, err := s.st.ListStudioContents(ctx, params)
	if err != nil {
		return StudioPage{}, fmt.Errorf("content: list studio contents: %w", err)
	}

	page := StudioPage{Items: make([]StudioContentItem, 0, min(len(rows), limit))}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		page.NextCursor = encodeStudioCursor(last.UpdatedAt, last.ID)
	}
	for _, row := range rows {
		item := StudioContentItem{
			ID:             row.ID,
			PublicID:       row.PublicID,
			Type:           row.Type,
			Status:         row.Status,
			RevisionStatus: row.RevisionStatus,
			Title:          row.Title,
			MyLevel:        row.MyLevel,
			AuthorsCount:   int(row.AuthorsCount),
			UpdatedAt:      row.UpdatedAt,
		}
		if row.CoverMediaID != nil {
			item.Cover = newImageRef(*row.CoverMediaID, row.CoverWidth, row.CoverHeight, row.CoverBlurhash, row.CoverDominantColor)
		}
		page.Items = append(page.Items, item)
	}
	return page, nil
}

// Summary returns the Studio badge counts for the actor.
func (s *Service) Summary(ctx context.Context, actor Actor) (StudioSummary, error) {
	principal, err := s.principal(actor)
	if err != nil {
		return StudioSummary{}, err
	}

	rows, err := s.st.CountStudioContentsByStatus(ctx, principal.UserID)
	if err != nil {
		return StudioSummary{}, fmt.Errorf("content: count studio contents: %w", err)
	}
	counts := make(map[string]int, len(statuses))
	for _, status := range statuses {
		counts[status] = 0
	}
	for _, row := range rows {
		counts[row.Status] = int(row.Count)
	}

	invitations, err := s.st.CountPendingInvitations(ctx, principal.UserID)
	if err != nil {
		return StudioSummary{}, fmt.Errorf("content: count pending invitations: %w", err)
	}

	return StudioSummary{Counts: counts, PendingInvitations: int(invitations)}, nil
}

// getStudioQ builds the Studio view using the given query handle, so callers
// inside a transaction see their own writes.
func (s *Service) getStudioQ(ctx context.Context, q *store.Queries, id, actorID uuid.UUID) (StudioContent, error) {
	content, err := s.loadContent(ctx, q, id)
	if err != nil {
		return StudioContent{}, err
	}
	author, err := s.authorAccess(ctx, q, id, actorID)
	if err != nil {
		return StudioContent{}, err
	}

	draft, err := q.GetContentDraft(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return StudioContent{}, fmt.Errorf("content: draft missing for %s", id)
	}
	if err != nil {
		return StudioContent{}, fmt.Errorf("content: load draft: %w", err)
	}
	draftView, err := buildDraft(ctx, q, draft)
	if err != nil {
		return StudioContent{}, err
	}

	authorsCount, err := q.CountAcceptedAuthors(ctx, id)
	if err != nil {
		return StudioContent{}, fmt.Errorf("content: count authors: %w", err)
	}
	claimed, err := q.HasClaimedModerationCase(ctx, id)
	if err != nil {
		return StudioContent{}, fmt.Errorf("content: check claimed moderation: %w", err)
	}
	moderation, err := latestModeration(ctx, q, id, author.Level)
	if err != nil {
		return StudioContent{}, err
	}

	return StudioContent{
		ID:               content.ID,
		PublicID:         content.PublicID,
		Type:             content.Type,
		Status:           content.Status,
		RevisionStatus:   content.RevisionStatus,
		Slug:             content.Slug,
		Title:            content.Title,
		MyLevel:          author.Level,
		Draft:            draftView,
		LastModeration:   moderation,
		HasLiveVersion:   content.LiveVersionID != nil,
		InReviewClaimed:  claimed,
		AuthorsCount:     int(authorsCount),
		FirstPublishedAt: content.FirstPublishedAt,
		ArchivedAt:       content.ArchivedAt,
		DeletedAt:        content.DeletedAt,
		PurgeAfter:       content.PurgeAfter,
		CreatedAt:        content.CreatedAt,
		UpdatedAt:        content.UpdatedAt,
	}, nil
}

func buildDraft(ctx context.Context, q *store.Queries, draft store.ContentDraft) (Draft, error) {
	view := Draft{
		LockVersion: int(draft.LockVersion),
		Title:       draft.Title,
		Excerpt:     draft.Excerpt,
		CategoryID:  draft.CategoryID,
		Body:        decodeObject(draft.Body),
		Meta:        decodeObject(draft.Meta),
		Tags:        []TaxonomyTerm{},
		Tools:       []TaxonomyTerm{},
		Skills:      []TaxonomyTerm{},
		UpdatedAt:   draft.UpdatedAt,
	}

	if draft.CoverMediaID != nil {
		cover, err := loadImageRef(ctx, q, *draft.CoverMediaID)
		if err != nil {
			return Draft{}, err
		}
		view.Cover = cover
	}

	if updatedBy, err := q.GetAuthorMini(ctx, draft.UpdatedBy); err == nil {
		view.UpdatedBy = &AuthorMini{
			UserID:      updatedBy.ID,
			Username:    updatedBy.Username,
			DisplayName: updatedBy.DisplayName,
		}
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return Draft{}, fmt.Errorf("content: load draft editor: %w", err)
	}

	if len(draft.TagIds) > 0 {
		rows, err := q.GetTagsByIDs(ctx, draft.TagIds)
		if err != nil {
			return Draft{}, fmt.Errorf("content: load draft tags: %w", err)
		}
		for _, row := range rows {
			view.Tags = append(view.Tags, TaxonomyTerm{Slug: row.Slug, Name: row.Name})
		}
	}
	if len(draft.ToolIds) > 0 {
		rows, err := q.GetToolsByIDs(ctx, draft.ToolIds)
		if err != nil {
			return Draft{}, fmt.Errorf("content: load draft tools: %w", err)
		}
		for _, row := range rows {
			category := row.Category
			view.Tools = append(view.Tools, TaxonomyTerm{Slug: row.Slug, Name: row.Name, Category: &category})
		}
	}
	if len(draft.SkillIds) > 0 {
		rows, err := q.GetSkillsByIDs(ctx, draft.SkillIds)
		if err != nil {
			return Draft{}, fmt.Errorf("content: load draft skills: %w", err)
		}
		for _, row := range rows {
			category := row.Category
			view.Skills = append(view.Skills, TaxonomyTerm{Slug: row.Slug, Name: row.Name, Category: &category})
		}
	}

	return view, nil
}

func loadImageRef(ctx context.Context, q *store.Queries, id uuid.UUID) (*ImageRef, error) {
	row, err := q.GetImageRef(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("content: load image ref: %w", err)
	}
	return newImageRef(row.ID, row.Width, row.Height, row.Blurhash, row.DominantColor), nil
}

func newImageRef(id uuid.UUID, width, height *int32, blurhash, dominantColor *string) *ImageRef {
	ref := &ImageRef{
		MediaID:       id,
		Blurhash:      blurhash,
		DominantColor: dominantColor,
	}
	if width != nil {
		converted := int(*width)
		ref.Width = &converted
	}
	if height != nil {
		converted := int(*height)
		ref.Height = &converted
	}
	return ref
}

func latestModeration(ctx context.Context, q *store.Queries, contentID uuid.UUID, level string) (*ModerationFeedback, error) {
	row, err := q.GetLatestModerationDecision(ctx, contentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("content: load moderation decision: %w", err)
	}

	decision := ""
	if row.Decision != nil {
		decision = *row.Decision
	}
	decidedAt := time.Time{}
	if row.DecidedAt != nil {
		decidedAt = *row.DecidedAt
	}

	annotationRows, err := q.ListModerationAnnotations(ctx, row.CaseID)
	if err != nil {
		return nil, fmt.Errorf("content: load moderation annotations: %w", err)
	}
	annotations := make([]ModerationAnnotation, 0, len(annotationRows))
	for _, annotation := range annotationRows {
		annotations = append(annotations, ModerationAnnotation{
			TargetRef: annotation.TargetRef,
			Note:      annotation.Note,
		})
	}

	return &ModerationFeedback{
		Decision:     decision,
		ReasonLabel:  row.ReasonLabel,
		Note:         row.DecisionNote,
		Annotations:  annotations,
		DecidedAt:    decidedAt,
		CanAppeal:    decision == "rejected" && row.AppealID == nil && level == LevelOwner,
		AppealStatus: row.AppealStatus,
	}, nil
}

func decodeObject(raw json.RawMessage) map[string]any {
	out := map[string]any{}
	if len(raw) == 0 {
		return out
	}
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

func knownStatus(status string) bool {
	for _, candidate := range statuses {
		if candidate == status {
			return true
		}
	}
	return false
}

type studioCursor struct {
	Time string `json:"t"`
	ID   string `json:"i"`
}

func encodeStudioCursor(updatedAt time.Time, id uuid.UUID) string {
	raw, err := json.Marshal(studioCursor{
		Time: updatedAt.UTC().Format(time.RFC3339Nano),
		ID:   id.String(),
	})
	if err != nil {
		panic("content: encode studio cursor: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeStudioCursor(value string) (time.Time, uuid.UUID, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	var payload studioCursor
	if err := json.Unmarshal(raw, &payload); err != nil {
		return time.Time{}, uuid.Nil, err
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, payload.Time)
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	id, err := uuid.Parse(payload.ID)
	if err != nil || id == uuid.Nil {
		return time.Time{}, uuid.Nil, fmt.Errorf("content: invalid cursor id")
	}
	return updatedAt, id, nil
}
