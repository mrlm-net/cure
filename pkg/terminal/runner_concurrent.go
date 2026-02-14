package terminal

import (
	"context"
	"errors"
	"runtime"
	"sync"
)

// ConcurrentRunner executes commands concurrently using goroutines.
// Each command runs in its own goroutine with an independent copy of the
// execution context (separate writers to avoid data races).
//
// Configure concurrency with [WithMaxWorkers]. Default: runtime.NumCPU().
//
// Errors from all commands are aggregated using [errors.Join].
// Respects context cancellation -- no new commands start after ctx.Done().
type ConcurrentRunner struct {
	// MaxWorkers limits the number of concurrent goroutines.
	// Zero or negative means runtime.NumCPU().
	MaxWorkers int
}

// WithMaxWorkers returns a ConcurrentRunner with the specified worker limit.
func WithMaxWorkers(n int) *ConcurrentRunner {
	return &ConcurrentRunner{MaxWorkers: n}
}

// Execute runs commands concurrently, respecting the MaxWorkers limit.
// Errors from all commands are aggregated using errors.Join.
// Returns nil if all commands succeed or the command list is empty.
func (r *ConcurrentRunner) Execute(ctx context.Context, commands []Command, execCtx *Context) error {
	if len(commands) == 0 {
		return nil
	}

	maxWorkers := r.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}

	// Semaphore channel limits concurrency
	sem := make(chan struct{}, maxWorkers)

	var (
		mu   sync.Mutex
		errs []error
		wg   sync.WaitGroup
	)

	for _, cmd := range commands {
		// Check context before starting new work
		select {
		case <-ctx.Done():
			mu.Lock()
			errs = append(errs, ctx.Err())
			mu.Unlock()
			goto done
		default:
		}

		wg.Add(1)
		go func(c Command) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				mu.Lock()
				errs = append(errs, ctx.Err())
				mu.Unlock()
				return
			}

			// Each command gets its own Context to avoid data races
			cmdCtx := &Context{
				Args:   execCtx.Args,
				Flags:  execCtx.Flags,
				Stdin:  execCtx.Stdin,
				Stdout: execCtx.Stdout,
				Stderr: execCtx.Stderr,
				Logger: execCtx.Logger,
			}

			if err := c.Run(ctx, cmdCtx); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}(cmd)
	}

done:
	wg.Wait()
	return errors.Join(errs...)
}
