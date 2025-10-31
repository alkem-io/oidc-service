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
	logger.Sync() // flush buffers
}

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry()
	require.NotNil(t, reg)
}
