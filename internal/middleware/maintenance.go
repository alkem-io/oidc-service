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

// Maintenance short-circuits requests with a 503 response when maintenance mode is enabled.
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
