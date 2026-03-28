package agent_test

import (
	"encoding/json"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
)

func TestToolCallEvent_Fields(t *testing.T) {
	ev := agent.ToolCallEvent{
		ID:        "tc_001",
		ToolName:  "calculator",
		InputJSON: `{"expr":"1+1"}`,
	}

	if ev.ID != "tc_001" {
		t.Errorf("ID = %q, want %q", ev.ID, "tc_001")
	}
	if ev.ToolName != "calculator" {
		t.Errorf("ToolName = %q, want %q", ev.ToolName, "calculator")
	}
	if ev.InputJSON != `{"expr":"1+1"}` {
		t.Errorf("InputJSON = %q, want %q", ev.InputJSON, `{"expr":"1+1"}`)
	}
}

func TestToolResultEvent_Fields(t *testing.T) {
	ev := agent.ToolResultEvent{
		ID:       "tc_001",
		ToolName: "calculator",
		Result:   "2",
		IsError:  false,
	}

	if ev.ID != "tc_001" {
		t.Errorf("ID = %q, want %q", ev.ID, "tc_001")
	}
	if ev.ToolName != "calculator" {
		t.Errorf("ToolName = %q, want %q", ev.ToolName, "calculator")
	}
	if ev.Result != "2" {
		t.Errorf("Result = %q, want %q", ev.Result, "2")
	}
	if ev.IsError {
		t.Error("IsError should be false")
	}
}

func TestEvent_WithToolCall_JSON(t *testing.T) {
	ev := agent.Event{
		Kind: agent.EventKindToolCall,
		ToolCall: &agent.ToolCallEvent{
			ID:        "tc_1",
			ToolName:  "search",
			InputJSON: `{"q":"test"}`,
		},
	}

	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got["kind"] != "tool_call" {
		t.Errorf("kind = %v, want tool_call", got["kind"])
	}
	if _, hasToolCall := got["tool_call"]; !hasToolCall {
		t.Error("expected tool_call field in JSON")
	}
	// ToolResult must be absent (omitempty)
	if _, hasToolResult := got["tool_result"]; hasToolResult {
		t.Error("tool_result should be absent (omitempty) when nil")
	}
}

func TestEvent_WithToolResult_JSON(t *testing.T) {
	ev := agent.Event{
		Kind: agent.EventKindToolResult,
		ToolResult: &agent.ToolResultEvent{
			ID:       "tc_1",
			ToolName: "search",
			Result:   "found",
			IsError:  false,
		},
	}

	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got["kind"] != "tool_result" {
		t.Errorf("kind = %v, want tool_result", got["kind"])
	}
	if _, hasToolResult := got["tool_result"]; !hasToolResult {
		t.Error("expected tool_result field in JSON")
	}
	// ToolCall must be absent (omitempty)
	if _, hasToolCall := got["tool_call"]; hasToolCall {
		t.Error("tool_call should be absent (omitempty) when nil")
	}
}

func TestEventKindConstants_ToolEvents(t *testing.T) {
	if agent.EventKindToolCall != "tool_call" {
		t.Errorf("EventKindToolCall = %q, want %q", agent.EventKindToolCall, "tool_call")
	}
	if agent.EventKindToolResult != "tool_result" {
		t.Errorf("EventKindToolResult = %q, want %q", agent.EventKindToolResult, "tool_result")
	}
}
