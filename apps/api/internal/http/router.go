package http

import (
	"log/slog"
	"net/http"
	"time"

	"grafikarsa/api/internal/observability"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func baseMiddleware() *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	// Tracing OpenTelemetry (PRD §13.1): span per request. Tracing no-op
	// kalau collector tidak dikonfigurasi, jadi tidak ada overhead berarti.
	r.Use(observability.TraceMiddleware)
	r.Use(requestLogger)
	r.Use(middleware.Timeout(30 * time.Second))
	return r
}

// requestLogger: satu baris JSON per request (request_id, method, path,
// status, durasi) tanpa mencatat body/PII.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Jam sistem (bukan App.Clock) sengaja dipakai di sini: yang
		// diukur adalah durasi request yang sebenarnya.
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		observability.Record(r.URL.Path, ww.Status(), time.Since(start))
		slog.Info("http",
			"id", middleware.GetReqID(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"ms", time.Since(start).Milliseconds(),
		)
	})
}

var _ = http.MethodGet
