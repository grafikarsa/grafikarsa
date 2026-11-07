package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/problem"
)

const (
	// CSRFCookieName is the double-submit cookie name. The __Host- prefix
	// requires a Secure cookie served from the root path with no Domain.
	CSRFCookieName = "__Host-gk_csrf"
	// CSRFHeaderName is the header the browser JS must echo from the cookie.
	CSRFHeaderName = "X-CSRF-Token"
)

// CSRFOptions configures the CSRF middleware.
type CSRFOptions struct {
	// AllowedOrigins: if a request carries an Origin header, it must be one of these
	// (exact match, e.g. "https://grafikarsa.com"). Empty = skip the Origin check.
	AllowedOrigins []string
	// ExemptPaths: exact request paths that skip the check (e.g. view beacons).
	ExemptPaths []string
}

// CSRF enforces the double-submit cookie pattern for unsafe HTTP methods. Safe
// methods pass through; everything else needs a cookie/header token match and,
// when configured, an allowed Origin.
func CSRF(opts CSRFOptions) func(http.Handler) http.Handler {
	exemptPaths := make(map[string]struct{}, len(opts.ExemptPaths))
	for _, path := range opts.ExemptPaths {
		exemptPaths[path] = struct{}{}
	}

	allowedOrigins := make(map[string]struct{}, len(opts.AllowedOrigins))
	for _, origin := range opts.AllowedOrigins {
		allowedOrigins[origin] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			if _, ok := exemptPaths[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}

			if origin := r.Header.Get("Origin"); origin != "" && len(allowedOrigins) > 0 {
				if _, ok := allowedOrigins[origin]; !ok {
					writeCSRFFailed(w, r)
					return
				}
			}

			cookie, err := r.Cookie(CSRFCookieName)
			if err != nil || cookie.Value == "" {
				writeCSRFFailed(w, r)
				return
			}

			header := r.Header.Get(CSRFHeaderName)
			if header == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(header)) != 1 {
				writeCSRFFailed(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SetCSRFCookie writes the CSRF token cookie. The cookie is readable by
// JavaScript (HttpOnly false) so the client can echo it in the header.
func SetCSRFCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		Secure:   true,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   0,
	})
}

func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}

func writeCSRFFailed(w http.ResponseWriter, r *http.Request) {
	problem.Write(w, r, problem.Problem{
		Status: http.StatusForbidden,
		Code:   "CSRF_FAILED",
		Title:  "Permintaan ditolak (token CSRF tidak valid)",
	})
}
