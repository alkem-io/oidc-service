package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRequestContextGeneratesHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	var capturedID string
	var loggerPresent bool

	h := RequestContext(zap.NewNop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = RequestID(r.Context())
		logger := Logger(r.Context())
		loggerPresent = logger != nil
		w.WriteHeader(http.StatusOK)
	}))

	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotEmpty(t, capturedID)
	require.Equal(t, capturedID, rec.Header().Get(RequestIDHeader))
	require.True(t, loggerPresent)
}

func TestRequestContextRespectsIncomingID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(RequestIDHeader, "incoming")
	rec := httptest.NewRecorder()

	h := RequestContext(zap.NewNop())(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "incoming", RequestID(r.Context()))
		w.WriteHeader(http.StatusNoContent)
	}))

	h.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "incoming", rec.Header().Get(RequestIDHeader))
}
