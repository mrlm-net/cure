package event

import "time"

// Event represents a single trace event in the lifecycle of a network operation.
type Event struct {
	// Type identifies the event category (e.g., "dns_start", "tcp_connect", "http_request").
	Type string `json:"type"`

	// Timestamp is the Unix timestamp (nanoseconds) when the event occurred.
	Timestamp int64 `json:"timestamp"`

	// TraceID correlates events from the same trace session.
	TraceID string `json:"trace_id"`

	// Data contains event-specific fields (e.g., resolved IP, status code, latency).
	Data map[string]interface{} `json:"data"`
}

// NewEvent creates an Event with the current timestamp.
func NewEvent(typ, traceID string, data map[string]interface{}) Event {
	return Event{
		Type:      typ,
		Timestamp: time.Now().UnixNano(),
		TraceID:   traceID,
		Data:      data,
	}
}

// Emitter consumes trace events. Implementations may write to stdout,
// buffer for HTML generation, or integrate with external systems.
type Emitter interface {
	// Emit processes a single event. Returns an error if emission fails.
	Emit(event Event) error

	// Close finalizes any buffered output. Not all emitters require cleanup.
	Close() error
}
