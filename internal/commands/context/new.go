package ctxcmd

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// stringSliceFlag is a flag.Value implementation that accumulates repeated
// --tag flag values into a string slice.
type stringSliceFlag []string

func (f *stringSliceFlag) String() string { return strings.Join(*f, ",") }

func (f *stringSliceFlag) Set(v string) error {
	if v == "" {
		return fmt.Errorf("tag value cannot be empty")
	}
	if len(v) > 128 {
		return fmt.Errorf("tag value too long (max 128 characters)")
	}
	for _, r := range v {
		if r < 0x20 {
			return fmt.Errorf("tag value must not contain control characters")
		}
	}
	*f = append(*f, v)
	return nil
}

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
	tags         []string // set by repeated --tag flags
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
  --tag             Tag for this session (may be repeated)

Examples:
  cure context new --provider claude
  cure context new --provider claude --message "Hello, world"
  cure context new --provider claude --system-prompt "You are a Go expert" --session-name "go-help"
  cure context new --provider claude --tag project:myapp --tag sprint:3
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
	fs.Var((*stringSliceFlag)(&c.tags), "tag", "Tag for this session (may be repeated)")
	return fs
}

func (c *NewCommand) Run(ctx context.Context, tc *terminal.Context) error {
	if c.provider == "" {
		return fmt.Errorf("context new: --provider is required")
	}

	a, err := agent.New(c.provider, nil)
	if err != nil {
		return fmt.Errorf("context new: %w", err)
	}

	sess := agent.NewSession(c.provider, defaultModel())
	if c.systemPrompt != "" {
		sess.SystemPrompt = c.systemPrompt
	}
	if c.sessionName != "" {
		sess.Tags = append(sess.Tags, "name:"+c.sessionName)
	}
	if len(c.tags) > 0 {
		sess.Tags = append(sess.Tags, c.tags...)
	}

	return runTurn(ctx, tc, a, c.store, sess, c.message, c.format)
}
