// Package api implements the server interfaces generated from
// packages/contracts/openapi.yaml.
package api

import (
	"context"
	"log/slog"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/academic"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/audit"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/media"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/roles"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/taxonomy"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/users"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
)

var _ apigen.StrictServerInterface = (*Server)(nil)

// Deps carries the services the handlers need.
type Deps struct {
	Auth     *auth.Service
	Academic *academic.Service
	Users    *users.Service
	Imports  *users.Importer
	Roles    *roles.Service
	Audit    *audit.Service
	Media    *media.Service
	Taxonomy *taxonomy.Service
	Logger   *slog.Logger
}

// Server implements apigen.StrictServerInterface.
type Server struct {
	auth     *auth.Service
	academic *academic.Service
	users    *users.Service
	imports  *users.Importer
	roles    *roles.Service
	audit    *audit.Service
	media    *media.Service
	taxonomy *taxonomy.Service
	logger   *slog.Logger
}

// NewServer returns a ready-to-use Server.
func NewServer(d Deps) *Server {
	logger := d.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Server{
		auth:     d.Auth,
		academic: d.Academic,
		users:    d.Users,
		imports:  d.Imports,
		roles:    d.Roles,
		audit:    d.Audit,
		media:    d.Media,
		taxonomy: d.Taxonomy,
		logger:   logger,
	}
}

// GetPing reports that the API is alive.
func (s *Server) GetPing(_ context.Context, _ apigen.GetPingRequestObject) (apigen.GetPingResponseObject, error) {
	return apigen.GetPing200JSONResponse{Pong: true}, nil
}
