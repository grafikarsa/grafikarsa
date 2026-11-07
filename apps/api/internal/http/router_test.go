package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRouterHealthz(t *testing.T) {
	rec := httptest.NewRecorder()
	NewRouter(testLogger()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestRouterNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	NewRouter(testLogger()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", got)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["code"] != "NOT_FOUND" {
		t.Errorf("code = %v, want NOT_FOUND", body["code"])
	}
}

func TestRouterPing(t *testing.T) {
	rec := httptest.NewRecorder()
	NewRouter(testLogger()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var body map[string]bool
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !body["pong"] {
		t.Errorf("pong = %t, want true", body["pong"])
	}
}

func TestRouterAPINotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	NewRouter(testLogger()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/tidak-ada", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", got)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["code"] != "NOT_FOUND" {
		t.Errorf("code = %v, want NOT_FOUND", body["code"])
	}
}

func TestRouterMethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	NewRouter(testLogger()).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/healthz", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", got)
	}
}

func TestRouter_InvalidPathParamIsProblemJSON(t *testing.T) {
	h := NewRouter(testLogger())
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/auth/sessions/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("content type = %q", ct)
	}
}

func TestRouter_PlaceholderAuthEndpointIs501(t *testing.T) {
	h := NewRouter(testLogger())
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/session", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", rec.Code)
	}
}
