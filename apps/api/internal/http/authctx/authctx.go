// Package authctx stores the request-scoped authentication state: the
// authenticated principal, request metadata used for auditing, and cookies
// that handlers want the middleware to set on the response.
package authctx

import (
	"context"
	"net/http"
	"sync"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/domain/auth"
)

type (
	principalKey struct{}
	metaKey      struct{}
	jarKey       struct{}
)

// WithPrincipal returns a context carrying the authenticated principal.
func WithPrincipal(ctx context.Context, p *auth.Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}

// Principal returns the authenticated principal, or nil for anonymous
// requests.
func Principal(ctx context.Context) *auth.Principal {
	p, _ := ctx.Value(principalKey{}).(*auth.Principal)
	return p
}

// Meta is request metadata recorded on security-sensitive actions.
type Meta struct {
	IP        string
	UserAgent string
}

// WithMeta returns a context carrying the request metadata.
func WithMeta(ctx context.Context, m Meta) context.Context {
	return context.WithValue(ctx, metaKey{}, m)
}

// MetaFrom returns the request metadata stored in ctx, or the zero value.
func MetaFrom(ctx context.Context) Meta {
	m, _ := ctx.Value(metaKey{}).(Meta)
	return m
}

// Jar collects the cookies handlers want to set. The CookieJar middleware
// flushes them once before the response headers are written.
type Jar struct {
	mu      sync.Mutex
	cookies []*http.Cookie
}

// Add queues a cookie. A nil cookie or a nil jar is ignored.
func (j *Jar) Add(c *http.Cookie) {
	if j == nil || c == nil {
		return
	}

	j.mu.Lock()
	defer j.mu.Unlock()
	j.cookies = append(j.cookies, c)
}

// Cookies returns a copy of the queued cookies.
func (j *Jar) Cookies() []*http.Cookie {
	if j == nil {
		return nil
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	out := make([]*http.Cookie, len(j.cookies))
	copy(out, j.cookies)
	return out
}

// WithJar returns a context carrying the cookie jar.
func WithJar(ctx context.Context, j *Jar) context.Context {
	return context.WithValue(ctx, jarKey{}, j)
}

// JarFrom returns the cookie jar stored in ctx, or nil.
func JarFrom(ctx context.Context) *Jar {
	j, _ := ctx.Value(jarKey{}).(*Jar)
	return j
}

// SetCookie queues c in the request's jar. It is a no-op when the request has
// no jar, for example in tests that call handlers without the middleware.
func SetCookie(ctx context.Context, c *http.Cookie) {
	JarFrom(ctx).Add(c)
}
