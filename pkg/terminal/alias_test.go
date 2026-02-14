package terminal

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
)

// aliasCommand implements AliasProvider for testing.
type aliasCommand struct {
	mockCommand
	aliases []string
}

func (c *aliasCommand) Aliases() []string { return c.aliases }

func TestRegisterWithAliases_RoutesAlias(t *testing.T) {
	router := New(WithStdout(io.Discard), WithStderr(io.Discard))
	cmd := &mockCommand{name: "version", desc: "Show version"}
	router.RegisterWithAliases(cmd, "v", "ver")

	err := router.RunArgs([]string{"v"})
	if err != nil {
		t.Fatalf("RunArgs(v) error = %v", err)
	}
	if !cmd.called {
		t.Error("command was not executed via alias 'v'")
	}

	cmd.called = false
	err = router.RunArgs([]string{"ver"})
	if err != nil {
		t.Fatalf("RunArgs(ver) error = %v", err)
	}
	if !cmd.called {
		t.Error("command was not executed via alias 'ver'")
	}
}

func TestRegisterWithAliases_PanicsOnEmptyAlias(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on empty alias")
		}
	}()

	router := New(WithStdout(io.Discard))
	router.RegisterWithAliases(&mockCommand{name: "version"}, "")
}

func TestRegisterWithAliases_PanicsOnAliasSameAsName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when alias equals command name")
		}
	}()

	router := New(WithStdout(io.Discard))
	router.RegisterWithAliases(&mockCommand{name: "version"}, "version")
}

func TestRegisterWithAliases_PanicsOnDuplicate(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate alias")
		}
	}()

	router := New(WithStdout(io.Discard))
	router.Register(&mockCommand{name: "v"})
	router.RegisterWithAliases(&mockCommand{name: "version"}, "v") // conflicts with existing "v"
}

func TestRegister_AutoDetectsAliasProvider(t *testing.T) {
	router := New(WithStdout(io.Discard), WithStderr(io.Discard))
	cmd := &aliasCommand{
		mockCommand: mockCommand{name: "version", desc: "Show version"},
		aliases:     []string{"v", "ver"},
	}
	router.Register(cmd)

	err := router.RunArgs([]string{"v"})
	if err != nil {
		t.Fatalf("RunArgs(v) error = %v", err)
	}
	if !cmd.called {
		t.Error("command was not executed via auto-detected alias 'v'")
	}
}

func TestCommands_DeduplicatesAliased(t *testing.T) {
	router := New(WithStdout(io.Discard))
	router.RegisterWithAliases(&mockCommand{name: "version", desc: "Show version"}, "v", "ver")
	router.Register(&mockCommand{name: "help", desc: "Show help"})

	cmds := router.Commands()
	if len(cmds) != 2 {
		names := make([]string, len(cmds))
		for i, c := range cmds {
			names[i] = c.Name()
		}
		t.Fatalf("Commands() len = %d (%v), want 2", len(cmds), names)
	}
}

func TestAliasesFor(t *testing.T) {
	router := New(WithStdout(io.Discard))
	router.RegisterWithAliases(&mockCommand{name: "version"}, "v", "ver")
	router.Register(&mockCommand{name: "help"})

	aliases := router.AliasesFor("version")
	if len(aliases) != 2 {
		t.Fatalf("AliasesFor(version) len = %d, want 2", len(aliases))
	}
	if aliases[0] != "v" || aliases[1] != "ver" {
		t.Errorf("AliasesFor(version) = %v, want [v ver]", aliases)
	}

	aliases = router.AliasesFor("help")
	if aliases != nil {
		t.Errorf("AliasesFor(help) = %v, want nil", aliases)
	}
}

func TestHelpCommand_ListShowsAliases(t *testing.T) {
	router := New(WithStdout(io.Discard), WithStderr(io.Discard))
	router.RegisterWithAliases(&mockCommand{name: "version", desc: "Show version"}, "v", "ver")
	router.Register(NewHelpCommand(router))

	var buf bytes.Buffer
	helpCmd := NewHelpCommand(router)
	tc := &Context{Stdout: &buf, Stderr: io.Discard}

	err := helpCmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "(aliases: v, ver)") {
		t.Errorf("output missing aliases, got: %s", output)
	}
}

func TestHelpCommand_DetailShowsAliases(t *testing.T) {
	router := New(WithStdout(io.Discard), WithStderr(io.Discard))
	router.RegisterWithAliases(&mockCommand{name: "version", desc: "Show version", usage: "Usage: cure version"}, "v", "ver")

	var buf bytes.Buffer
	helpCmd := NewHelpCommand(router)
	tc := &Context{Args: []string{"version"}, Stdout: &buf, Stderr: io.Discard}

	err := helpCmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Aliases: v, ver") {
		t.Errorf("output missing aliases line, got: %s", output)
	}
}

func TestLookup_ViaAlias(t *testing.T) {
	router := New(WithStdout(io.Discard))
	router.RegisterWithAliases(&mockCommand{name: "version"}, "v")

	cmd, ok := router.Lookup("v")
	if !ok {
		t.Fatal("Lookup(v) should find the command")
	}
	if cmd.Name() != "version" {
		t.Errorf("Lookup(v) name = %q, want %q", cmd.Name(), "version")
	}
}

// Verify Router implements AliasRegistry.
var _ AliasRegistry = (*Router)(nil)

func BenchmarkRouter_RunArgs_Alias(b *testing.B) {
	router := New(WithStdout(io.Discard), WithStderr(io.Discard))
	cmd := &mockCommand{name: "version"}
	router.RegisterWithAliases(cmd, "v", "ver")
	args := []string{"v"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd.called = false
		_ = router.RunArgs(args)
	}
}
