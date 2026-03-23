package ctxcmd

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// DeleteCommand implements "cure context delete <session-id>".
// With --yes it deletes immediately; without --yes on a TTY it prompts first.
type DeleteCommand struct {
	store agent.SessionStore

	// Flags
	yes bool
}

func (c *DeleteCommand) Name() string        { return "delete" }
func (c *DeleteCommand) Description() string { return "Delete an AI conversation session" }

func (c *DeleteCommand) Usage() string {
	return `Usage: cure context delete <session-id> [--yes]

Delete a persisted AI conversation session. Without --yes the command prompts
for confirmation when running interactively. Use --yes to skip confirmation in
scripts or CI pipelines.

Arguments:
  <session-id>    ID of the session to delete (required)

Flags:
  --yes           Skip confirmation prompt

Examples:
  cure context delete abc123def456
  cure context delete abc123def456 --yes
`
}

func (c *DeleteCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("context-delete", flag.ContinueOnError)
	fs.BoolVar(&c.yes, "yes", false, "Skip confirmation prompt")
	return fs
}

func (c *DeleteCommand) Run(ctx context.Context, tc *terminal.Context) error {
	if len(tc.Args) == 0 {
		return fmt.Errorf("context delete: missing <session-id> argument")
	}
	sessionID := tc.Args[0]

	// Verify the session exists before prompting.
	if _, err := c.store.Load(ctx, sessionID); err != nil {
		if errors.Is(err, agent.ErrSessionNotFound) {
			return fmt.Errorf("context delete: session %q not found", sessionID)
		}
		return fmt.Errorf("context delete: %w", err)
	}

	if !c.yes {
		confirmed, err := confirmDelete(tc, sessionID)
		if err != nil {
			return fmt.Errorf("context delete: %w", err)
		}
		if !confirmed {
			fmt.Fprintln(tc.Stdout, "Aborted.")
			return nil
		}
	}

	if err := c.store.Delete(ctx, sessionID); err != nil {
		if errors.Is(err, agent.ErrSessionNotFound) {
			return fmt.Errorf("context delete: session %q not found", sessionID)
		}
		return fmt.Errorf("context delete: %w", err)
	}

	fmt.Fprintf(tc.Stdout, "Session %q deleted.\n", sessionID)
	return nil
}

// confirmDelete writes a prompt to tc.Stdout and reads a line from tc.Stdin
// (or os.Stdin if tc.Stdin is nil). Returns true only if the user types "y" or "yes".
func confirmDelete(tc *terminal.Context, sessionID string) (bool, error) {
	fmt.Fprintf(tc.Stdout, "Delete session %q? [y/N] ", sessionID)
	r := stdinReader(tc)
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, err
		}
		return false, nil
	}
	answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
	return answer == "y" || answer == "yes", nil
}
