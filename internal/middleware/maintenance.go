package middleware

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alkem-io/oidc-service/internal/challenge"
	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
	"github.com/alkem-io/oidc-service/pkg/telemetry"
)

// SkipFunc determines whether a request should bypass maintenance checks.
type SkipFunc func(*http.Request) bool

// MaintenanceOptions configure the maintenance middleware behaviour.
type MaintenanceOptions struct {
	State   *maintenance.State
	Skip    SkipFunc
	Metrics telemetry.ChallengeRecorder
}

// Maintenance returns an HTTP middleware that short-circuits requests and writes a maintenance challenge response when maintenance mode is enabled.
// If the maintenance state provides a retry interval the middleware sets the `Retry-After` header; if a Metrics recorder is provided it records maintenance challenge timing for recognized flows (login, consent).
// The middleware respects the provided Skip function to bypass maintenance checks for specific requests.
func Maintenance(opts MaintenanceOptions) func(http.Handler) http.Handler {
	state := opts.State
	if state == nil {
		state = maintenance.NewState(config.MaintenanceState{})
	}

	skip := opts.Skip
	if skip == nil {
		skip = func(*http.Request) bool { return false }
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			if skip(r) {
				next.ServeHTTP(w, r)
				return
			}

			snapshot := state.Snapshot()
			if !snapshot.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			if value := retryAfter(snapshot); value != "" {
				w.Header().Set("Retry-After", value)
			}

			if recorder := opts.Metrics; recorder != nil {
				if flow := flowFromPath(r.URL.Path); flow != "" {
					recorder.ObserveChallenge(flow, "maintenance", "maintenance_mode", time.Since(started))
				}
			}

			WriteChallengeError(w, r, challenge.NewMaintenanceError(snapshot.Message))
		})
	}
}

// flowFromPath maps an HTTP request path to a maintenance flow name.
// It returns "login" for paths starting with "/v1/oidc/login", "consent" for paths starting with "/v1/oidc/consent", and the empty string otherwise.
func flowFromPath(path string) string {
	if strings.HasPrefix(path, "/v1/oidc/login") {
		return "login"
	}
	if strings.HasPrefix(path, "/v1/oidc/consent") {
		return "consent"
	}
	return ""
}

// SkipAll always returns true and can be used to disable maintenance checks.
func SkipAll(*http.Request) bool { return true }

// SkipNone always returns false and forces maintenance checks.
func SkipNone(*http.Request) bool { return false }

// retryAfter converts the state's RetryAfter duration into a string suitable for an HTTP
// Retry-After header. If state.RetryAfter is nil it returns an empty string. The duration
// is rounded up to the next whole second; if the resulting seconds value is less than or
// equal to zero it returns "0", otherwise it returns the decimal seconds as a string.
func retryAfter(state config.MaintenanceState) string {
	if state.RetryAfter == nil {
		return ""
	}
	seconds := int(math.Ceil(state.RetryAfter.Seconds()))
	if seconds <= 0 {
		return "0"
	}
	return strconv.Itoa(seconds)
}