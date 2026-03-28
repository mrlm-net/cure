package ctxcmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// NewCommand implements "cure context new".
// It creates a fresh session for the given provider and optionally sends an
// initial message (or enters REPL mode if no message is provided).
type NewCommand struct {
	store agent.SessionStore

	// Flags
	provider     string
	message      string
	format       string
	systemPrompt string
	sessionName  string
	model        string
	maxTokens    int
}

func (c *NewCommand) Name() string        { return "new" }
func (c *NewCommand) Description() string { return "Start a new AI conversation session" }

func (c *NewCommand) Usage() string {
	return `Usage: cure context new --provider <provider> [options]

Start a new AI conversation session. With --message the command sends a
single turn and exits. Without --message and connected to a terminal the
command enters an interactive REPL. Piped stdin is read as a single message.

Flags:
  --provider        AI provider name (required, e.g. "claude")
  --message         Initial user message (optional; triggers single-turn mode)
  --format          Output format: "text" (default) or "ndjson"
  --system-prompt   System prompt to set for the session
  --session-name    Human-readable name tag stored with the session
  --model           Model name (provider-specific; uses provider default if empty)
  --max-tokens      Maximum tokens for the response (uses provider default if 0)

Examples:
  cure context new --provider claude
  cure context new --provider claude --message "Hello, world"
  cure context new --provider claude --system-prompt "You are a Go expert" --session-name "go-help"
  cure context new --provider openai --model "gpt-4o-mini" --max-tokens 2048
  echo "Explain goroutines" | cure context new --provider claude
`
}

func (c *NewCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("context-new", flag.ContinueOnError)
	fs.StringVar(&c.provider, "provider", "", "AI provider name (required)")
	fs.StringVar(&c.message, "message", "", "Initial user message")
	fs.StringVar(&c.format, "format", "text", `Output format: "text" or "ndjson"`)
	fs.StringVar(&c.systemPrompt, "system-prompt", "", "System prompt for the session")
	fs.StringVar(&c.sessionName, "session-name", "", "Human-readable name tag for the session")
	fs.StringVar(&c.model, "model", "", "Model name (provider-specific; uses provider default if empty)")
	fs.IntVar(&c.maxTokens, "max-tokens", 0, "Maximum tokens for the response (uses provider default if 0)")
	return fs
}

func (c *NewCommand) Run(ctx context.Context, tc *terminal.Context) error {
	if c.provider == "" {
		return fmt.Errorf("context new: --provider is required")
	}

	cfg := map[string]any{}
	if c.model != "" {
		cfg["model"] = c.model
	}
	if c.maxTokens > 0 {
		cfg["max_tokens"] = c.maxTokens
	}

	a, err := agent.New(c.provider, cfg)
	if err != nil {
		return fmt.Errorf("context new: %w", err)
	}

	model := defaultModel()
	if c.model != "" {
		model = c.model
	}

	sess := agent.NewSession(c.provider, model)
	if c.systemPrompt != "" {
		sess.SystemPrompt = c.systemPrompt
	}
	if c.sessionName != "" {
		sess.Tags = append(sess.Tags, "name:"+c.sessionName)
	}

	return runTurn(ctx, tc, a, c.store, sess, c.message, c.format)
}
