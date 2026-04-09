// Package claudestream provides a streaming text adapter for pkg/agent.
// It invokes the `claude` CLI with --output-format text inside a PTY so
// that Node.js sees process.stdout.isTTY = true and streams response text
// character-by-character as it is generated rather than buffering the full
// response before writing.
//
// Import with a blank import to register the "claude-stream" provider:
//
//	import _ "github.com/mrlm-net/cure/internal/agent/claudestream"
//
// # Streaming model
//
// When stdout is a plain pipe, Node.js sets process.stdout.isTTY = false
// and the Claude CLI accumulates the full response before writing.  By
// spawning the CLI inside a PTY (pseudo-terminal), isTTY = true and the
// CLI streams each token immediately, giving true progressive output.
//
// If PTY creation fails (unsupported platform, permission error), the
// adapter falls back to pipe-based communication; streaming is then
// line-granular rather than token-granular.
//
// # Trade-offs vs. claudecode adapter
//
// This adapter does NOT emit tool-call or tool-result events; it only
// emits start / token / done / error events.  This is acceptable for a
// chat UI that does not require agentic tool use.
//
// # Multi-turn sessions
//
// History is collapsed into a single Human/Assistant transcript and passed
// as the -p prompt.  --max-turns is fixed at 1; multi-turn context is
// provided via the transcript.
package claudestream

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"iter"
	"os"
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
// output mode for real streaming via PTY.
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
// output mode.  The subprocess is spawned inside a PTY so that Node.js
// treats stdout as a TTY and streams tokens progressively.
//
// The caller iterates events with:
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

		// Try PTY first — makes Node.js stream tokens progressively.
		// Fall back to pipe if PTY creation fails (unsupported platform, etc.).
		master, slaveName, ptyErr := openPTY()
		if ptyErr != nil {
			a.streamViaPipe(ctx, args, yield)
			return
		}

		slave, err := os.OpenFile(slaveName, os.O_RDWR, 0)
		if err != nil {
			master.Close()
			a.streamViaPipe(ctx, args, yield)
			return
		}

		a.streamViaPTY(ctx, args, master, slave, yield)
	}
}

// streamViaPTY runs the CLI subprocess with a PTY for real token streaming.
// slave is the PTY slave device; master is the read side (parent).
func (a *claudeStreamAdapter) streamViaPTY(
	ctx context.Context,
	args []string,
	master, slave *os.File,
	yield func(agent.Event, error) bool,
) {
	defer master.Close()

	cmd := exec.CommandContext(ctx, a.claudeBin, args...) //nolint:gosec
	cmd.Stdin = slave
	cmd.Stdout = slave
	cmd.Stderr = slave
	// ptyProcAttr sets Setsid + Setctty so the slave PTY becomes the child's
	// controlling terminal, making Node.js see process.stdout.isTTY = true.
	cmd.SysProcAttr = ptyProcAttr()

	if err := cmd.Start(); err != nil {
		slave.Close()
		msg := fmt.Sprintf("claude-stream: start subprocess: %v", err)
		yield(agent.Event{Kind: agent.EventKindError, Err: msg},
			fmt.Errorf("%s", msg)) //nolint:goerr113
		return
	}
	// Parent must close its slave reference after the child is forked;
	// otherwise the master will never see EOF when the child exits.
	slave.Close()

	if !yield(agent.Event{Kind: agent.EventKindStart}, nil) {
		cmd.Cancel()
		return
	}

	a.drainReader(ctx, master, yield)

	// Collect exit status; EIO from the master is expected when slave closes.
	if err := cmd.Wait(); err != nil && ctx.Err() == nil {
		msg := fmt.Sprintf("claude-stream: subprocess exited with error: %v", err)
		yield(agent.Event{Kind: agent.EventKindError, Err: msg},
			fmt.Errorf("%s", msg)) //nolint:goerr113
		return
	}

	yield(agent.Event{Kind: agent.EventKindDone, StopReason: "end_turn"}, nil)
}

// streamViaPipe runs the CLI with a plain stdout pipe.  Node.js will buffer
// output (isTTY = false) so tokens arrive in bulk; this is the fallback
// when PTY creation is not available.
func (a *claudeStreamAdapter) streamViaPipe(
	ctx context.Context,
	args []string,
	yield func(agent.Event, error) bool,
) {
	cmd := exec.CommandContext(ctx, a.claudeBin, args...) //nolint:gosec
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

	if !yield(agent.Event{Kind: agent.EventKindStart}, nil) {
		return
	}

	a.drainReader(ctx, stdout, yield)

	if err := cmd.Wait(); err != nil && ctx.Err() == nil {
		msg := fmt.Sprintf("claude-stream: subprocess exited with error: %v", err)
		yield(agent.Event{Kind: agent.EventKindError, Err: msg},
			fmt.Errorf("%s", msg)) //nolint:goerr113
		return
	}

	yield(agent.Event{Kind: agent.EventKindDone, StopReason: "end_turn"}, nil)
}

// drainReader reads text from r line-by-line and yields each non-empty
// line as an EventKindToken.  ANSI escape sequences are stripped so the
// chat UI receives clean text even when the CLI emits colour codes.
//
// ScanLines handles both \n and \r\n (PTY cooked mode) correctly.
func (a *claudeStreamAdapter) drainReader(
	ctx context.Context,
	r io.Reader,
	yield func(agent.Event, error) bool,
) {
	const maxBuf = 4 * 1024 * 1024 // 4 MiB — matches claudecode adapter
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, maxBuf), maxBuf)

	lineNum := 0
	for scanner.Scan() {
		if ctx.Err() != nil {
			return
		}
		line := stripANSI(scanner.Text())
		// Prepend \n for every line after the first to preserve paragraph
		// structure when tokens are concatenated in the chat UI.
		text := line
		if lineNum > 0 {
			text = "\n" + line
		}
		lineNum++
		if !yield(agent.Event{Kind: agent.EventKindToken, Text: text}, nil) {
			return
		}
	}
	// scanner.Err() is nil for clean EOF and EIO (PTY slave closed = normal).
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
			// Handled via --system-prompt; skip from transcript.
		}
	}
	return b.String()
}

// stripANSI removes ANSI escape sequences (ESC [ ... letter) from text.
// This is needed when the CLI is run inside a PTY and emits colour codes
// or cursor movement sequences.  The fast path skips strings with no ESC.
func stripANSI(s string) string {
	if !strings.ContainsRune(s, '\x1b') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// Skip ESC [ <params> <letter>
			i += 2
			for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == ';') {
				i++
			}
			if i < len(s) {
				i++ // skip terminating letter
			}
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}
