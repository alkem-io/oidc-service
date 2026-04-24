// In local-dev the events are written as one JSON line per record to stdout.
// In k8s a Filebeat → Logstash → Elasticsearch pipeline collects the same
// stream. The Emitter type is transport-agnostic; callers pass the writer.
package audit

import (
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

type Outcome string

const (
	OutcomeSuccess Outcome = "success"
	OutcomeFailure Outcome = "failure"
	OutcomeWarn    Outcome = "warn"
)

// Omitted pointer / empty fields are dropped from the emitted JSON via the
// omitempty tag so records stay PII-minimal by default.
type Event struct {
	EventType      string  `json:"event_type"`
	Outcome        Outcome `json:"outcome"`
	Sub            string  `json:"sub,omitempty"`
	ClientID       string  `json:"client_id,omitempty"`
	CorrelationID  string  `json:"correlation_id"`
	RequestID      string  `json:"request_id"`
	Timestamp      string  `json:"timestamp"`
	ErrorCode      string  `json:"error_code,omitempty"`
	RequestedScope string  `json:"requested_scope,omitempty"`
	GrantedScope   string  `json:"granted_scope,omitempty"`
	TruncatedInput string  `json:"truncated_input,omitempty"`
	RpID           string  `json:"rp_id,omitempty"`
}

// Emitter serialises one Event to one JSON line and writes it to the
// underlying writer. Safe for concurrent use.
type Emitter struct {
	w  io.Writer
	mu sync.Mutex
}

// NewEmitter builds an Emitter that writes to w. When w is nil it falls back
// to os.Stdout so that callers can wire the default in one line.
func NewEmitter(w io.Writer) *Emitter {
	if w == nil {
		w = os.Stdout
	}
	return &Emitter{w: w}
}

// Emit serialises the event, appends a newline, and writes atomically.
// Timestamp is populated when absent.
func (e *Emitter) Emit(ev Event) error {
	if ev.Timestamp == "" {
		ev.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err = e.w.Write(data)
	return err
}
