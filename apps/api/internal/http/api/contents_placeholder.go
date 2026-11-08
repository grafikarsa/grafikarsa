// Code below is a temporary placeholder: content/studio endpoints return 501 until
// the content domain is wired in (M4). Replace each method with a real
// implementation in contents.go / studio.go and delete it from this file.
package api

import (
	"context"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
)

// GetContents is not implemented yet.
func (s *Server) GetContents(_ context.Context, _ apigen.GetContentsRequestObject) (apigen.GetContentsResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// GetContent is not implemented yet.
func (s *Server) GetContent(_ context.Context, _ apigen.GetContentRequestObject) (apigen.GetContentResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostStudioContentAppeal is not implemented yet.
func (s *Server) PostStudioContentAppeal(_ context.Context, _ apigen.PostStudioContentAppealRequestObject) (apigen.PostStudioContentAppealResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// GetStudioContentAuthors is not implemented yet.
func (s *Server) GetStudioContentAuthors(_ context.Context, _ apigen.GetStudioContentAuthorsRequestObject) (apigen.GetStudioContentAuthorsResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostStudioContentAuthor is not implemented yet.
func (s *Server) PostStudioContentAuthor(_ context.Context, _ apigen.PostStudioContentAuthorRequestObject) (apigen.PostStudioContentAuthorResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// DeleteStudioContentAuthor is not implemented yet.
func (s *Server) DeleteStudioContentAuthor(_ context.Context, _ apigen.DeleteStudioContentAuthorRequestObject) (apigen.DeleteStudioContentAuthorResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PatchStudioContentAuthor is not implemented yet.
func (s *Server) PatchStudioContentAuthor(_ context.Context, _ apigen.PatchStudioContentAuthorRequestObject) (apigen.PatchStudioContentAuthorResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostStudioContentLeave is not implemented yet.
func (s *Server) PostStudioContentLeave(_ context.Context, _ apigen.PostStudioContentLeaveRequestObject) (apigen.PostStudioContentLeaveResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostStudioContentTransferOwnership is not implemented yet.
func (s *Server) PostStudioContentTransferOwnership(_ context.Context, _ apigen.PostStudioContentTransferOwnershipRequestObject) (apigen.PostStudioContentTransferOwnershipResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// GetStudioInvitations is not implemented yet.
func (s *Server) GetStudioInvitations(_ context.Context, _ apigen.GetStudioInvitationsRequestObject) (apigen.GetStudioInvitationsResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostStudioInvitationAccept is not implemented yet.
func (s *Server) PostStudioInvitationAccept(_ context.Context, _ apigen.PostStudioInvitationAcceptRequestObject) (apigen.PostStudioInvitationAcceptResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostStudioInvitationDecline is not implemented yet.
func (s *Server) PostStudioInvitationDecline(_ context.Context, _ apigen.PostStudioInvitationDeclineRequestObject) (apigen.PostStudioInvitationDeclineResponseObject, error) {
	return nil, apperr.NotImplemented()
}
