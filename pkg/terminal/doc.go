// Package terminal provides a reusable CLI framework for building multi-command
// applications with declarative command registration, automatic help generation,
// and pluggable execution modes.
//
// The package defines two core types:
//
// [Command] is an interface representing a single executable command. Implementations
// define the command's identity (name, description), accepted flags, and execution
// logic. Commands are designed to be defined in isolation and registered with a
// Router for dispatch.
//
// [Context] is a struct providing the execution environment for a command — parsed
// arguments, flags, and output streams. Commands receive a Context and must use its
// Stdout and Stderr writers instead of os.Stdout/os.Stderr, enabling testability
// and output redirection.
//
// # Example
//
//	type GreetCommand struct{}
//
//	func (g *GreetCommand) Name() string        { return "greet" }
//	func (g *GreetCommand) Description() string { return "Print a greeting" }
//	func (g *GreetCommand) Usage() string       { return "Usage: app greet <name>" }
//	func (g *GreetCommand) Flags() *flag.FlagSet { return nil }
//
//	func (g *GreetCommand) Run(ctx context.Context, c *terminal.Context) error {
//		if len(c.Args) == 0 {
//			return fmt.Errorf("missing name argument")
//		}
//		fmt.Fprintf(c.Stdout, "Hello, %s!\n", c.Args[0])
//		return nil
//	}
package terminal
