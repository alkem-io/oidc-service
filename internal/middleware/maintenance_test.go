package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/stretchr/testify/require"
)

func TestMaintenanceMiddlewareAllowsRequestsWhenDisabled(t *testing.T) {
	handlerCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusTeapot)
	})

	middleware := Maintenance(MaintenanceOptions{State: maintenance.NewState(config.MaintenanceState{})})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	middleware(next).ServeHTTP(rec, req)

	require.True(t, handlerCalled)
	require.Equal(t, http.StatusTeapot, rec.Code)
}

func TestMaintenanceMiddlewareReturnsServiceUnavailable(t *testing.T) {
	retry := 5 * time.Minute
	state := maintenance.NewState(config.MaintenanceState{
		Enabled:    true,
		Message:    "scheduled upgrade",
		RetryAfter: &retry,
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next handler should not be called")
	})

	middleware := Maintenance(MaintenanceOptions{State: state})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	middleware(next).ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Equal(t, "300", rec.Header().Get("Retry-After"))

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, "maintenance_mode", payload["error"])
	require.Equal(t, "scheduled upgrade", payload["message"])
}

func TestMaintenanceMiddlewareSkipped(t *testing.T) {
	state := maintenance.NewState(config.MaintenanceState{Enabled: true})
	called := false

	middleware := Maintenance(MaintenanceOptions{
		State: state,
		Skip: func(r *http.Request) bool {
			return true
		},
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusAccepted)
	})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rec := httptest.NewRecorder()

	middleware(next).ServeHTTP(rec, req)

	require.True(t, called)
	require.Equal(t, http.StatusAccepted, rec.Code)
}
