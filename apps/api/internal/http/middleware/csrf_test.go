package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/problem"
)

const testCSRFToken = "csrf-token-123"

func csrfTestHandler() (http.Handler, *bool) {
	called := false
	handler := CSRF(CSRFOptions{
		AllowedOrigins: []string{"https://grafikarsa.com"},
		ExemptPaths:    []string{"/api/v1/views"},
	})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	return handler, &called
}

func csrfRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: testCSRFToken})
	req.Header.Set(CSRFHeaderName, testCSRFToken)
	return req
}

func assertCSRFFailed(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.StatusCode)
	}

	var body problem.Problem
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "CSRF_FAILED" {
		t.Errorf("Code = %q, want CSRF_FAILED", body.Code)
	}
	if body.Status != http.StatusForbidden {
		t.Errorf("Status = %d, want 403", body.Status)
	}
}

func TestCSRFSafeMethodPassesWithoutToken(t *testing.T) {
	handler, called := csrfTestHandler()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/items", nil))

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if !*called {
		t.Error("next handler was not called")
	}
}

func TestCSRFRejectsPostWithoutCookie(t *testing.T) {
	handler, called := csrfTestHandler()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/items", nil)
	req.Header.Set(CSRFHeaderName, testCSRFToken)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertCSRFFailed(t, rec)
	if *called {
		t.Error("next handler was called")
	}
}

func TestCSRFRejectsMismatchedHeader(t *testing.T) {
	handler, called := csrfTestHandler()

	req := csrfRequest(http.MethodPost, "/api/v1/items")
	req.Header.Set(CSRFHeaderName, "another-token")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertCSRFFailed(t, rec)
	if *called {
		t.Error("next handler was called")
	}
}

func TestCSRFAllowsMatchingCookieAndHeader(t *testing.T) {
	handler, called := csrfTestHandler()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, csrfRequest(http.MethodPost, "/api/v1/items"))

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if !*called {
		t.Error("next handler was not called")
	}
}

func TestCSRFExemptPathPassesWithoutToken(t *testing.T) {
	handler, called := csrfTestHandler()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/views", nil))

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if !*called {
		t.Error("next handler was not called")
	}
}

func TestCSRFRejectsDisallowedOrigin(t *testing.T) {
	handler, called := csrfTestHandler()

	req := csrfRequest(http.MethodPost, "/api/v1/items")
	req.Header.Set("Origin", "https://evil.example")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assertCSRFFailed(t, rec)
	if *called {
		t.Error("next handler was called")
	}
}

func TestCSRFAllowsAllowedOrigin(t *testing.T) {
	handler, called := csrfTestHandler()

	req := csrfRequest(http.MethodPost, "/api/v1/items")
	req.Header.Set("Origin", "https://grafikarsa.com")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if !*called {
		t.Error("next handler was not called")
	}
}

func TestSetCSRFCookie(t *testing.T) {
	rec := httptest.NewRecorder()
	SetCSRFCookie(rec, testCSRFToken)

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}

	cookie := cookies[0]
	if cookie.Name != CSRFCookieName {
		t.Errorf("Name = %q, want %q", cookie.Name, CSRFCookieName)
	}
	if cookie.Value != testCSRFToken {
		t.Errorf("Value = %q, want %q", cookie.Value, testCSRFToken)
	}
	if cookie.Path != "/" {
		t.Errorf("Path = %q, want /", cookie.Path)
	}
	if !cookie.Secure {
		t.Error("Secure = false, want true")
	}
	if cookie.HttpOnly {
		t.Error("HttpOnly = true, want false")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", cookie.SameSite)
	}
	if cookie.MaxAge != 0 {
		t.Errorf("MaxAge = %d, want 0", cookie.MaxAge)
	}
}
