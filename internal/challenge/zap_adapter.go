package challenge

import "go.uber.org/zap"

// zapLoggerAdapter adapts a zap.Logger to the challenge.Logger interface.
type zapLoggerAdapter struct {
	logger *zap.Logger
}

// NewZapLoggerAdapter creates a challenge.Logger from a zap.Logger.
func NewZapLoggerAdapter(logger *zap.Logger) Logger {
	if logger == nil {
		return noopLogger{}
	}
	return &zapLoggerAdapter{logger: logger}
}

// Info forwards structured info logs to the underlying zap logger.
func (z *zapLoggerAdapter) Info(msg string, fields ...interface{}) {
	z.logger.Info(msg, z.convertFields(fields...)...)
}

// Warn forwards structured warning logs to the underlying zap logger.
func (z *zapLoggerAdapter) Warn(msg string, fields ...interface{}) {
	z.logger.Warn(msg, z.convertFields(fields...)...)
}

// Error forwards structured error logs to the underlying zap logger.
func (z *zapLoggerAdapter) Error(msg string, fields ...interface{}) {
	z.logger.Error(msg, z.convertFields(fields...)...)
}

// Debug forwards structured debug logs to the underlying zap logger.
func (z *zapLoggerAdapter) Debug(msg string, fields ...interface{}) {
	z.logger.Debug(msg, z.convertFields(fields...)...)
}

// convertFields converts interface{} key-value pairs to zap.Field.
func (z *zapLoggerAdapter) convertFields(fields ...interface{}) []zap.Field {
	if len(fields)%2 != 0 {
		// Odd number of fields, add the last one as a value with "unknown" key
		fields = append(fields, "unknown")
	}

	zapFields := make([]zap.Field, 0, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			key = "unknown"
		}
		value := fields[i+1]
		zapFields = append(zapFields, zap.Any(key, value))
	}
	return zapFields
}
