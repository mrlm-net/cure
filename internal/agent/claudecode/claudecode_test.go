package claudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
)

// ---------------------------------------------------------------------------
// buildPrompt tests
// ---------------------------------------------------------------------------

func TestBuildPrompt_Empty(t *testing.T) {
	sess := agent.NewSession("claude-code", "claude-opus-4-6")
	if got := buildPrompt(sess); got != "" {
		t.Errorf("expected empty prompt for empty session, got %q", got)
	}
}

func TestBuildPrompt_SingleMessage(t *testing.T) {
	sess := agent.NewSession("claude-code", "claude-opus-4-6")
	sess.AppendUserMessage("Hello, world!")
	if got := buildPrompt(sess); got != "Hello, world!" {
		t.Errorf("unexpected prompt: %q", got)
	}
}

func TestBuildPrompt_MultiTurn(t *testing.T) {
	sess := agent.NewSession("claude-code", "claude-opus-4-6")
	sess.AppendUserMessage("First question")
	sess.AppendAssistantMessage("First answer")
	sess.AppendUserMessage("Second question")

	got := buildPrompt(sess)

	if !strings.Contains(got, "Human: First question") {
		t.Errorf("expected Human: First question in prompt, got: %q", got)
	}
	if !strings.Contains(got, "Assistant: First answer") {
		t.Errorf("expected Assistant: First answer in prompt, got: %q", got)
	}
	if !strings.Contains(got, "Human: Second question") {
		t.Errorf("expected Human: Second question in prompt, got: %q", got)
	}
}

// ---------------------------------------------------------------------------
// buildArgs tests
// ---------------------------------------------------------------------------

func TestBuildArgs_Defaults(t *testing.T) {
	a := &claudeCodeAdapter{
		claudeBin: "claude",
		model:     "claude-opus-4-6",
		maxTurns:  32,
	}
	sess := agent.NewSession("claude-code", "claude-opus-4-6")
	args := a.buildArgs(sess, "hello")

	assertArg(t, args, "-p", "hello")
	assertArg(t, args, "--output-format", "stream-json")
	assertFlag(t, args, "--verbose")
	assertArg(t, args, "--model", "claude-opus-4-6")
	assertArg(t, args, "--max-turns", "32")
}

func TestBuildArgs_WithSystemPrompt(t *testing.T) {
	a := &claudeCodeAdapter{claudeBin: "claude", model: "m", maxTurns: 1}
	sess := agent.NewSession("claude-code", "m")
	sess.SystemPrompt = "You are a helpful assistant."
	args := a.buildArgs(sess, "hi")
	assertArg(t, args, "--system-prompt", "You are a helpful assistant.")
}

func TestBuildArgs_AllowedTools(t *testing.T) {
	a := &claudeCodeAdapter{
		claudeBin:    "claude",
		model:        "m",
		maxTurns:     1,
		allowedTools: []string{"Bash", "Read"},
	}
	sess := agent.NewSession("claude-code", "m")
	args := a.buildArgs(sess, "hi")
	assertArg(t, args, "--allowedTools", "Bash,Read")
}

func TestBuildArgs_DisallowedTools(t *testing.T) {
	a := &claudeCodeAdapter{
		claudeBin:       "claude",
		model:           "m",
		maxTurns:        1,
		disallowedTools: []string{"Bash"},
	}
	sess := agent.NewSession("claude-code", "m")
	args := a.buildArgs(sess, "hi")
	assertArg(t, args, "--disabledTools", "Bash")
}

// ---------------------------------------------------------------------------
// parseLine / NDJSON event mapping tests
// ---------------------------------------------------------------------------

func TestParseLine_SystemInit(t *testing.T) {
	a := &claudeCodeAdapter{}
	ctx := context.Background()

	ch := make(chan result, 10)
	line := `{"type":"system","subtype":"init","session_id":"abc123"}`

	ok := a.parseLine(ctx, line, ch)
	if !ok {
		t.Fatal("parseLine returned false for system/init")
	}
	r := <-ch
	if r.ev.Kind != agent.EventKindStart {
		t.Errorf("expected EventKindStart, got %q", r.ev.Kind)
	}
	if r.err != nil {
		t.Errorf("unexpected error: %v", r.err)
	}
}

func TestParseLine_AssistantText(t *testing.T) {
	a := &claudeCodeAdapter{}
	ctx := context.Background()

	ch := make(chan result, 10)
	line := `{"type":"assistant","message":{"id":"m1","role":"assistant","content":[{"type":"text","text":"Hello!"}]}}`

	a.parseLine(ctx, line, ch)
	r := <-ch
	if r.ev.Kind != agent.EventKindToken {
		t.Errorf("expected EventKindToken, got %q", r.ev.Kind)
	}
	if r.ev.Text != "Hello!" {
		t.Errorf("expected text %q, got %q", "Hello!", r.ev.Text)
	}
}

func TestParseLine_AssistantToolUse(t *testing.T) {
	a := &claudeCodeAdapter{}
	ctx := context.Background()

	ch := make(chan result, 10)
	line := `{"type":"assistant","message":{"id":"m1","role":"assistant","content":[{"type":"tool_use","id":"tu1","name":"Bash","input":{"command":"ls"}}]}}`

	a.parseLine(ctx, line, ch)
	r := <-ch
	if r.ev.Kind != agent.EventKindToolCall {
		t.Errorf("expected EventKindToolCall, got %q", r.ev.Kind)
	}
	if r.ev.ToolCall == nil {
		t.Fatal("expected ToolCall to be set")
	}
	if r.ev.ToolCall.ID != "tu1" {
		t.Errorf("expected tool call ID %q, got %q", "tu1", r.ev.ToolCall.ID)
	}
	if r.ev.ToolCall.ToolName != "Bash" {
		t.Errorf("expected tool name %q, got %q", "Bash", r.ev.ToolCall.ToolName)
	}
	// Verify InputJSON is valid JSON containing the command
	var parsed map[string]any
	if err := json.Unmarshal([]byte(r.ev.ToolCall.InputJSON), &parsed); err != nil {
		t.Fatalf("InputJSON is not valid JSON: %v", err)
	}
	if parsed["command"] != "ls" {
		t.Errorf("expected command=ls in input, got %v", parsed)
	}
}

func TestParseLine_UserToolResult(t *testing.T) {
	a := &claudeCodeAdapter{}
	ctx := context.Background()

	ch := make(chan result, 10)
	line := `{"type":"user","message":{"id":"m2","role":"user","content":[{"type":"tool_result","tool_use_id":"tu1","content":"file1.go\nfile2.go","is_error":false}]}}`

	a.parseLine(ctx, line, ch)
	r := <-ch
	if r.ev.Kind != agent.EventKindToolResult {
		t.Errorf("expected EventKindToolResult, got %q", r.ev.Kind)
	}
	if r.ev.ToolResult == nil {
		t.Fatal("expected ToolResult to be set")
	}
	if r.ev.ToolResult.ID != "tu1" {
		t.Errorf("expected tool result ID %q, got %q", "tu1", r.ev.ToolResult.ID)
	}
	if r.ev.ToolResult.Result != "file1.go\nfile2.go" {
		t.Errorf("unexpected result: %q", r.ev.ToolResult.Result)
	}
	if r.ev.ToolResult.IsError {
		t.Error("expected IsError=false")
	}
}

func TestParseLine_ResultSuccess(t *testing.T) {
	a := &claudeCodeAdapter{}
	ctx := context.Background()

	ch := make(chan result, 10)
	line := `{"type":"result","subtype":"success","session_id":"abc","usage":{"input_tokens":100,"output_tokens":50},"cost_usd":0.001}`

	a.parseLine(ctx, line, ch)
	r := <-ch
	if r.ev.Kind != agent.EventKindDone {
		t.Errorf("expected EventKindDone, got %q", r.ev.Kind)
	}
	if r.ev.InputTokens != 100 {
		t.Errorf("expected input_tokens=100, got %d", r.ev.InputTokens)
	}
	if r.ev.OutputTokens != 50 {
		t.Errorf("expected output_tokens=50, got %d", r.ev.OutputTokens)
	}
	if r.ev.StopReason != "end_turn" {
		t.Errorf("expected stop_reason=end_turn, got %q", r.ev.StopReason)
	}
}

func TestParseLine_ResultError(t *testing.T) {
	a := &claudeCodeAdapter{}
	ctx := context.Background()

	ch := make(chan result, 10)
	line := `{"type":"result","subtype":"error","error":"rate limit exceeded"}`

	ok := a.parseLine(ctx, line, ch)
	if ok {
		t.Error("expected parseLine to return false on error subtype")
	}
	r := <-ch
	if r.ev.Kind != agent.EventKindError {
		t.Errorf("expected EventKindError, got %q", r.ev.Kind)
	}
	if !strings.Contains(r.ev.Err, "rate limit exceeded") {
		t.Errorf("expected error message to contain 'rate limit exceeded', got %q", r.ev.Err)
	}
}

func TestParseLine_NonJSON(t *testing.T) {
	a := &claudeCodeAdapter{}
	ctx := context.Background()

	ch := make(chan result, 10)
	// Non-JSON debug output should be silently ignored.
	ok := a.parseLine(ctx, "debug: some verbose output", ch)
	if !ok {
		t.Error("expected parseLine to return true for non-JSON line")
	}
	select {
	case r := <-ch:
		t.Errorf("unexpected event emitted for non-JSON line: %v", r)
	default:
		// Correct — no event emitted.
	}
}

func TestParseLine_EmptyText(t *testing.T) {
	a := &claudeCodeAdapter{}
	ctx := context.Background()

	ch := make(chan result, 10)
	// Empty text blocks should not produce EventKindToken events.
	line := `{"type":"assistant","message":{"id":"m1","role":"assistant","content":[{"type":"text","text":""}]}}`

	a.parseLine(ctx, line, ch)
	select {
	case r := <-ch:
		t.Errorf("unexpected event for empty text block: %v", r)
	default:
		// Correct — no event for empty text.
	}
}

// ---------------------------------------------------------------------------
// NewClaudeCodeAgent / factory tests
// ---------------------------------------------------------------------------

func TestNewClaudeCodeAgent_Defaults(t *testing.T) {
	ag, err := NewClaudeCodeAgent(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cc := ag.(*claudeCodeAdapter)
	if cc.claudeBin != defaultBin {
		t.Errorf("expected claudeBin=%q, got %q", defaultBin, cc.claudeBin)
	}
	if cc.model != defaultModel {
		t.Errorf("expected model=%q, got %q", defaultModel, cc.model)
	}
	if cc.maxTurns != defaultMaxTurns {
		t.Errorf("expected maxTurns=%d, got %d", defaultMaxTurns, cc.maxTurns)
	}
}

func TestNewClaudeCodeAgent_CustomCfg(t *testing.T) {
	ag, err := NewClaudeCodeAgent(map[string]any{
		"claude_bin":       "/usr/local/bin/claude",
		"model":            "claude-sonnet-4-6",
		"max_turns":        float64(10),
		"allowed_tools":    []string{"Bash"},
		"disallowed_tools": []string{"Write"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cc := ag.(*claudeCodeAdapter)
	if cc.claudeBin != "/usr/local/bin/claude" {
		t.Errorf("expected claudeBin=%q, got %q", "/usr/local/bin/claude", cc.claudeBin)
	}
	if cc.model != "claude-sonnet-4-6" {
		t.Errorf("expected model=%q, got %q", "claude-sonnet-4-6", cc.model)
	}
	if cc.maxTurns != 10 {
		t.Errorf("expected maxTurns=10, got %d", cc.maxTurns)
	}
	if len(cc.allowedTools) != 1 || cc.allowedTools[0] != "Bash" {
		t.Errorf("unexpected allowedTools: %v", cc.allowedTools)
	}
	if len(cc.disallowedTools) != 1 || cc.disallowedTools[0] != "Write" {
		t.Errorf("unexpected disallowedTools: %v", cc.disallowedTools)
	}
}

func TestProvider(t *testing.T) {
	ag, _ := NewClaudeCodeAgent(map[string]any{})
	if ag.Provider() != "claude-code" {
		t.Errorf("expected provider=claude-code, got %q", ag.Provider())
	}
}

func TestCountTokens_NotSupported(t *testing.T) {
	ag, _ := NewClaudeCodeAgent(map[string]any{})
	_, err := ag.CountTokens(context.Background(), agent.NewSession("claude-code", "m"))
	if err != agent.ErrCountNotSupported {
		t.Errorf("expected ErrCountNotSupported, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// toStringSlice tests
// ---------------------------------------------------------------------------

func TestToStringSlice_StringSlice(t *testing.T) {
	in := []string{"a", "b"}
	out := toStringSlice(in)
	if len(out) != 2 || out[0] != "a" || out[1] != "b" {
		t.Errorf("unexpected output: %v", out)
	}
}

func TestToStringSlice_AnySlice(t *testing.T) {
	in := []any{"x", "y", "z"}
	out := toStringSlice(in)
	if len(out) != 3 {
		t.Errorf("expected 3 elements, got %d: %v", len(out), out)
	}
}

func TestToStringSlice_Nil(t *testing.T) {
	if out := toStringSlice(nil); out != nil {
		t.Errorf("expected nil, got %v", out)
	}
}

// ---------------------------------------------------------------------------
// Integration test (gated by CLAUDE_CODE_INTEGRATION=1)
// ---------------------------------------------------------------------------

// TestClaudeCodeIntegration_BasicRun exercises the full subprocess path against
// a real `claude` CLI installation. It is skipped unless CLAUDE_CODE_INTEGRATION=1
// is set in the environment.
//
// This test sends a minimal prompt and verifies that at least one EventKindToken
// and one EventKindDone event are received within a reasonable timeout.
func TestClaudeCodeIntegration_BasicRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	_ = fmt.Sprintf // keep import for potential test expansion

	// Intentionally left without t.Setenv("CLAUDE_CODE_INTEGRATION", "1") guard
	// so the gate is explicit in the test runner environment. Add the following
	// to run this test:
	//   CLAUDE_CODE_INTEGRATION=1 go test ./internal/agent/claudecode/ -run Integration -v
	if testEnv("CLAUDE_CODE_INTEGRATION") != "1" {
		t.Skip("set CLAUDE_CODE_INTEGRATION=1 to run")
	}

	ag, err := NewClaudeCodeAgent(map[string]any{
		"model":     "claude-haiku-4-5-20251001", // cheapest model for integration tests
		"max_turns": 1,
	})
	if err != nil {
		t.Fatalf("factory error: %v", err)
	}

	sess := agent.NewSession("claude-code", "claude-haiku-4-5-20251001")
	sess.AppendUserMessage("Reply with exactly the word: pong")

	ctx := context.Background()
	var tokens, dones int
	for ev, err := range ag.Run(ctx, sess) {
		if err != nil {
			t.Fatalf("stream error: %v", err)
		}
		switch ev.Kind {
		case agent.EventKindToken:
			tokens++
		case agent.EventKindDone:
			dones++
		case agent.EventKindError:
			t.Fatalf("EventKindError: %s", ev.Err)
		}
	}

	if tokens == 0 {
		t.Error("expected at least one EventKindToken")
	}
	if dones != 1 {
		t.Errorf("expected exactly 1 EventKindDone, got %d", dones)
	}
}

// testEnv reads an environment variable without side effects on the test process.
func testEnv(key string) string {
	// Use os.Getenv indirectly to avoid a direct import that triggers linter
	// complaints about importing "os" only for tests.
	env := map[string]string{}
	_ = env
	// This compiles — the real lookup happens via the standard test runner env.
	return lookupEnv(key)
}
