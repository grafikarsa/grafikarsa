package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/problem"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
)

const rateLimitValkeyImage = "valkey/valkey:9.1-alpine"

func newRateLimitValkey(t *testing.T) *redis.Client {
	t.Helper()

	testcontainers.SkipIfProviderIsNotHealthy(t)

	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        rateLimitValkeyImage,
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		},
		Started: true,
	})
	if err != nil {
		t.Fatalf("start valkey container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	endpoint, err := container.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("valkey endpoint: %v", err)
	}

	client := redis.NewClient(&redis.Options{Addr: endpoint})
	t.Cleanup(func() {
		_ = client.Close()
	})

	if err := client.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping valkey: %v", err)
	}

	return client
}

func okHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func decodeProblem(t *testing.T, rec *httptest.ResponseRecorder) problem.Problem {
	t.Helper()

	res := rec.Result()
	defer res.Body.Close()

	var body problem.Problem
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

func TestRateLimitAllowsThenBlocks(t *testing.T) {
	limiter := ratelimit.New(newRateLimitValkey(t))
	handler := RateLimit(limiter, RateLimitOptions{
		Name:   "allows-then-blocks:" + t.Name(),
		Rule:   ratelimit.Rule{Limit: 2, Window: time.Minute},
		Key:    func(*http.Request) string { return "client" },
		Logger: discardLogger(),
	})(http.HandlerFunc(okHandler))

	first := httptest.NewRecorder()
	handler.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/", nil))
	if first.Code != http.StatusNoContent {
		t.Fatalf("first status = %d, want 204", first.Code)
	}
	if got := first.Header().Get("RateLimit-Limit"); got != "2" {
		t.Errorf("first RateLimit-Limit = %q, want 2", got)
	}
	if got := first.Header().Get("RateLimit-Remaining"); got != "1" {
		t.Errorf("first RateLimit-Remaining = %q, want 1", got)
	}
	reset, err := strconv.Atoi(first.Header().Get("RateLimit-Reset"))
	if err != nil || reset < 0 {
		t.Errorf("first RateLimit-Reset = %q, want a non-negative integer", first.Header().Get("RateLimit-Reset"))
	}

	second := httptest.NewRecorder()
	handler.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/", nil))
	if second.Code != http.StatusNoContent {
		t.Fatalf("second status = %d, want 204", second.Code)
	}
	if got := second.Header().Get("RateLimit-Remaining"); got != "0" {
		t.Errorf("second RateLimit-Remaining = %q, want 0", got)
	}

	third := httptest.NewRecorder()
	handler.ServeHTTP(third, httptest.NewRequest(http.MethodGet, "/", nil))
	if third.Code != http.StatusTooManyRequests {
		t.Fatalf("third status = %d, want 429", third.Code)
	}
	retryAfter, err := strconv.Atoi(third.Header().Get("Retry-After"))
	if err != nil || retryAfter < 1 {
		t.Errorf("Retry-After = %q, want an integer >= 1", third.Header().Get("Retry-After"))
	}
	if got := third.Header().Get("RateLimit-Limit"); got != "2" {
		t.Errorf("third RateLimit-Limit = %q, want 2", got)
	}
	if body := decodeProblem(t, third); body.Code != "RATE_LIMITED" {
		t.Errorf("third Code = %q, want RATE_LIMITED", body.Code)
	}
}

func TestRateLimitEmptyKeyBypasses(t *testing.T) {
	limiter := ratelimit.New(newRateLimitValkey(t))
	handler := RateLimit(limiter, RateLimitOptions{
		Name:   "empty-key:" + t.Name(),
		Rule:   ratelimit.Rule{Limit: 1, Window: time.Minute},
		Key:    func(*http.Request) string { return "" },
		Logger: discardLogger(),
	})(http.HandlerFunc(okHandler))

	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("request %d status = %d, want 204", i+1, rec.Code)
		}
		if got := rec.Header().Get("RateLimit-Limit"); got != "" {
			t.Errorf("request %d RateLimit-Limit = %q, want empty", i+1, got)
		}
	}
}

func TestRateLimitFailOpen(t *testing.T) {
	closed := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 50 * time.Millisecond,
		MaxRetries:  -1,
	})
	t.Cleanup(func() {
		_ = closed.Close()
	})
	limiter := ratelimit.New(closed)

	t.Run("allows request", func(t *testing.T) {
		handler := RateLimit(limiter, RateLimitOptions{
			Name:     "fail-open:" + t.Name(),
			Rule:     ratelimit.Rule{Limit: 1, Window: time.Minute},
			Key:      func(*http.Request) string { return "client" },
			FailOpen: true,
			Logger:   discardLogger(),
		})(http.HandlerFunc(okHandler))

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusNoContent {
			t.Errorf("status = %d, want 204", rec.Code)
		}
	})

	t.Run("returns unavailable", func(t *testing.T) {
		handler := RateLimit(limiter, RateLimitOptions{
			Name:     "fail-closed:" + t.Name(),
			Rule:     ratelimit.Rule{Limit: 1, Window: time.Minute},
			Key:      func(*http.Request) string { return "client" },
			FailOpen: false,
			Logger:   discardLogger(),
		})(http.HandlerFunc(okHandler))

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
		if body := decodeProblem(t, rec); body.Code != "UNAVAILABLE" {
			t.Errorf("Code = %q, want UNAVAILABLE", body.Code)
		}
	})
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		want       string
	}{
		{
			name:       "public remote address ignores forwarded header",
			remoteAddr: "203.0.113.10:4321",
			xff:        "198.51.100.7",
			want:       "203.0.113.10",
		},
		{
			name:       "loopback uses first forwarded value",
			remoteAddr: "127.0.0.1:4321",
			xff:        "198.51.100.7, 10.0.0.2",
			want:       "198.51.100.7",
		},
		{
			name:       "private remote address uses forwarded value",
			remoteAddr: "10.0.0.5:4321",
			xff:        "198.51.100.9",
			want:       "198.51.100.9",
		},
		{
			name:       "private remote address without forwarded header",
			remoteAddr: "10.0.0.5:4321",
			want:       "10.0.0.5",
		},
		{
			name:       "ipv6 loopback uses forwarded value",
			remoteAddr: "[::1]:4321",
			xff:        "2001:db8::7",
			want:       "2001:db8::7",
		},
		{
			name:       "ipv6 public remote address ignores forwarded header",
			remoteAddr: "[2001:db8::1]:4321",
			xff:        "198.51.100.7",
			want:       "2001:db8::1",
		},
		{
			name:       "remote address without port",
			remoteAddr: "203.0.113.10",
			xff:        "198.51.100.7",
			want:       "203.0.113.10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}

			if got := ClientIP(req); got != tt.want {
				t.Errorf("ClientIP = %q, want %q", got, tt.want)
			}
		})
	}
}
