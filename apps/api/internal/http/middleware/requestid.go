// Package middleware contains the HTTP middleware stack used by the API.
package middleware

import (
	"net/http"
	"regexp"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/reqctx"
)

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

// RequestID reuses a valid incoming X-Request-Id or generates a new one. The
// chosen ID is stored in the request context and echoed in the response.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if !requestIDPattern.MatchString(id) {
			id = uuid.Must(uuid.NewV7()).String()
		}

		w.Header().Set("X-Request-Id", id)
		ctx := reqctx.WithRequestID(r.Context(), id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
