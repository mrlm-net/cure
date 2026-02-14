package terminal

import (
	"context"
	"errors"
	"flag"
	"io"
	"testing"
)

// mockCommand is a test helper implementing Command.
type mockCommand struct {
	name   string
	desc   string
	usage  string
	flags  *flag.FlagSet
	err    error
	called bool
}

func (m *mockCommand) Name() string        { return m.name }
func (m *mockCommand) Description() string { return m.desc }
func (m *mockCommand) Usage() string       { return m.usage }
func (m *mockCommand) Flags() *flag.FlagSet { return m.flags }

func (m *mockCommand) Run(_ context.Context, _ *Context) error {
	m.called = true
	return m.err
}

func newExecCtx() *Context {
	return &Context{Stdout: io.Discard, Stderr: io.Discard}
}

func TestSerialRunner(t *testing.T) {
	tests := []struct {
		name      string
		commands  []*mockCommand
		wantErr   bool
		wantCalls []bool
	}{
		{
			name:      "single command success",
			commands:  []*mockCommand{{name: "a"}},
			wantErr:   false,
			wantCalls: []bool{true},
		},
		{
			name:      "multiple commands success",
			commands:  []*mockCommand{{name: "a"}, {name: "b"}, {name: "c"}},
			wantErr:   false,
			wantCalls: []bool{true, true, true},
		},
		{
			name:      "stops on first error",
			commands:  []*mockCommand{{name: "a"}, {name: "b", err: errors.New("fail")}, {name: "c"}},
			wantErr:   true,
			wantCalls: []bool{true, true, false},
		},
		{
			name:      "first command fails",
			commands:  []*mockCommand{{name: "a", err: errors.New("fail")}, {name: "b"}},
			wantErr:   true,
			wantCalls: []bool{true, false},
		},
		{
			name:      "empty command list",
			commands:  []*mockCommand{},
			wantErr:   false,
			wantCalls: []bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &SerialRunner{}
			cmds := make([]Command, len(tt.commands))
			for i, c := range tt.commands {
				cmds[i] = c
			}

			err := runner.Execute(context.Background(), cmds, newExecCtx())

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			for i, want := range tt.wantCalls {
				if tt.commands[i].called != want {
					t.Errorf("command[%d] called = %v, want %v", i, tt.commands[i].called, want)
				}
			}
		})
	}
}

func TestSerialRunner_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cmd := &mockCommand{name: "a"}
	runner := &SerialRunner{}

	err := runner.Execute(ctx, []Command{cmd}, newExecCtx())

	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	if cmd.called {
		t.Error("command should not have been called after cancellation")
	}
}

func TestConcurrentRunner_NotImplemented(t *testing.T) {
	runner := &ConcurrentRunner{}
	err := runner.Execute(context.Background(), nil, nil)
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("error = %v, want ErrNotImplemented", err)
	}
}

func TestPipelineRunner_NotImplemented(t *testing.T) {
	runner := &PipelineRunner{}
	err := runner.Execute(context.Background(), nil, nil)
	if !errors.Is(err, ErrNotImplemented) {
		t.Errorf("error = %v, want ErrNotImplemented", err)
	}
}

func BenchmarkSerialRunner_Execute(b *testing.B) {
	runner := &SerialRunner{}
	cmd := &mockCommand{name: "bench"}
	cmds := []Command{cmd}
	execCtx := newExecCtx()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd.called = false
		_ = runner.Execute(ctx, cmds, execCtx)
	}
}
