package middleware

import (
	"math"
	"net/http"
	"strconv"

	"github.com/alkem-io/oidc-service/internal/challenge"
	"github.com/alkem-io/oidc-service/internal/config"
	"github.com/alkem-io/oidc-service/internal/maintenance"
)

// SkipFunc determines whether a request should bypass maintenance checks.
type SkipFunc func(*http.Request) bool

// MaintenanceOptions configure the maintenance middleware behaviour.
type MaintenanceOptions struct {
	State *maintenance.State
	Skip  SkipFunc
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

			WriteChallengeError(w, r, challenge.NewMaintenanceError(snapshot.Message))
		})
	}
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
