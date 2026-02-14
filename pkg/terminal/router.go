package terminal

import (
	"context"
	"fmt"
	"io"
	"os"
)

// Router dispatches CLI commands using an internal radix tree for fast lookup.
// Commands are registered via [Router.Register] and executed via [Router.Run]
// or [Router.RunContext].
//
// Configure output streams and execution strategy with functional options:
//
//	router := terminal.New(
//		terminal.WithStdout(os.Stdout),
//		terminal.WithStderr(os.Stderr),
//		terminal.WithRunner(&terminal.SerialRunner{}),
//	)
//	router.Register(&VersionCommand{})
//	if err := router.Run(os.Args[1:]); err != nil {
//		fmt.Fprintf(os.Stderr, "error: %v\n", err)
//		os.Exit(1)
//	}
type Router struct {
	root   *node
	stdout io.Writer
	stderr io.Writer
	runner Runner
}

// Option is a functional option for configuring a [Router].
type Option func(*Router)

// WithStdout sets the standard output stream for command execution.
// Commands receive this writer via [Context].Stdout.
//
// Default: os.Stdout
func WithStdout(w io.Writer) Option {
	return func(r *Router) {
		r.stdout = w
	}
}

// WithStderr sets the standard error stream for command execution.
// Commands receive this writer via [Context].Stderr.
//
// Default: os.Stderr
func WithStderr(w io.Writer) Option {
	return func(r *Router) {
		r.stderr = w
	}
}

// WithRunner sets the execution strategy for matched commands.
//
// Default: &[SerialRunner]{}
func WithRunner(runner Runner) Option {
	return func(r *Router) {
		r.runner = runner
	}
}

// New creates a new Router with the provided options.
// Defaults: stdout=os.Stdout, stderr=os.Stderr, runner=&SerialRunner{}.
func New(opts ...Option) *Router {
	r := &Router{
		root:   &node{children: make(map[byte]*node)},
		stdout: os.Stdout,
		stderr: os.Stderr,
		runner: &SerialRunner{},
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Register adds a command to the router's radix tree.
// Panics if cmd.Name() is empty or if a command with the same name is
// already registered.
func (r *Router) Register(cmd Command) {
	name := cmd.Name()
	if name == "" {
		panic("terminal: command name cannot be empty")
	}
	r.root.insert(name, cmd)
}

// Run executes the command identified by the first element of args.
// Equivalent to RunContext with context.Background().
func (r *Router) Run(args []string) error {
	return r.RunContext(context.Background(), args)
}

// RunContext executes the command identified by the first element of args,
// using ctx for cancellation and deadlines.
//
// The method parses the command name from args[0], looks it up in the radix
// tree, parses any flags declared by the command, and delegates execution
// to the configured [Runner].
//
// Returns an error if no args are provided, the command is not found,
// flag parsing fails, or the command itself returns an error.
func (r *Router) RunContext(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	cmdName := args[0]
	cmdArgs := args[1:]

	cmd, found := r.root.search(cmdName)
	if !found {
		return fmt.Errorf("unknown command: %s", cmdName)
	}

	execCtx := &Context{
		Stdout: r.stdout,
		Stderr: r.stderr,
	}

	if fs := cmd.Flags(); fs != nil {
		if err := fs.Parse(cmdArgs); err != nil {
			return fmt.Errorf("flag parsing failed: %w", err)
		}
		execCtx.Flags = fs
		execCtx.Args = fs.Args()
	} else {
		execCtx.Args = cmdArgs
	}

	return r.runner.Execute(ctx, []Command{cmd}, execCtx)
}

// Lookup finds a registered command by exact name match.
// Returns the command and true if found, nil and false otherwise.
func (r *Router) Lookup(name string) (Command, bool) {
	return r.root.search(name)
}

// Commands returns all registered commands in no guaranteed order.
// Use this to build help text or command listings.
func (r *Router) Commands() []Command {
	return r.root.collectCommands()
}
