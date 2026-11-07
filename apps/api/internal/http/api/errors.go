package api

import (
	"log/slog"
	stdhttp "net/http"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/apigen"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/problem"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/reqctx"
)

// StrictOptions returns strict-server error handlers that render RFC 9457
// problem responses instead of the generated defaults.
func StrictOptions(logger *slog.Logger) apigen.StrictHTTPServerOptions {
	return apigen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w stdhttp.ResponseWriter, r *stdhttp.Request, err error) {
			problem.Write(w, r, problem.Problem{
				Status: stdhttp.StatusBadRequest,
				Code:   "BAD_REQUEST",
				Title:  "Permintaan tidak valid",
				Detail: err.Error(),
			})
		},
		ResponseErrorHandlerFunc: func(w stdhttp.ResponseWriter, r *stdhttp.Request, err error) {
			WriteError(w, r, logger, err)
		},
	}
}

// WriteError renders err as a problem response. *apperr.Error values are
// mapped to their status and code; anything else becomes a logged 500.
func WriteError(w stdhttp.ResponseWriter, r *stdhttp.Request, logger *slog.Logger, err error) {
	ae, ok := apperr.As(err)
	if !ok {
		logger.Error("unhandled error", "error", err, "request_id", reqctx.RequestID(r.Context()),
			"method", r.Method, "path", r.URL.Path)
		problem.Write(w, r, problem.Internal())
		return
	}
	if ae.Cause != nil {
		logger.Warn("request failed", "code", ae.Code, "error", ae.Cause,
			"request_id", reqctx.RequestID(r.Context()))
	}
	p := problem.Problem{Status: ae.Status, Code: ae.Code, Title: ae.Title, Detail: ae.Detail}
	for _, f := range ae.Fields {
		p.Errors = append(p.Errors, problem.FieldError{Field: f.Field, Ref: f.Ref, Code: f.Code, Message: f.Message})
	}
	problem.Write(w, r, p)
}
