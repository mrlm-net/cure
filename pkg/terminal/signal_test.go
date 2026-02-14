package terminal

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestWithSignalHandler_SetsFlag(t *testing.T) {
	router := New(WithSignalHandler())
	if !router.handleSignal {
		t.Error("WithSignalHandler did not set handleSignal flag")
	}
}

func TestWithTimeout_SetsTimeout(t *testing.T) {
	router := New(WithTimeout(3 * time.Second))
	if router.timeout != 3*time.Second {
		t.Errorf("timeout = %v, want 3s", router.timeout)
	}
}

func TestWithGracePeriod_SetsGracePeriod(t *testing.T) {
	router := New(WithGracePeriod(10 * time.Second))
	if router.gracePeriod != 10*time.Second {
		t.Errorf("gracePeriod = %v, want 10s", router.gracePeriod)
	}
}

func TestWithGracePeriod_Default(t *testing.T) {
	router := New()
	if router.gracePeriod != 5*time.Second {
		t.Errorf("default gracePeriod = %v, want 5s", router.gracePeriod)
	}
}

func TestWithTimeout_CancelsContext(t *testing.T) {
	router := New(
		WithStdout(io.Discard),
		WithStderr(io.Discard),
		WithTimeout(50*time.Millisecond),
	)

	// Command that blocks until context is done
	slowCmd := &mockCommand{name: "slow"}
	slowCmd.err = nil
	blocker := &blockingCommand{mockCommand: mockCommand{name: "block"}}
	router.Register(blocker)

	err := router.RunArgs([]string{"block"})
	if err == nil {
		t.Fatal("expected error from timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want context.DeadlineExceeded", err)
	}
}

func TestWithTimeout_CompletesBeforeTimeout(t *testing.T) {
	router := New(
		WithStdout(io.Discard),
		WithStderr(io.Discard),
		WithTimeout(5*time.Second),
	)

	cmd := &mockCommand{name: "fast"}
	router.Register(cmd)

	err := router.RunArgs([]string{"fast"})
	if err != nil {
		t.Fatalf("RunArgs() error = %v", err)
	}
	if !cmd.called {
		t.Error("command was not executed")
	}
}

// blockingCommand blocks until its context is cancelled.
type blockingCommand struct {
	mockCommand
}

func (c *blockingCommand) Run(ctx context.Context, tc *Context) error {
	c.called = true
	<-ctx.Done()
	return ctx.Err()
}

func BenchmarkRunContext_WithTimeout(b *testing.B) {
	router := New(
		WithStdout(io.Discard),
		WithStderr(io.Discard),
		WithTimeout(10*time.Second),
	)
	cmd := &mockCommand{name: "bench"}
	router.Register(cmd)
	args := []string{"bench"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd.called = false
		_ = router.RunArgs(args)
	}
}

func BenchmarkSignalSetup(b *testing.B) {
	r := &Router{gracePeriod: 5 * time.Second}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use a fresh context per iteration so cleanup works cleanly.
		ctx, baseCancel := context.WithCancel(context.Background())
		_, cleanup := r.setupSignalHandler(ctx)
		cleanup()
		baseCancel()
	}
}
