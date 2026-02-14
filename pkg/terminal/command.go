package terminal

import (
	"context"
	"flag"
)

// Command represents a single executable command with metadata and execution logic.
// Commands are registered with a Router and invoked based on CLI input.
//
// Implementations define the command's identity (Name, Description), usage information
// (Usage, Flags), and execution logic (Run). The Router dispatches to the appropriate
// command based on Name and builds a [Context] with parsed flags and output streams.
//
// Example implementation:
//
//	type VersionCommand struct{}
//
//	func (c *VersionCommand) Name() string        { return "version" }
//	func (c *VersionCommand) Description() string { return "Print version information" }
//	func (c *VersionCommand) Usage() string       { return "Usage: cure version" }
//	func (c *VersionCommand) Flags() *flag.FlagSet { return nil }
//
//	func (c *VersionCommand) Run(ctx context.Context, tc *Context) error {
//		fmt.Fprintln(tc.Stdout, "cure version dev")
//		return nil
//	}
type Command interface {
	// Name returns the command name as invoked from the CLI.
	// Should be a short, lowercase command name (e.g., "version", "help", "gen-config").
	Name() string

	// Description returns a short one-line description shown in help output.
	Description() string

	// Usage returns detailed usage information including flags, arguments,
	// and examples. Shown when the user runs "help <command>".
	// Return empty string to use auto-generated usage from flags.
	Usage() string

	// Flags returns a flag.FlagSet defining all flags accepted by this command.
	// The Router parses CLI arguments against this FlagSet before calling Run.
	// Return nil if the command accepts no flags.
	Flags() *flag.FlagSet

	// Run executes the command logic.
	// ctx carries cancellation signals and deadlines.
	// c contains parsed arguments, flags, and output streams.
	// Return nil on success, or an error describing the failure.
	Run(ctx context.Context, c *Context) error
}
