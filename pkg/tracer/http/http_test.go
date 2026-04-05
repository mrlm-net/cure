package http

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	nethttp "net/http"

	"github.com/mrlm-net/cure/pkg/tracer/event"
	"github.com/mrlm-net/cure/pkg/tracer/formatter"
)

func TestTraceURL_Success(t *testing.T) {
	// Start test server
	ts := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	// Capture events
	var buf bytes.Buffer
	em := formatter.NewNDJSONEmitter(&buf)

	err := TraceURL(context.Background(), ts.URL, WithEmitter(em))
	if err != nil {
		t.Fatalf("TraceURL() error = %v", err)
	}
	em.Close()

	// Parse events
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 4 {
		t.Fatalf("got %d events, want at least 4 (dns_start, dns_done, tcp_connect*, http_*)", len(lines))
	}

	// Validate events
	var events []event.Event
	for _, line := range lines {
		if line == "" {
			continue
		}
		var ev event.Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		events = append(events, ev)
	}

	// Check that we have dns_start and http_response_done events
	hasKeyEvents := false
	for _, ev := range events {
		if ev.Type == "dns_start" || ev.Type == "http_response_done" {
			hasKeyEvents = true
			break
		}
	}
	if !hasKeyEvents {
		t.Error("missing expected event types (dns_start or http_response_done)")
	}
}

func TestTraceURL_DryRun(t *testing.T) {
	var buf bytes.Buffer
	em := formatter.NewNDJSONEmitter(&buf)

	err := TraceURL(context.Background(), "https://example.com", WithEmitter(em), WithDryRun(true))
	if err != nil {
		t.Fatalf("TraceURL() error = %v", err)
	}
	em.Close()

	// Parse events
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 8 {
		t.Fatalf("got %d events, want at least 8", len(lines))
	}

	// Validate synthetic events
	for _, line := range lines {
		if line == "" {
			continue
		}
		var ev event.Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if ev.TraceID == "" {
			t.Error("event TraceID is empty")
		}
	}
}

func TestTraceURL_Redact(t *testing.T) {
	ts := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.Header().Set("Set-Cookie", "session=secret")
		w.WriteHeader(200)
	}))
	defer ts.Close()

	var buf bytes.Buffer
	em := formatter.NewNDJSONEmitter(&buf)

	headers := map[string]string{
		"Authorization": "Bearer secret-token",
		"X-Custom":      "visible",
	}

	err := TraceURL(context.Background(), ts.URL,
		WithEmitter(em),
		WithHeaders(headers),
		WithRedact(true),
	)
	if err != nil {
		t.Fatalf("TraceURL() error = %v", err)
	}
	em.Close()

	output := buf.String()

	// Verify redaction
	if strings.Contains(output, "secret-token") {
		t.Error("Authorization header not redacted")
	}
	if strings.Contains(output, "session=secret") {
		t.Error("Set-Cookie header not redacted")
	}
	if !strings.Contains(output, "[REDACTED]") {
		t.Error("expected [REDACTED] in output")
	}
	if !strings.Contains(output, "visible") {
		t.Error("X-Custom header should be visible")
	}
}

func TestTraceURL_Redirect(t *testing.T) {
	// Set up a redirect chain: /redirect → /final
	ts := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		if r.URL.Path == "/redirect" {
			nethttp.Redirect(w, r, "/final", nethttp.StatusFound)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte("final"))
	}))
	defer ts.Close()

	var buf bytes.Buffer
	em := formatter.NewNDJSONEmitter(&buf)

	err := TraceURL(context.Background(), ts.URL+"/redirect", WithEmitter(em))
	if err != nil {
		t.Fatalf("TraceURL() error = %v", err)
	}
	em.Close()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	var events []event.Event
	for _, line := range lines {
		if line == "" {
			continue
		}
		var ev event.Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		events = append(events, ev)
	}

	// Collect event types seen
	seen := make(map[string]bool)
	for _, ev := range events {
		seen[ev.Type] = true
	}

	for _, want := range []string{"http_request_start", "http_redirect", "http_response_done"} {
		if !seen[want] {
			t.Errorf("missing expected event %q; got types: %v", want, eventTypes(events))
		}
	}

	// Validate http_redirect fields
	for _, ev := range events {
		if ev.Type != "http_redirect" {
			continue
		}
		if ev.Data["from"] == nil || ev.Data["to"] == nil {
			t.Errorf("http_redirect missing from/to fields: %v", ev.Data)
		}
		statusCode, ok := ev.Data["status_code"].(float64)
		if !ok || statusCode != 302 {
			t.Errorf("http_redirect status_code = %v, want 302", ev.Data["status_code"])
		}
	}
}

func TestTraceURL_NewEvents(t *testing.T) {
	ts := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		w.WriteHeader(200)
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	var buf bytes.Buffer
	em := formatter.NewNDJSONEmitter(&buf)

	err := TraceURL(context.Background(), ts.URL, WithEmitter(em))
	if err != nil {
		t.Fatalf("TraceURL() error = %v", err)
	}
	em.Close()

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	var events []event.Event
	for _, line := range lines {
		if line == "" {
			continue
		}
		var ev event.Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		events = append(events, ev)
	}

	seen := make(map[string]bool)
	for _, ev := range events {
		seen[ev.Type] = true
	}

	for _, want := range []string{"conn_reused", "request_written", "ttfb"} {
		if !seen[want] {
			t.Errorf("missing expected event %q; got types: %v", want, eventTypes(events))
		}
	}
}

func TestTraceURL_DryRun_ContainsNewEvents(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantTypes []string
	}{
		{
			name: "http URL includes redirect event",
			url:  "http://example.com",
			wantTypes: []string{
				"http_request_start", "conn_reused", "dns_start", "dns_done",
				"tcp_connect_start", "tcp_connect_done", "request_written",
				"http_redirect", "ttfb", "http_response_done",
			},
		},
		{
			name: "https URL includes TLS events, no redirect",
			url:  "https://example.com",
			wantTypes: []string{
				"http_request_start", "conn_reused", "dns_start", "dns_done",
				"tcp_connect_start", "tcp_connect_done", "tls_handshake_start",
				"tls_handshake_done", "request_written", "ttfb", "http_response_done",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			em := formatter.NewNDJSONEmitter(&buf)

			err := TraceURL(context.Background(), tc.url, WithEmitter(em), WithDryRun(true))
			if err != nil {
				t.Fatalf("TraceURL() error = %v", err)
			}
			em.Close()

			lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
			var events []event.Event
			for _, line := range lines {
				if line == "" {
					continue
				}
				var ev event.Event
				if err := json.Unmarshal([]byte(line), &ev); err != nil {
					t.Fatalf("json.Unmarshal() error = %v", err)
				}
				events = append(events, ev)
			}

			seen := make(map[string]bool)
			for _, ev := range events {
				seen[ev.Type] = true
			}

			for _, want := range tc.wantTypes {
				if !seen[want] {
					t.Errorf("missing expected event %q; got types: %v", want, eventTypes(events))
				}
			}
		})
	}
}

// eventTypes returns a deduplicated list of event type strings for test output.
func eventTypes(events []event.Event) []string {
	seen := make(map[string]bool)
	var out []string
	for _, ev := range events {
		if !seen[ev.Type] {
			seen[ev.Type] = true
			out = append(out, ev.Type)
		}
	}
	return out
}

func TestTraceURL_CustomHeaders(t *testing.T) {
	receivedHeaders := make(nethttp.Header)
	ts := httptest.NewServer(nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(200)
	}))
	defer ts.Close()

	headers := map[string]string{
		"X-Custom-Header": "test-value",
		"User-Agent":      "cure-tracer",
	}

	err := TraceURL(context.Background(), ts.URL,
		WithHeaders(headers),
		WithEmitter(formatter.NewNDJSONEmitter(bytes.NewBuffer(nil))),
	)
	if err != nil {
		t.Fatalf("TraceURL() error = %v", err)
	}

	if receivedHeaders.Get("X-Custom-Header") != "test-value" {
		t.Errorf("X-Custom-Header = %q, want %q", receivedHeaders.Get("X-Custom-Header"), "test-value")
	}
	if receivedHeaders.Get("User-Agent") != "cure-tracer" {
		t.Errorf("User-Agent = %q, want %q", receivedHeaders.Get("User-Agent"), "cure-tracer")
	}
}
