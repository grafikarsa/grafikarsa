// Package reqctx stores values shared across an HTTP request lifetime.
package reqctx

import "context"

type requestIDKey struct{}

// WithRequestID returns a context carrying the given request ID.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

// RequestID returns the request ID stored in ctx, or an empty string.
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}
