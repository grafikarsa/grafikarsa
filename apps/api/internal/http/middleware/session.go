package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/authctx"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/problem"
)

// SessionCookieName is the HttpOnly session cookie. The __Host- prefix
// requires Secure, Path=/ and no Domain.
const SessionCookieName = "__Host-gk_session"

// SessionCookie builds the cookie that carries a session token.
func SessionCookie(token string, expires time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expires,
		MaxAge:   maxAgeUntil(expires),
	}
}

// ClearSessionCookie builds the cookie that deletes the session cookie.
func ClearSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	}
}

// SessionAuthenticator resolves an opaque session token to a principal.
type SessionAuthenticator interface {
	Authenticate(ctx context.Context, token string) (*auth.Principal, error)
}

// Session attaches the principal behind the session cookie to the request
// context. A missing or rejected cookie continues anonymously; an unexpected
// authentication failure responds 503 instead of silently downgrading.
func Session(svc SessionAuthenticator, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(SessionCookieName)
			if err != nil || cookie.Value == "" {
				next.ServeHTTP(w, r)
				return
			}

			principal, err := svc.Authenticate(r.Context(), cookie.Value)
			if err != nil {
				if apperr.Is(err, "UNAUTHENTICATED") {
					authctx.SetCookie(r.Context(), ClearSessionCookie())
					next.ServeHTTP(w, r)
					return
				}

				logger.ErrorContext(r.Context(), "session authentication failed", "error", err)
				problem.Write(w, r, problem.Unavailable("sesi"))
				return
			}

			next.ServeHTTP(w, r.WithContext(authctx.WithPrincipal(r.Context(), principal)))
		})
	}
}

// maxAgeUntil returns the whole seconds from now until expires, never negative.
func maxAgeUntil(expires time.Time) int {
	seconds := int(time.Until(expires).Seconds())
	if seconds < 0 {
		return 0
	}
	return seconds
}
