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

// DeleteStudioContent is not implemented yet.
func (s *Server) DeleteStudioContent(_ context.Context, _ apigen.DeleteStudioContentRequestObject) (apigen.DeleteStudioContentResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostStudioContentAppeal is not implemented yet.
func (s *Server) PostStudioContentAppeal(_ context.Context, _ apigen.PostStudioContentAppealRequestObject) (apigen.PostStudioContentAppealResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostStudioContentArchive is not implemented yet.
func (s *Server) PostStudioContentArchive(_ context.Context, _ apigen.PostStudioContentArchiveRequestObject) (apigen.PostStudioContentArchiveResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostStudioContentRestore is not implemented yet.
func (s *Server) PostStudioContentRestore(_ context.Context, _ apigen.PostStudioContentRestoreRequestObject) (apigen.PostStudioContentRestoreResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// DeleteStudioContentRevision is not implemented yet.
func (s *Server) DeleteStudioContentRevision(_ context.Context, _ apigen.DeleteStudioContentRevisionRequestObject) (apigen.DeleteStudioContentRevisionResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostStudioContentRevision is not implemented yet.
func (s *Server) PostStudioContentRevision(_ context.Context, _ apigen.PostStudioContentRevisionRequestObject) (apigen.PostStudioContentRevisionResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostStudioContentSubmit is not implemented yet.
func (s *Server) PostStudioContentSubmit(_ context.Context, _ apigen.PostStudioContentSubmitRequestObject) (apigen.PostStudioContentSubmitResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostStudioContentUnarchive is not implemented yet.
func (s *Server) PostStudioContentUnarchive(_ context.Context, _ apigen.PostStudioContentUnarchiveRequestObject) (apigen.PostStudioContentUnarchiveResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// GetStudioContentVersions is not implemented yet.
func (s *Server) GetStudioContentVersions(_ context.Context, _ apigen.GetStudioContentVersionsRequestObject) (apigen.GetStudioContentVersionsResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// GetStudioContentVersion is not implemented yet.
func (s *Server) GetStudioContentVersion(_ context.Context, _ apigen.GetStudioContentVersionRequestObject) (apigen.GetStudioContentVersionResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostStudioContentWithdraw is not implemented yet.
func (s *Server) PostStudioContentWithdraw(_ context.Context, _ apigen.PostStudioContentWithdrawRequestObject) (apigen.PostStudioContentWithdrawResponseObject, error) {
	return nil, apperr.NotImplemented()
}
