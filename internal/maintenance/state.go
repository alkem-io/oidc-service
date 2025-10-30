package maintenance

import (
	"sync/atomic"
	"time"

	"github.com/alkem-io/oidc-service/internal/config"
)

// State tracks the live maintenance toggle consulted by HTTP middleware.
type State struct {
	enabled    atomic.Bool
	message    atomic.Value
	retryAfter atomic.Value
}

// NewState constructs a State instance from the current configuration snapshot.
func NewState(initial config.MaintenanceState) *State {
	s := &State{}
	s.Update(initial)
	return s
}

// Update swaps the maintenance state atomically.
func (s *State) Update(next config.MaintenanceState) {
	s.enabled.Store(next.Enabled)
	if next.Message == "" {
		s.message.Store("OIDC service temporarily unavailable")
	} else {
		s.message.Store(next.Message)
	}
	s.retryAfter.Store(copyDuration(next.RetryAfter))
}

// Snapshot returns the current maintenance toggle metadata.
func (s *State) Snapshot() config.MaintenanceState {
	value := config.MaintenanceState{
		Enabled: s.enabled.Load(),
		Message: s.message.Load().(string),
	}

	if stored := s.retryAfter.Load(); stored != nil {
		if d, ok := stored.(*time.Duration); ok && d != nil {
			value.RetryAfter = copyDuration(d)
		}
	}

	return value
}

func copyDuration(src *time.Duration) *time.Duration {
	if src == nil {
		return nil
	}
	d := *src
	return &d
}
