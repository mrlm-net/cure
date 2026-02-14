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

	err := router.RunArgs([]string{"greet"})
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

	err := router.RunArgs([]string{"unknown"})
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

	err := router.RunArgs(nil)
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

	err := router.RunArgs([]string{"generate", "--type", "yaml", "output.yaml"})
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

	err := router.RunArgs([]string{"echo", "hello", "world"})
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
	// The context.Canceled error is now wrapped in a CommandError
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	var cmdErr *CommandError
	if !errors.As(err, &cmdErr) {
		t.Errorf("error should be *CommandError, got %T", err)
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

// -- Subcommand tests (Phase 4) --

func TestRouter_Subcommand_Dispatch(t *testing.T) {
	var buf bytes.Buffer

	configSet := &argsCapture{
		mockCommand: mockCommand{name: "set", desc: "Set config value"},
		captured:    new([]string),
	}

	config := New(
		WithName("config"),
		WithDescription("Manage configuration"),
		WithStdout(io.Discard),
		WithStderr(io.Discard),
	)
	config.Register(configSet)

	root := New(WithStdout(&buf), WithStderr(io.Discard))
	root.Register(config)

	err := root.RunArgs([]string{"config", "set", "key", "value"})
	if err != nil {
		t.Fatalf("RunArgs() error = %v", err)
	}
	if !configSet.called {
		t.Error("subcommand was not executed")
	}
	if len(*configSet.captured) != 2 || (*configSet.captured)[0] != "key" || (*configSet.captured)[1] != "value" {
		t.Errorf("subcommand args = %v, want [key value]", *configSet.captured)
	}
}

func TestRouter_Subcommand_InheritsStreams(t *testing.T) {
	var buf bytes.Buffer

	outputCmd := &writerMockCommand{
		mockCommand: mockCommand{name: "show", desc: "Show output"},
		output:      "config output",
	}

	config := New(
		WithName("config"),
		WithDescription("Manage configuration"),
		WithStdout(io.Discard),
		WithStderr(io.Discard),
	)
	config.Register(outputCmd)

	root := New(WithStdout(&buf), WithStderr(io.Discard))
	root.Register(config)

	err := root.RunArgs([]string{"config", "show"})
	if err != nil {
		t.Fatalf("RunArgs() error = %v", err)
	}
	if buf.String() != "config output" {
		t.Errorf("output = %q, want %q", buf.String(), "config output")
	}
}

func TestRouter_Subcommand_EmptyArgs(t *testing.T) {
	config := New(
		WithName("config"),
		WithDescription("Manage configuration"),
		WithStdout(io.Discard),
		WithStderr(io.Discard),
	)
	config.Register(&mockCommand{name: "set"})

	root := New(WithStdout(io.Discard), WithStderr(io.Discard))
	root.Register(config)

	err := root.RunArgs([]string{"config"})
	if err == nil {
		t.Fatal("expected error for empty subcommand args")
	}
	var noCmd *NoCommandError
	if !errors.As(err, &noCmd) {
		t.Errorf("error should contain NoCommandError, got %T: %v", err, err)
	}
}

func TestRouter_Subcommand_Unknown(t *testing.T) {
	config := New(
		WithName("config"),
		WithDescription("Manage configuration"),
		WithStdout(io.Discard),
		WithStderr(io.Discard),
	)
	config.Register(&mockCommand{name: "set"})

	root := New(WithStdout(io.Discard), WithStderr(io.Discard))
	root.Register(config)

	err := root.RunArgs([]string{"config", "nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	var notFound *CommandNotFoundError
	if !errors.As(err, &notFound) {
		t.Errorf("error should contain CommandNotFoundError, got %T: %v", err, err)
	}
}

func TestRouter_Subcommand_DeeplyNested(t *testing.T) {
	var buf bytes.Buffer

	leaf := &argsCapture{
		mockCommand: mockCommand{name: "value", desc: "Show value"},
		captured:    new([]string),
	}

	level2 := New(
		WithName("get"),
		WithDescription("Get operations"),
		WithStdout(io.Discard),
		WithStderr(io.Discard),
	)
	level2.Register(leaf)

	level1 := New(
		WithName("config"),
		WithDescription("Configuration"),
		WithStdout(io.Discard),
		WithStderr(io.Discard),
	)
	level1.Register(level2)

	root := New(WithStdout(&buf), WithStderr(io.Discard))
	root.Register(level1)

	err := root.RunArgs([]string{"config", "get", "value", "mykey"})
	if err != nil {
		t.Fatalf("RunArgs() error = %v", err)
	}
	if !leaf.called {
		t.Error("deeply nested command was not executed")
	}
	if len(*leaf.captured) != 1 || (*leaf.captured)[0] != "mykey" {
		t.Errorf("leaf args = %v, want [mykey]", *leaf.captured)
	}
}

func TestRouter_Subcommand_CommandsIncludesSubRouter(t *testing.T) {
	config := New(
		WithName("config"),
		WithDescription("Manage configuration"),
	)
	root := New(WithStdout(io.Discard))
	root.Register(config)
	root.Register(&mockCommand{name: "version"})

	cmds := root.Commands()
	if len(cmds) != 2 {
		t.Fatalf("Commands() len = %d, want 2", len(cmds))
	}

	found := false
	for _, cmd := range cmds {
		if cmd.Name() == "config" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Commands() should include the sub-router")
	}
}

func TestRouter_ImplementsCommand(t *testing.T) {
	var _ Command = (*Router)(nil)
}

func TestRouter_WithNameAndDescription(t *testing.T) {
	r := New(WithName("config"), WithDescription("Manage configuration"))
	if r.Name() != "config" {
		t.Errorf("Name() = %q, want %q", r.Name(), "config")
	}
	if r.Description() != "Manage configuration" {
		t.Errorf("Description() = %q, want %q", r.Description(), "Manage configuration")
	}
}

func TestRouter_Usage(t *testing.T) {
	r := New(WithName("config"))
	r.Register(&mockCommand{name: "get"})
	r.Register(&mockCommand{name: "set"})

	usage := r.Usage()
	if usage == "" {
		t.Fatal("Usage() should not be empty")
	}
	if !contains(usage, "get") || !contains(usage, "set") {
		t.Errorf("Usage() = %q, should contain 'get' and 'set'", usage)
	}
}

func TestRouter_Usage_RootRouter(t *testing.T) {
	r := New() // no WithName
	if r.Usage() != "" {
		t.Errorf("Usage() for root router = %q, want empty", r.Usage())
	}
}

func TestRouter_Flags_ReturnsNil(t *testing.T) {
	r := New(WithName("config"))
	if r.Flags() != nil {
		t.Error("Flags() should return nil for Router")
	}
}

// writerMockCommand writes a fixed string to stdout.
type writerMockCommand struct {
	mockCommand
	output string
}

func (c *writerMockCommand) Run(_ context.Context, tc *Context) error {
	c.called = true
	_, _ = fmt.Fprint(tc.Stdout, c.output)
	return c.err
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func BenchmarkRouter_Subcommand_Dispatch(b *testing.B) {
	configSet := &mockCommand{name: "set"}
	config := New(
		WithName("config"),
		WithDescription("Config"),
		WithStdout(io.Discard),
		WithStderr(io.Discard),
	)
	config.Register(configSet)

	root := New(WithStdout(io.Discard), WithStderr(io.Discard))
	root.Register(config)
	args := []string{"config", "set", "key"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		configSet.called = false
		_ = root.RunArgs(args)
	}
}

func BenchmarkRouter_Run(b *testing.B) {
	router := New(WithStdout(io.Discard), WithStderr(io.Discard))
	router.Register(&mockCommand{name: "test"})
	args := []string{"test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = router.RunArgs(args)
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
		_ = router.RunArgs(args)
	}
}
