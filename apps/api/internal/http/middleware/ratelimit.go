package middleware

import (
	"log/slog"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/problem"
	"github.com/grafikarsa/grafikarsa/apps/api/internal/platform/ratelimit"
)

// KeyFunc derives the rate limit key for a request. Return "" to skip
// limiting this request.
type KeyFunc func(r *http.Request) string

// RateLimitOptions configures the rate limit middleware.
type RateLimitOptions struct {
	Name     string // logical bucket name, e.g. "read", "login_ip"
	Rule     ratelimit.Rule
	Key      KeyFunc
	FailOpen bool // on Valkey error: true = allow request, false = respond 503
	Logger   *slog.Logger
}

// RateLimit rejects requests that exceed the configured rule, writing
// RateLimit-* headers on every limited response.
func RateLimit(l *ratelimit.Limiter, opts RateLimitOptions) func(http.Handler) http.Handler {
	keyFunc := opts.Key
	if keyFunc == nil {
		keyFunc = KeyByIP("")
	}

	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			res, err := l.Allow(r.Context(), opts.Name+":"+key, opts.Rule)
			if err != nil {
				logger.WarnContext(r.Context(), "rate limiter unavailable", "bucket", opts.Name, "error", err)
				if opts.FailOpen {
					next.ServeHTTP(w, r)
					return
				}

				problem.Write(w, r, problem.Problem{
					Status: http.StatusServiceUnavailable,
					Code:   "UNAVAILABLE",
					Title:  "Layanan belum siap",
				})
				return
			}

			w.Header().Set("RateLimit-Limit", strconv.Itoa(res.Limit))
			w.Header().Set("RateLimit-Remaining", strconv.Itoa(res.Remaining))
			w.Header().Set("RateLimit-Reset", strconv.FormatInt(ceilSeconds(res.ResetAfter), 10))

			if !res.Allowed {
				retryAfter := ceilSeconds(res.RetryAfter)
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", strconv.FormatInt(retryAfter, 10))

				problem.Write(w, r, problem.Problem{
					Status: http.StatusTooManyRequests,
					Code:   "RATE_LIMITED",
					Title:  "Terlalu banyak permintaan, coba lagi nanti",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP returns the client IP: the first X-Forwarded-For value when the
// request came from a private or loopback address (our Caddy), otherwise the
// RemoteAddr host. Public clients can never spoof X-Forwarded-For.
func ClientIP(r *http.Request) string {
	host := r.RemoteAddr
	if h, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		host = h
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return host
	}
	if !ip.IsPrivate() && !ip.IsLoopback() {
		return host
	}

	first, _, _ := strings.Cut(r.Header.Get("X-Forwarded-For"), ",")
	first = strings.TrimSpace(first)
	if first == "" {
		return host
	}

	return first
}

// KeyByIP builds a KeyFunc that keys requests by client IP with the given
// prefix.
func KeyByIP(prefix string) KeyFunc {
	return func(r *http.Request) string {
		return prefix + ":" + ClientIP(r)
	}
}

func ceilSeconds(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	return int64(math.Ceil(d.Seconds()))
}
