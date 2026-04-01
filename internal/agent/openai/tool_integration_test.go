package openai

// tool_integration_test.go exercises the full tool-use session round-trip for
// the OpenAI provider. It spins up an httptest.Server that serves a two-turn
// conversation:
//   - Turn 1: model requests the "get_time" tool via an SSE tool_calls delta.
//   - Turn 2: model returns a plain-text final answer after receiving the tool result.
//
// Scope note: openai_test.go has tool-loop tests (TestRun_ToolLoop_*) that
// verify the loop mechanism itself. This file adds integration-level assertions:
// strict first/last event ordering, exact HTTP call count, and tool call ID
// correlation across the full turn boundary.

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
)

// getTimeTool returns a FuncTool that returns a fixed time string.
func getTimeTool() agent.Tool {
	return agent.FuncTool(
		"get_time",
		"Get the current time",
		map[string]any{"type": "object", "properties": map[string]any{}},
		func(_ context.Context, _ map[string]any) (string, error) {
			return "12:00", nil
		},
	)
}

// TestOpenAIToolIntegration_GetTimeTool is a full round-trip integration test
// for the tool-use loop using a "get_time" FuncTool. The mock server serves:
//   - Request 1: SSE stream with tool_calls delta requesting "get_time"
//   - Request 2: SSE stream with the final text answer
//
// Assertions:
//   - EventKindStart is the first event
//   - EventKindToolCall appears with ToolName == "get_time"
//   - EventKindToolResult appears with Result == "12:00" and IsError == false
//   - At least one EventKindToken with the final answer text
//   - EventKindDone is the last event
//   - The HTTP handler was called exactly twice
func TestOpenAIToolIntegration_GetTimeTool(t *testing.T) {
	var requestCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := requestCount.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if n == 1 {
			// Turn 1: model requests the get_time tool.
			fmt.Fprint(w, buildSSEToolCallResponse("call_time_001", "get_time", `{}`))
		} else {
			// Turn 2: model returns the final answer.
			fmt.Fprint(w, buildSSEResponse([]string{"The time is 12:00"}))
		}
	}))
	defer srv.Close()

	a := NewAdapterForTest("sk-integ-test", srv.URL, "gpt-4o")

	sess := agent.NewSession("openai", "gpt-4o")
	sess.Tools = []agent.Tool{getTimeTool()}
	sess.AppendUserMessage("What time is it?")

	var (
		firstEventKind agent.EventKind
		gotToolCall    *agent.ToolCallEvent
		gotToolResult  *agent.ToolResultEvent
		gotFinalToken  bool
		lastEventKind  agent.EventKind
		eventCount     int
	)

	for ev, err := range a.Run(context.Background(), sess) {
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

// TestOpenAIToolIntegration_ToolCallIDPropagated verifies that the tool call ID
// from the model's SSE delta (e.g., "call_abc123") is correctly propagated
// through the ToolCallEvent and ToolResultEvent, ensuring correlation between
// request and response.
func TestOpenAIToolIntegration_ToolCallIDPropagated(t *testing.T) {
	const wantToolID = "call_abc123"

	var callN atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callN.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if n == 1 {
			fmt.Fprint(w, buildSSEToolCallResponse(wantToolID, "get_time", `{}`))
		} else {
			fmt.Fprint(w, buildSSEResponse([]string{"The time is 12:00"}))
		}
	}))
	defer srv.Close()

	a := NewAdapterForTest("sk-id-test", srv.URL, "gpt-4o")

	sess := agent.NewSession("openai", "gpt-4o")
	sess.Tools = []agent.Tool{getTimeTool()}
	sess.AppendUserMessage("What time is it?")

	var (
		toolCallID   string
		toolResultID string
	)
	for ev, err := range a.Run(context.Background(), sess) {
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
