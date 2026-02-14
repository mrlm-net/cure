package terminal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"runtime"
	"testing"
	"time"
)

// writerCommand writes a fixed string to stdout.
type writerCommand struct {
	mockCommand
	output string
}

func (c *writerCommand) Run(_ context.Context, tc *Context) error {
	c.called = true
	_, err := fmt.Fprint(tc.Stdout, c.output)
	return err
}

// readerCommand reads from stdin and writes to stdout.
type readerCommand struct {
	mockCommand
	prefix string
}

func (c *readerCommand) Run(_ context.Context, tc *Context) error {
	c.called = true
	if tc.Stdin == nil {
		return errors.New("stdin is nil")
	}
	data, err := io.ReadAll(tc.Stdin)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(tc.Stdout, "%s%s", c.prefix, string(data))
	return err
}

// uppercaseCommand reads from stdin, uppercases, and writes to stdout.
type uppercaseCommand struct {
	mockCommand
}

func (c *uppercaseCommand) Run(_ context.Context, tc *Context) error {
	c.called = true
	if tc.Stdin == nil {
		return errors.New("stdin is nil")
	}
	data, err := io.ReadAll(tc.Stdin)
	if err != nil {
		return err
	}
	result := bytes.ToUpper(data)
	_, err = tc.Stdout.Write(result)
	return err
}

func TestPipelineRunner_TwoCommands(t *testing.T) {
	var buf bytes.Buffer
	execCtx := &Context{Stdout: &buf, Stderr: io.Discard}

	writer := &writerCommand{mockCommand: mockCommand{name: "write"}, output: "hello"}
	reader := &readerCommand{mockCommand: mockCommand{name: "read"}, prefix: "got: "}

	runner := &PipelineRunner{}
	err := runner.Execute(context.Background(), []Command{writer, reader}, execCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := "got: hello"
	if buf.String() != want {
		t.Errorf("output = %q, want %q", buf.String(), want)
	}
	if !writer.called {
		t.Error("writer command not called")
	}
	if !reader.called {
		t.Error("reader command not called")
	}
}

func TestPipelineRunner_ThreeCommands(t *testing.T) {
	var buf bytes.Buffer
	execCtx := &Context{Stdout: &buf, Stderr: io.Discard}

	writer := &writerCommand{mockCommand: mockCommand{name: "write"}, output: "hello"}
	upper := &uppercaseCommand{mockCommand: mockCommand{name: "upper"}}
	reader := &readerCommand{mockCommand: mockCommand{name: "read"}, prefix: "result: "}

	runner := &PipelineRunner{}
	err := runner.Execute(context.Background(), []Command{writer, upper, reader}, execCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	want := "result: HELLO"
	if buf.String() != want {
		t.Errorf("output = %q, want %q", buf.String(), want)
	}
}

func TestPipelineRunner_SingleCommand(t *testing.T) {
	var buf bytes.Buffer
	execCtx := &Context{Stdout: &buf, Stderr: io.Discard}

	writer := &writerCommand{mockCommand: mockCommand{name: "write"}, output: "solo"}
	runner := &PipelineRunner{}

	err := runner.Execute(context.Background(), []Command{writer}, execCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if buf.String() != "solo" {
		t.Errorf("output = %q, want %q", buf.String(), "solo")
	}
}

func TestPipelineRunner_EmptyCommands(t *testing.T) {
	runner := &PipelineRunner{}
	err := runner.Execute(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestPipelineRunner_FirstCommandFails(t *testing.T) {
	var buf bytes.Buffer
	execCtx := &Context{Stdout: &buf, Stderr: io.Discard}

	failing := &mockCommand{name: "fail", err: errors.New("write failed")}
	reader := &readerCommand{mockCommand: mockCommand{name: "read"}, prefix: "got: "}

	runner := &PipelineRunner{}
	err := runner.Execute(context.Background(), []Command{failing, reader}, execCtx)
	if err == nil {
		t.Fatal("expected error from failing command")
	}
	if err.Error() != "write failed" {
		t.Errorf("error = %q, want %q", err.Error(), "write failed")
	}
}

func TestPipelineRunner_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	var buf bytes.Buffer
	execCtx := &Context{Stdout: &buf, Stderr: io.Discard}

	// blockingWriter blocks until context is cancelled, then returns
	blocker := &blockingWriterCommand{mockCommand: mockCommand{name: "block"}}
	reader := &readerCommand{mockCommand: mockCommand{name: "read"}, prefix: "got: "}

	runner := &PipelineRunner{}

	done := make(chan error, 1)
	go func() {
		done <- runner.Execute(ctx, []Command{blocker, reader}, execCtx)
	}()

	// Cancel after a short delay
	time.Sleep(20 * time.Millisecond)
	cancel()

	err := <-done
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// blockingWriterCommand blocks until context is cancelled.
type blockingWriterCommand struct {
	mockCommand
}

func (c *blockingWriterCommand) Run(ctx context.Context, tc *Context) error {
	c.called = true
	<-ctx.Done()
	return ctx.Err()
}

func TestPipelineRunner_NoGoroutineLeaks(t *testing.T) {
	// Baseline goroutine count
	runtime.GC()
	time.Sleep(10 * time.Millisecond)
	before := runtime.NumGoroutine()

	var buf bytes.Buffer
	execCtx := &Context{Stdout: &buf, Stderr: io.Discard}

	writer := &writerCommand{mockCommand: mockCommand{name: "write"}, output: "hello"}
	reader := &readerCommand{mockCommand: mockCommand{name: "read"}, prefix: "got: "}

	runner := &PipelineRunner{}
	for i := 0; i < 10; i++ {
		buf.Reset()
		writer.called = false
		reader.called = false
		_ = runner.Execute(context.Background(), []Command{writer, reader}, execCtx)
	}

	runtime.GC()
	time.Sleep(10 * time.Millisecond)
	after := runtime.NumGoroutine()

	// Allow some variance for runtime goroutines
	if after > before+5 {
		t.Errorf("goroutine leak: before=%d, after=%d", before, after)
	}
}

func BenchmarkPipelineRunner_3Commands(b *testing.B) {
	runner := &PipelineRunner{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		execCtx := &Context{Stdout: &buf, Stderr: io.Discard}

		writer := &writerCommand{mockCommand: mockCommand{name: "write"}, output: "bench data"}
		upper := &uppercaseCommand{mockCommand: mockCommand{name: "upper"}}
		reader := &readerCommand{mockCommand: mockCommand{name: "read"}, prefix: ""}

		_ = runner.Execute(context.Background(), []Command{writer, upper, reader}, execCtx)
	}
}
