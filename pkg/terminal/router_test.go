package terminal

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"testing"
)

func TestRouter_RegisterAndRun(t *testing.T) {
	var buf bytes.Buffer
	router := New(WithStdout(&buf), WithStderr(io.Discard))

	cmd := &mockCommand{name: "greet"}
	router.Register(cmd)

	err := router.Run([]string{"greet"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !cmd.called {
		t.Error("command was not executed")
	}
}

func TestRouter_RunUnknownCommand(t *testing.T) {
	router := New(WithStdout(io.Discard), WithStderr(io.Discard))
	router.Register(&mockCommand{name: "version"})

	err := router.Run([]string{"unknown"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	want := "unknown command: unknown"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestRouter_RunNoArgs(t *testing.T) {
	router := New(WithStdout(io.Discard))

	err := router.Run(nil)
	if err == nil {
		t.Fatal("expected error for empty args")
	}
	want := "no command specified"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

func TestRouter_RegisterDuplicatePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate registration")
		}
	}()

	router := New(WithStdout(io.Discard))
	router.Register(&mockCommand{name: "version"})
	router.Register(&mockCommand{name: "version"})
}

func TestRouter_RegisterEmptyNamePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on empty command name")
		}
	}()

	router := New(WithStdout(io.Discard))
	router.Register(&mockCommand{name: ""})
}

func TestRouter_RunWithFlags(t *testing.T) {
	var gotType string

	cmd := &flagCommand{
		mockCommand: mockCommand{name: "generate"},
		typeFlag:    &gotType,
	}

	router := New(WithStdout(io.Discard), WithStderr(io.Discard))
	router.Register(cmd)

	err := router.Run([]string{"generate", "--type", "yaml", "output.yaml"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if gotType != "yaml" {
		t.Errorf("type flag = %q, want %q", gotType, "yaml")
	}
	if !cmd.called {
		t.Error("command was not executed")
	}
}

func TestRouter_RunPositionalArgs(t *testing.T) {
	var capturedArgs []string

	cmd := &argsCapture{
		mockCommand: mockCommand{name: "echo"},
		captured:    &capturedArgs,
	}

	router := New(WithStdout(io.Discard), WithStderr(io.Discard))
	router.Register(cmd)

	err := router.Run([]string{"echo", "hello", "world"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(capturedArgs) != 2 || capturedArgs[0] != "hello" || capturedArgs[1] != "world" {
		t.Errorf("args = %v, want [hello world]", capturedArgs)
	}
}

func TestRouter_RunContext_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	router := New(WithStdout(io.Discard), WithStderr(io.Discard))
	router.Register(&mockCommand{name: "slow"})

	err := router.RunContext(ctx, []string{"slow"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestRouter_Commands(t *testing.T) {
	router := New(WithStdout(io.Discard))
	router.Register(&mockCommand{name: "version", desc: "show version"})
	router.Register(&mockCommand{name: "help", desc: "show help"})
	router.Register(&mockCommand{name: "generate", desc: "generate files"})

	cmds := router.Commands()
	if len(cmds) != 3 {
		t.Fatalf("Commands() len = %d, want 3", len(cmds))
	}

	names := make(map[string]bool)
	for _, cmd := range cmds {
		names[cmd.Name()] = true
	}
	for _, want := range []string{"version", "help", "generate"} {
		if !names[want] {
			t.Errorf("Commands() missing %q", want)
		}
	}
}

func TestRouter_WithOptions(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	runner := &SerialRunner{}

	router := New(WithStdout(stdout), WithStderr(stderr), WithRunner(runner))

	if router.stdout != stdout {
		t.Error("WithStdout not applied")
	}
	if router.stderr != stderr {
		t.Error("WithStderr not applied")
	}
	if router.runner != runner {
		t.Error("WithRunner not applied")
	}
}

func TestRouter_DefaultOptions(t *testing.T) {
	router := New()

	if router.stdout == nil {
		t.Error("default stdout is nil")
	}
	if router.stderr == nil {
		t.Error("default stderr is nil")
	}
	if router.runner == nil {
		t.Error("default runner is nil")
	}
}

// -- test helpers --

// flagCommand is a Command that declares a --type flag.
type flagCommand struct {
	mockCommand
	typeFlag *string
}

func (c *flagCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet(c.name, flag.ContinueOnError)
	fs.StringVar(c.typeFlag, "type", "json", "output type")
	return fs
}

func (c *flagCommand) Run(ctx context.Context, tc *Context) error {
	c.called = true
	return c.err
}

// argsCapture is a Command that captures positional args.
type argsCapture struct {
	mockCommand
	captured *[]string
}

func (c *argsCapture) Run(_ context.Context, tc *Context) error {
	c.called = true
	*c.captured = tc.Args
	return nil
}

func BenchmarkRouter_Run(b *testing.B) {
	router := New(WithStdout(io.Discard), WithStderr(io.Discard))
	router.Register(&mockCommand{name: "test"})
	args := []string{"test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = router.Run(args)
	}
}

func BenchmarkRouter_Run_ManyCommands(b *testing.B) {
	router := New(WithStdout(io.Discard), WithStderr(io.Discard))
	for i := 0; i < 100; i++ {
		router.Register(&mockCommand{name: fmt.Sprintf("command-%d", i)})
	}
	args := []string{"command-50"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = router.Run(args)
	}
}
