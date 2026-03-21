package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// sendAndReceive runs serveLoop in a goroutine with the given input and returns
// the output written to the buffer.
func sendAndReceive(t *testing.T, srv *Server, input string) []map[string]any {
	t.Helper()
	r := strings.NewReader(input)
	var buf bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := srv.serveLoop(ctx, r, &buf)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Fatalf("serveLoop error: %v", err)
	}

	var results []map[string]any
	dec := json.NewDecoder(&buf)
	for {
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			break
		}
		results = append(results, m)
	}
	return results
}

func TestServeStdio_Ping(t *testing.T) {
	srv := New()
	req := `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n"
	results := sendAndReceive(t, srv, req)
	if len(results) != 1 {
		t.Fatalf("expected 1 response, got %d", len(results))
	}
	if results[0]["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", results[0]["jsonrpc"])
	}
	if results[0]["error"] != nil {
		t.Errorf("unexpected error: %v", results[0]["error"])
	}
}

func TestServeStdio_Initialize(t *testing.T) {
	srv := New(WithName("stdio-test"), WithVersion("0.1.0"))
	srv.RegisterTool(&noopTool{name: "echo"})

	req := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n"
	results := sendAndReceive(t, srv, req)
	if len(results) != 1 {
		t.Fatalf("expected 1 response, got %d", len(results))
	}
	result := results[0]["result"].(map[string]any)
	if result["protocolVersion"] != "2025-03-26" {
		t.Errorf("protocolVersion = %v, want 2025-03-26", result["protocolVersion"])
	}
}

func TestServeStdio_MultipleRequests(t *testing.T) {
	srv := New()
	srv.RegisterTool(&noopTool{name: "t"})

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"initialize","params":{}}`,
	}, "\n") + "\n"

	results := sendAndReceive(t, srv, input)
	if len(results) != 3 {
		t.Fatalf("expected 3 responses, got %d", len(results))
	}
}

func TestServeStdio_Notification_NoResponse(t *testing.T) {
	srv := New()
	// notifications/initialized has no ID and must not produce a response.
	input := `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n" +
		`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n"

	results := sendAndReceive(t, srv, input)
	// Only the ping produces a response — the notification is silently consumed.
	if len(results) != 1 {
		t.Fatalf("expected 1 response (ping only), got %d", len(results))
	}
}

func TestServeStdio_ParseError(t *testing.T) {
	srv := New()
	input := "not json at all\n" +
		`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n"

	results := sendAndReceive(t, srv, input)
	// Parse error produces an error response; ping produces a success response.
	if len(results) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(results))
	}
	// First must be a parse error.
	first := results[0]
	errObj, ok := first["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object in first response, got: %v", first)
	}
	if errObj["code"].(float64) != codeParseError {
		t.Errorf("error code = %v, want %d", errObj["code"], codeParseError)
	}
}

func TestServeStdio_UnknownMethod(t *testing.T) {
	srv := New()
	req := `{"jsonrpc":"2.0","id":99,"method":"does/not/exist"}` + "\n"
	results := sendAndReceive(t, srv, req)
	if len(results) != 1 {
		t.Fatalf("expected 1 response, got %d", len(results))
	}
	errObj, ok := results[0]["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error in response")
	}
	if errObj["code"].(float64) != codeMethodNotFound {
		t.Errorf("code = %v, want %d", errObj["code"], codeMethodNotFound)
	}
}

func TestServeStdio_ContextCancellation(t *testing.T) {
	srv := New()
	// Use an io.Pipe so the server blocks waiting for input.
	pr, _ := io.Pipe()
	var buf bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- srv.serveLoop(ctx, pr, &buf)
	}()

	// Cancel the context while the server is blocked.
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("serveLoop did not stop after context cancellation")
	}
	pr.Close()
}

func TestServeStdio_BlankLines(t *testing.T) {
	srv := New()
	// Blank lines between requests should be skipped.
	input := "\n\n" + `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n\n"
	results := sendAndReceive(t, srv, input)
	if len(results) != 1 {
		t.Fatalf("expected 1 response, got %d: %v", len(results), results)
	}
}

func TestServeStdio_ToolCall_E2E(t *testing.T) {
	srv := New()
	srv.RegisterTool(FuncTool(
		"add", "Add",
		Schema().Number("a", "a", Required()).Number("b", "b", Required()).Build(),
		func(_ context.Context, args map[string]any) ([]Content, error) {
			a, _ := args["a"].(float64)
			b, _ := args["b"].(float64)
			return Textf("%.0f", a+b), nil
		},
	))

	params, _ := json.Marshal(map[string]any{
		"name":      "add",
		"arguments": map[string]any{"a": 3, "b": 4},
	})
	// Build a complete JSON-RPC request: {"jsonrpc":"2.0","id":1,"method":"tools/call","params":<params>}
	reqObj, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  json.RawMessage(params),
	})
	req := string(append(reqObj, '\n'))

	results := sendAndReceive(t, srv, req)
	if len(results) != 1 {
		t.Fatalf("expected 1 response, got %d", len(results))
	}
	result := results[0]["result"].(map[string]any)
	if result["isError"].(bool) {
		t.Error("isError must be false")
	}
	content := result["content"].([]any)
	if len(content) == 0 {
		t.Fatal("content must not be empty")
	}
	tc := content[0].(map[string]any)
	if tc["text"] != "7" {
		t.Errorf("text = %v, want %q", tc["text"], "7")
	}
}

// ---- Benchmarks ----

// BenchmarkStdioRoundTrip measures the throughput of a single ping round-trip
// through the stdio loop.
func BenchmarkStdioRoundTrip(b *testing.B) {
	srv := New()
	pingLine := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(pingLine)
		var buf bytes.Buffer
		ctx, cancel := context.WithCancel(context.Background())
		_ = srv.serveLoop(ctx, r, &buf)
		cancel()
	}
}

// BenchmarkStdioRoundTrip_ToolCall measures a tools/call round-trip.
func BenchmarkStdioRoundTrip_ToolCall(b *testing.B) {
	srv := New()
	srv.RegisterTool(FuncTool(
		"noop", "Noop",
		Schema().Build(),
		func(_ context.Context, _ map[string]any) ([]Content, error) {
			return Text("ok"), nil
		},
	))

	params, _ := json.Marshal(map[string]any{"name": "noop", "arguments": map[string]any{}})
	line := append(
		[]byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":`),
		append(params, '\n')...,
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(line)
		var buf bytes.Buffer
		ctx, cancel := context.WithCancel(context.Background())
		_ = srv.serveLoop(ctx, r, &buf)
		cancel()
	}
}
