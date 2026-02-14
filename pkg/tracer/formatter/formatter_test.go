package formatter

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/tracer/event"
)

func TestNDJSONEmitter(t *testing.T) {
	var buf bytes.Buffer
	em := NewNDJSONEmitter(&buf)

	ev1 := event.NewEvent("dns_start", "trace1", map[string]interface{}{"host": "example.com"})
	ev2 := event.NewEvent("dns_done", "trace1", map[string]interface{}{"ip": "93.184.216.34"})

	if err := em.Emit(ev1); err != nil {
		t.Fatalf("Emit(ev1) error = %v", err)
	}
	if err := em.Emit(ev2); err != nil {
		t.Fatalf("Emit(ev2) error = %v", err)
	}
	if err := em.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Validate NDJSON output
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}

	// Parse first line
	var decoded1 event.Event
	if err := json.Unmarshal([]byte(lines[0]), &decoded1); err != nil {
		t.Fatalf("json.Unmarshal(line1) error = %v", err)
	}
	if decoded1.Type != "dns_start" {
		t.Errorf("line1 Type = %q, want %q", decoded1.Type, "dns_start")
	}

	// Parse second line
	var decoded2 event.Event
	if err := json.Unmarshal([]byte(lines[1]), &decoded2); err != nil {
		t.Fatalf("json.Unmarshal(line2) error = %v", err)
	}
	if decoded2.Type != "dns_done" {
		t.Errorf("line2 Type = %q, want %q", decoded2.Type, "dns_done")
	}
}

func TestHTMLEmitter_Buffer(t *testing.T) {
	var buf bytes.Buffer
	em := NewHTMLEmitter(&buf)

	ev := event.NewEvent("dns_start", "trace1", map[string]interface{}{"host": "example.com"})
	if err := em.Emit(ev); err != nil {
		t.Fatalf("Emit() error = %v", err)
	}

	// Before Close, buffer should be empty
	if buf.Len() > 0 {
		t.Errorf("buffer len = %d before Close(), want 0", buf.Len())
	}

	// After Close, buffer should contain HTML
	if err := em.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if buf.Len() == 0 {
		t.Error("buffer len = 0 after Close(), want > 0")
	}
}

func TestHTMLEmitter_Output(t *testing.T) {
	var buf bytes.Buffer
	em := NewHTMLEmitter(&buf)

	ev1 := event.NewEvent("dns_start", "trace1", map[string]interface{}{"host": "example.com"})
	ev2 := event.NewEvent("dns_done", "trace1", map[string]interface{}{"ip": "93.184.216.34", "duration_ms": 100})

	if err := em.Emit(ev1); err != nil {
		t.Fatalf("Emit(ev1) error = %v", err)
	}
	if err := em.Emit(ev2); err != nil {
		t.Fatalf("Emit(ev2) error = %v", err)
	}
	if err := em.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	html := buf.String()

	// Validate HTML structure
	if !strings.Contains(html, "<!DOCTYPE html>") {
		t.Error("HTML output missing DOCTYPE")
	}
	if !strings.Contains(html, "Network Trace Report") {
		t.Error("HTML output missing title")
	}
	if !strings.Contains(html, "dns_start") {
		t.Error("HTML output missing dns_start event")
	}
	if !strings.Contains(html, "dns_done") {
		t.Error("HTML output missing dns_done event")
	}
	if !strings.Contains(html, "example.com") {
		t.Error("HTML output missing host data")
	}
	if !strings.Contains(html, "93.184.216.34") {
		t.Error("HTML output missing ip data")
	}
}
