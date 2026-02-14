package terminal

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestWithLogger_AppliesLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := New(WithLogger(logger))

	if router.logger != logger {
		t.Error("WithLogger did not set the logger")
	}
}

func TestContext_LoggerPopulated(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := New(
		WithStdout(io.Discard),
		WithStderr(io.Discard),
		WithLogger(logger),
	)

	var gotLogger *slog.Logger
	cmd := &logCheckCommand{
		mockCommand: mockCommand{name: "check"},
		gotLogger:   &gotLogger,
	}
	router.Register(cmd)

	err := router.RunArgs([]string{"check"})
	if err != nil {
		t.Fatalf("RunArgs() error = %v", err)
	}

	if gotLogger == nil {
		t.Fatal("Context.Logger was nil, expected non-nil")
	}
	if gotLogger != logger {
		t.Error("Context.Logger does not match the configured logger")
	}
}

func TestContext_LoggerNilWhenNotConfigured(t *testing.T) {
	router := New(WithStdout(io.Discard), WithStderr(io.Discard))

	var gotLogger *slog.Logger
	cmd := &logCheckCommand{
		mockCommand: mockCommand{name: "check"},
		gotLogger:   &gotLogger,
	}
	router.Register(cmd)

	err := router.RunArgs([]string{"check"})
	if err != nil {
		t.Fatalf("RunArgs() error = %v", err)
	}

	// gotLogger should remain nil since it was never assigned
	if gotLogger != nil {
		t.Error("Context.Logger should be nil when no logger is configured")
	}
}

func TestLogger_OutputsJSON(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	router := New(
		WithStdout(io.Discard),
		WithStderr(io.Discard),
		WithLogger(logger),
	)
	router.Register(&mockCommand{name: "test"})

	err := router.RunArgs([]string{"test"})
	if err != nil {
		t.Fatalf("RunArgs() error = %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("expected log output, got empty")
	}

	// Parse JSON lines
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 log lines, got %d: %s", len(lines), output)
	}

	// Check first log entry (dispatching command)
	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatalf("failed to parse log JSON: %v", err)
	}
	if entry["msg"] != "dispatching command" {
		t.Errorf("first log msg = %q, want %q", entry["msg"], "dispatching command")
	}
	if entry["command"] != "test" {
		t.Errorf("first log command = %q, want %q", entry["command"], "test")
	}

	// Check last log entry (command completed)
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &entry); err != nil {
		t.Fatalf("failed to parse log JSON: %v", err)
	}
	if entry["msg"] != "command completed" {
		t.Errorf("last log msg = %q, want %q", entry["msg"], "command completed")
	}
}

func TestLogger_NoOutputWithoutLogger(t *testing.T) {
	router := New(WithStdout(io.Discard), WithStderr(io.Discard))
	router.Register(&mockCommand{name: "test"})

	// No logger configured -- should produce no output anywhere
	err := router.RunArgs([]string{"test"})
	if err != nil {
		t.Fatalf("RunArgs() error = %v", err)
	}
	// If we got here without panic, the nil logger checks are working
}

// logCheckCommand captures the Logger from Context.
type logCheckCommand struct {
	mockCommand
	gotLogger **slog.Logger
}

func (c *logCheckCommand) Run(_ context.Context, tc *Context) error {
	c.called = true
	*c.gotLogger = tc.Logger
	return nil
}

func BenchmarkRunContext_WithLogger(b *testing.B) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
	router := New(
		WithStdout(io.Discard),
		WithStderr(io.Discard),
		WithLogger(logger),
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

func BenchmarkRunContext_WithoutLogger(b *testing.B) {
	router := New(
		WithStdout(io.Discard),
		WithStderr(io.Discard),
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
