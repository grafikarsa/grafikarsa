// Code below is a temporary placeholder: every auth endpoint returns 501 until
// the auth domain is wired in. Replace each method with a real implementation
// in auth.go and delete it from this file.
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

// GetAuthCsrf is not implemented yet.
func (s *Server) GetAuthCsrf(_ context.Context, _ apigen.GetAuthCsrfRequestObject) (apigen.GetAuthCsrfResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostAuthLogin is not implemented yet.
func (s *Server) PostAuthLogin(_ context.Context, _ apigen.PostAuthLoginRequestObject) (apigen.PostAuthLoginResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostAuthLogout is not implemented yet.
func (s *Server) PostAuthLogout(_ context.Context, _ apigen.PostAuthLogoutRequestObject) (apigen.PostAuthLogoutResponseObject, error) {
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

// GetAuthSession is not implemented yet.
func (s *Server) GetAuthSession(_ context.Context, _ apigen.GetAuthSessionRequestObject) (apigen.GetAuthSessionResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// GetAuthSessions is not implemented yet.
func (s *Server) GetAuthSessions(_ context.Context, _ apigen.GetAuthSessionsRequestObject) (apigen.GetAuthSessionsResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// PostAuthSessionsRevokeOthers is not implemented yet.
func (s *Server) PostAuthSessionsRevokeOthers(_ context.Context, _ apigen.PostAuthSessionsRevokeOthersRequestObject) (apigen.PostAuthSessionsRevokeOthersResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// DeleteAuthSession is not implemented yet.
func (s *Server) DeleteAuthSession(_ context.Context, _ apigen.DeleteAuthSessionRequestObject) (apigen.DeleteAuthSessionResponseObject, error) {
	return nil, apperr.NotImplemented()
}

// GetAuthUsernameAvailability is not implemented yet.
func (s *Server) GetAuthUsernameAvailability(_ context.Context, _ apigen.GetAuthUsernameAvailabilityRequestObject) (apigen.GetAuthUsernameAvailabilityResponseObject, error) {
	return nil, apperr.NotImplemented()
}
