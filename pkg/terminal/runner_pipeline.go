package terminal

import (
	"context"
	"io"
	"sync"
)

// PipelineRunner executes commands in sequence, piping the stdout of each
// command to the stdin of the next. The first command receives Stdin from
// the execution context (may be nil). The last command writes to the
// execution context's Stdout.
//
// Each pair of adjacent commands is connected by an [io.Pipe]. If any
// command fails, downstream commands are not started and the pipeline
// returns the error. Upstream commands that have already started will
// have their pipe closed, causing them to receive a write error if they
// are still producing output.
//
// PipelineRunner uses one goroutine per command to enable concurrent
// pipe processing. All goroutines are cleaned up before Execute returns.
type PipelineRunner struct{}

// Execute runs commands in a pipeline. Each command's stdout is connected
// to the next command's stdin via an io.Pipe.
// Returns nil if all commands succeed or the command list is empty.
// Returns the first non-nil error from any stage.
func (r *PipelineRunner) Execute(ctx context.Context, commands []Command, execCtx *Context) error {
	if len(commands) == 0 {
		return nil
	}
	if len(commands) == 1 {
		return commands[0].Run(ctx, execCtx)
	}

	// Create pipes: N-1 pipes for N commands
	type pipe struct {
		reader *io.PipeReader
		writer *io.PipeWriter
	}
	pipes := make([]pipe, len(commands)-1)
	for i := range pipes {
		pipes[i].reader, pipes[i].writer = io.Pipe()
	}

	// stageResult collects the error from each stage
	type stageResult struct {
		index int
		err   error
	}
	results := make(chan stageResult, len(commands))

	// Launch all commands in goroutines
	var wg sync.WaitGroup
	for i, cmd := range commands {
		wg.Add(1)
		go func(idx int, c Command) {
			defer wg.Done()

			cmdCtx := &Context{
				Args:   execCtx.Args,
				Flags:  execCtx.Flags,
				Stderr: execCtx.Stderr,
				Logger: execCtx.Logger,
			}

			// First command: stdin from execCtx
			// Middle/last commands: stdin from previous pipe
			if idx == 0 {
				cmdCtx.Stdin = execCtx.Stdin
			} else {
				cmdCtx.Stdin = pipes[idx-1].reader
			}

			// Last command: stdout to execCtx
			// First/middle commands: stdout to next pipe
			if idx == len(commands)-1 {
				cmdCtx.Stdout = execCtx.Stdout
			} else {
				cmdCtx.Stdout = pipes[idx].writer
			}

			err := c.Run(ctx, cmdCtx)

			// Close the write end of our pipe (if any) so downstream
			// readers get EOF or error
			if idx < len(commands)-1 {
				if err != nil {
					pipes[idx].writer.CloseWithError(err)
				} else {
					pipes[idx].writer.Close()
				}
			}

			results <- stageResult{index: idx, err: err}
		}(i, cmd)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(results)

	// Collect errors in stage order
	errs := make([]error, len(commands))
	for res := range results {
		errs[res.index] = res.err
	}

	// Return the first non-nil error (pipeline fails at first broken stage)
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
