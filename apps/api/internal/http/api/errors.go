package api

import (
	stdhttp "net/http"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/problem"
)

// StrictOptions returns strict-server error handlers that render RFC 9457
// problem responses instead of the generated defaults.
func StrictOptions() apigen.StrictHTTPServerOptions {
	return apigen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w stdhttp.ResponseWriter, r *stdhttp.Request, err error) {
			problem.Write(w, r, problem.Problem{
				Status: stdhttp.StatusBadRequest,
				Code:   "BAD_REQUEST",
				Title:  "Permintaan tidak valid",
				Detail: err.Error(),
			})
		},
		ResponseErrorHandlerFunc: func(w stdhttp.ResponseWriter, r *stdhttp.Request, _ error) {
			problem.Write(w, r, problem.Internal())
		},
	}
}
