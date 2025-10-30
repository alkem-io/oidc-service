package telemetry

import (
	"fmt"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a production-grade zap.Logger configured for JSON output.
// It uses ISO8601 timestamps under the "timestamp" key. If the provided level
// string is non-empty it is applied to the logger; an invalid level returns an error.
// On success it returns the constructed *zap.Logger ready for use.
func NewLogger(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.Encoding = "json"

	if level != "" {
		if err := assignLevel(level, &cfg.Level); err != nil {
			return nil, err
		}
	}

	logger, err := cfg.Build()
	if err != nil {
		return nil, fmt.Errorf("build zap logger: %w", err)
	}

	return logger, nil
}

// assignLevel sets atom to the zap log level corresponding to level.
// Accepted values for level are "debug", "info", "warn", and "error".
// Returns an error if level is not one of the accepted values.
func assignLevel(level string, atom *zap.AtomicLevel) error {
	switch strings.ToLower(level) {
	case "debug":
		atom.SetLevel(zapcore.DebugLevel)
	case "info":
		atom.SetLevel(zapcore.InfoLevel)
	case "warn":
		atom.SetLevel(zapcore.WarnLevel)
	case "error":
		atom.SetLevel(zapcore.ErrorLevel)
	default:
		return fmt.Errorf("invalid log level: %s", level)
	}
	return nil
}

// NewRegistry creates a new Prometheus registry. Standard collectors are not registered by default, allowing callers to register only the collectors they need.
func NewRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	return reg
}