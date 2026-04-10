package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"
)

// TestE2EProtocolFlow exercises the full MCP handshake and tool round-trip
// over an io.Pipe()-simulated client connection:
//
//  1. initialize          -> verify protocolVersion, serverInfo, capabilities
//  2. notifications/initialized -> verify no response (notification)
//  3. tools/list          -> verify echo tool is present
//  4. tools/call (echo)   -> verify content matches "hello"
func TestE2EProtocolFlow(t *testing.T) {
	// Build the server under test.
	srv := New(WithName("test"), WithVersion("0.0.0"))
	srv.RegisterTool(FuncTool(
		"echo",
		"Echo the text argument",
		Schema().String("text", "Text to echo", Required()).Build(),
		func(_ context.Context, args map[string]any) ([]Content, error) {
			text, _ := args["text"].(string)
			return Text(text), nil
		},
	))

	// Assemble the four client messages as newline-delimited JSON.
	initMsg := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"test","version":"0.0.0"},"capabilities":{}}}` + "\n"
	notifMsg := `{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n"
	listMsg := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n"

	callParams, _ := json.Marshal(map[string]any{
		"name":      "echo",
		"arguments": map[string]any{"text": "hello"},
	})
	callReqJSON, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      float64(3),
		"method":  "tools/call",
		"params":  json.RawMessage(callParams),
	})
	callMsg := string(append(callReqJSON, '\n'))

	input := strings.Join([]string{initMsg, notifMsg, listMsg, callMsg}, "")

	// Run serve loop with a timeout.
	r := strings.NewReader(input)
	var buf bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := srv.serveLoop(ctx, r, &buf)
	if err != nil {
		// EOF on clean reader is expected — serveLoop returns nil on EOF.
		// Any other error (including context deadline) is a test failure.
		t.Fatalf("serveLoop returned unexpected error: %v", err)
	}

	// Decode all responses.
	var responses []map[string]any
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		var m map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &m); err != nil {
			t.Fatalf("failed to decode response line: %v (line: %s)", err, scanner.Text())
		}
		responses = append(responses, m)
	}

	// The notification must not produce a response, so we expect exactly 3
	// responses: initialize (id=1), tools/list (id=2), tools/call (id=3).
	if len(responses) != 3 {
		t.Fatalf("expected 3 responses (notification must not produce one), got %d: %v", len(responses), responses)
	}

	// ---- Response 1: initialize ----
	t.Run("initialize_response", func(t *testing.T) {
		resp := findByID(responses, 1)
		if resp == nil {
			t.Fatal("no response with id=1 found")
		}
		if resp["jsonrpc"] != "2.0" {
			t.Errorf("jsonrpc = %v, want 2.0", resp["jsonrpc"])
		}
		if resp["error"] != nil {
			t.Errorf("unexpected error: %v", resp["error"])
		}
		result, ok := resp["result"].(map[string]any)
		if !ok {
			t.Fatalf("result is not an object: %v", resp["result"])
		}
		if result["protocolVersion"] != "2025-03-26" {
			t.Errorf("protocolVersion = %v, want 2025-03-26", result["protocolVersion"])
		}
		si, ok := result["serverInfo"].(map[string]any)
		if !ok {
			t.Fatal("serverInfo missing or wrong type")
		}
		if si["name"] != "test" {
			t.Errorf("serverInfo.name = %v, want test", si["name"])
		}
		if si["version"] != "0.0.0" {
			t.Errorf("serverInfo.version = %v, want 0.0.0", si["version"])
		}
		caps, ok := result["capabilities"].(map[string]any)
		if !ok {
			t.Fatal("capabilities missing or wrong type")
		}
		if caps["tools"] == nil {
			t.Error("capabilities.tools must be present because a tool is registered")
		}
	})

	// ---- Response 2: tools/list ----
	t.Run("tools_list_response", func(t *testing.T) {
		resp := findByID(responses, 2)
		if resp == nil {
			t.Fatal("no response with id=2 found")
		}
		if resp["error"] != nil {
			t.Errorf("unexpected error: %v", resp["error"])
		}
		result, ok := resp["result"].(map[string]any)
		if !ok {
			t.Fatalf("result is not an object: %v", resp["result"])
		}
		toolsRaw, ok := result["tools"].([]any)
		if !ok {
			t.Fatalf("tools field missing or wrong type: %v", result["tools"])
		}
		if len(toolsRaw) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(toolsRaw))
		}
		tool, ok := toolsRaw[0].(map[string]any)
		if !ok {
			t.Fatal("first tool is not an object")
		}
		if tool["name"] != "echo" {
			t.Errorf("tool name = %v, want echo", tool["name"])
		}
		if tool["description"] == nil || tool["description"] == "" {
			t.Error("tool description must not be empty")
		}
		schema, ok := tool["inputSchema"].(map[string]any)
		if !ok {
			t.Fatal("inputSchema missing or wrong type")
		}
		if schema["type"] != "object" {
			t.Errorf("inputSchema.type = %v, want object", schema["type"])
		}
	})

	// ---- Response 3: tools/call (echo "hello") ----
	t.Run("tools_call_echo_response", func(t *testing.T) {
		resp := findByID(responses, 3)
		if resp == nil {
			t.Fatal("no response with id=3 found")
		}
		if resp["error"] != nil {
			t.Errorf("unexpected error: %v", resp["error"])
		}
		result, ok := resp["result"].(map[string]any)
		if !ok {
			t.Fatalf("result is not an object: %v", resp["result"])
		}
		isErr, _ := result["isError"].(bool)
		if isErr {
			t.Errorf("isError must be false for a successful call")
		}
		contentRaw, ok := result["content"].([]any)
		if !ok {
			t.Fatalf("content missing or wrong type: %v", result["content"])
		}
		if len(contentRaw) == 0 {
			t.Fatal("content slice must not be empty")
		}
		first, ok := contentRaw[0].(map[string]any)
		if !ok {
			t.Fatal("first content item is not an object")
		}
		if first["type"] != "text" {
			t.Errorf("content[0].type = %v, want text", first["type"])
		}
		if first["text"] != "hello" {
			t.Errorf("content[0].text = %v, want hello", first["text"])
		}
	})
}

// TestE2EProtocolFlow_PipeSimulated mirrors TestE2EProtocolFlow but uses io.Pipe
// to simulate a true streaming client where bytes arrive asynchronously.
func TestE2EProtocolFlow_PipeSimulated(t *testing.T) {
	srv := New(WithName("test"), WithVersion("0.0.0"))
	srv.RegisterTool(FuncTool(
		"echo",
		"Echo the text argument",
		Schema().String("text", "Text to echo", Required()).Build(),
		func(_ context.Context, args map[string]any) ([]Content, error) {
			text, _ := args["text"].(string)
			return Text(text), nil
		},
	))

	pr, pw := io.Pipe()
	var buf bytes.Buffer

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.serveLoop(ctx, pr, &buf)
	}()

	// Write messages and close the write end to signal EOF.
	callParams, _ := json.Marshal(map[string]any{
		"name":      "echo",
		"arguments": map[string]any{"text": "hello"},
	})
	callReqJSON, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      float64(3),
		"method":  "tools/call",
		"params":  json.RawMessage(callParams),
	})
	callMsg := string(append(callReqJSON, '\n'))

	messages := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","clientInfo":{"name":"test","version":"0.0.0"},"capabilities":{}}}` + "\n",
		`{"jsonrpc":"2.0","method":"notifications/initialized"}` + "\n",
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n",
		callMsg,
	}

	for _, msg := range messages {
		if _, err := io.WriteString(pw, msg); err != nil {
			t.Fatalf("failed to write to pipe: %v", err)
		}
	}
	pw.Close() // signal EOF to the server

	if err := <-serverDone; err != nil {
		t.Fatalf("serveLoop returned unexpected error: %v", err)
	}

	// Decode responses and verify same 3-response shape as the string-reader test.
	var responses []map[string]any
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		var m map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &m); err != nil {
			t.Fatalf("failed to decode response line: %v", err)
		}
		responses = append(responses, m)
	}

	if len(responses) != 3 {
		t.Fatalf("expected 3 responses, got %d", len(responses))
	}

	// Spot-check: call response text == "hello"
	resp := findByID(responses, 3)
	if resp == nil {
		t.Fatal("no tools/call response (id=3)")
	}
	result := resp["result"].(map[string]any)
	content := result["content"].([]any)
	first := content[0].(map[string]any)
	if first["text"] != "hello" {
		t.Errorf("pipe round-trip: text = %v, want hello", first["text"])
	}
}

// findByID returns the response map whose "id" field equals the given numeric id,
// or nil if not found. JSON numbers decode as float64.
func findByID(responses []map[string]any, id float64) map[string]any {
	for _, r := range responses {
		if r["id"] == id {
			return r
		}
	}
	return nil
}
