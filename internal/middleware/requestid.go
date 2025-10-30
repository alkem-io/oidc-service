package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const (
	requestIDKey contextKey = "request-id"
	loggerKey    contextKey = "logger"

	// RequestIDHeader is the header used to propagate correlation identifiers.
	RequestIDHeader = "X-Request-Id"
)

// RequestContext returns middleware that ensures each HTTP request carries a correlation
// ID and a request-scoped logger stored in the request context.
// 
// If the incoming request includes an X-Request-Id header that value is used; otherwise
// a new UUID is generated. The correlation ID is stored in the context under requestIDKey
// and also set on the response header. A child logger derived from the provided base
// logger (or a no-op logger if base is nil) is stored in the context under loggerKey
// with fields "requestId", "method", and "path". The middleware then calls the next
// handler with the updated context.
func RequestContext(base *zap.Logger) func(http.Handler) http.Handler {
	if base == nil {
		base = zap.NewNop()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := r.Header.Get(RequestIDHeader)
			if reqID == "" {
				reqID = uuid.NewString()
			}

			ctx := context.WithValue(r.Context(), requestIDKey, reqID)
			child := base.With(
				zap.String("requestId", reqID),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
			)
			ctx = context.WithValue(ctx, loggerKey, child)

			w.Header().Set(RequestIDHeader, reqID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestID returns the correlation/request ID stored in ctx, or an empty string if none.
func RequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// Logger returns the request-scoped logger stored in ctx or a no-op logger if none is present.
// If the context contains a non-nil *zap.Logger under the middleware key, that logger is returned; otherwise zap.NewNop() is returned.
func Logger(ctx context.Context) *zap.Logger {
	if v, ok := ctx.Value(loggerKey).(*zap.Logger); ok && v != nil {
		return v
	}
	return zap.NewNop()
}