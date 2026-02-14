package commands

import (
	"context"
	"flag"
	"fmt"

	"github.com/mrlm-net/cure/pkg/terminal"
)

// VersionCommand prints the cure version.
type VersionCommand struct{}

// Name returns "version".
func (c *VersionCommand) Name() string { return "version" }

// Description returns a short description for the version command.
func (c *VersionCommand) Description() string { return "Print version information" }

// Usage returns detailed usage information.
func (c *VersionCommand) Usage() string { return "Usage: cure version" }

// Flags returns nil — the version command accepts no flags.
func (c *VersionCommand) Flags() *flag.FlagSet { return nil }

// Run executes the version command, printing version information to stdout.
func (c *VersionCommand) Run(_ context.Context, tc *terminal.Context) error {
	fmt.Fprintln(tc.Stdout, "cure version dev")
	return nil
}
