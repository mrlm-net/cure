// Package claudestream provides a streaming text adapter for pkg/agent.
// It invokes the `claude` CLI with --output-format text so that response
// text is piped character-by-character as it is generated, giving true
// token streaming without requiring an Anthropic API key (uses Claude
// subscription auth via the local claude binary).
//
// Import with a blank import to register the "claude-stream" provider:
//
//	import _ "github.com/mrlm-net/cure/internal/agent/claudestream"
//
// # Streaming model
//
// Unlike the claudecode adapter (which uses --output-format stream-json and
// emits ONE complete assistant message per turn), this adapter uses
// --output-format text. The claude CLI streams response text to stdout as
// it is generated, so the reader sees incremental output rather than a
// single bulk write at the end.
//
// # Trade-offs vs. claudecode adapter
//
// This adapter does NOT emit tool-call or tool-result events; it only
// emits start / token / done / error events. This is acceptable for a
// chat UI that does not require agentic tool use.
//
// # Multi-turn sessions
//
// History is collapsed into a single Human/Assistant transcript and passed
// as the -p prompt. --max-turns is fixed at 1 since each GUI request is a
// single response turn; multi-turn context is provided via the transcript.
package claudestream

import (
	"bufio"
	"context"
	"fmt"
	"iter"
	"os/exec"
	"strings"

	"github.com/mrlm-net/cure/pkg/agent"
)

const (
	defaultModel = "claude-opus-4-6"
	defaultBin   = "claude"
)

func init() {
	agent.Register("claude-stream", NewClaudeStreamAgent)
}

// claudeStreamAdapter implements agent.Agent using the claude CLI in text
// output mode for real streaming.
type claudeStreamAdapter struct {
	claudeBin string
	model     string
}

// NewClaudeStreamAgent is the AgentFactory for the "claude-stream" provider.
//
// Recognised cfg keys:
//   - "claude_bin" string — path to claude binary (default: "claude")
//   - "model"      string — model name (default: "claude-opus-4-6")
func NewClaudeStreamAgent(cfg map[string]any) (agent.Agent, error) {
	bin := defaultBin
	if v, ok := cfg["claude_bin"].(string); ok && v != "" {
		bin = v
	}
	model := defaultModel
	if v, ok := cfg["model"].(string); ok && v != "" {
		model = v
	}
	return &claudeStreamAdapter{claudeBin: bin, model: model}, nil
}

// Provider returns the provider name "claude-stream".
func (a *claudeStreamAdapter) Provider() string { return "claude-stream" }

// CountTokens is not supported by the text streaming adapter.
func (a *claudeStreamAdapter) CountTokens(_ context.Context, _ *agent.Session) (int, error) {
	return 0, agent.ErrCountNotSupported
}

// Run streams a response for the given session using the claude CLI in text
// output mode. The caller iterates events with:
//
//	for ev, err := range a.Run(ctx, session) { ... }
//
// Cancelling ctx terminates the subprocess cleanly.
func (a *claudeStreamAdapter) Run(ctx context.Context, sess *agent.Session) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		prompt := buildPrompt(sess)
		if prompt == "" {
			msg := "claude-stream: session has no user message"
			yield(agent.Event{Kind: agent.EventKindError, Err: msg},
				fmt.Errorf("%s", msg)) //nolint:goerr113
			return
		}

		args := buildArgs(a, sess, prompt)
		cmd := exec.CommandContext(ctx, a.claudeBin, args...) //nolint:gosec // validated path, not user input
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			msg := fmt.Sprintf("claude-stream: stdout pipe: %v", err)
			yield(agent.Event{Kind: agent.EventKindError, Err: msg},
				fmt.Errorf("%s", msg)) //nolint:goerr113
			return
		}

		if err := cmd.Start(); err != nil {
			msg := fmt.Sprintf("claude-stream: start subprocess: %v", err)
			yield(agent.Event{Kind: agent.EventKindError, Err: msg},
				fmt.Errorf("%s", msg)) //nolint:goerr113
			return
		}

		// Emit start before any text arrives.
		if !yield(agent.Event{Kind: agent.EventKindStart}, nil) {
			return
		}

		// Stream text line-by-line.  ScanLines strips the newline; we restore it
		// so the UI can render paragraphs correctly.  The last line (which may
		// not have a trailing newline) is still emitted without modification.
		scanner := bufio.NewScanner(stdout)
		const maxScanBuf = 4 * 1024 * 1024 // 4 MiB — matches claudecode adapter
		scanner.Buffer(make([]byte, maxScanBuf), maxScanBuf)

		lineNum := 0
		for scanner.Scan() {
			if ctx.Err() != nil {
				break
			}
			line := scanner.Text()
			// Prepend \n for every line after the first so that multi-line
			// responses are reconstructed correctly from individual token events.
			text := line
			if lineNum > 0 {
				text = "\n" + line
			}
			lineNum++
			if !yield(agent.Event{Kind: agent.EventKindToken, Text: text}, nil) {
				return
			}
		}

		// Wait for the process to exit; report errors only when context is live.
		if err := cmd.Wait(); err != nil && ctx.Err() == nil {
			msg := fmt.Sprintf("claude-stream: subprocess exited with error: %v", err)
			yield(agent.Event{Kind: agent.EventKindError, Err: msg},
				fmt.Errorf("%s", msg)) //nolint:goerr113
			return
		}

		yield(agent.Event{Kind: agent.EventKindDone, StopReason: "end_turn"}, nil)
	}
}

// buildArgs constructs the CLI argument list.
//
//   - Always: -p <prompt> --output-format text --model <model> --max-turns 1
//   - Optional: --system-prompt <sys> when sess.SystemPrompt is non-empty
func buildArgs(a *claudeStreamAdapter, sess *agent.Session, prompt string) []string {
	args := []string{
		"-p", prompt,
		"--output-format", "text",
		"--model", a.model,
		"--max-turns", "1",
	}
	if sess.SystemPrompt != "" {
		args = append(args, "--system-prompt", sess.SystemPrompt)
	}
	return args
}

// buildPrompt creates a prompt string from the session history.
//
// Single user message: returned verbatim.
// Multi-turn history: formatted as Human/Assistant transcript so Claude has
// full conversational context.
func buildPrompt(sess *agent.Session) string {
	if len(sess.History) == 0 {
		return ""
	}
	if len(sess.History) == 1 {
		return agent.TextOf(sess.History[0].Content)
	}
	var b strings.Builder
	for i, msg := range sess.History {
		text := agent.TextOf(msg.Content)
		if text == "" {
			continue
		}
		if i > 0 {
			b.WriteString("\n\n")
		}
		switch msg.Role {
		case agent.RoleUser:
			b.WriteString("Human: ")
			b.WriteString(text)
		case agent.RoleAssistant:
			b.WriteString("Assistant: ")
			b.WriteString(text)
		case agent.RoleSystem:
			// System messages handled via --system-prompt; skip from transcript.
		}
	}
	return b.String()
}
