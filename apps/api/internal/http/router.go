// Package http assembles the API HTTP router and its middleware stack.
package http

import (
	"log/slog"
	stdhttp "net/http"

	"github.com/go-chi/chi/v5"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/api"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/health"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/middleware"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/problem"
)

// NewRouter builds the full HTTP handler with the standard middleware stack.
func NewRouter(logger *slog.Logger, checkers ...health.Checker) stdhttp.Handler {
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
	r.Get("/readyz", health.Readyz(logger, checkers...))

	r.Route("/api/v1", func(r chi.Router) {
		apigen.HandlerFromMux(apigen.NewStrictHandlerWithOptions(api.NewServer(), nil, api.StrictOptions()), r)
	})

	return r
}
