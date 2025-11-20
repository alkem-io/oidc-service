package support

import (
	"sync"

	"github.com/alkem-io/oidc-service/internal/challenge"
)

// LogEntry captures a structured log emitted during tests.
type LogEntry struct {
	Level   string
	Message string
	Fields  map[string]any
}

// LoggerStub stores log calls for assertions in tests.
type LoggerStub struct {
	mu      sync.Mutex
	entries []LogEntry
}

func (l *LoggerStub) Info(msg string, fields ...interface{}) {
	l.append("info", msg, fields...)
}

func (l *LoggerStub) Warn(msg string, fields ...interface{}) {
	l.append("warn", msg, fields...)
}

func (l *LoggerStub) Error(msg string, fields ...interface{}) {
	l.append("error", msg, fields...)
}

func (l *LoggerStub) Debug(msg string, fields ...interface{}) {
	l.append("debug", msg, fields...)
}

// Entries returns a copy of all captured log entries.
func (l *LoggerStub) Entries() []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]LogEntry, len(l.entries))
	copy(out, l.entries)
	return out
}

// EntriesByLevel returns log entries filtered by level.
func (l *LoggerStub) EntriesByLevel(level string) []LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	var filtered []LogEntry
	for _, entry := range l.entries {
		if entry.Level == level {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func (l *LoggerStub) append(level, msg string, fields ...interface{}) {
	if l == nil {
		return
	}

	entry := LogEntry{Level: level, Message: msg, Fields: map[string]any{}}
	if msg == "" {
		entry.Message = ""
	}

	entry.Fields = normalizeFields(fields...)

	l.mu.Lock()
	l.entries = append(l.entries, entry)
	l.mu.Unlock()
}

func normalizeFields(fields ...interface{}) map[string]any {
	normalized := make(map[string]any)
	for i := 0; i+1 < len(fields); i += 2 {
		key, _ := fields[i].(string)
		if key == "" {
			key = "unknown"
		}
		normalized[key] = fields[i+1]
	}
	return normalized
}

var _ challenge.Logger = (*LoggerStub)(nil)
