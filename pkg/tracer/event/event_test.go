package event

import (
	"encoding/json"
	"testing"
)

func TestNewEvent(t *testing.T) {
	data := map[string]interface{}{"key": "value"}
	ev := NewEvent("test_type", "trace123", data)

	if ev.Type != "test_type" {
		t.Errorf("NewEvent() Type = %q, want %q", ev.Type, "test_type")
	}
	if ev.TraceID != "trace123" {
		t.Errorf("NewEvent() TraceID = %q, want %q", ev.TraceID, "trace123")
	}
	if ev.Timestamp == 0 {
		t.Error("NewEvent() Timestamp = 0, want non-zero")
	}
	if ev.Data == nil {
		t.Error("NewEvent() Data = nil, want non-nil")
	}
	if ev.Data["key"] != "value" {
		t.Errorf("NewEvent() Data[key] = %q, want %q", ev.Data["key"], "value")
	}
}

func TestEvent_JSON(t *testing.T) {
	data := map[string]interface{}{
		"host":   "example.com",
		"port":   443,
		"secure": true,
	}
	ev := Event{
		Type:      "dns_start",
		Timestamp: 1676432100123456789,
		TraceID:   "abc123",
		Data:      data,
	}

	// Marshal
	jsonData, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	// Unmarshal
	var decoded Event
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	// Verify round-trip
	if decoded.Type != ev.Type {
		t.Errorf("decoded Type = %q, want %q", decoded.Type, ev.Type)
	}
	if decoded.Timestamp != ev.Timestamp {
		t.Errorf("decoded Timestamp = %d, want %d", decoded.Timestamp, ev.Timestamp)
	}
	if decoded.TraceID != ev.TraceID {
		t.Errorf("decoded TraceID = %q, want %q", decoded.TraceID, ev.TraceID)
	}
	if decoded.Data["host"] != "example.com" {
		t.Errorf("decoded Data[host] = %q, want %q", decoded.Data["host"], "example.com")
	}
	// JSON unmarshals numbers as float64
	if decoded.Data["port"].(float64) != 443 {
		t.Errorf("decoded Data[port] = %v, want 443", decoded.Data["port"])
	}
	if decoded.Data["secure"] != true {
		t.Errorf("decoded Data[secure] = %v, want true", decoded.Data["secure"])
	}
}
