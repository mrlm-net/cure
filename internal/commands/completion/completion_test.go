package completion

import (
	"bytes"
	"context"
	"flag"
	"io"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/terminal"
)

// mockCommand is a test command implementation.
type mockCommand struct {
	name  string
	desc  string
	usage string
	flags *flag.FlagSet
}

func (c *mockCommand) Name() string                                     { return c.name }
func (c *mockCommand) Description() string                              { return c.desc }
func (c *mockCommand) Usage() string                                    { return c.usage }
func (c *mockCommand) Flags() *flag.FlagSet                             { return c.flags }
func (c *mockCommand) Run(_ context.Context, _ *terminal.Context) error { return nil }

// mockRegistry implements CommandRegistry for testing.
type mockRegistry struct {
	commands []terminal.Command
}

func (r *mockRegistry) Commands() []terminal.Command {
	return r.commands
}

func (r *mockRegistry) Lookup(name string) (terminal.Command, bool) {
	for _, cmd := range r.commands {
		if cmd.Name() == name {
			return cmd, true
		}
	}
	return nil, false
}

func TestNewCompletionCommand(t *testing.T) {
	registry := &mockRegistry{}
	cmd := NewCompletionCommand(registry)

	if cmd == nil {
		t.Fatal("NewCompletionCommand returned nil")
	}

	// Should be a Router
	router, ok := cmd.(*terminal.Router)
	if !ok {
		t.Fatalf("NewCompletionCommand returned %T, want *terminal.Router", cmd)
	}

	if router.Name() != "completion" {
		t.Errorf("Name() = %q, want %q", router.Name(), "completion")
	}

	if router.Description() == "" {
		t.Error("Description() is empty")
	}

	// Should have bash and zsh subcommands
	cmds := router.Commands()
	if len(cmds) != 2 {
		t.Fatalf("got %d subcommands, want 2", len(cmds))
	}

	names := make(map[string]bool)
	for _, subCmd := range cmds {
		names[subCmd.Name()] = true
	}

	if !names["bash"] {
		t.Error("missing bash subcommand")
	}
	if !names["zsh"] {
		t.Error("missing zsh subcommand")
	}
}

func TestBashCommand_Metadata(t *testing.T) {
	cmd := &BashCommand{registry: &mockRegistry{}}

	if cmd.Name() != "bash" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "bash")
	}
	if cmd.Description() == "" {
		t.Error("Description() is empty")
	}
	if cmd.Usage() == "" {
		t.Error("Usage() is empty")
	}
	if cmd.Flags() != nil {
		t.Error("Flags() should be nil")
	}
}

func TestBashCommand_GenerateScript(t *testing.T) {
	// Create mock registry with commands
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.String("format", "json", "output format")

	registry := &mockRegistry{
		commands: []terminal.Command{
			&mockCommand{name: "version", desc: "Print version", flags: fs},
			&mockCommand{name: "help", desc: "Show help"},
		},
	}

	cmd := &BashCommand{registry: registry}
	var buf bytes.Buffer
	tc := &terminal.Context{Stdout: &buf, Stderr: io.Discard}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()

	// Check for bash completion essentials
	if !strings.Contains(output, "_cure_completions()") {
		t.Error("output missing completion function")
	}
	if !strings.Contains(output, "complete -F _cure_completions cure") {
		t.Error("output missing complete directive")
	}
	if !strings.Contains(output, "_init_completion") {
		t.Error("output missing _init_completion")
	}

	// Check for command names
	if !strings.Contains(output, "version") {
		t.Error("output missing 'version' command")
	}
	if !strings.Contains(output, "help") {
		t.Error("output missing 'help' command")
	}

	// Check for flag completion
	if !strings.Contains(output, "--format") {
		t.Error("output missing '--format' flag")
	}

	// Check for flag value completion (from FlagValues map)
	if !strings.Contains(output, "json") || !strings.Contains(output, "html") {
		t.Error("output missing format flag values")
	}
}

func TestBashCommand_WithSubcommands(t *testing.T) {
	// Create a nested router (like trace command)
	traceRouter := terminal.New(
		terminal.WithName("trace"),
		terminal.WithDescription("Trace network connections"),
	)
	traceRouter.Register(&mockCommand{name: "http", desc: "Trace HTTP"})
	traceRouter.Register(&mockCommand{name: "tcp", desc: "Trace TCP"})

	registry := &mockRegistry{
		commands: []terminal.Command{
			traceRouter,
			&mockCommand{name: "version", desc: "Print version"},
		},
	}

	cmd := &BashCommand{registry: registry}
	var buf bytes.Buffer
	tc := &terminal.Context{Stdout: &buf, Stderr: io.Discard}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()

	// Check for parent command
	if !strings.Contains(output, "trace") {
		t.Error("output missing 'trace' command")
	}

	// Check for subcommand completion
	if !strings.Contains(output, "# Subcommand completion") {
		t.Error("output missing subcommand completion section")
	}
	if !strings.Contains(output, "http") {
		t.Error("output missing 'http' subcommand")
	}
	if !strings.Contains(output, "tcp") {
		t.Error("output missing 'tcp' subcommand")
	}

	// Should have case statement for trace
	if !strings.Contains(output, "case ${words[1]} in") {
		t.Error("output missing case statement for subcommands")
	}
}

func TestBashCommand_EmptyRegistry(t *testing.T) {
	cmd := &BashCommand{registry: &mockRegistry{}}
	var buf bytes.Buffer
	tc := &terminal.Context{Stdout: &buf, Stderr: io.Discard}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()

	// Should still generate valid script structure
	if !strings.Contains(output, "_cure_completions()") {
		t.Error("output missing completion function")
	}
	if !strings.Contains(output, "complete -F _cure_completions cure") {
		t.Error("output missing complete directive")
	}
}

func TestZshCommand_Metadata(t *testing.T) {
	cmd := &ZshCommand{registry: &mockRegistry{}}

	if cmd.Name() != "zsh" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "zsh")
	}
	if cmd.Description() == "" {
		t.Error("Description() is empty")
	}
	if cmd.Usage() == "" {
		t.Error("Usage() is empty")
	}
	if cmd.Flags() != nil {
		t.Error("Flags() should be nil")
	}
}

func TestZshCommand_GenerateScript(t *testing.T) {
	// Create mock registry with commands
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	fs.String("format", "json", "output format")

	registry := &mockRegistry{
		commands: []terminal.Command{
			&mockCommand{name: "version", desc: "Print version", flags: fs},
			&mockCommand{name: "help", desc: "Show help"},
		},
	}

	cmd := &ZshCommand{registry: registry}
	var buf bytes.Buffer
	tc := &terminal.Context{Stdout: &buf, Stderr: io.Discard}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()

	// Check for zsh completion essentials
	if !strings.Contains(output, "#compdef cure") {
		t.Error("output missing #compdef directive")
	}
	if !strings.Contains(output, "_cure()") {
		t.Error("output missing completion function")
	}
	if !strings.Contains(output, "_cure\n") {
		t.Error("output missing function call")
	}

	// Check for command names with descriptions
	if !strings.Contains(output, "'version:Print version'") {
		t.Error("output missing 'version' command with description")
	}
	if !strings.Contains(output, "'help:Show help'") {
		t.Error("output missing 'help' command with description")
	}

	// Check for _arguments usage
	if !strings.Contains(output, "_arguments") {
		t.Error("output missing _arguments")
	}
	if !strings.Contains(output, "_describe") {
		t.Error("output missing _describe")
	}

	// Check for flag completion
	if !strings.Contains(output, "--format") {
		t.Error("output missing '--format' flag")
	}
}

func TestZshCommand_WithSubcommands(t *testing.T) {
	// Create a nested router
	traceRouter := terminal.New(
		terminal.WithName("trace"),
		terminal.WithDescription("Trace network connections"),
	)
	traceRouter.Register(&mockCommand{name: "http", desc: "Trace HTTP"})
	traceRouter.Register(&mockCommand{name: "tcp", desc: "Trace TCP"})

	registry := &mockRegistry{
		commands: []terminal.Command{
			traceRouter,
			&mockCommand{name: "version", desc: "Print version"},
		},
	}

	cmd := &ZshCommand{registry: registry}
	var buf bytes.Buffer
	tc := &terminal.Context{Stdout: &buf, Stderr: io.Discard}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()

	// Check for parent command
	if !strings.Contains(output, "'trace:Trace network connections'") {
		t.Error("output missing 'trace' command")
	}

	// Check for subcommands
	if !strings.Contains(output, "'http:Trace HTTP'") {
		t.Error("output missing 'http' subcommand")
	}
	if !strings.Contains(output, "'tcp:Trace TCP'") {
		t.Error("output missing 'tcp' subcommand")
	}

	// Should have case for trace with subcommands
	if !strings.Contains(output, "trace)") {
		t.Error("output missing case for trace command")
	}
}

func TestZshCommand_EscapeDescription(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Simple description", "Simple description"},
		{"Has: colon", "Has\\: colon"},
		{"Has [brackets]", "Has \\[brackets\\]"},
		{"Has 'quotes'", "Has \\'quotes\\'"},
		{"Mix: of [special] 'chars'", "Mix\\: of \\[special\\] \\'chars\\'"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeZshDesc(tt.input)
			if got != tt.want {
				t.Errorf("escapeZshDesc(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestZshCommand_EmptyRegistry(t *testing.T) {
	cmd := &ZshCommand{registry: &mockRegistry{}}
	var buf bytes.Buffer
	tc := &terminal.Context{Stdout: &buf, Stderr: io.Discard}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()

	// Should still generate valid script structure
	if !strings.Contains(output, "#compdef cure") {
		t.Error("output missing #compdef directive")
	}
	if !strings.Contains(output, "_cure()") {
		t.Error("output missing completion function")
	}
}

func TestBashCommand_CollectCommands(t *testing.T) {
	registry := &mockRegistry{
		commands: []terminal.Command{
			&mockCommand{name: "version", desc: "Version"},
			&mockCommand{name: "help", desc: "Help"},
			&mockCommand{name: "generate", desc: "Generate"},
		},
	}

	cmd := &BashCommand{registry: registry}
	names := cmd.collectCommands()

	if len(names) != 3 {
		t.Fatalf("got %d commands, want 3", len(names))
	}

	// Should be sorted
	expected := []string{"generate", "help", "version"}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("names[%d] = %q, want %q", i, name, expected[i])
		}
	}
}

func TestBashCommand_CollectSubcommands(t *testing.T) {
	// Create nested router
	traceRouter := terminal.New(
		terminal.WithName("trace"),
		terminal.WithDescription("Trace network"),
	)
	traceRouter.Register(&mockCommand{name: "http", desc: "HTTP"})
	traceRouter.Register(&mockCommand{name: "tcp", desc: "TCP"})

	configRouter := terminal.New(
		terminal.WithName("config"),
		terminal.WithDescription("Manage config"),
	)
	configRouter.Register(&mockCommand{name: "get", desc: "Get"})
	configRouter.Register(&mockCommand{name: "set", desc: "Set"})

	registry := &mockRegistry{
		commands: []terminal.Command{
			traceRouter,
			configRouter,
			&mockCommand{name: "version", desc: "Version"},
		},
	}

	cmd := &BashCommand{registry: registry}
	subcommands := cmd.collectSubcommands()

	if len(subcommands) != 2 {
		t.Fatalf("got %d parent commands with subcommands, want 2", len(subcommands))
	}

	// Check trace subcommands
	traceSubs, ok := subcommands["trace"]
	if !ok {
		t.Fatal("missing trace subcommands")
	}
	if len(traceSubs) != 2 {
		t.Fatalf("got %d trace subcommands, want 2", len(traceSubs))
	}
	if traceSubs[0] != "http" || traceSubs[1] != "tcp" {
		t.Errorf("trace subcommands = %v, want [http tcp]", traceSubs)
	}

	// Check config subcommands
	configSubs, ok := subcommands["config"]
	if !ok {
		t.Fatal("missing config subcommands")
	}
	if len(configSubs) != 2 {
		t.Fatalf("got %d config subcommands, want 2", len(configSubs))
	}
	if configSubs[0] != "get" || configSubs[1] != "set" {
		t.Errorf("config subcommands = %v, want [get set]", configSubs)
	}
}

func TestBashCommand_CollectFlags(t *testing.T) {
	fs1 := flag.NewFlagSet("cmd1", flag.ContinueOnError)
	fs1.String("format", "json", "format")
	fs1.Bool("verbose", false, "verbose")

	fs2 := flag.NewFlagSet("cmd2", flag.ContinueOnError)
	fs2.String("output", "-", "output")
	fs2.String("format", "json", "format") // duplicate, should appear once

	registry := &mockRegistry{
		commands: []terminal.Command{
			&mockCommand{name: "cmd1", desc: "Command 1", flags: fs1},
			&mockCommand{name: "cmd2", desc: "Command 2", flags: fs2},
			&mockCommand{name: "cmd3", desc: "Command 3"}, // no flags
		},
	}

	cmd := &BashCommand{registry: registry}
	flags := cmd.collectFlags()

	// Should have 3 unique flags, sorted
	expected := []string{"--format", "--output", "--verbose"}
	if len(flags) != len(expected) {
		t.Fatalf("got %d flags, want %d", len(flags), len(expected))
	}

	for i, flag := range flags {
		if flag != expected[i] {
			t.Errorf("flags[%d] = %q, want %q", i, flag, expected[i])
		}
	}
}

func TestBashCommand_CollectFlags_Recursive(t *testing.T) {
	// Create nested router with flags
	subFs := flag.NewFlagSet("sub", flag.ContinueOnError)
	subFs.String("suboption", "val", "sub option")

	subRouter := terminal.New(
		terminal.WithName("parent"),
		terminal.WithDescription("Parent command"),
	)
	subRouter.Register(&mockCommand{name: "sub", desc: "Sub", flags: subFs})

	topFs := flag.NewFlagSet("top", flag.ContinueOnError)
	topFs.String("topoption", "val", "top option")

	registry := &mockRegistry{
		commands: []terminal.Command{
			&mockCommand{name: "top", desc: "Top", flags: topFs},
			subRouter,
		},
	}

	cmd := &BashCommand{registry: registry}
	flags := cmd.collectFlags()

	// Should collect flags from both top-level and nested commands
	if len(flags) != 2 {
		t.Fatalf("got %d flags, want 2", len(flags))
	}

	// Check both flags are present
	hasSubOption := false
	hasTopOption := false
	for _, flag := range flags {
		if flag == "--suboption" {
			hasSubOption = true
		}
		if flag == "--topoption" {
			hasTopOption = true
		}
	}

	if !hasSubOption {
		t.Error("missing --suboption flag from nested command")
	}
	if !hasTopOption {
		t.Error("missing --topoption flag from top-level command")
	}
}
