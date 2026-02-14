package terminal

import (
	"context"
	"errors"
)

// Runner executes one or more commands with a defined execution strategy.
// Implementations control how commands are executed — serially, concurrently,
// or in a pipeline with output chaining.
//
// The Router uses a Runner to execute matched commands. The default is
// [SerialRunner], which executes commands sequentially and stops on first error.
type Runner interface {
	// Execute runs the provided commands with the given context.
	// The execution strategy is implementation-defined.
	//
	// Execute must respect ctx.Done() and halt execution if the context is cancelled.
	// Returns nil if all commands succeed, or the first error encountered.
	Execute(ctx context.Context, commands []Command, execCtx *Context) error
}

// SerialRunner executes commands sequentially, stopping on the first error.
// Each command receives the same [Context]. If any command returns an error,
// execution halts and the error is returned. Respects context cancellation
// by checking ctx.Done() before each command.
//
// This is the default Runner used by [Router].
type SerialRunner struct{}

// Execute runs commands one at a time in order.
// Returns nil if all succeed, or the first error encountered.
// If ctx is cancelled before a command starts, returns ctx.Err().
func (r *SerialRunner) Execute(ctx context.Context, commands []Command, execCtx *Context) error {
	for _, cmd := range commands {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := cmd.Run(ctx, execCtx); err != nil {
			return err
		}
	}
	return nil
}

// ErrNotImplemented is returned by runner stubs that are designed but not
// yet implemented. See issues #10 (ConcurrentRunner) and #11 (PipelineRunner).
var ErrNotImplemented = errors.New("terminal: runner not implemented")

// ConcurrentRunner executes commands concurrently using goroutines.
//
// Not yet implemented — returns [ErrNotImplemented].
// See issue #10 for implementation tracking.
type ConcurrentRunner struct{}

// Execute returns [ErrNotImplemented].
func (r *ConcurrentRunner) Execute(_ context.Context, _ []Command, _ *Context) error {
	return ErrNotImplemented
}

// PipelineRunner executes commands in a pipeline, chaining the output
// of each command to the input of the next.
//
// Not yet implemented — returns [ErrNotImplemented].
// See issue #11 for implementation tracking.
type PipelineRunner struct{}

// Execute returns [ErrNotImplemented].
func (r *PipelineRunner) Execute(_ context.Context, _ []Command, _ *Context) error {
	return ErrNotImplemented
}
