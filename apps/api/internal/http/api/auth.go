package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/authctx"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/middleware"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/securetoken"
)

// requirePrincipal returns the authenticated principal attached by the
// session middleware.
func requirePrincipal(ctx context.Context) (*auth.Principal, error) {
	principal := authctx.Principal(ctx)
	if principal == nil {
		return nil, apperr.Unauthenticated()
	}
	return principal, nil
}

// toSession maps the domain principal to the API session representation.
func toSession(p *auth.Principal) apigen.Session {
	grants := make([]apigen.RoleGrant, 0, len(p.Grants))
	for _, grant := range p.Grants {
		majorIDs := make([]openapi_types.UUID, len(grant.MajorIDs))
		copy(majorIDs, grant.MajorIDs)
		grants = append(grants, apigen.RoleGrant{
			Global:     grant.Global,
			MajorIds:   majorIDs,
			Permission: grant.Permission,
		})
	}

	isMigrated := p.IsMigrated

	return apigen.Session{
		ExpiresAt:         p.ExpiresAt,
		Grants:            grants,
		IsMigratedAccount: &isMigrated,
		NextStep:          apigen.NextStep(p.NextStep),
		User: apigen.SessionUser{
			AccountType: apigen.AccountType(p.AccountType),
			DisplayName: p.DisplayName,
			Id:          p.UserID,
			Status:      apigen.AccountStatus(p.Status),
			Username:    p.Username,
		},
	}
}

// csrfCookie mirrors middleware.SetCSRFCookie so a token queued through the
// request jar carries the same attributes as one written directly.
func csrfCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     middleware.CSRFCookieName,
		Value:    token,
		Path:     "/",
		Secure:   true,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	}
}

// GetAuthCsrf sets the CSRF cookie and returns its token.
func (s *Server) GetAuthCsrf(ctx context.Context, _ apigen.GetAuthCsrfRequestObject) (apigen.GetAuthCsrfResponseObject, error) {
	token, err := securetoken.CSRFToken()
	if err != nil {
		return nil, fmt.Errorf("api: generate csrf token: %w", err)
	}

	authctx.SetCookie(ctx, csrfCookie(token))

	return apigen.GetAuthCsrf200JSONResponse{CsrfToken: token}, nil
}

// PostAuthLogin authenticates a user, sets the session cookie, and rotates the
// CSRF token.
func (s *Server) PostAuthLogin(ctx context.Context, request apigen.PostAuthLoginRequestObject) (apigen.PostAuthLoginResponseObject, error) {
	if request.Body == nil {
		return nil, apperr.BadRequest("Body permintaan kosong")
	}

	// Generate the rotated token before creating a session so a failure here
	// cannot leave a usable session without its cookie.
	csrfToken, err := securetoken.CSRFToken()
	if err != nil {
		return nil, fmt.Errorf("api: generate csrf token: %w", err)
	}

	meta := authctx.MetaFrom(ctx)
	result, err := s.auth.Login(ctx, auth.LoginInput{
		Identifier: request.Body.Identifier,
		Password:   request.Body.Password,
		IP:         meta.IP,
		UserAgent:  meta.UserAgent,
	})
	if err != nil {
		return nil, err
	}

	authctx.SetCookie(ctx, middleware.SessionCookie(result.Token, result.Principal.ExpiresAt))
	authctx.SetCookie(ctx, csrfCookie(csrfToken))

	return apigen.PostAuthLogin200JSONResponse(toSession(result.Principal)), nil
}

// PostAuthLogout revokes the current session and clears the session cookie.
func (s *Server) PostAuthLogout(ctx context.Context, _ apigen.PostAuthLogoutRequestObject) (apigen.PostAuthLogoutResponseObject, error) {
	if principal := authctx.Principal(ctx); principal != nil {
		if err := s.auth.Logout(ctx, principal); err != nil {
			return nil, err
		}
	}

	authctx.SetCookie(ctx, middleware.ClearSessionCookie())

	return apigen.PostAuthLogout204Response{}, nil
}

// GetAuthSession returns the current session.
func (s *Server) GetAuthSession(ctx context.Context, _ apigen.GetAuthSessionRequestObject) (apigen.GetAuthSessionResponseObject, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}

	return apigen.GetAuthSession200JSONResponse(toSession(principal)), nil
}

// GetAuthSessions lists the principal's active sessions.
func (s *Server) GetAuthSessions(ctx context.Context, _ apigen.GetAuthSessionsRequestObject) (apigen.GetAuthSessionsResponseObject, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}

	devices, err := s.auth.ListSessions(ctx, principal)
	if err != nil {
		return nil, err
	}

	items := make([]apigen.SessionDevice, 0, len(devices))
	for _, device := range devices {
		items = append(items, apigen.SessionDevice{
			CreatedAt:  device.CreatedAt,
			Current:    device.Current,
			Id:         device.ID,
			IpPrefix:   device.IPPrefix,
			LastSeenAt: device.LastSeenAt,
			UserAgent:  device.UserAgent,
		})
	}

	return apigen.GetAuthSessions200JSONResponse{Items: items}, nil
}

// DeleteAuthSession revokes one of the principal's sessions.
func (s *Server) DeleteAuthSession(ctx context.Context, request apigen.DeleteAuthSessionRequestObject) (apigen.DeleteAuthSessionResponseObject, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.auth.RevokeSession(ctx, principal, uuid.UUID(request.Id)); err != nil {
		return nil, err
	}

	return apigen.DeleteAuthSession204Response{}, nil
}

// PostAuthSessionsRevokeOthers revokes every other session of the principal.
func (s *Server) PostAuthSessionsRevokeOthers(ctx context.Context, _ apigen.PostAuthSessionsRevokeOthersRequestObject) (apigen.PostAuthSessionsRevokeOthersResponseObject, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.auth.RevokeOtherSessions(ctx, principal); err != nil {
		return nil, err
	}

	return apigen.PostAuthSessionsRevokeOthers204Response{}, nil
}
