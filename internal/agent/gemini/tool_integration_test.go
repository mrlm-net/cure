package gemini_test

// tool_integration_test.go exercises the full tool-use session round-trip for
// the Gemini provider. It spins up an httptest.Server that serves a two-turn
// conversation:
//   - Turn 1: model requests the "get_time" tool via a functionCall part.
//   - Turn 2: model returns a plain-text final answer after receiving the tool result.
//
// Scope note: gemini_test.go has tool-loop tests (TestRun_ToolLoop_*) that
// verify the loop mechanism itself. This file adds integration-level assertions:
// strict first/last event ordering, exact HTTP call count, and tool call ID
// correlation across the full turn boundary.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	gemini "github.com/mrlm-net/cure/internal/agent/gemini"
	"github.com/mrlm-net/cure/pkg/agent"
)

// intValidStreamBody returns a complete SSE stream simulating a Gemini
// plain-text response (duplicated from gemini_test.go for the _test package).
func intValidStreamBody() string {
	first := map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"parts": []map[string]any{{"text": "The time is 12:00"}},
					"role":  "model",
				},
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     10,
			"candidatesTokenCount": 0,
		},
	}
	second := map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"parts": []map[string]any{{"text": ""}},
					"role":  "model",
				},
				"finishReason": "STOP",
			},
		},
		"usageMetadata": map[string]any{
			"promptTokenCount":     10,
			"candidatesTokenCount": 5,
		},
	}
	b1, _ := json.Marshal(first)
	b2, _ := json.Marshal(second)
	return intSSEData(string(b1)) + intSSEData(string(b2))
}

// intSSEData formats a data line for SSE.
func intSSEData(data string) string {
	return fmt.Sprintf("data: %s\n\n", data)
}

// intBuildFunctionCallStream returns an SSE stream containing a single
// functionCall part (duplicated from gemini_test.go for the _test package).
func intBuildFunctionCallStream(toolName string, args map[string]any) string {
	ev := map[string]any{
		"candidates": []map[string]any{
			{
				"content": map[string]any{
					"parts": []map[string]any{
						{"functionCall": map[string]any{"name": toolName, "args": args}},
					},
					"role": "model",
				},
				"finishReason": "FUNCTION_CALL",
			},
		},
	}
	b, _ := json.Marshal(ev)
	return intSSEData(string(b))
}

// intGetTimeTool returns a FuncTool that returns a fixed time string.
func intGetTimeTool() agent.Tool {
	return agent.FuncTool(
		"get_time",
		"Get the current time",
		map[string]any{"type": "object", "properties": map[string]any{}},
		func(_ context.Context, _ map[string]any) (string, error) {
			return "12:00", nil
		},
	)
}

// TestGeminiToolIntegration_GetTimeTool is a full round-trip integration test
// for the tool-use loop using a "get_time" FuncTool. The mock server serves:
//   - Request 1: SSE stream with a functionCall part requesting "get_time"
//   - Request 2: SSE stream with the final text answer
//
// Assertions:
//   - EventKindStart is the first event
//   - EventKindToolCall appears with ToolName == "get_time"
//   - EventKindToolResult appears with Result == "12:00" and IsError == false
//   - At least one EventKindToken with the final answer text
//   - EventKindDone is the last event
//   - The HTTP handler was called exactly twice
func TestGeminiToolIntegration_GetTimeTool(t *testing.T) {
	var requestCount atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if n == 1 {
			// Turn 1: model requests the get_time tool.
			fmt.Fprint(w, intBuildFunctionCallStream("get_time", map[string]any{}))
		} else {
			// Turn 2: model returns the final answer.
			fmt.Fprint(w, intValidStreamBody())
		}
	}))
	defer ts.Close()

	a := gemini.NewAdapterForTest("test-key", "gemini-2.5-pro", 8192, ts.URL, nil)

	ctx := context.Background()
	sess := agent.NewSession("gemini", "gemini-2.5-pro")
	sess.Tools = []agent.Tool{intGetTimeTool()}
	sess.AppendUserMessage("What time is it?")

	var (
		firstEventKind agent.EventKind
		gotToolCall    *agent.ToolCallEvent
		gotToolResult  *agent.ToolResultEvent
		gotFinalToken  bool
		lastEventKind  agent.EventKind
		eventCount     int
	)

	for ev, err := range a.Run(ctx, sess) {
		if err != nil {
			t.Fatalf("unexpected error at event %d: %v", eventCount, err)
		}
		if eventCount == 0 {
			firstEventKind = ev.Kind
		}
		lastEventKind = ev.Kind
		eventCount++

		switch ev.Kind {
		case agent.EventKindToolCall:
			if ev.ToolCall != nil {
				gotToolCall = ev.ToolCall
			}
		case agent.EventKindToolResult:
			if ev.ToolResult != nil {
				gotToolResult = ev.ToolResult
			}
		case agent.EventKindToken:
			gotFinalToken = true
		case agent.EventKindError:
			t.Fatalf("unexpected error event: %s", ev.Err)
		}
	}

	// Verify event count is sane (at least 5: start + tool_call + tool_result + start + token + done).
	if eventCount < 5 {
		t.Errorf("event count = %d, want >= 5", eventCount)
	}

	// EventKindStart must be first.
	if firstEventKind != agent.EventKindStart {
		t.Errorf("first event kind = %q, want %q", firstEventKind, agent.EventKindStart)
	}

	// EventKindDone must be last.
	if lastEventKind != agent.EventKindDone {
		t.Errorf("last event kind = %q, want %q", lastEventKind, agent.EventKindDone)
	}

	// EventKindToolCall must have appeared with ToolName == "get_time".
	if gotToolCall == nil {
		t.Fatal("no EventKindToolCall received")
	}
	if gotToolCall.ToolName != "get_time" {
		t.Errorf("ToolCall.ToolName = %q, want %q", gotToolCall.ToolName, "get_time")
	}

	// EventKindToolResult must have appeared with Result == "12:00".
	if gotToolResult == nil {
		t.Fatal("no EventKindToolResult received")
	}
	if gotToolResult.Result != "12:00" {
		t.Errorf("ToolResult.Result = %q, want %q", gotToolResult.Result, "12:00")
	}
	if gotToolResult.IsError {
		t.Error("ToolResult.IsError should be false")
	}

	// A final text token must have been emitted.
	if !gotFinalToken {
		t.Error("no EventKindToken received after tool loop")
	}

	// The server must have been called exactly twice (turn 1 + turn 2).
	if n := int(requestCount.Load()); n != 2 {
		t.Errorf("HTTP request count = %d, want 2", n)
	}
}

// TestGeminiToolIntegration_ToolCallIDPropagated verifies that the locally
// generated tool call ID (format "fc_{turn}_{i}") appears in both the
// ToolCallEvent and ToolResultEvent, ensuring correlation between request and
// response.
func TestGeminiToolIntegration_ToolCallIDPropagated(t *testing.T) {
	// Gemini generates IDs as "fc_{turn}_{index}". For turn 0, index 0 the
	// expected ID is "fc_0_0".
	const wantToolID = "fc_0_0"

	var callN atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callN.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if n == 1 {
			fmt.Fprint(w, intBuildFunctionCallStream("get_time", map[string]any{}))
		} else {
			fmt.Fprint(w, intValidStreamBody())
		}
	}))
	defer ts.Close()

	a := gemini.NewAdapterForTest("test-key", "gemini-2.5-pro", 8192, ts.URL, nil)

	ctx := context.Background()
	sess := agent.NewSession("gemini", "gemini-2.5-pro")
	sess.Tools = []agent.Tool{intGetTimeTool()}
	sess.AppendUserMessage("What time is it?")

	var (
		toolCallID   string
		toolResultID string
	)
	for ev, err := range a.Run(ctx, sess) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ev.Kind == agent.EventKindToolCall && ev.ToolCall != nil {
			toolCallID = ev.ToolCall.ID
		}
		if ev.Kind == agent.EventKindToolResult && ev.ToolResult != nil {
			toolResultID = ev.ToolResult.ID
		}
	}

	// Guard against silent missing events before comparing IDs.
	if toolCallID == "" {
		t.Fatal("no EventKindToolCall with ToolCall.ID received")
	}
	if toolResultID == "" {
		t.Fatal("no EventKindToolResult with ToolResult.ID received")
	}
	if toolCallID != wantToolID {
		t.Errorf("ToolCall.ID = %q, want %q", toolCallID, wantToolID)
	}
	if toolResultID != wantToolID {
		t.Errorf("ToolResult.ID = %q, want %q", toolResultID, wantToolID)
	}
	if toolCallID != toolResultID {
		t.Errorf("ToolCall.ID %q != ToolResult.ID %q — IDs must match", toolCallID, toolResultID)
	}
}
