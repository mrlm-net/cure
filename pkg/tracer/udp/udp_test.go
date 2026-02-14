package udp

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/tracer/event"
	"github.com/mrlm-net/cure/pkg/tracer/formatter"
)

func TestTraceAddr_Success(t *testing.T) {
	// Start mock UDP server
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.ListenPacket() error = %v", err)
	}
	defer conn.Close()

	go func() {
		buf := make([]byte, 1024)
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			return
		}
		conn.WriteTo([]byte("response"), addr)
		_ = n
	}()

	var buf bytes.Buffer
	em := formatter.NewNDJSONEmitter(&buf)

	addr := conn.LocalAddr().String()
	err = TraceAddr(context.Background(), addr,
		WithEmitter(em),
		WithDataString("test data"),
	)
	if err != nil {
		t.Fatalf("TraceAddr() error = %v", err)
	}
	em.Close()

	// Parse events
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("got %d events, want at least 2", len(lines))
	}

	// Validate events
	var hasSend bool
	for _, line := range lines {
		if line == "" {
			continue
		}
		var ev event.Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if ev.Type == "udp_send" {
			hasSend = true
		}
	}

	if !hasSend {
		t.Error("missing udp_send event")
	}
}

func TestTraceAddr_DryRun(t *testing.T) {
	var buf bytes.Buffer
	em := formatter.NewNDJSONEmitter(&buf)

	err := TraceAddr(context.Background(), "1.1.1.1:53",
		WithEmitter(em),
		WithDryRun(true),
	)
	if err != nil {
		t.Fatalf("TraceAddr() error = %v", err)
	}
	em.Close()

	// Parse events
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("got %d events, want at least 2", len(lines))
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
