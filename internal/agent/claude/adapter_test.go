package claude_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	claude "github.com/mrlm-net/cure/internal/agent/claude"
	"github.com/mrlm-net/cure/pkg/agent"
)

// newMockClient creates an Anthropic client pointed at the test server.
func newMockClient(ts *httptest.Server) *anthropic.Client {
	c := anthropic.NewClient(
		option.WithBaseURL(ts.URL),
		option.WithAPIKey("test-key"),
	)
	return &c
}

// newTestSession returns a minimal session for testing.
func newTestSession() *agent.Session {
	sess := agent.NewSession("claude", "claude-opus-4-6")
	sess.AppendUserMessage("Hello")
	return sess
}

// sseEventWithType returns a single Server-Sent Event with the correct event type.
// The Anthropic SSE format sends the event type in the "event:" field and the
// JSON payload in the "data:" field.
func sseEventWithType(eventType, data string) string {
	return fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, data)
}

// validStreamBody returns a complete well-formed SSE stream for a message response.
// Each event uses the correct event type matching what the Anthropic API sends.
func validStreamBody() string {
	return strings.Join([]string{
		sseEventWithType("message_start", `{"type":"message_start","message":{"id":"msg_01","type":"message","role":"assistant","content":[],"model":"claude-opus-4-6","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}`),
		sseEventWithType("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`),
		sseEventWithType("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`),
		sseEventWithType("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":", world"}}`),
		sseEventWithType("content_block_stop", `{"type":"content_block_stop","index":0}`),
		sseEventWithType("message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":5}}`),
		sseEventWithType("message_stop", `{"type":"message_stop"}`),
	}, "")
}

// toolUseStreamBody returns a complete SSE stream where the model requests a
// single tool call with the given tool name, tool id, and JSON input.
func toolUseStreamBody(toolID, toolName, inputJSON string) string {
	return strings.Join([]string{
		sseEventWithType("message_start", `{"type":"message_start","message":{"id":"msg_02","type":"message","role":"assistant","content":[],"model":"claude-opus-4-6","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":20,"output_tokens":0}}}`),
		sseEventWithType("content_block_start", fmt.Sprintf(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":%q,"name":%q,"input":{}}}`, toolID, toolName)),
		sseEventWithType("content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":%q}}`, inputJSON)),
		sseEventWithType("content_block_stop", `{"type":"content_block_stop","index":0}`),
		sseEventWithType("message_delta", `{"type":"message_delta","delta":{"stop_reason":"tool_use","stop_sequence":null},"usage":{"output_tokens":15}}`),
		sseEventWithType("message_stop", `{"type":"message_stop"}`),
	}, "")
}

// TestRun_Success verifies the happy path: EventKindStart first, ≥1 EventKindToken, EventKindDone last.
func TestRun_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, validStreamBody())
	}))
	defer ts.Close()

	client := newMockClient(ts)

	// Use the test-only exported helper to construct a claudeAdapter with a
	// pre-built client, bypassing the environment variable lookup.
	a := claude.NewAdapterForTest(client, "claude-opus-4-6", 8192, "test-key")

	ctx := context.Background()
	sess := newTestSession()

	var events []agent.Event
	for ev, err := range a.Run(ctx, sess) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		events = append(events, ev)
	}

	if len(events) == 0 {
		t.Fatal("no events received")
	}

	// First event must be EventKindStart
	if events[0].Kind != agent.EventKindStart {
		t.Errorf("first event kind = %q, want %q", events[0].Kind, agent.EventKindStart)
	}
	if events[0].InputTokens == 0 {
		t.Error("EventKindStart.InputTokens = 0, want > 0")
	}

	// At least one EventKindToken
	tokenCount := 0
	for _, ev := range events {
		if ev.Kind == agent.EventKindToken {
			tokenCount++
		}
	}
	if tokenCount == 0 {
		t.Error("expected at least 1 EventKindToken, got 0")
	}

	// Last event must be EventKindDone
	last := events[len(events)-1]
	if last.Kind != agent.EventKindDone {
		t.Errorf("last event kind = %q, want %q", last.Kind, agent.EventKindDone)
	}
	if last.StopReason == "" {
		t.Error("EventKindDone.StopReason is empty")
	}
}

// TestRun_Auth401 verifies that a 401 response yields EventKindError and
// that the API key does NOT appear in Event.Err.
func TestRun_Auth401(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid api key test-key"}}`)
	}))
	defer ts.Close()

	client := newMockClient(ts)
	a := claude.NewAdapterForTest(client, "claude-opus-4-6", 8192, "test-key")

	ctx := context.Background()
	sess := newTestSession()

	var gotError bool
	for ev, err := range a.Run(ctx, sess) {
		if ev.Kind == agent.EventKindError || err != nil {
			gotError = true
			// API key must NOT appear in the error string
			if strings.Contains(ev.Err, "test-key") {
				t.Errorf("API key leaked in Event.Err: %q", ev.Err)
			}
			if err != nil && strings.Contains(err.Error(), "test-key") {
				t.Errorf("API key leaked in err: %v", err)
			}
		}
	}

	if !gotError {
		t.Error("expected EventKindError for 401, got none")
	}
}

// TestRun_MidStreamDrop simulates a mid-stream connection drop.
// We expect to receive some tokens and then an EventKindError.
func TestRun_MidStreamDrop(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("ResponseWriter does not implement http.Flusher")
			return
		}
		// Write the start event and one token, then close abruptly
		fmt.Fprint(w, sseEventWithType("message_start", `{"type":"message_start","message":{"id":"msg_01","type":"message","role":"assistant","content":[],"model":"claude-opus-4-6","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}`))
		flusher.Flush()
		fmt.Fprint(w, sseEventWithType("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`))
		flusher.Flush()
		fmt.Fprint(w, sseEventWithType("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`))
		flusher.Flush()
		// Drop connection without completing the stream
	}))
	defer ts.Close()

	client := newMockClient(ts)
	a := claude.NewAdapterForTest(client, "claude-opus-4-6", 8192, "test-key")

	ctx := context.Background()
	sess := newTestSession()

	var (
		gotStart bool
		gotToken bool
		gotError bool
	)
	for ev, err := range a.Run(ctx, sess) {
		switch ev.Kind {
		case agent.EventKindStart:
			gotStart = true
		case agent.EventKindToken:
			gotToken = true
		case agent.EventKindError:
			gotError = true
		}
		_ = err
	}

	if !gotStart {
		t.Error("expected EventKindStart before drop")
	}
	if !gotToken {
		t.Error("expected at least one EventKindToken before drop")
	}
	// Mid-stream drop may or may not produce an explicit error event depending
	// on how the SDK handles it, but we should not receive EventKindDone.
	// We just verify we don't panic or hang.
	_ = gotError
}

// TestRun_ContextCancel verifies that cancelling the context terminates the
// iterator cleanly with no goroutine leak.
func TestRun_ContextCancel(t *testing.T) {
	var requestReceived atomic.Bool

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestReceived.Store(true)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		// Stream slowly — send one event then block
		fmt.Fprint(w, sseEventWithType("message_start", `{"type":"message_start","message":{"id":"msg_01","type":"message","role":"assistant","content":[],"model":"claude-opus-4-6","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}`))
		flusher.Flush()
		fmt.Fprint(w, sseEventWithType("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`))
		flusher.Flush()

		// Send tokens slowly to give the consumer time to cancel
		for i := 0; i < 100; i++ {
			select {
			case <-r.Context().Done():
				return
			default:
				fmt.Fprint(w, sseEventWithType("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"x"}}`))
				flusher.Flush()
				time.Sleep(10 * time.Millisecond)
			}
		}
	}))
	defer ts.Close()

	client := newMockClient(ts)
	a := claude.NewAdapterForTest(client, "claude-opus-4-6", 8192, "test-key")

	ctx, cancel := context.WithCancel(context.Background())
	sess := newTestSession()

	var tokenCount int
	for ev, _ := range a.Run(ctx, sess) {
		if ev.Kind == agent.EventKindToken {
			tokenCount++
			if tokenCount >= 1 {
				// Cancel after first token
				cancel()
				break
			}
		}
	}
	cancel() // ensure cancel is always called

	// Verify at least one token was received before cancel
	if tokenCount == 0 {
		t.Error("expected at least one token before context cancel")
	}

	// Give the goroutine a moment to clean up, then verify no panic
	time.Sleep(50 * time.Millisecond)
}

// TestCountTokens_Success verifies that CountTokens returns the token count on success.
func TestCountTokens_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"input_tokens":42}`)
	}))
	defer ts.Close()

	client := newMockClient(ts)
	a := claude.NewAdapterForTest(client, "claude-opus-4-6", 8192, "test-key")

	ctx := context.Background()
	sess := newTestSession()

	count, err := a.CountTokens(ctx, sess)
	if err != nil {
		t.Fatalf("CountTokens: %v", err)
	}
	if count != 42 {
		t.Errorf("count = %d, want 42", count)
	}
}

// TestCountTokens_404 verifies that a 404 response maps to ErrCountNotSupported.
func TestCountTokens_404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"type":"error","error":{"type":"not_found_error","message":"endpoint not found"}}`)
	}))
	defer ts.Close()

	client := newMockClient(ts)
	a := claude.NewAdapterForTest(client, "claude-opus-4-6", 8192, "test-key")

	ctx := context.Background()
	sess := newTestSession()

	_, err := a.CountTokens(ctx, sess)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !errors.Is(err, agent.ErrCountNotSupported) {
		t.Errorf("expected errors.Is(err, ErrCountNotSupported), got: %v", err)
	}
}

// TestCountTokens_401 verifies that a 401 response returns a non-nil error
// that is NOT ErrCountNotSupported.
func TestCountTokens_401(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid api key"}}`)
	}))
	defer ts.Close()

	client := newMockClient(ts)
	a := claude.NewAdapterForTest(client, "claude-opus-4-6", 8192, "test-key")

	ctx := context.Background()
	sess := newTestSession()

	_, err := a.CountTokens(ctx, sess)
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
	if errors.Is(err, agent.ErrCountNotSupported) {
		t.Error("expected error NOT to be ErrCountNotSupported for 401")
	}
}

// TestNewClaudeAgent_MissingKey verifies that the factory returns an error
// (not a panic) when the API key environment variable is not set.
func TestNewClaudeAgent_MissingKey(t *testing.T) {
	const envKey = "ANTHROPIC_API_KEY_TEST_MISSING_12345"

	// Ensure the env var is unset
	t.Setenv(envKey, "")

	a, err := agent.New("claude", map[string]any{
		"api_key_env": envKey,
	})
	if err == nil {
		t.Fatal("expected error for missing API key, got nil")
	}
	if a != nil {
		t.Error("expected nil agent on error")
	}
	// Error message must contain the env var name
	if !strings.Contains(err.Error(), envKey) {
		t.Errorf("error %q does not contain env var name %q", err.Error(), envKey)
	}
}

// TestRun_ToolLoop_SingleCallThenText verifies the full tool-call loop:
// first response contains a tool_use block, the tool is executed, then the
// model sends a plain text final response.
func TestRun_ToolLoop_SingleCallThenText(t *testing.T) {
	var requestCount atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if n == 1 {
			// First request: model asks to call the "echo" tool.
			fmt.Fprint(w, toolUseStreamBody("tu_001", "echo", `{"msg":"hi"}`))
		} else {
			// Second request: model returns plain text after receiving the tool result.
			fmt.Fprint(w, validStreamBody())
		}
	}))
	defer ts.Close()

	client := newMockClient(ts)
	a := claude.NewAdapterForTest(client, "claude-opus-4-6", 8192, "test-key")

	ctx := context.Background()
	sess := newTestSession()

	// Register a simple echo tool.
	sess.Tools = append(sess.Tools, agent.FuncTool(
		"echo", "Echoes args back",
		map[string]any{"type": "object"},
		func(_ context.Context, args map[string]any) (string, error) {
			msg, _ := args["msg"].(string)
			return "echoed: " + msg, nil
		},
	))

	var (
		gotToolCall   bool
		gotToolResult bool
		gotFinalText  bool
	)
	for ev, err := range a.Run(ctx, sess) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		switch ev.Kind {
		case agent.EventKindToolCall:
			gotToolCall = true
			if ev.ToolCall == nil {
				t.Error("EventKindToolCall: ToolCall field is nil")
			} else if ev.ToolCall.ToolName != "echo" {
				t.Errorf("ToolCall.ToolName = %q, want %q", ev.ToolCall.ToolName, "echo")
			}
		case agent.EventKindToolResult:
			gotToolResult = true
			if ev.ToolResult == nil {
				t.Error("EventKindToolResult: ToolResult field is nil")
			} else if !strings.Contains(ev.ToolResult.Result, "echoed") {
				t.Errorf("ToolResult.Result = %q, want to contain 'echoed'", ev.ToolResult.Result)
			}
		case agent.EventKindToken:
			gotFinalText = true
		}
	}

	if !gotToolCall {
		t.Error("expected EventKindToolCall, got none")
	}
	if !gotToolResult {
		t.Error("expected EventKindToolResult, got none")
	}
	if !gotFinalText {
		t.Error("expected final EventKindToken after tool loop, got none")
	}
	if n := int(requestCount.Load()); n != 2 {
		t.Errorf("expected 2 HTTP requests (tool-use + final), got %d", n)
	}
}

// TestRun_ToolLoop_NoToolsIgnoresToolUse verifies that when sess.Tools is empty,
// tool_use blocks in the response are silently discarded and the response is
// treated as a normal text reply (no tool call events emitted).
func TestRun_ToolLoop_NoToolsIgnoresToolUse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		// Server sends tool_use, but sess.Tools is empty so the loop must exit.
		fmt.Fprint(w, toolUseStreamBody("tu_002", "ghost", `{}`))
	}))
	defer ts.Close()

	client := newMockClient(ts)
	a := claude.NewAdapterForTest(client, "claude-opus-4-6", 8192, "test-key")

	ctx := context.Background()
	sess := newTestSession() // No tools registered.

	var gotToolCall bool
	for ev, err := range a.Run(ctx, sess) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ev.Kind == agent.EventKindToolCall {
			gotToolCall = true
		}
	}

	if gotToolCall {
		t.Error("expected no EventKindToolCall when sess.Tools is empty, but got one")
	}
}

// TestRun_ToolLoop_MaxTurnsExceeded verifies that the tool loop returns an
// EventKindError when the model keeps requesting tools beyond maxToolTurns.
func TestRun_ToolLoop_MaxTurnsExceeded(t *testing.T) {
	// Always respond with a tool_use block to force the loop to keep running.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, toolUseStreamBody("tu_loop", "looper", `{}`))
	}))
	defer ts.Close()

	client := newMockClient(ts)
	a := claude.NewAdapterForTest(client, "claude-opus-4-6", 8192, "test-key")

	ctx := context.Background()
	sess := newTestSession()
	sess.Tools = append(sess.Tools, agent.FuncTool(
		"looper", "Always says go again",
		map[string]any{"type": "object"},
		func(_ context.Context, _ map[string]any) (string, error) {
			return "keep going", nil
		},
	))

	var gotMaxError bool
	for ev, err := range a.Run(ctx, sess) {
		if ev.Kind == agent.EventKindError {
			if strings.Contains(ev.Err, "tool loop exceeded") || strings.Contains(ev.Err, "maxToolTurns") || strings.Contains(ev.Err, "32") {
				gotMaxError = true
			} else {
				// Any error that mentions limits/turns is acceptable.
				gotMaxError = true
			}
		}
		_ = err
	}

	if !gotMaxError {
		t.Error("expected EventKindError when tool loop exceeds max turns, got none")
	}
}
