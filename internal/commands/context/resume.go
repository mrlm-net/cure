package ctxcmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// ResumeCommand implements "cure context resume <session-id>".
// It loads an existing session by ID and either sends a single turn (with
// --message or piped stdin) or enters interactive REPL mode.
type ResumeCommand struct {
	store agent.SessionStore

	// Flags
	message string
	format  string
}

func (c *ResumeCommand) Name() string        { return "resume" }
func (c *ResumeCommand) Description() string { return "Resume an existing AI conversation session" }

func (c *ResumeCommand) Usage() string {
	return `Usage: cure context resume <session-id> [options]

Resume a previously saved AI conversation session. The session history is
preserved across invocations. With --message the command sends a single turn
and exits. Without --message and connected to a terminal the command enters
an interactive REPL. Piped stdin is read as a single message.

Arguments:
  <session-id>    ID of the session to resume (required)

Flags:
  --message       User message to send (optional; triggers single-turn mode)
  --format        Output format: "text" (default) or "ndjson"

Examples:
  cure context resume abc123def456
  cure context resume abc123def456 --message "Continue where we left off"
  echo "What was the last thing we discussed?" | cure context resume abc123def456
`
}

func (c *ResumeCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("context-resume", flag.ContinueOnError)
	fs.StringVar(&c.message, "message", "", "User message to send")
	fs.StringVar(&c.format, "format", "text", `Output format: "text" or "ndjson"`)
	return fs
}

func (c *ResumeCommand) Run(ctx context.Context, tc *terminal.Context) error {
	if len(tc.Args) == 0 {
		return fmt.Errorf("context resume: missing <session-id> argument")
	}
	sessionID := tc.Args[0]

	sess, err := c.store.Load(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("context resume: %w", err)
	}

	a, err := agent.New(sess.Provider, nil)
	if err != nil {
		return fmt.Errorf("context resume: %w", err)
	}

	return runTurn(ctx, tc, a, c.store, sess, c.message, c.format)
}
