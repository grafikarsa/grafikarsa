// Package health provides liveness and readiness HTTP handlers.
package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

const readinessTimeout = 2 * time.Second

// Checker verifies that a dependency is available.
type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

// CheckerFunc adapts a function into a named Checker.
type CheckerFunc struct {
	N string
	F func(context.Context) error
}

// Name returns the checker name.
func (c CheckerFunc) Name() string {
	return c.N
}

// Check runs the checker function.
func (c CheckerFunc) Check(ctx context.Context) error {
	return c.F(ctx)
}

// Healthz reports that the process is alive.
func Healthz() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// Readyz checks every dependency concurrently and reports whether the API is
// ready to serve traffic. Dependency error details are logged, never returned.
func Readyz(logger *slog.Logger, checkers ...Checker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readinessTimeout)
		defer cancel()

		var (
			mu     sync.Mutex
			wg     sync.WaitGroup
			checks = make(map[string]string, len(checkers))
		)

		for _, checker := range checkers {
			wg.Add(1)
			go func(checker Checker) {
				defer wg.Done()

				err := checker.Check(ctx)

				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					checks[checker.Name()] = "error"
					logger.Warn("readiness check failed", "check", checker.Name(), "error", err)
					return
				}
				checks[checker.Name()] = "ok"
			}(checker)
		}
		wg.Wait()

		status := "ok"
		statusCode := http.StatusOK
		for _, result := range checks {
			if result != "ok" {
				status = "unavailable"
				statusCode = http.StatusServiceUnavailable
				break
			}
		}

		writeJSON(w, statusCode, map[string]any{
			"status": status,
			"checks": checks,
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
