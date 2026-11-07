// Package api implements the server interfaces generated from
// packages/contracts/openapi.yaml.
package api

import (
	"context"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
)

var _ apigen.StrictServerInterface = (*Server)(nil)

// Server implements apigen.StrictServerInterface.
type Server struct{}

// NewServer returns a ready-to-use Server.
func NewServer() *Server {
	return &Server{}
}

// GetPing reports that the API is alive.
func (s *Server) GetPing(_ context.Context, _ apigen.GetPingRequestObject) (apigen.GetPingResponseObject, error) {
	return apigen.GetPing200JSONResponse{Pong: true}, nil
}
