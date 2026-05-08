package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// FR-036a contract — oidc-service health probes
//   /health/live  -> 200, {status:"ok"} regardless of dep state
//   /health/ready -> 200 when Kratos + Hydra alive endpoints respond 2xx;
//                    503 when either fails;
//                    cached ≤2 s

func TestHealthHandlerLive_AlwaysOK(t *testing.T) {
	// Even with both deps unreachable, /live MUST return 200 and MUST NOT
	// hit either upstream.
	handler := NewHealthHandler(HealthConfig{
		Logger:         zap.NewNop(),
		HydraAdminURL:  "http://127.0.0.1:1", // unreachable
		KratosAdminURL: "http://127.0.0.1:1",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)

	handler.ServeLive(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var body LivenessResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "ok", body.Status)
}

func TestHealthHandlerReady_BothUp(t *testing.T) {
	kratos := stubAlive(t, http.StatusOK)
	defer kratos.Close()
	hydra := stubAlive(t, http.StatusOK)
	defer hydra.Close()

	handler := NewHealthHandler(HealthConfig{
		Logger:         zap.NewNop(),
		HydraAdminURL:  hydra.URL,
		KratosAdminURL: kratos.URL,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	handler.ServeReady(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body ReadinessResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, checkStatusOK, body.Status)
	require.Equal(t, checkStatusOK, body.Checks.Kratos.Status)
	require.Equal(t, checkStatusOK, body.Checks.Hydra.Status)
}

func TestHealthHandlerReady_KratosDown(t *testing.T) {
	kratos := stubAlive(t, http.StatusInternalServerError)
	defer kratos.Close()
	hydra := stubAlive(t, http.StatusOK)
	defer hydra.Close()

	handler := NewHealthHandler(HealthConfig{
		Logger:         zap.NewNop(),
		HydraAdminURL:  hydra.URL,
		KratosAdminURL: kratos.URL,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	handler.ServeReady(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	var body ReadinessResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, checkStatusUnhealthy, body.Status)
	require.Equal(t, checkStatusUnhealthy, body.Checks.Kratos.Status)
	require.Equal(t, "kratos_unreachable", body.Checks.Kratos.Error)
	// Hydra was up — checks are independent.
	require.Equal(t, checkStatusOK, body.Checks.Hydra.Status)
}

func TestHealthHandlerReady_HydraDown(t *testing.T) {
	kratos := stubAlive(t, http.StatusOK)
	defer kratos.Close()
	// Use a closed listener URL — this exercises the network-error path
	// rather than the non-2xx path covered by TestHealthHandlerReady_KratosDown.
	hydra := stubAlive(t, http.StatusOK)
	hydraURL := hydra.URL
	hydra.Close() // immediately close so probes fail with connection refused.

	handler := NewHealthHandler(HealthConfig{
		Logger:         zap.NewNop(),
		HydraAdminURL:  hydraURL,
		KratosAdminURL: kratos.URL,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	handler.ServeReady(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	var body ReadinessResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, checkStatusUnhealthy, body.Status)
	require.Equal(t, checkStatusOK, body.Checks.Kratos.Status)
	require.Equal(t, checkStatusUnhealthy, body.Checks.Hydra.Status)
	require.Equal(t, "hydra_unreachable", body.Checks.Hydra.Error)
}

func TestHealthHandlerReady_UnconfiguredURLsAreUnhealthy(t *testing.T) {
	handler := NewHealthHandler(HealthConfig{Logger: zap.NewNop()})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	handler.ServeReady(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	var body ReadinessResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "url_not_configured", body.Checks.Kratos.Error)
	require.Equal(t, "url_not_configured", body.Checks.Hydra.Error)
}

func TestHealthHandlerReady_CachesResultFor2Seconds(t *testing.T) {
	var kratosCalls, hydraCalls atomic.Int32
	kratos := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		kratosCalls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer kratos.Close()
	hydra := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hydraCalls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer hydra.Close()

	now := time.Unix(1_700_000_000, 0)
	handler := NewHealthHandler(HealthConfig{
		Logger:         zap.NewNop(),
		HydraAdminURL:  hydra.URL,
		KratosAdminURL: kratos.URL,
		Now:            func() time.Time { return now },
	})

	// 5 back-to-back probes within the cache window — only ONE upstream
	// call per dep should fire.
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		handler.ServeReady(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	require.Equal(t, int32(1), kratosCalls.Load(), "kratos probed once within cache window")
	require.Equal(t, int32(1), hydraCalls.Load(), "hydra probed once within cache window")

	// Advance the clock past the cache window — next probe MUST re-check.
	now = now.Add(depCheckCacheTTL + time.Second)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	handler.ServeReady(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Equal(t, int32(2), kratosCalls.Load(), "kratos re-probed after cache expiry")
	require.Equal(t, int32(2), hydraCalls.Load(), "hydra re-probed after cache expiry")
}

func TestHealthHandlerReady_EnforcesPerProbeTimeout(t *testing.T) {
	// A stub that hangs forever — the 500 ms per-probe timeout must
	// short-circuit the wait so the probe surfaces as unhealthy quickly.
	hangCh := make(chan struct{})
	hangingKratos := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		<-hangCh // never closed within the test
	}))
	defer func() {
		close(hangCh)
		hangingKratos.Close()
	}()
	hydra := stubAlive(t, http.StatusOK)
	defer hydra.Close()

	handler := NewHealthHandler(HealthConfig{
		Logger:         zap.NewNop(),
		HydraAdminURL:  hydra.URL,
		KratosAdminURL: hangingKratos.URL,
	})

	start := time.Now()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	handler.ServeReady(rec, req)
	elapsed := time.Since(start)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	// Timeout must fire well before 1 s — give CI 750 ms slack on the
	// 500 ms ceiling.
	require.Less(t, elapsed, 1500*time.Millisecond, "probe must respect 500 ms timeout")
	var body ReadinessResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "kratos_unreachable", body.Checks.Kratos.Error)
}

// stubAlive returns a httptest server that responds with `code` on
// `/admin/health/alive` and `/health/alive`.
func stubAlive(t *testing.T, code int) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/admin/health/alive", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/health/alive", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(code)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	return httptest.NewServer(mux)
}
