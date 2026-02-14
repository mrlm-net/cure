package terminal

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"testing"
)

// mockRegistry implements CommandRegistry for testing.
type mockRegistry struct {
	commands []Command
}

func (r *mockRegistry) Commands() []Command {
	return r.commands
}

func (r *mockRegistry) Lookup(name string) (Command, bool) {
	for _, cmd := range r.commands {
		if cmd.Name() == name {
			return cmd, true
		}
	}
	return nil, false
}

func TestHelpCommand_Interface(t *testing.T) {
	var _ Command = (*HelpCommand)(nil)
}

func TestHelpCommand_Metadata(t *testing.T) {
	cmd := NewHelpCommand(&mockRegistry{})

	if cmd.Name() != "help" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "help")
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

func TestHelpCommand_ListCommands(t *testing.T) {
	registry := &mockRegistry{
		commands: []Command{
			&mockCommand{name: "version", desc: "Print version"},
			&mockCommand{name: "generate", desc: "Generate files"},
			&mockCommand{name: "help", desc: "Show help"},
		},
	}

	cmd := NewHelpCommand(registry)
	var buf bytes.Buffer
	tc := &Context{Stdout: &buf, Stderr: io.Discard}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()

	// Should contain header
	if !strings.Contains(output, "Available commands:") {
		t.Error("output missing 'Available commands:' header")
	}

	// Should list all three commands
	for _, name := range []string{"generate", "help", "version"} {
		if !strings.Contains(output, name) {
			t.Errorf("output missing command %q", name)
		}
	}

	// Should be sorted alphabetically — generate before help before version
	genIdx := strings.Index(output, "generate")
	helpIdx := strings.Index(output, "help")
	verIdx := strings.Index(output, "version")
	if genIdx > helpIdx || helpIdx > verIdx {
		t.Error("commands not sorted alphabetically")
	}

	// Should contain footer
	if !strings.Contains(output, "Use \"help <command>\"") {
		t.Error("output missing usage footer")
	}
}

func TestHelpCommand_ListCommands_Aligned(t *testing.T) {
	registry := &mockRegistry{
		commands: []Command{
			&mockCommand{name: "a", desc: "Short name"},
			&mockCommand{name: "long-name", desc: "Long name"},
		},
	}

	cmd := NewHelpCommand(registry)
	var buf bytes.Buffer
	tc := &Context{Stdout: &buf, Stderr: io.Discard}

	_ = cmd.Run(context.Background(), tc)

	output := buf.String()
	lines := strings.Split(output, "\n")

	// Find the command lines and verify alignment
	var cmdLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, "  ") && strings.Contains(line, "name") {
			cmdLines = append(cmdLines, line)
		}
	}
	if len(cmdLines) != 2 {
		t.Fatalf("expected 2 command lines, got %d", len(cmdLines))
	}

	// Both descriptions should start at the same column
	desc0 := strings.Index(cmdLines[0], "Short name")
	desc1 := strings.Index(cmdLines[1], "Long name")
	if desc0 != desc1 {
		t.Errorf("descriptions not aligned: positions %d and %d", desc0, desc1)
	}
}

func TestHelpCommand_ListCommands_Empty(t *testing.T) {
	registry := &mockRegistry{}
	cmd := NewHelpCommand(registry)
	var buf bytes.Buffer
	tc := &Context{Stdout: &buf, Stderr: io.Discard}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !strings.Contains(buf.String(), "No commands registered.") {
		t.Error("expected empty message")
	}
}

func TestHelpCommand_ShowCommand(t *testing.T) {
	registry := &mockRegistry{
		commands: []Command{
			&mockCommand{name: "version", desc: "Print version", usage: "Usage: cure version"},
		},
	}

	cmd := NewHelpCommand(registry)
	var buf bytes.Buffer
	tc := &Context{Args: []string{"version"}, Stdout: &buf, Stderr: io.Discard}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "version — Print version") {
		t.Errorf("output missing command header, got: %s", output)
	}
	if !strings.Contains(output, "Usage: cure version") {
		t.Errorf("output missing usage, got: %s", output)
	}
}

func TestHelpCommand_ShowCommand_WithFlags(t *testing.T) {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.String("type", "json", "output format")
	fs.Bool("force", false, "overwrite existing files")

	registry := &mockRegistry{
		commands: []Command{
			&mockCommand{
				name:  "generate",
				desc:  "Generate files",
				usage: "Usage: cure generate [flags] <output>",
				flags: fs,
			},
		},
	}

	cmd := NewHelpCommand(registry)
	var buf bytes.Buffer
	tc := &Context{Args: []string{"generate"}, Stdout: &buf, Stderr: io.Discard}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Flags:") {
		t.Error("output missing 'Flags:' section")
	}
	if !strings.Contains(output, "-type") {
		t.Error("output missing -type flag")
	}
	if !strings.Contains(output, "-force") {
		t.Error("output missing -force flag")
	}
}

func TestHelpCommand_ShowCommand_NoUsage(t *testing.T) {
	registry := &mockRegistry{
		commands: []Command{
			&mockCommand{name: "version", desc: "Print version"},
		},
	}

	cmd := NewHelpCommand(registry)
	var buf bytes.Buffer
	tc := &Context{Args: []string{"version"}, Stdout: &buf, Stderr: io.Discard}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "version — Print version") {
		t.Errorf("output missing header, got: %s", output)
	}
	// With empty usage, output should just be the header line
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line for command with no usage, got %d: %q", len(lines), output)
	}
}

func TestHelpCommand_UnknownCommand(t *testing.T) {
	registry := &mockRegistry{
		commands: []Command{
			&mockCommand{name: "version", desc: "Print version"},
		},
	}

	cmd := NewHelpCommand(registry)
	tc := &Context{Args: []string{"unknown"}, Stdout: io.Discard, Stderr: io.Discard}

	err := cmd.Run(context.Background(), tc)
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	want := "unknown command: unknown"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestHelpCommand_Integration(t *testing.T) {
	router := New(WithStdout(io.Discard), WithStderr(io.Discard))
	router.Register(&mockCommand{name: "version", desc: "Print version"})
	router.Register(NewHelpCommand(router))

	// Test via Router.Run — "help" lists commands
	var buf bytes.Buffer
	router.stdout = &buf
	err := router.Run([]string{"help"})
	if err != nil {
		t.Fatalf("Run(help) error = %v", err)
	}
	if !strings.Contains(buf.String(), "version") {
		t.Error("help output missing 'version' command")
	}

	// Test via Router.Run — "help version" shows detail
	buf.Reset()
	err = router.Run([]string{"help", "version"})
	if err != nil {
		t.Fatalf("Run(help version) error = %v", err)
	}
	if !strings.Contains(buf.String(), "version — Print version") {
		t.Error("help version output missing header")
	}

	// Test unknown — "help nonexistent" returns error
	err = router.Run([]string{"help", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestRouter_Lookup(t *testing.T) {
	router := New(WithStdout(io.Discard))
	router.Register(&mockCommand{name: "version", desc: "Print version"})
	router.Register(&mockCommand{name: "help", desc: "Show help"})

	tests := []struct {
		name   string
		want   bool
	}{
		{"version", true},
		{"help", true},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, ok := router.Lookup(tt.name)
			if ok != tt.want {
				t.Errorf("Lookup(%q) ok = %v, want %v", tt.name, ok, tt.want)
			}
			if ok && cmd.Name() != tt.name {
				t.Errorf("Lookup(%q) name = %q", tt.name, cmd.Name())
			}
		})
	}
}

// Verify Router implements CommandRegistry.
var _ CommandRegistry = (*Router)(nil)

func BenchmarkHelpCommand_List(b *testing.B) {
	cmds := make([]Command, 20)
	for i := range cmds {
		cmds[i] = &mockCommand{name: fmt.Sprintf("command-%02d", i), desc: "A command"}
	}
	registry := &mockRegistry{commands: cmds}
	cmd := NewHelpCommand(registry)
	tc := &Context{Stdout: io.Discard, Stderr: io.Discard}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cmd.Run(ctx, tc)
	}
}

func BenchmarkHelpCommand_Show(b *testing.B) {
	registry := &mockRegistry{
		commands: []Command{
			&mockCommand{name: "version", desc: "Print version", usage: "Usage: cure version"},
		},
	}
	cmd := NewHelpCommand(registry)
	tc := &Context{Args: []string{"version"}, Stdout: io.Discard, Stderr: io.Discard}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cmd.Run(ctx, tc)
	}
}
