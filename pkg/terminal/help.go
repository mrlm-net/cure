package terminal

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strings"
)

// CommandRegistry provides read access to registered commands.
// [Router] implements this interface.
type CommandRegistry interface {
	// Commands returns all registered commands.
	Commands() []Command

	// Lookup finds a command by exact name match.
	Lookup(name string) (Command, bool)
}

// HelpCommand is the built-in help command that introspects a [CommandRegistry]
// to generate help text dynamically.
//
// With no arguments, it lists all registered commands alphabetically with
// their descriptions. With a command name argument, it shows that command's
// description, usage, and flags.
//
// Create with [NewHelpCommand]:
//
//	router := terminal.New()
//	router.Register(terminal.NewHelpCommand(router))
type HelpCommand struct {
	registry CommandRegistry
}

// NewHelpCommand creates a HelpCommand that introspects the given registry.
// The registry is typically a [Router].
func NewHelpCommand(registry CommandRegistry) *HelpCommand {
	return &HelpCommand{registry: registry}
}

// Name returns "help".
func (c *HelpCommand) Name() string { return "help" }

// Description returns a short description for the help command.
func (c *HelpCommand) Description() string { return "Show help for commands" }

// Usage returns detailed usage information.
func (c *HelpCommand) Usage() string { return "Usage: help [command]" }

// Flags returns nil — the help command accepts no flags.
func (c *HelpCommand) Flags() *flag.FlagSet { return nil }

// Run executes the help command.
// With no args, lists all commands. With one arg, shows help for that command.
func (c *HelpCommand) Run(_ context.Context, tc *Context) error {
	if len(tc.Args) == 0 {
		return c.listCommands(tc)
	}
	return c.showCommand(tc, tc.Args[0])
}

// listCommands writes an alphabetical listing of all registered commands.
func (c *HelpCommand) listCommands(tc *Context) error {
	cmds := c.registry.Commands()
	if len(cmds) == 0 {
		fmt.Fprintln(tc.Stdout, "No commands registered.")
		return nil
	}

	sort.Slice(cmds, func(i, j int) bool {
		return cmds[i].Name() < cmds[j].Name()
	})

	// Calculate padding for aligned output.
	maxLen := 0
	for _, cmd := range cmds {
		if len(cmd.Name()) > maxLen {
			maxLen = len(cmd.Name())
		}
	}

	fmt.Fprintln(tc.Stdout, "Available commands:")
	fmt.Fprintln(tc.Stdout)
	for _, cmd := range cmds {
		line := fmt.Sprintf("  %-*s  %s", maxLen, cmd.Name(), cmd.Description())
		if ar, ok := c.registry.(AliasRegistry); ok {
			if aliases := ar.AliasesFor(cmd.Name()); len(aliases) > 0 {
				line += fmt.Sprintf(" (aliases: %s)", strings.Join(aliases, ", "))
			}
		}
		fmt.Fprintln(tc.Stdout, line)
	}
	fmt.Fprintln(tc.Stdout)
	fmt.Fprintln(tc.Stdout, "Use \"help <command>\" for more information about a command.")
	return nil
}

// showCommand writes detailed help for a single command.
func (c *HelpCommand) showCommand(tc *Context, name string) error {
	cmd, found := c.registry.Lookup(name)
	if !found {
		return &CommandNotFoundError{Name: name}
	}

	// If looking up a sub-router and there are more args, delegate to its help
	if subRouter, ok := cmd.(*Router); ok && len(tc.Args) > 1 {
		subHelp := NewHelpCommand(subRouter)
		subCtx := &Context{
			Args:   tc.Args[1:],
			Stdout: tc.Stdout,
			Stderr: tc.Stderr,
			Logger: tc.Logger,
		}
		return subHelp.Run(context.Background(), subCtx)
	}

	fmt.Fprintf(tc.Stdout, "%s \u2014 %s\n", cmd.Name(), cmd.Description())

	if ar, ok := c.registry.(AliasRegistry); ok {
		if aliases := ar.AliasesFor(cmd.Name()); len(aliases) > 0 {
			fmt.Fprintf(tc.Stdout, "Aliases: %s\n", strings.Join(aliases, ", "))
		}
	}

	if usage := cmd.Usage(); usage != "" {
		fmt.Fprintln(tc.Stdout)
		fmt.Fprintln(tc.Stdout, usage)
	}

	if fs := cmd.Flags(); fs != nil {
		fmt.Fprintln(tc.Stdout)
		fmt.Fprintln(tc.Stdout, "Flags:")
		fs.SetOutput(tc.Stdout)
		fs.PrintDefaults()
	}

	return nil
}
