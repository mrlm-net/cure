package terminal

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrentRunner_AllExecute(t *testing.T) {
	cmds := []*mockCommand{
		{name: "a"},
		{name: "b"},
		{name: "c"},
	}
	commands := make([]Command, len(cmds))
	for i, c := range cmds {
		commands[i] = c
	}

	runner := &ConcurrentRunner{MaxWorkers: 2}
	err := runner.Execute(context.Background(), commands, newExecCtx())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for i, c := range cmds {
		if !c.called {
			t.Errorf("command[%d] %q was not executed", i, c.name)
		}
	}
}

func TestConcurrentRunner_AggregatesErrors(t *testing.T) {
	errA := errors.New("error a")
	errB := errors.New("error b")

	cmds := []Command{
		&mockCommand{name: "a", err: errA},
		&mockCommand{name: "b"},
		&mockCommand{name: "c", err: errB},
	}

	runner := &ConcurrentRunner{MaxWorkers: 4}
	err := runner.Execute(context.Background(), cmds, newExecCtx())
	if err == nil {
		t.Fatal("expected error from concurrent runner")
	}

	if !errors.Is(err, errA) {
		t.Error("error does not contain errA")
	}
	if !errors.Is(err, errB) {
		t.Error("error does not contain errB")
	}
}

func TestConcurrentRunner_EmptyCommands(t *testing.T) {
	runner := &ConcurrentRunner{}
	err := runner.Execute(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestConcurrentRunner_SingleCommand(t *testing.T) {
	cmd := &mockCommand{name: "solo"}
	runner := &ConcurrentRunner{}

	err := runner.Execute(context.Background(), []Command{cmd}, newExecCtx())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !cmd.called {
		t.Error("command was not executed")
	}
}

func TestConcurrentRunner_MaxWorkersLimitsConcurrency(t *testing.T) {
	var peak int64
	var current int64

	// Command that tracks concurrent execution
	type counterCmd struct {
		mockCommand
		peak    *int64
		current *int64
	}

	makeCmd := func(name string) *counterCmd {
		return &counterCmd{
			mockCommand: mockCommand{name: name},
			peak:        &peak,
			current:     &current,
		}
	}

	cmds := make([]Command, 10)
	for i := range cmds {
		cc := makeCmd("cmd")
		// Override Run to track concurrency
		cmds[i] = &concurrencyTracker{
			mockCommand: mockCommand{name: cc.name},
			current:     &current,
			peak:        &peak,
		}
	}

	runner := &ConcurrentRunner{MaxWorkers: 2}
	err := runner.Execute(context.Background(), cmds, newExecCtx())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if atomic.LoadInt64(&peak) > 2 {
		t.Errorf("peak concurrency = %d, want <= 2", atomic.LoadInt64(&peak))
	}
}

// concurrencyTracker tracks peak concurrent goroutines.
type concurrencyTracker struct {
	mockCommand
	current *int64
	peak    *int64
}

func (c *concurrencyTracker) Run(_ context.Context, _ *Context) error {
	c.called = true
	cur := atomic.AddInt64(c.current, 1)
	for {
		old := atomic.LoadInt64(c.peak)
		if cur <= old || atomic.CompareAndSwapInt64(c.peak, old, cur) {
			break
		}
	}
	time.Sleep(10 * time.Millisecond) // Hold the slot briefly
	atomic.AddInt64(c.current, -1)
	return nil
}

func TestConcurrentRunner_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cmd := &mockCommand{name: "a"}
	runner := &ConcurrentRunner{MaxWorkers: 2}

	err := runner.Execute(ctx, []Command{cmd}, newExecCtx())
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
}

func TestWithMaxWorkers(t *testing.T) {
	runner := WithMaxWorkers(4)
	if runner.MaxWorkers != 4 {
		t.Errorf("MaxWorkers = %d, want 4", runner.MaxWorkers)
	}
}

func BenchmarkConcurrentRunner_10Commands(b *testing.B) {
	cmds := make([]Command, 10)
	for i := range cmds {
		cmds[i] = &mockCommand{name: "bench"}
	}
	runner := &ConcurrentRunner{MaxWorkers: 4}
	execCtx := &Context{Stdout: io.Discard, Stderr: io.Discard}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, c := range cmds {
			c.(*mockCommand).called = false
		}
		_ = runner.Execute(ctx, cmds, execCtx)
	}
}
