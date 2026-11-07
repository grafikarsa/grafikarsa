package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/authctx"
)

func TestCookieJarWritesQueuedCookiesOnWriteHeader(t *testing.T) {
	var meta authctx.Meta
	handler := CookieJar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meta = authctx.MetaFrom(r.Context())
		authctx.SetCookie(r.Context(), &http.Cookie{Name: "a", Value: "1", Path: "/"})
		authctx.SetCookie(r.Context(), &http.Cookie{Name: "b", Value: "2", Path: "/"})
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("User-Agent", "uji/1.0")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("cookies = %d, want 2", len(cookies))
	}
	if cookies[0].Name != "a" || cookies[0].Value != "1" {
		t.Errorf("first cookie = %s=%s, want a=1", cookies[0].Name, cookies[0].Value)
	}
	if cookies[1].Name != "b" || cookies[1].Value != "2" {
		t.Errorf("second cookie = %s=%s, want b=2", cookies[1].Name, cookies[1].Value)
	}

	if meta.UserAgent != "uji/1.0" {
		t.Errorf("UserAgent = %q, want uji/1.0", meta.UserAgent)
	}
	if meta.IP == "" {
		t.Error("IP is empty")
	}
}

func TestCookieJarWritesQueuedCookiesOnWrite(t *testing.T) {
	handler := CookieJar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authctx.SetCookie(r.Context(), &http.Cookie{Name: "a", Value: "1", Path: "/"})
		_, _ = w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := len(rec.Result().Cookies()); got != 1 {
		t.Fatalf("cookies = %d, want 1", got)
	}
}

func TestCookieJarWritesCookiesOnlyOnce(t *testing.T) {
	handler := CookieJar(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authctx.SetCookie(r.Context(), &http.Cookie{Name: "a", Value: "1", Path: "/"})
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := len(rec.Result().Cookies()); got != 1 {
		t.Fatalf("cookies = %d, want 1", got)
	}
	if got := rec.Header().Values("Set-Cookie"); len(got) != 1 {
		t.Fatalf("Set-Cookie headers = %d, want 1", len(got))
	}
}

func TestCookieJarWithoutCookiesSetsNothing(t *testing.T) {
	handler := CookieJar(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if got := rec.Header().Values("Set-Cookie"); len(got) != 0 {
		t.Fatalf("Set-Cookie headers = %v, want none", got)
	}
}
