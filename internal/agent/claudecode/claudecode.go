// Package claudecode provides a Claude Code CLI adapter for pkg/agent.
// It invokes the `claude` CLI as a subprocess and streams NDJSON events.
//
// Import this package with a blank import to register the "claude-code" provider:
//
//	import _ "github.com/mrlm-net/cure/internal/agent/claudecode"
//
// # CLI requirements
//
// The adapter requires the `claude` CLI (Claude Code) to be installed and
// available on PATH (or at the path configured via "claude_bin"). The
// claude CLI must be version 1.x or later.
//
// # Conversation model
//
// Unlike the API-backed providers (claude, openai, gemini) which send the full
// session history on every request, this adapter builds a formatted conversation
// from the session history and passes it as the prompt to `claude -p`. For
// multi-turn sessions, all prior history is included as context so the model
// has full conversational awareness.
//
// Future work (Phase 2): When session.Tools is non-empty the adapter will start
// an in-process MCP stdio server exposing those tools and pass it to
// `claude --mcpServers` so that Claude Code can invoke them.
package claudecode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"os/exec"
	"strings"

	"github.com/mrlm-net/cure/pkg/agent"
)

const (
	defaultModel    = "claude-opus-4-6"
	defaultMaxTurns = 32
	defaultBin      = "claude"
)

func init() {
	agent.Register("claude-code", NewClaudeCodeAgent)
}

// claudeCodeAdapter implements agent.Agent by spawning the claude CLI.
type claudeCodeAdapter struct {
	claudeBin       string
	model           string
	maxTurns        int
	allowedTools    []string
	disallowedTools []string
}

// NewClaudeCodeAgent is the AgentFactory for the "claude-code" provider.
//
// Recognised cfg keys:
//   - "claude_bin"       string   — path to claude binary (default: "claude")
//   - "model"            string   — model name (default: "claude-opus-4-6")
//   - "max_turns"        int/int64/float64 — max agentic turns (default: 32)
//   - "allowed_tools"    []string — whitelist of tools Claude Code may use
//   - "disallowed_tools" []string — blacklist of tools Claude Code may not use
func NewClaudeCodeAgent(cfg map[string]any) (agent.Agent, error) {
	bin := defaultBin
	if v, ok := cfg["claude_bin"].(string); ok && v != "" {
		bin = v
	}

	model := defaultModel
	if v, ok := cfg["model"].(string); ok && v != "" {
		model = v
	}

	maxTurns := defaultMaxTurns
	switch v := cfg["max_turns"].(type) {
	case int:
		maxTurns = v
	case int64:
		maxTurns = int(v)
	case float64:
		maxTurns = int(v)
	}

	var allowedTools, disallowedTools []string
	if raw, ok := cfg["allowed_tools"]; ok {
		allowedTools = toStringSlice(raw)
	}
	if raw, ok := cfg["disallowed_tools"]; ok {
		disallowedTools = toStringSlice(raw)
	}

	return &claudeCodeAdapter{
		claudeBin:       bin,
		model:           model,
		maxTurns:        maxTurns,
		allowedTools:    allowedTools,
		disallowedTools: disallowedTools,
	}, nil
}

// Provider returns the provider name "claude-code".
func (a *claudeCodeAdapter) Provider() string { return "claude-code" }

// CountTokens is not supported by the Claude Code CLI adapter.
func (a *claudeCodeAdapter) CountTokens(_ context.Context, _ *agent.Session) (int, error) {
	return 0, agent.ErrCountNotSupported
}

// Run streams a response for the given session by invoking the claude CLI.
// The caller iterates events with: for ev, err := range a.Run(ctx, session) { ... }
// Cancelling ctx terminates the subprocess cleanly via cmd.Cancel.
//
// When the context is cancelled the goroutine exits and the channel is closed;
// the caller's range loop will exit on the next iteration.
func (a *claudeCodeAdapter) Run(ctx context.Context, sess *agent.Session) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		ch := make(chan result)
		go func() {
			defer close(ch)
			a.streamInto(ctx, sess, ch)
		}()
		for r := range ch {
			if !yield(r.ev, r.err) {
				return
			}
		}
	}
}

// streamInto builds the CLI command, starts the subprocess, and translates
// NDJSON output lines into agent.Event values sent on ch.
func (a *claudeCodeAdapter) streamInto(ctx context.Context, sess *agent.Session, ch chan<- result) {
	prompt := buildPrompt(sess)
	if prompt == "" {
		send(ctx, ch, result{
			ev:  agent.Event{Kind: agent.EventKindError, Err: "claude-code: session has no user message"},
			err: fmt.Errorf("claude-code: session has no user message"),
		})
		return
	}

	args := a.buildArgs(sess, prompt)
	cmd := exec.CommandContext(ctx, a.claudeBin, args...) //nolint:gosec // validated path, not user input
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		msg := fmt.Sprintf("claude-code: failed to create stdout pipe: %v", err)
		send(ctx, ch, result{
			ev:  agent.Event{Kind: agent.EventKindError, Err: msg},
			err: fmt.Errorf("%s", msg), //nolint:goerr113
		})
		return
	}

	if err := cmd.Start(); err != nil {
		msg := fmt.Sprintf("claude-code: failed to start subprocess: %v", err)
		send(ctx, ch, result{
			ev:  agent.Event{Kind: agent.EventKindError, Err: msg},
			err: fmt.Errorf("%s", msg), //nolint:goerr113
		})
		return
	}

	scanner := bufio.NewScanner(stdout)
	// Increase scanner buffer for large tool result payloads.
	const maxScanBuf = 4 * 1024 * 1024 // 4 MiB
	scanner.Buffer(make([]byte, maxScanBuf), maxScanBuf)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if ctx.Err() != nil {
			break
		}
		if !a.parseLine(ctx, line, ch) {
			break
		}
	}

	// Wait for the process to exit. Errors here are usually context cancellation
	// or non-zero exit codes from the CLI; we report them only when the context
	// is still live (a cancelled context always causes a non-zero exit).
	if err := cmd.Wait(); err != nil && ctx.Err() == nil {
		msg := fmt.Sprintf("claude-code: subprocess exited with error: %v", err)
		send(ctx, ch, result{
			ev:  agent.Event{Kind: agent.EventKindError, Err: msg},
			err: fmt.Errorf("%s", msg), //nolint:goerr113
		})
	}
}

// buildArgs constructs the CLI argument list for the claude subprocess.
//
//   - Always adds: -p <prompt> --output-format stream-json --verbose --model <model> --max-turns <n>
//   - Adds --system-prompt <sys> when sess.SystemPrompt is non-empty
//   - Adds --allowedTools and --disabledTools flags when configured
func (a *claudeCodeAdapter) buildArgs(sess *agent.Session, prompt string) []string {
	args := []string{
		"-p", prompt,
		"--output-format", "stream-json",
		"--verbose",
		"--model", a.model,
		"--max-turns", fmt.Sprintf("%d", a.maxTurns),
	}
	if sess.SystemPrompt != "" {
		args = append(args, "--system-prompt", sess.SystemPrompt)
	}
	if len(a.allowedTools) > 0 {
		args = append(args, "--allowedTools", strings.Join(a.allowedTools, ","))
	}
	if len(a.disallowedTools) > 0 {
		args = append(args, "--disabledTools", strings.Join(a.disallowedTools, ","))
	}
	return args
}

// parseLine decodes a single NDJSON line and sends the corresponding event on ch.
// Returns false if the caller should stop (context cancelled or channel closed).
func (a *claudeCodeAdapter) parseLine(ctx context.Context, line string, ch chan<- result) bool {
	var env ndjsonEnvelope
	if err := json.Unmarshal([]byte(line), &env); err != nil {
		// Ignore non-JSON lines (e.g. debug output when --verbose is active).
		return true
	}

	switch env.Type {
	case "system":
		if env.Subtype == "init" {
			// Session start — emit EventKindStart with input token info when available.
			return send(ctx, ch, result{ev: agent.Event{
				Kind: agent.EventKindStart,
			}})
		}

	case "assistant":
		if env.Message == nil {
			return true
		}
		for _, block := range env.Message.Content {
			switch block.Type {
			case "text":
				if block.Text != "" {
					if !send(ctx, ch, result{ev: agent.Event{
						Kind: agent.EventKindToken,
						Text: block.Text,
					}}) {
						return false
					}
				}
			case "tool_use":
				inputJSON, _ := json.Marshal(block.Input)
				if !send(ctx, ch, result{ev: agent.Event{
					Kind: agent.EventKindToolCall,
					ToolCall: &agent.ToolCallEvent{
						ID:        block.ID,
						ToolName:  block.Name,
						InputJSON: string(inputJSON),
					},
				}}) {
					return false
				}
			}
		}

	case "user":
		// Tool results fed back by Claude Code after executing tools.
		if env.Message == nil {
			return true
		}
		for _, block := range env.Message.Content {
			if block.Type == "tool_result" {
				if !send(ctx, ch, result{ev: agent.Event{
					Kind: agent.EventKindToolResult,
					ToolResult: &agent.ToolResultEvent{
						ID:       block.ToolUseID,
						ToolName: block.ToolName,
						Result:   block.Content,
						IsError:  block.IsError,
					},
				}}) {
					return false
				}
			}
		}

	case "result":
		if env.Subtype == "success" {
			var inputTokens, outputTokens int
			if env.Usage != nil {
				inputTokens = env.Usage.InputTokens
				outputTokens = env.Usage.OutputTokens
			}
			return send(ctx, ch, result{ev: agent.Event{
				Kind:         agent.EventKindDone,
				InputTokens:  inputTokens,
				OutputTokens: outputTokens,
				StopReason:   "end_turn",
			}})
		}
		if env.Subtype == "error" {
			msg := "claude-code: execution error"
			if env.Error != "" {
				msg = fmt.Sprintf("claude-code: %s", env.Error)
			}
			send(ctx, ch, result{ //nolint:errcheck // best-effort on error path
				ev:  agent.Event{Kind: agent.EventKindError, Err: msg},
				err: fmt.Errorf("%s", msg), //nolint:goerr113
			})
			return false
		}
	}

	return true
}

// buildPrompt extracts a prompt string from the session history.
//
// For a single-message session it returns the plain text of that message.
// For multi-turn sessions it formats all history as a Human/Assistant
// transcript so Claude Code has full conversational context.
func buildPrompt(sess *agent.Session) string {
	if len(sess.History) == 0 {
		return ""
	}

	// Single user message — pass it verbatim.
	if len(sess.History) == 1 {
		return agent.TextOf(sess.History[0].Content)
	}

	// Multi-turn history — format as a readable transcript.
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
			// System messages are handled via --system-prompt; skip from transcript.
		}
	}
	return b.String()
}

// result is the internal channel payload type.
type result struct {
	ev  agent.Event
	err error
}

// send delivers r to ch, aborting if ctx is already cancelled.
// Returns false if the send could not be delivered.
func send(ctx context.Context, ch chan<- result, r result) bool {
	select {
	case ch <- r:
		return true
	case <-ctx.Done():
		return false
	}
}

// toStringSlice coerces an interface{} value to []string.
// Accepts []string and []interface{} (the latter from JSON unmarshalling).
func toStringSlice(v any) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		out := make([]string, 0, len(s))
		for _, item := range s {
			if str, ok := item.(string); ok {
				out = append(out, str)
			}
		}
		return out
	}
	return nil
}
