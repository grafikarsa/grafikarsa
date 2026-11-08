package api

import (
	"context"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/content"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/authctx"
)

// studioActor builds the service actor for an authenticated Studio request.
func (s *Server) studioActor(ctx context.Context) (content.Actor, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return content.Actor{}, err
	}
	return content.Actor{Principal: principal, IP: authctx.MetaFrom(ctx).IP}, nil
}

// PostStudioContent creates an empty draft owned by the caller.
func (s *Server) PostStudioContent(ctx context.Context, request apigen.PostStudioContentRequestObject) (apigen.PostStudioContentResponseObject, error) {
	actor, err := s.studioActor(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	result, err := s.content.Create(ctx, actor, content.CreateInput{Type: string(request.Body.Type)})
	if err != nil {
		return nil, err
	}
	return apigen.PostStudioContent201JSONResponse(toStudioContent(result)), nil
}

// GetStudioContent returns the editor view of one content.
func (s *Server) GetStudioContent(ctx context.Context, request apigen.GetStudioContentRequestObject) (apigen.GetStudioContentResponseObject, error) {
	actor, err := s.studioActor(ctx)
	if err != nil {
		return nil, err
	}

	result, err := s.content.GetStudio(ctx, actor, uuid.UUID(request.Id))
	if err != nil {
		return nil, err
	}
	return apigen.GetStudioContent200JSONResponse(toStudioContent(result)), nil
}

// PutStudioContentDraft saves the whole editable draft with optimistic locking.
func (s *Server) PutStudioContentDraft(ctx context.Context, request apigen.PutStudioContentDraftRequestObject) (apigen.PutStudioContentDraftResponseObject, error) {
	actor, err := s.studioActor(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	input := content.DraftInput{
		LockVersion: request.Body.LockVersion,
		Title:       request.Body.Title,
		Excerpt:     request.Body.Excerpt,
		Body:        request.Body.Body,
		Meta:        request.Body.Meta,
	}
	if request.Body.CategoryId != nil {
		id := uuid.UUID(*request.Body.CategoryId)
		input.CategoryID = &id
	}
	if request.Body.CoverMediaId != nil {
		id := uuid.UUID(*request.Body.CoverMediaId)
		input.CoverMediaID = &id
	}
	if request.Body.Tags != nil {
		input.Tags = *request.Body.Tags
	}
	if request.Body.Tools != nil {
		input.Tools = *request.Body.Tools
	}
	if request.Body.Skills != nil {
		input.Skills = *request.Body.Skills
	}

	result, err := s.content.SaveDraft(ctx, actor, uuid.UUID(request.Id), input)
	if err != nil {
		return nil, err
	}
	return apigen.PutStudioContentDraft200JSONResponse(toStudioContent(result)), nil
}

// GetStudioContents lists contents the caller authors.
func (s *Server) GetStudioContents(ctx context.Context, request apigen.GetStudioContentsRequestObject) (apigen.GetStudioContentsResponseObject, error) {
	actor, err := s.studioActor(ctx)
	if err != nil {
		return nil, err
	}

	filter := content.ListFilter{Cursor: queryParam(request.Params.Cursor)}
	if request.Params.Status != nil {
		filter.Status = string(*request.Params.Status)
	}
	if request.Params.Type != nil {
		filter.Type = string(*request.Params.Type)
	}
	if request.Params.Limit != nil {
		filter.Limit = *request.Params.Limit
	}

	page, err := s.content.ListStudio(ctx, actor, filter)
	if err != nil {
		return nil, err
	}

	items := make([]apigen.StudioContentItem, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, toStudioContentItem(item))
	}
	var next *string
	if page.NextCursor != "" {
		next = &page.NextCursor
	}
	return apigen.GetStudioContents200JSONResponse{Items: items, NextCursor: next}, nil
}

// GetStudioSummary returns per-status counts and pending invitation count.
func (s *Server) GetStudioSummary(ctx context.Context, _ apigen.GetStudioSummaryRequestObject) (apigen.GetStudioSummaryResponseObject, error) {
	actor, err := s.studioActor(ctx)
	if err != nil {
		return nil, err
	}

	summary, err := s.content.Summary(ctx, actor)
	if err != nil {
		return nil, err
	}
	return apigen.GetStudioSummary200JSONResponse{
		Counts:             summary.Counts,
		PendingInvitations: summary.PendingInvitations,
	}, nil
}

func toStudioContent(c content.StudioContent) apigen.StudioContent {
	return apigen.StudioContent{
		Id:               c.ID,
		PublicId:         c.PublicID,
		Type:             apigen.ContentType(c.Type),
		Status:           apigen.ContentStatus(c.Status),
		RevisionStatus:   toRevisionStatus(c.RevisionStatus),
		Slug:             c.Slug,
		Title:            c.Title,
		MyLevel:          apigen.AuthorLevel(c.MyLevel),
		Draft:            toDraft(c.Draft),
		LastModeration:   toModerationFeedback(c.LastModeration),
		HasLiveVersion:   c.HasLiveVersion,
		InReviewClaimed:  c.InReviewClaimed,
		AuthorsCount:     c.AuthorsCount,
		FirstPublishedAt: c.FirstPublishedAt,
		ArchivedAt:       c.ArchivedAt,
		DeletedAt:        c.DeletedAt,
		PurgeAfter:       c.PurgeAfter,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}
}

func toStudioContentItem(item content.StudioContentItem) apigen.StudioContentItem {
	return apigen.StudioContentItem{
		Id:             item.ID,
		PublicId:       item.PublicID,
		Type:           apigen.ContentType(item.Type),
		Status:         apigen.ContentStatus(item.Status),
		RevisionStatus: toRevisionStatus(item.RevisionStatus),
		Title:          item.Title,
		Cover:          toImageRef(item.Cover),
		MyLevel:        apigen.AuthorLevel(item.MyLevel),
		AuthorsCount:   item.AuthorsCount,
		UpdatedAt:      item.UpdatedAt,
	}
}

func toDraft(draft content.Draft) apigen.Draft {
	body := draft.Body
	if body == nil {
		body = map[string]any{}
	}
	meta := draft.Meta
	if meta == nil {
		meta = map[string]any{}
	}
	return apigen.Draft{
		LockVersion: draft.LockVersion,
		Title:       draft.Title,
		Excerpt:     draft.Excerpt,
		CategoryId:  toOptionalOpenAPIUUID(draft.CategoryID),
		Cover:       toImageRef(draft.Cover),
		Body:        body,
		Meta:        meta,
		Tags:        toContentTaxonomyTerms(draft.Tags),
		Tools:       toContentTaxonomyTerms(draft.Tools),
		Skills:      toContentTaxonomyTerms(draft.Skills),
		UpdatedAt:   draft.UpdatedAt,
		UpdatedBy:   toAuthorMini(draft.UpdatedBy),
	}
}

func toContentTaxonomyTerms(terms []content.TaxonomyTerm) []apigen.TaxonomyTerm {
	items := make([]apigen.TaxonomyTerm, 0, len(terms))
	for _, term := range terms {
		items = append(items, apigen.TaxonomyTerm{
			Slug:     term.Slug,
			Name:     term.Name,
			Category: term.Category,
		})
	}
	return items
}

func toImageRef(ref *content.ImageRef) *apigen.ImageRef {
	if ref == nil {
		return nil
	}
	return &apigen.ImageRef{
		MediaId:       ref.MediaID,
		Url:           ref.URL,
		W:             ref.Width,
		H:             ref.Height,
		Blurhash:      ref.Blurhash,
		DominantColor: ref.DominantColor,
	}
}

func toAuthorMini(author *content.AuthorMini) *apigen.AuthorMini {
	if author == nil {
		return nil
	}
	return &apigen.AuthorMini{
		UserId:      author.UserID,
		Username:    author.Username,
		DisplayName: author.DisplayName,
		AvatarUrl:   author.AvatarURL,
	}
}

func toModerationFeedback(feedback *content.ModerationFeedback) *apigen.ModerationFeedback {
	if feedback == nil {
		return nil
	}
	annotations := make([]apigen.ModerationAnnotation, 0, len(feedback.Annotations))
	for _, annotation := range feedback.Annotations {
		annotations = append(annotations, apigen.ModerationAnnotation{
			TargetRef: annotation.TargetRef,
			Note:      annotation.Note,
		})
	}
	var appealStatus *apigen.ModerationFeedbackAppealStatus
	if feedback.AppealStatus != nil {
		converted := apigen.ModerationFeedbackAppealStatus(*feedback.AppealStatus)
		appealStatus = &converted
	}
	return &apigen.ModerationFeedback{
		Decision:     apigen.ModerationFeedbackDecision(feedback.Decision),
		ReasonLabel:  feedback.ReasonLabel,
		Note:         feedback.Note,
		Annotations:  annotations,
		DecidedAt:    feedback.DecidedAt,
		CanAppeal:    feedback.CanAppeal,
		AppealStatus: appealStatus,
	}
}

func toRevisionStatus(status *string) *apigen.RevisionStatus {
	if status == nil {
		return nil
	}
	converted := apigen.RevisionStatus(*status)
	return &converted
}
