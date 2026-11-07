package health

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type readyzResponse struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	Healthz().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}
	if got := res.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %q, want ok", body["status"])
	}
}

func TestReadyzAllOK(t *testing.T) {
	handler := Readyz(testLogger(),
		CheckerFunc{N: "postgres", F: func(context.Context) error { return nil }},
		CheckerFunc{N: "valkey", F: func(context.Context) error { return nil }},
		CheckerFunc{N: "storage", F: func(context.Context) error { return nil }},
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}

	var body readyzResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("status = %q, want ok", body.Status)
	}
	for _, name := range []string{"postgres", "valkey", "storage"} {
		if body.Checks[name] != "ok" {
			t.Errorf("checks[%q] = %q, want ok", name, body.Checks[name])
		}
	}
}

func TestReadyzFailure(t *testing.T) {
	handler := Readyz(testLogger(),
		CheckerFunc{N: "postgres", F: func(context.Context) error { return nil }},
		CheckerFunc{N: "valkey", F: func(context.Context) error { return errors.New("boom") }},
	)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", res.StatusCode)
	}

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if strings.Contains(string(raw), "boom") {
		t.Errorf("body leaks checker error: %s", raw)
	}

	var body readyzResponse
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "unavailable" {
		t.Errorf("status = %q, want unavailable", body.Status)
	}
	if body.Checks["postgres"] != "ok" {
		t.Errorf("checks[postgres] = %q, want ok", body.Checks["postgres"])
	}
	if body.Checks["valkey"] != "error" {
		t.Errorf("checks[valkey] = %q, want error", body.Checks["valkey"])
	}
}

func TestReadyzTimeout(t *testing.T) {
	handler := Readyz(testLogger(),
		CheckerFunc{N: "slow", F: func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}},
	)

	start := time.Now()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	elapsed := time.Since(start)

	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
	if elapsed > 3*time.Second {
		t.Errorf("elapsed = %v, want about 2s", elapsed)
	}

	var body readyzResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "unavailable" {
		t.Errorf("status = %q, want unavailable", body.Status)
	}
	if body.Checks["slow"] != "error" {
		t.Errorf("checks[slow] = %q, want error", body.Checks["slow"])
	}
}
