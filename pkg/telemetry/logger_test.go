package telemetry

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLoggerInvalidLevel(t *testing.T) {
	logger, err := NewLogger("nope")
	require.Error(t, err)
	require.Nil(t, logger)
}

func TestNewLoggerValidLevel(t *testing.T) {
	logger, err := NewLogger("debug")
	require.NoError(t, err)
	require.NotNil(t, logger)
	// flush buffers; Sync can return benign errors on non-file sinks, so just log them
	if err := logger.Sync(); err != nil {
		t.Logf("logger.Sync() returned: %v", err)
	}
}

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	require.NotNil(t, reg)
}
