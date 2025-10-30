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

// RequestContext ensures every request has a correlation ID and contextual logger.
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

// RequestID extracts the correlation ID from context.
func RequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// Logger extracts the request-scoped logger from context.
func Logger(ctx context.Context) *zap.Logger {
	if v, ok := ctx.Value(loggerKey).(*zap.Logger); ok && v != nil {
		return v
	}
	return zap.NewNop()
}
