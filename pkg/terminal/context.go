package terminal

import (
	"flag"
	"io"
)

// Context provides the execution environment for a command, including parsed
// arguments, flags, and output streams.
//
// A new Context is built by the Router for each command invocation. Commands
// must write all output to Stdout and Stderr rather than using os.Stdout or
// os.Stderr directly, enabling testability and output redirection.
type Context struct {
	// Args contains positional arguments remaining after flag parsing.
	// For "cure generate --type yaml config.yaml", Args would be ["config.yaml"].
	Args []string

	// Flags is the parsed flag.FlagSet for this command.
	// Access flag values via Flags.Lookup("name").Value.String() or
	// by using typed variables bound during flag definition.
	// May be nil if the command declared no flags.
	Flags *flag.FlagSet

	// Stdout is the standard output stream for the command.
	// Commands must write normal output here, not to os.Stdout.
	Stdout io.Writer

	// Stderr is the standard error stream for the command.
	// Commands must write error and diagnostic messages here, not to os.Stderr.
	Stderr io.Writer
}
