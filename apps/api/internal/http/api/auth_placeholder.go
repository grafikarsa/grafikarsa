// Code below is a temporary placeholder: the auth endpoints that are not wired
// in yet return 501. Replace each method with a real implementation in auth.go
// and delete it from this file.
package api

import (
	"context"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
)

// PostAuthActivate is not implemented yet.
func (s *Server) PostAuthActivate(_ context.Context, _ apigen.PostAuthActivateRequestObject) (apigen.PostAuthActivateResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostAuthPasswordChange is not implemented yet.
func (s *Server) PostAuthPasswordChange(_ context.Context, _ apigen.PostAuthPasswordChangeRequestObject) (apigen.PostAuthPasswordChangeResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostAuthPasswordRecover is not implemented yet.
func (s *Server) PostAuthPasswordRecover(_ context.Context, _ apigen.PostAuthPasswordRecoverRequestObject) (apigen.PostAuthPasswordRecoverResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostAuthRecoveryCodesConfirm is not implemented yet.
func (s *Server) PostAuthRecoveryCodesConfirm(_ context.Context, _ apigen.PostAuthRecoveryCodesConfirmRequestObject) (apigen.PostAuthRecoveryCodesConfirmResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostAuthRecoveryCodesRegenerate is not implemented yet.
func (s *Server) PostAuthRecoveryCodesRegenerate(_ context.Context, _ apigen.PostAuthRecoveryCodesRegenerateRequestObject) (apigen.PostAuthRecoveryCodesRegenerateResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// GetAuthUsernameAvailability is not implemented yet.
func (s *Server) GetAuthUsernameAvailability(_ context.Context, _ apigen.GetAuthUsernameAvailabilityRequestObject) (apigen.GetAuthUsernameAvailabilityResponseObject, error) {
	return nil, apperr.NotImplemented()
}
