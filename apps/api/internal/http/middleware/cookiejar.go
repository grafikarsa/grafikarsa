package middleware

import (
	"net/http"

	"github.com/grafikarsa/grafikarsa/apps/api/internal/http/authctx"
)

// CookieJar puts a fresh cookie jar and the request metadata in the context,
// then writes every queued cookie on the response before the handler's first
// header or body write.
func CookieJar(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jar := &authctx.Jar{}

		ctx := authctx.WithJar(r.Context(), jar)
		ctx = authctx.WithMeta(ctx, authctx.Meta{
			IP:        ClientIP(r),
			UserAgent: r.UserAgent(),
		})

		next.ServeHTTP(&cookieFlushWriter{ResponseWriter: w, jar: jar}, r.WithContext(ctx))
	})
}

// cookieFlushWriter writes the pending cookies exactly once, right before the
// response headers are committed.
type cookieFlushWriter struct {
	http.ResponseWriter
	jar     *authctx.Jar
	flushed bool
}

func (w *cookieFlushWriter) flush() {
	if w.flushed {
		return
	}
	w.flushed = true

	for _, cookie := range w.jar.Cookies() {
		http.SetCookie(w.ResponseWriter, cookie)
	}
}

func (w *cookieFlushWriter) WriteHeader(status int) {
	w.flush()
	w.ResponseWriter.WriteHeader(status)
}

func (w *cookieFlushWriter) Write(b []byte) (int, error) {
	w.flush()
	return w.ResponseWriter.Write(b)
}

// Unwrap lets http.ResponseController reach the underlying writer.
func (w *cookieFlushWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
