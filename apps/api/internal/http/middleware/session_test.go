package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/apperr"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/authctx"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/problem"
)

type fakeAuthenticator struct {
	principal *auth.Principal
	err       error
	called    bool
	token     string
}

func (f *fakeAuthenticator) Authenticate(_ context.Context, token string) (*auth.Principal, error) {
	f.called = true
	f.token = token
	return f.principal, f.err
}

func TestSessionWithoutCookieContinuesAnonymous(t *testing.T) {
	authn := &fakeAuthenticator{}
	var principal *auth.Principal

	handler := Session(authn, discardLogger())(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		principal = authctx.Principal(r.Context())
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if authn.called {
		t.Error("Authenticate was called without a cookie")
	}
	if principal != nil {
		t.Errorf("principal = %v, want nil", principal)
	}
}

func TestSessionAttachesPrincipal(t *testing.T) {
	want := &auth.Principal{SessionID: uuid.New(), UserID: uuid.New(), NextStep: auth.NextStepNone}
	authn := &fakeAuthenticator{principal: want}
	var principal *auth.Principal

	handler := CookieJar(Session(authn, discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal = authctx.Principal(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "token-123"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if principal != want {
		t.Fatalf("principal = %v, want %v", principal, want)
	}
	if authn.token != "token-123" {
		t.Errorf("token = %q, want token-123", authn.token)
	}
}

func TestSessionUnauthenticatedClearsCookieAndContinuesAnonymous(t *testing.T) {
	authn := &fakeAuthenticator{err: apperr.Unauthenticated()}
	var principal *auth.Principal

	handler := CookieJar(Session(authn, discardLogger())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal = authctx.Principal(r.Context())
		w.WriteHeader(http.StatusUnauthorized)
	})))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "stale"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if principal != nil {
		t.Errorf("principal = %v, want nil", principal)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if cookies[0].Name != SessionCookieName {
		t.Errorf("cookie name = %q, want %q", cookies[0].Name, SessionCookieName)
	}
	if cookies[0].Value != "" {
		t.Errorf("cookie value = %q, want empty", cookies[0].Value)
	}
	if cookies[0].MaxAge >= 0 {
		t.Errorf("MaxAge = %d, want negative", cookies[0].MaxAge)
	}
}

func TestSessionUnexpectedErrorReturns503(t *testing.T) {
	authn := &fakeAuthenticator{err: errors.New("database down")}
	called := false

	handler := Session(authn, discardLogger())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "token"})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Error("next handler was called")
	}

	res := rec.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", res.StatusCode)
	}

	var body problem.Problem
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != "UNAVAILABLE" {
		t.Errorf("code = %q, want UNAVAILABLE", body.Code)
	}
}

func TestSessionCookieAttributes(t *testing.T) {
	expires := time.Now().Add(time.Hour)
	cookie := SessionCookie("token-123", expires)

	if cookie.Name != SessionCookieName {
		t.Errorf("Name = %q, want %q", cookie.Name, SessionCookieName)
	}
	if cookie.Value != "token-123" {
		t.Errorf("Value = %q, want token-123", cookie.Value)
	}
	if cookie.Path != "/" {
		t.Errorf("Path = %q, want /", cookie.Path)
	}
	if !cookie.Secure {
		t.Error("Secure = false, want true")
	}
	if !cookie.HttpOnly {
		t.Error("HttpOnly = false, want true")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", cookie.SameSite)
	}
	if !cookie.Expires.Equal(expires) {
		t.Errorf("Expires = %v, want %v", cookie.Expires, expires)
	}
	if cookie.MaxAge <= 0 {
		t.Errorf("MaxAge = %d, want a positive value", cookie.MaxAge)
	}
}

func TestClearSessionCookieAttributes(t *testing.T) {
	cookie := ClearSessionCookie()

	if cookie.Name != SessionCookieName {
		t.Errorf("Name = %q, want %q", cookie.Name, SessionCookieName)
	}
	if cookie.Value != "" {
		t.Errorf("Value = %q, want empty", cookie.Value)
	}
	if cookie.Path != "/" {
		t.Errorf("Path = %q, want /", cookie.Path)
	}
	if !cookie.Secure {
		t.Error("Secure = false, want true")
	}
	if !cookie.HttpOnly {
		t.Error("HttpOnly = false, want true")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite = %v, want Lax", cookie.SameSite)
	}
	if cookie.MaxAge != -1 {
		t.Errorf("MaxAge = %d, want -1", cookie.MaxAge)
	}
}
