package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"grafikarsa/api/internal/cache"
	"grafikarsa/api/internal/db"
	"grafikarsa/api/internal/storage"
)

type Deps struct {
	Pool    *pgxpool.Pool
	Cache   *cache.Cache
	Storage *storage.Storage
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func ok(data any, r *http.Request) map[string]any {
	return map[string]any{
		"success":    true,
		"data":       data,
		"request_id": middleware.GetReqID(r.Context()),
	}
}

func fail(code, message string, r *http.Request, status int) map[string]any {
	return map[string]any{
		"success": false,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
		"request_id": middleware.GetReqID(r.Context()),
	}
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
	})

	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		checks := map[string]string{}
		status := http.StatusOK
		if err := db.Health(r.Context(), d.Pool); err != nil {
			checks["db"] = "error"
			status = http.StatusServiceUnavailable
		} else {
			checks["db"] = "ok"
		}
		if err := d.Cache.Ping(r.Context()); err != nil {
			checks["dragonfly"] = "error"
			status = http.StatusServiceUnavailable
		} else {
			checks["dragonfly"] = "ok"
		}
		if err := d.Storage.Health(r.Context()); err != nil {
			checks["storage"] = "error"
			status = http.StatusServiceUnavailable
		} else {
			checks["storage"] = "ok"
		}
		if status != http.StatusOK {
			writeJSON(w, status, fail("SERVICE_UNAVAILABLE", "Salah satu dependensi belum siap.", r, status))
			return
		}
		writeJSON(w, status, ok(checks, r))
	})

	r.Route("/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, ok(map[string]string{"status": "ok"}, r))
		})
	})
	return r
}
