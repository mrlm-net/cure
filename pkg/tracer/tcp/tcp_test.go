package tcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/mrlm-net/cure/pkg/tracer/event"
	"github.com/mrlm-net/cure/pkg/tracer/formatter"
)

func TestTraceAddr_Success(t *testing.T) {
	// Start mock TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 1024)
		conn.Read(buf)
		conn.Write([]byte("response"))
	}()

	var buf bytes.Buffer
	em := formatter.NewNDJSONEmitter(&buf)

	addr := listener.Addr().String()
	err = TraceAddr(context.Background(), addr,
		WithEmitter(em),
		WithDataString("test data"),
		WithTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("TraceAddr() error = %v", err)
	}
	em.Close()

	// Parse events
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 4 {
		t.Fatalf("got %d events, want at least 4", len(lines))
	}

	// Validate events
	var hasConnectDone, hasSend bool
	for _, line := range lines {
		if line == "" {
			continue
		}
		var ev event.Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if ev.Type == "tcp_connect_done" {
			hasConnectDone = true
		}
		if ev.Type == "tcp_send" {
			hasSend = true
		}
	}

	if !hasConnectDone {
		t.Error("missing tcp_connect_done event")
	}
	if !hasSend {
		t.Error("missing tcp_send event")
	}
}

func TestTraceAddr_DryRun(t *testing.T) {
	var buf bytes.Buffer
	em := formatter.NewNDJSONEmitter(&buf)

	err := TraceAddr(context.Background(), "example.com:443",
		WithEmitter(em),
		WithDryRun(true),
	)
	if err != nil {
		t.Fatalf("TraceAddr() error = %v", err)
	}
	em.Close()

	// Parse events
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) < 4 {
		t.Fatalf("got %d events, want at least 4", len(lines))
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

func TestTraceAddr_SendData(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	defer listener.Close()

	receivedData := make(chan []byte, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 1024)
		n, _ := conn.Read(buf)
		receivedData <- buf[:n]
		conn.Write([]byte("ack"))
	}()

	testData := "test message"
	err = TraceAddr(context.Background(), listener.Addr().String(),
		WithDataString(testData),
		WithTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("TraceAddr() error = %v", err)
	}

	select {
	case data := <-receivedData:
		if string(data) != testData {
			t.Errorf("received data = %q, want %q", string(data), testData)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for received data")
	}
}
