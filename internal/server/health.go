package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// FR-036a — uniform ≤500 ms timeout per dependency check, ≤2 s TTL cache on
// the dep-check result so probe storms don't saturate the dependency.
const (
	depCheckTimeout = 500 * time.Millisecond
	depCheckCacheTTL = 2 * time.Second

	// Kratos and Hydra both expose `/admin/health/alive` returning
	// `{"status":"ok"}`. Hydra's path is the same shape under its admin host.
	// Using the alive endpoint (not `/ready`) is intentional — `/admin/health/ready`
	// on Hydra/Kratos transitively checks their database, which makes the
	// readiness probe fail-cascade through three services. We only need to
	// know "the admin HTTP plane responds"; downstream dep-of-dep failures
	// surface through the actual login/consent flows, not the probe.
	kratosAliveProbePath = "/admin/health/alive"
	hydraAliveProbePath  = "/health/alive"
)

// CheckStatus is the per-dep summary string in the JSON probe response.
type CheckStatus string

const (
	checkStatusOK        CheckStatus = "ok"
	checkStatusUnhealthy CheckStatus = "unhealthy"
)

// ReadinessChecks is the per-dep map exposed in the readiness response.
type ReadinessChecks struct {
	Kratos CheckResult `json:"kratos"`
	Hydra  CheckResult `json:"hydra"`
}

// CheckResult bundles status + optional error code per dep.
type CheckResult struct {
	Status CheckStatus `json:"status"`
	Error  string      `json:"error,omitempty"`
}

// ReadinessResult is the top-level readiness response shape.
type ReadinessResult struct {
	Status CheckStatus     `json:"status"`
	Checks ReadinessChecks `json:"checks"`
}

// LivenessResult is the liveness response — handler responsive only,
// no dep calls.
type LivenessResult struct {
	Status string `json:"status"`
}

// HealthConfig wires the URLs + HTTP client used to probe Hydra and Kratos.
// Zero values are filled with sensible defaults at construction time.
type HealthConfig struct {
	Logger         *zap.Logger
	HydraAdminURL  string
	KratosAdminURL string
	HTTPClient     *http.Client
	// Now overrides the clock for tests; nil → time.Now.
	Now func() time.Time
}

// HealthHandler implements FR-036a probe endpoints for oidc-service.
//
// Endpoints:
//   - GET /health/live  — handler responsive only; never hits a dep.
//   - GET /health/ready — Kratos `/admin/health/alive` AND Hydra
//     `/health/alive`. 200 when both ok, 503 otherwise. Each call gated by
//     ≤500 ms timeout; results cached ≤2 s.
type HealthHandler struct {
	logger         *zap.Logger
	httpClient     *http.Client
	hydraAdminURL  string
	kratosAdminURL string

	now func() time.Time

	mu          sync.Mutex
	cachedReady *cachedReadyResult
}

type cachedReadyResult struct {
	result   ReadinessResult
	storedAt time.Time
}

// NewHealthHandler constructs a HealthHandler; a nil cfg.HTTPClient gets a
// per-request 500 ms-timeout client. Logger is replaced with Nop when nil.
func NewHealthHandler(cfg HealthConfig) *HealthHandler {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: depCheckTimeout}
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &HealthHandler{
		logger:         logger,
		httpClient:     httpClient,
		hydraAdminURL:  strings.TrimRight(cfg.HydraAdminURL, "/"),
		kratosAdminURL: strings.TrimRight(cfg.KratosAdminURL, "/"),
		now:            now,
	}
}

// ServeLive handles GET /health/live — handler responsive only.
func (h *HealthHandler) ServeLive(w http.ResponseWriter, _ *http.Request) {
	writeHealthJSON(w, http.StatusOK, LivenessResult{Status: "ok"})
}

// ServeReady handles GET /health/ready — Kratos + Hydra reachability.
func (h *HealthHandler) ServeReady(w http.ResponseWriter, r *http.Request) {
	result := h.evaluateReadiness(r.Context())
	code := http.StatusOK
	if result.Status != checkStatusOK {
		code = http.StatusServiceUnavailable
	}
	writeHealthJSON(w, code, result)
}

func (h *HealthHandler) evaluateReadiness(ctx context.Context) ReadinessResult {
	h.mu.Lock()
	if h.cachedReady != nil && h.now().Sub(h.cachedReady.storedAt) < depCheckCacheTTL {
		cached := h.cachedReady.result
		h.mu.Unlock()
		return cached
	}
	h.mu.Unlock()

	// Run the probes outside the mutex so a 500 ms hang here doesn't
	// serialise concurrent probes — they'll all see the same cache miss
	// and may race the same upstream call. That's an acceptable trade-off
	// for sub-second probes; a singleflight would buy us at most 2 dropped
	// upstream calls per 2 s window.
	kratos := h.probeURL(ctx, h.kratosAdminURL, kratosAliveProbePath, "kratos_unreachable")
	hydra := h.probeURL(ctx, h.hydraAdminURL, hydraAliveProbePath, "hydra_unreachable")

	overall := checkStatusOK
	if kratos.Status != checkStatusOK || hydra.Status != checkStatusOK {
		overall = checkStatusUnhealthy
	}
	result := ReadinessResult{
		Status: overall,
		Checks: ReadinessChecks{Kratos: kratos, Hydra: hydra},
	}

	h.mu.Lock()
	h.cachedReady = &cachedReadyResult{result: result, storedAt: h.now()}
	h.mu.Unlock()

	return result
}

// probeURL issues a GET against `baseURL+path` with a 500 ms deadline.
// Anything other than 2xx (or a successful TCP+read inside the timeout) is
// treated as a probe failure — the caller will surface this as `unhealthy`
// with the supplied errorCode. An empty baseURL fails fast with
// `url_not_configured` so deploy misconfiguration is immediately visible
// in the probe response.
func (h *HealthHandler) probeURL(ctx context.Context, baseURL, path, errorCode string) CheckResult {
	if baseURL == "" {
		return CheckResult{Status: checkStatusUnhealthy, Error: "url_not_configured"}
	}
	probeCtx, cancel := context.WithTimeout(ctx, depCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return CheckResult{Status: checkStatusUnhealthy, Error: errorCode}
	}
	req.Header.Set("Accept", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return CheckResult{Status: checkStatusUnhealthy, Error: errorCode}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return CheckResult{Status: checkStatusUnhealthy, Error: errorCode}
	}
	return CheckResult{Status: checkStatusOK}
}

func writeHealthJSON(w http.ResponseWriter, status int, payload any) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(buf.Bytes())
}
