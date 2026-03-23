package ctxcmd

import (
	"context"
	"errors"
	"flag"
	"fmt"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// ForkCommand implements "cure context fork <session-id>".
// It creates an independent deep copy of an existing session with a new ID.
type ForkCommand struct {
	store agent.SessionStore
}

func (c *ForkCommand) Name() string        { return "fork" }
func (c *ForkCommand) Description() string { return "Fork an existing AI conversation session" }

func (c *ForkCommand) Usage() string {
	return `Usage: cure context fork <session-id>

Create an independent copy of an existing session. The forked session receives
a new ID and records the source session ID in the ForkOf field. The two
sessions share no state after forking.

Arguments:
  <session-id>    ID of the session to fork (required)

Examples:
  cure context fork abc123def456
`
}

func (c *ForkCommand) Flags() *flag.FlagSet {
	return flag.NewFlagSet("context-fork", flag.ContinueOnError)
}

func (c *ForkCommand) Run(ctx context.Context, tc *terminal.Context) error {
	if len(tc.Args) == 0 {
		return fmt.Errorf("context fork: missing <session-id> argument")
	}
	sourceID := tc.Args[0]

	forked, err := c.store.Fork(ctx, sourceID)
	if err != nil {
		if errors.Is(err, agent.ErrSessionNotFound) {
			return fmt.Errorf("context fork: session %q not found", sourceID)
		}
		return fmt.Errorf("context fork: %w", err)
	}

	fmt.Fprintln(tc.Stdout, forked.ID)
	return nil
}
