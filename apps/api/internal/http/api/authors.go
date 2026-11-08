package api

import (
	"context"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/content"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
)

// GetStudioContentAuthors lists accepted authors and pending invitations of a
// content. Any accepted author may see the list.
func (s *Server) GetStudioContentAuthors(ctx context.Context, request apigen.GetStudioContentAuthorsRequestObject) (apigen.GetStudioContentAuthorsResponseObject, error) {
	actor, err := s.studioActor(ctx)
	if err != nil {
		return nil, err
	}

	authors, err := s.content.ListAuthors(ctx, actor, uuid.UUID(request.Id))
	if err != nil {
		return nil, err
	}
	return apigen.GetStudioContentAuthors200JSONResponse(toContentAuthorList(authors)), nil
}

// PostStudioContentAuthor invites a collaborator. Only the owner may invite.
func (s *Server) PostStudioContentAuthor(ctx context.Context, request apigen.PostStudioContentAuthorRequestObject) (apigen.PostStudioContentAuthorResponseObject, error) {
	actor, err := s.studioActor(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	invited, err := s.content.InviteAuthor(ctx, actor, uuid.UUID(request.Id), content.AuthorInviteInput{
		UserID:    uuid.UUID(request.Body.UserId),
		Level:     string(request.Body.Level),
		RoleLabel: request.Body.RoleLabel,
	})
	if err != nil {
		return nil, err
	}
	return apigen.PostStudioContentAuthor201JSONResponse(toContentAuthor(invited)), nil
}

// PatchStudioContentAuthor updates the level, role label, or position of an
// author. Only the owner may do it.
func (s *Server) PatchStudioContentAuthor(ctx context.Context, request apigen.PatchStudioContentAuthorRequestObject) (apigen.PatchStudioContentAuthorResponseObject, error) {
	actor, err := s.studioActor(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	input := content.AuthorUpdateInput{
		RoleLabel: request.Body.RoleLabel,
		Position:  request.Body.Position,
	}
	if request.Body.Level != nil {
		level := string(*request.Body.Level)
		input.Level = &level
	}

	updated, err := s.content.UpdateAuthor(ctx, actor, uuid.UUID(request.Id), uuid.UUID(request.UserId), input)
	if err != nil {
		return nil, err
	}
	return apigen.PatchStudioContentAuthor200JSONResponse(toContentAuthor(updated)), nil
}

// DeleteStudioContentAuthor removes an author or cancels an invitation. Only
// the owner may do it.
func (s *Server) DeleteStudioContentAuthor(ctx context.Context, request apigen.DeleteStudioContentAuthorRequestObject) (apigen.DeleteStudioContentAuthorResponseObject, error) {
	actor, err := s.studioActor(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.content.RemoveAuthor(ctx, actor, uuid.UUID(request.Id), uuid.UUID(request.UserId)); err != nil {
		return nil, err
	}
	return apigen.DeleteStudioContentAuthor204Response{}, nil
}

// PostStudioContentTransferOwnership moves ownership to an accepted author.
func (s *Server) PostStudioContentTransferOwnership(ctx context.Context, request apigen.PostStudioContentTransferOwnershipRequestObject) (apigen.PostStudioContentTransferOwnershipResponseObject, error) {
	actor, err := s.studioActor(ctx)
	if err != nil {
		return nil, err
	}
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	authors, err := s.content.TransferOwnership(ctx, actor, uuid.UUID(request.Id), uuid.UUID(request.Body.UserId))
	if err != nil {
		return nil, err
	}
	return apigen.PostStudioContentTransferOwnership200JSONResponse(toContentAuthorList(authors)), nil
}

// PostStudioContentLeave removes the caller from the author list.
func (s *Server) PostStudioContentLeave(ctx context.Context, request apigen.PostStudioContentLeaveRequestObject) (apigen.PostStudioContentLeaveResponseObject, error) {
	actor, err := s.studioActor(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.content.LeaveContent(ctx, actor, uuid.UUID(request.Id)); err != nil {
		return nil, err
	}
	return apigen.PostStudioContentLeave204Response{}, nil
}

// GetStudioInvitations lists pending collaboration invitations of the caller.
func (s *Server) GetStudioInvitations(ctx context.Context, request apigen.GetStudioInvitationsRequestObject) (apigen.GetStudioInvitationsResponseObject, error) {
	actor, err := s.studioActor(ctx)
	if err != nil {
		return nil, err
	}

	page, err := s.content.ListInvitations(ctx, actor, queryParam(request.Params.Cursor))
	if err != nil {
		return nil, err
	}

	items := make([]apigen.Invitation, 0, len(page.Items))
	for _, invitation := range page.Items {
		items = append(items, toInvitation(invitation))
	}
	var next *string
	if page.NextCursor != "" {
		next = &page.NextCursor
	}
	return apigen.GetStudioInvitations200JSONResponse{Items: items, NextCursor: next}, nil
}

// PostStudioInvitationAccept accepts a pending invitation.
func (s *Server) PostStudioInvitationAccept(ctx context.Context, request apigen.PostStudioInvitationAcceptRequestObject) (apigen.PostStudioInvitationAcceptResponseObject, error) {
	actor, err := s.studioActor(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.content.AcceptInvitation(ctx, actor, uuid.UUID(request.ContentId)); err != nil {
		return nil, err
	}
	return apigen.PostStudioInvitationAccept204Response{}, nil
}

// PostStudioInvitationDecline declines a pending invitation.
func (s *Server) PostStudioInvitationDecline(ctx context.Context, request apigen.PostStudioInvitationDeclineRequestObject) (apigen.PostStudioInvitationDeclineResponseObject, error) {
	actor, err := s.studioActor(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.content.DeclineInvitation(ctx, actor, uuid.UUID(request.ContentId)); err != nil {
		return nil, err
	}
	return apigen.PostStudioInvitationDecline204Response{}, nil
}

func toContentAuthorList(authors []content.ContentAuthorView) apigen.ContentAuthorList {
	items := make([]apigen.ContentAuthor, 0, len(authors))
	for _, author := range authors {
		items = append(items, toContentAuthor(author))
	}
	return apigen.ContentAuthorList{Items: items}
}

func toContentAuthor(author content.ContentAuthorView) apigen.ContentAuthor {
	return apigen.ContentAuthor{
		User:            *toAuthorMini(&author.User),
		Level:           apigen.AuthorLevel(author.Level),
		RoleLabel:       author.RoleLabel,
		Position:        author.Position,
		InviteStatus:    apigen.InviteStatus(author.InviteStatus),
		InvitedAt:       author.InvitedAt,
		InviteExpiresAt: author.InviteExpiresAt,
	}
}

func toInvitation(invitation content.Invitation) apigen.Invitation {
	return apigen.Invitation{
		Content: apigen.InvitationContent{
			Id:       invitation.Content.ID,
			PublicId: invitation.Content.PublicID,
			Type:     apigen.ContentType(invitation.Content.Type),
			Title:    invitation.Content.Title,
			Cover:    toImageRef(invitation.Content.Cover),
		},
		InvitedBy: *toAuthorMini(&invitation.InvitedBy),
		Level:     apigen.AuthorLevel(invitation.Level),
		RoleLabel: invitation.RoleLabel,
		InvitedAt: invitation.InvitedAt,
		ExpiresAt: invitation.ExpiresAt,
	}
}
