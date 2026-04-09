package claudestream

import (
	"context"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
)

// ---------------------------------------------------------------------------
// buildPrompt tests
// ---------------------------------------------------------------------------

func TestBuildPrompt_Empty(t *testing.T) {
	sess := agent.NewSession("claude-stream", "claude-opus-4-6")
	if got := buildPrompt(sess); got != "" {
		t.Errorf("expected empty prompt for empty session, got %q", got)
	}
}

func TestBuildPrompt_SingleMessage(t *testing.T) {
	sess := agent.NewSession("claude-stream", "claude-opus-4-6")
	sess.AppendUserMessage("Hello, world!")
	if got := buildPrompt(sess); got != "Hello, world!" {
		t.Errorf("unexpected prompt: %q", got)
	}
}

func TestBuildPrompt_MultiTurn(t *testing.T) {
	sess := agent.NewSession("claude-stream", "claude-opus-4-6")
	sess.AppendUserMessage("First question")
	sess.AppendAssistantMessage("First answer")
	sess.AppendUserMessage("Second question")

	got := buildPrompt(sess)

	if !strings.Contains(got, "Human: First question") {
		t.Errorf("missing Human: First question in prompt, got: %q", got)
	}
	if !strings.Contains(got, "Assistant: First answer") {
		t.Errorf("missing Assistant: First answer in prompt, got: %q", got)
	}
	if !strings.Contains(got, "Human: Second question") {
		t.Errorf("missing Human: Second question in prompt, got: %q", got)
	}
}

func TestBuildPrompt_SystemMessageSkipped(t *testing.T) {
	sess := agent.NewSession("claude-stream", "claude-opus-4-6")
	sess.SystemPrompt = "You are a bot"
	sess.AppendUserMessage("Hi")

	got := buildPrompt(sess)
	if strings.Contains(got, "You are a bot") {
		t.Errorf("system prompt should not appear in transcript, got: %q", got)
	}
}

// ---------------------------------------------------------------------------
// buildArgs tests
// ---------------------------------------------------------------------------

func TestBuildArgs_Defaults(t *testing.T) {
	a := &claudeStreamAdapter{claudeBin: "claude", model: "claude-opus-4-6"}
	sess := agent.NewSession("claude-stream", "claude-opus-4-6")
	args := buildArgs(a, sess, "hello")

	assertArg(t, args, "-p", "hello")
	assertArg(t, args, "--output-format", "text")
	assertArg(t, args, "--model", "claude-opus-4-6")
	assertArg(t, args, "--max-turns", "1")
	assertNoFlag(t, args, "--verbose")   // text mode does not need --verbose
	assertNoFlag(t, args, "stream-json") // must NOT use stream-json in text mode
}

func TestBuildArgs_WithSystemPrompt(t *testing.T) {
	a := &claudeStreamAdapter{claudeBin: "claude", model: "m"}
	sess := agent.NewSession("claude-stream", "m")
	sess.SystemPrompt = "You are a helpful assistant."
	args := buildArgs(a, sess, "hi")
	assertArg(t, args, "--system-prompt", "You are a helpful assistant.")
}

func TestBuildArgs_NoSystemPrompt(t *testing.T) {
	a := &claudeStreamAdapter{claudeBin: "claude", model: "m"}
	sess := agent.NewSession("claude-stream", "m")
	args := buildArgs(a, sess, "hi")
	for i, arg := range args {
		if arg == "--system-prompt" {
			t.Errorf("unexpected --system-prompt at index %d; args: %v", i, args)
		}
	}
}

// ---------------------------------------------------------------------------
// NewClaudeStreamAgent factory tests
// ---------------------------------------------------------------------------

func TestNewClaudeStreamAgent_Defaults(t *testing.T) {
	ag, err := NewClaudeStreamAgent(map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cs := ag.(*claudeStreamAdapter)
	if cs.claudeBin != defaultBin {
		t.Errorf("expected claudeBin=%q, got %q", defaultBin, cs.claudeBin)
	}
	if cs.model != defaultModel {
		t.Errorf("expected model=%q, got %q", defaultModel, cs.model)
	}
}

func TestNewClaudeStreamAgent_CustomCfg(t *testing.T) {
	ag, err := NewClaudeStreamAgent(map[string]any{
		"claude_bin": "/usr/local/bin/claude",
		"model":      "claude-sonnet-4-6",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cs := ag.(*claudeStreamAdapter)
	if cs.claudeBin != "/usr/local/bin/claude" {
		t.Errorf("expected claudeBin=/usr/local/bin/claude, got %q", cs.claudeBin)
	}
	if cs.model != "claude-sonnet-4-6" {
		t.Errorf("expected model=claude-sonnet-4-6, got %q", cs.model)
	}
}

func TestProvider(t *testing.T) {
	ag, _ := NewClaudeStreamAgent(map[string]any{})
	if ag.Provider() != "claude-stream" {
		t.Errorf("expected provider=claude-stream, got %q", ag.Provider())
	}
}

func TestCountTokens_NotSupported(t *testing.T) {
	ag, _ := NewClaudeStreamAgent(map[string]any{})
	_, err := ag.CountTokens(context.Background(), agent.NewSession("claude-stream", "m"))
	if err != agent.ErrCountNotSupported {
		t.Errorf("expected ErrCountNotSupported, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Run — empty session error path
// ---------------------------------------------------------------------------

func TestRun_EmptySession_EmitsError(t *testing.T) {
	ag, _ := NewClaudeStreamAgent(map[string]any{})
	sess := agent.NewSession("claude-stream", "m")

	ctx := context.Background()
	var gotError bool
	for ev, err := range ag.Run(ctx, sess) {
		if ev.Kind == agent.EventKindError || err != nil {
			gotError = true
		}
	}
	if !gotError {
		t.Error("expected an error event for empty session")
	}
}

// ---------------------------------------------------------------------------
// Integration test (gated by CLAUDE_STREAM_INTEGRATION=1)
// ---------------------------------------------------------------------------

func TestClaudeStreamIntegration_BasicRun(t *testing.T) {
	if lookupEnv("CLAUDE_STREAM_INTEGRATION") != "1" {
		t.Skip("set CLAUDE_STREAM_INTEGRATION=1 to run")
	}

	ag, err := NewClaudeStreamAgent(map[string]any{
		"model": "claude-haiku-4-5-20251001",
	})
	if err != nil {
		t.Fatalf("factory error: %v", err)
	}

	sess := agent.NewSession("claude-stream", "claude-haiku-4-5-20251001")
	sess.AppendUserMessage("Reply with exactly the word: pong")

	ctx := context.Background()
	var tokens, dones int
	var text strings.Builder
	for ev, err := range ag.Run(ctx, sess) {
		if err != nil {
			t.Fatalf("stream error: %v", err)
		}
		switch ev.Kind {
		case agent.EventKindToken:
			tokens++
			text.WriteString(ev.Text)
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
	t.Logf("streamed text: %q", text.String())
}
