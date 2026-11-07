package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/problem"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/reqctx"
)

// Recoverer turns panics into a logged 500 problem response. It keeps the
// standard net/http behavior for http.ErrAbortHandler.
func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				if recovered == http.ErrAbortHandler {
					panic(recovered)
				}

				logger.Error("panic recovered",
					"request_id", reqctx.RequestID(r.Context()),
					"panic", fmt.Sprint(recovered),
					"stack", string(debug.Stack()),
				)
				problem.Write(w, r, problem.Internal())
			}()

			next.ServeHTTP(w, r)
		})
	}
}
