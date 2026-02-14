package terminal

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"
)

// Router dispatches CLI commands using an internal radix tree for fast lookup.
// Router implements the [Command] interface, enabling nested routers for
// subcommand support. When used as a subcommand, args[0] is consumed as the
// subcommand group name and remaining args are dispatched to child commands.
//
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
	root    *node
	stdout  io.Writer
	stderr  io.Writer
	runner  Runner
	logger  *slog.Logger
	aliases map[string][]string // primary name -> []alias names

	// Subcommand identity (only set when Router is used as a Command)
	name string
	desc string

	// Signal handling and timeouts
	handleSignal bool
	timeout      time.Duration
	gracePeriod  time.Duration
}

// Verify Router satisfies Command interface at compile time.
var _ Command = (*Router)(nil)

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

// WithName sets the command name for a Router used as a subcommand group.
// Only relevant when the Router is registered as a Command in a parent Router.
func WithName(name string) Option {
	return func(r *Router) {
		r.name = name
	}
}

// WithDescription sets the description for a Router used as a subcommand group.
func WithDescription(desc string) Option {
	return func(r *Router) {
		r.desc = desc
	}
}

// New creates a new Router with the provided options.
// Defaults: stdout=os.Stdout, stderr=os.Stderr, runner=&SerialRunner{}.
func New(opts ...Option) *Router {
	r := &Router{
		root:        &node{children: make(map[byte]*node)},
		stdout:      os.Stdout,
		stderr:      os.Stderr,
		runner:      &SerialRunner{},
		aliases:     make(map[string][]string),
		gracePeriod: 5 * time.Second,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// Name returns the subcommand group name, or empty string for the root router.
func (r *Router) Name() string { return r.name }

// Description returns the subcommand group description.
func (r *Router) Description() string { return r.desc }

// Usage returns auto-generated usage showing available subcommands.
func (r *Router) Usage() string {
	if r.name == "" {
		return ""
	}
	cmds := r.Commands()
	if len(cmds) == 0 {
		return fmt.Sprintf("Usage: %s <command>", r.name)
	}
	names := make([]string, len(cmds))
	for i, cmd := range cmds {
		names[i] = cmd.Name()
	}
	sort.Strings(names)
	return fmt.Sprintf("Usage: %s <command>\n\nAvailable commands: %s",
		r.name, strings.Join(names, ", "))
}

// Flags returns nil -- subcommand groups do not accept flags directly.
func (r *Router) Flags() *flag.FlagSet { return nil }

// Run dispatches to child commands when the Router is used as a Command
// in a parent Router. It creates a child router context that inherits the
// parent's output streams without mutating the Router's fields, making it
// safe for concurrent use.
func (r *Router) Run(ctx context.Context, tc *Context) error {
	if tc == nil || len(tc.Args) == 0 {
		return &NoCommandError{}
	}
	return r.runContextWith(ctx, tc.Args, tc.Stdout, tc.Stderr)
}

// Register adds a command to the router's radix tree.
// Panics if cmd.Name() is empty or if a command with the same name is
// already registered.
//
// If the command implements [AliasProvider], its aliases are automatically
// registered.
func (r *Router) Register(cmd Command) {
	name := cmd.Name()
	if name == "" {
		panic("terminal: command name cannot be empty")
	}
	r.root.insert(name, cmd)
	// Auto-register aliases if the command declares them.
	if ap, ok := cmd.(AliasProvider); ok {
		for _, alias := range ap.Aliases() {
			if alias == "" || alias == name {
				continue
			}
			r.root.insert(alias, cmd)
			r.addAlias(name, alias)
		}
	}
}

// RegisterWithAliases adds a command and its aliases to the router.
// Each alias routes to the same command. Panics if any alias conflicts
// with an existing command or alias name.
//
// Example:
//
//	router.RegisterWithAliases(&VersionCommand{}, "v", "ver")
func (r *Router) RegisterWithAliases(cmd Command, aliases ...string) {
	r.Register(cmd)
	for _, alias := range aliases {
		if alias == "" {
			panic("terminal: alias cannot be empty")
		}
		if alias == cmd.Name() {
			panic(fmt.Sprintf("terminal: alias %q is the same as command name", alias))
		}
		r.root.insert(alias, cmd)
		r.addAlias(cmd.Name(), alias)
	}
}

// addAlias records an alias mapping for help display.
func (r *Router) addAlias(primary, alias string) {
	r.aliases[primary] = append(r.aliases[primary], alias)
}

// AliasesFor returns the registered aliases for a command name.
// Returns nil if the command has no aliases.
func (r *Router) AliasesFor(name string) []string {
	return r.aliases[name]
}

// RunArgs executes the command identified by the first element of args.
// Equivalent to RunContext with context.Background().
// Deprecated: Use [Router.Run] for subcommand dispatch or call RunContext directly.
func (r *Router) RunArgs(args []string) error {
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
	return r.runContextWith(ctx, args, r.stdout, r.stderr)
}

// runContextWith is the shared dispatch implementation that takes explicit
// output streams. This avoids mutating Router fields when sub-routers
// inherit streams from a parent context.
func (r *Router) runContextWith(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return &NoCommandError{}
	}

	cmdName := args[0]
	cmdArgs := args[1:]

	if r.logger != nil {
		r.logger.DebugContext(ctx, "dispatching command",
			slog.String("command", cmdName),
			slog.Int("argc", len(cmdArgs)),
		)
	}

	cmd, found := r.root.search(cmdName)
	if !found {
		if r.logger != nil {
			r.logger.InfoContext(ctx, "command not found",
				slog.String("command", cmdName),
			)
		}
		return &CommandNotFoundError{Name: cmdName, Suggestions: r.suggestNames(cmdName)}
	}

	execCtx := &Context{
		Stdout: stdout,
		Stderr: stderr,
		Logger: r.logger,
	}

	if fs := cmd.Flags(); fs != nil {
		if err := fs.Parse(cmdArgs); err != nil {
			return &FlagParseError{Command: cmdName, Err: err}
		}
		execCtx.Flags = fs
		execCtx.Args = fs.Args()
	} else {
		execCtx.Args = cmdArgs
	}

	// Set up signal handling if configured
	if r.handleSignal {
		var cancel context.CancelFunc
		ctx, cancel = r.setupSignalHandler(ctx)
		defer cancel()
	}

	// Apply timeout if configured
	if r.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.timeout)
		defer cancel()
	}

	start := time.Now()
	err := r.runner.Execute(ctx, []Command{cmd}, execCtx)
	if r.logger != nil {
		duration := time.Since(start)
		if err != nil {
			r.logger.InfoContext(ctx, "command failed",
				slog.String("command", cmdName),
				slog.Duration("duration", duration),
				slog.String("error", err.Error()),
			)
		} else {
			r.logger.InfoContext(ctx, "command completed",
				slog.String("command", cmdName),
				slog.Duration("duration", duration),
			)
		}
	}

	if err != nil {
		var cmdErr *CommandError
		if errors.As(err, &cmdErr) {
			return err
		}
		return &CommandError{Command: cmdName, Err: err}
	}
	return nil
}

// suggestNames returns similar command names for "did you mean?" suggestions.
func (r *Router) suggestNames(name string) []string {
	similar := r.root.findSimilar(name, 3)
	if len(similar) == 0 {
		return nil
	}
	names := make([]string, len(similar))
	for i, cmd := range similar {
		names[i] = cmd.Name()
	}
	return names
}

// Lookup finds a registered command by exact name match.
// Returns the command and true if found, nil and false otherwise.
func (r *Router) Lookup(name string) (Command, bool) {
	return r.root.search(name)
}

// Commands returns all registered commands, deduplicated by primary name.
// Use this to build help text or command listings.
func (r *Router) Commands() []Command {
	all := r.root.collectCommands()
	seen := make(map[string]bool, len(all))
	unique := make([]Command, 0, len(all))
	for _, cmd := range all {
		if !seen[cmd.Name()] {
			seen[cmd.Name()] = true
			unique = append(unique, cmd)
		}
	}
	return unique
}
