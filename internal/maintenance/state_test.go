package maintenance

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alkem-io/oidc-service/internal/config"
)

func TestStateSnapshotReflectsUpdates(t *testing.T) {
	initial := config.MaintenanceState{Enabled: false}
	state := NewState(initial)

	require.False(t, state.Snapshot().Enabled)

	retry := 10 * time.Second
	state.Update(config.MaintenanceState{
		Enabled:    true,
		Message:    "maintenance",
		RetryAfter: &retry,
	})

	snapshot := state.Snapshot()
	require.True(t, snapshot.Enabled)
	require.Equal(t, "maintenance", snapshot.Message)
	require.NotNil(t, snapshot.RetryAfter)
	require.Equal(t, retry, *snapshot.RetryAfter)

	*snapshot.RetryAfter = 5 * time.Second
	updated := state.Snapshot()
	require.Equal(t, retry, *updated.RetryAfter)
}

func TestStateUsesDefaultMessage(t *testing.T) {
	state := NewState(config.MaintenanceState{Enabled: true})

	snapshot := state.Snapshot()
	require.Equal(t, "OIDC service temporarily unavailable", snapshot.Message)
}
