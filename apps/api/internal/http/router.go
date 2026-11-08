// Package http assembles the API HTTP router and its middleware stack.
package http

import (
	"log/slog"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/academic"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/roles"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/users"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/api"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/authctx"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/health"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/middleware"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/problem"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
)

// Rate limit applied to every API request: 600 requests per minute per user
// (or per IP for anonymous requests).
const generalRateLimit = 600

// RouterDeps carries the services and settings the router needs.
type RouterDeps struct {
	Logger         *slog.Logger
	Checkers       []health.Checker
	Auth           *auth.Service
	Academic       *academic.Service
	Users          *users.Service
	Roles          *roles.Service
	Audit          *audit.Service
	Limiter        *ratelimit.Limiter
	AllowedOrigins []string
}

// NewRouter builds the full HTTP handler with the standard middleware stack.
func NewRouter(d RouterDeps) stdhttp.Handler {
	logger := d.Logger
	if logger == nil {
		logger = slog.Default()
	}

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer(logger))
	r.Use(middleware.AccessLog(logger))

	r.NotFound(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		problem.Write(w, r, problem.NotFound())
	})
	r.MethodNotAllowed(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		problem.Write(w, r, problem.MethodNotAllowed())
	})

	r.Get("/healthz", health.Healthz())
	r.Get("/readyz", health.Readyz(logger, d.Checkers...))

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.CookieJar)
		if d.Auth != nil {
			r.Use(middleware.Session(d.Auth, logger))
		}
		r.Use(middleware.CSRF(middleware.CSRFOptions{AllowedOrigins: d.AllowedOrigins}))
		if d.Limiter != nil {
			r.Use(middleware.RateLimit(d.Limiter, middleware.RateLimitOptions{
				Name: "api",
				Rule: ratelimit.Rule{Limit: generalRateLimit, Window: time.Minute},
				Key: func(r *stdhttp.Request) string {
					if principal := authctx.Principal(r.Context()); principal != nil {
						return principal.UserID.String()
					}
					return "ip:" + middleware.ClientIP(r)
				},
				FailOpen: true,
				Logger:   logger,
			}))
		}
		r.Use(middleware.Onboarding(logger))
		r.Use(noStoreAdminResponses)

		apigen.HandlerWithOptions(apigen.NewStrictHandlerWithOptions(api.NewServer(api.Deps{
			Auth:     d.Auth,
			Academic: d.Academic,
			Users:    d.Users,
			Roles:    d.Roles,
			Audit:    d.Audit,
			Logger:   logger,
		}), nil, api.StrictOptions(logger)), apigen.ChiServerOptions{
			BaseRouter: r,
			// Parameter binding errors (bad path/query values) must also be problem+json.
			ErrorHandlerFunc: api.StrictOptions(logger).RequestErrorHandlerFunc,
		})
	})

	return r
}

// noStoreAdminResponses prevents browsers and proxies from caching admin
// responses, which can carry one-time passwords.
func noStoreAdminResponses(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if r.URL.Path == "/api/v1/admin" || strings.HasPrefix(r.URL.Path, "/api/v1/admin/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}
