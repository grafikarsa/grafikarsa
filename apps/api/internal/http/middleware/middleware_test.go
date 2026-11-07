package middleware

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/problem"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/reqctx"
)

var requestIDPatternTest = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRequestIDGenerates(t *testing.T) {
	var fromContext string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fromContext = reqctx.RequestID(r.Context())
		_, _ = w.Write([]byte(fromContext))
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	header := rec.Header().Get("X-Request-Id")
	if header == "" {
		t.Fatal("X-Request-Id header is empty")
	}
	if !requestIDPatternTest.MatchString(header) {
		t.Errorf("X-Request-Id = %q, want a valid id", header)
	}
	if fromContext != header {
		t.Errorf("context id = %q, want %q", fromContext, header)
	}
	if rec.Body.String() != header {
		t.Errorf("body = %q, want %q", rec.Body.String(), header)
	}
}

func TestRequestIDReusesValidHeader(t *testing.T) {
	const incoming = "client-request-123"

	var fromContext string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fromContext = reqctx.RequestID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", incoming)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-Id"); got != incoming {
		t.Errorf("X-Request-Id = %q, want %q", got, incoming)
	}
	if fromContext != incoming {
		t.Errorf("context id = %q, want %q", fromContext, incoming)
	}
}

func TestRequestIDReplacesInvalidHeader(t *testing.T) {
	const invalid = "not a valid request id"

	var fromContext string
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fromContext = reqctx.RequestID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", invalid)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	header := rec.Header().Get("X-Request-Id")
	if header == invalid {
		t.Errorf("X-Request-Id = %q, want the invalid header to be replaced", header)
	}
	if !requestIDPatternTest.MatchString(header) {
		t.Errorf("X-Request-Id = %q, want a valid generated id", header)
	}
	if fromContext != header {
		t.Errorf("context id = %q, want %q", fromContext, header)
	}
}

func TestRecovererWritesProblem(t *testing.T) {
	handler := Recoverer(discardLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", res.StatusCode)
	}
	if got := res.Header.Get("Content-Type"); got != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", got)
	}

	var body problem.Problem
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "INTERNAL" {
		t.Errorf("Code = %q, want INTERNAL", body.Code)
	}
	if body.Status != http.StatusInternalServerError {
		t.Errorf("Status = %d, want 500", body.Status)
	}
}

func TestRecovererRepanicsAbortHandler(t *testing.T) {
	handler := Recoverer(discardLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	defer func() {
		if recovered := recover(); recovered != http.ErrAbortHandler {
			t.Errorf("recovered = %v, want http.ErrAbortHandler", recovered)
		}
	}()

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}
