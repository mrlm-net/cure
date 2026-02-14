package terminal

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// WithSignalHandler configures the Router to intercept OS signals (SIGINT,
// SIGTERM) and cancel the context passed to commands. When a signal is
// received, the context is cancelled and commands should return promptly.
//
// A second signal forces immediate termination via os.Exit(1).
//
// This option only takes effect when using [Router.RunArgs] or [Router.RunContext].
// The signal handler is active for the duration of command execution and is
// cleaned up afterward.
func WithSignalHandler() Option {
	return func(r *Router) {
		r.handleSignal = true
	}
}

// WithTimeout sets a per-command execution timeout. If a command does not
// complete within the specified duration, its context is cancelled.
//
// The timeout applies to the time spent in the Runner.Execute call.
// It does not include flag parsing or command lookup.
//
// A zero or negative duration disables the timeout.
func WithTimeout(d time.Duration) Option {
	return func(r *Router) {
		r.timeout = d
	}
}

// WithGracePeriod sets the duration to wait after context cancellation
// before forcefully terminating. During the grace period, commands can
// perform cleanup. After the grace period, the context's Done channel
// is closed with DeadlineExceeded.
//
// Default: 5 seconds.
// Only meaningful when combined with [WithSignalHandler] or [WithTimeout].
func WithGracePeriod(d time.Duration) Option {
	return func(r *Router) {
		r.gracePeriod = d
	}
}

// setupSignalHandler creates a context that is cancelled on SIGINT or SIGTERM.
// On first signal, the context is cancelled. On second signal, os.Exit(1).
func (r *Router) setupSignalHandler(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCh:
			// First signal: cancel context (grace period starts)
			if r.logger != nil {
				r.logger.InfoContext(ctx, "received signal, shutting down gracefully")
			}
			cancel()

			// Wait for second signal or grace period
			select {
			case <-sigCh:
				// Second signal: force exit
				os.Exit(1)
			case <-time.After(r.gracePeriod):
				// Grace period expired
				return
			}
		case <-ctx.Done():
			return
		}
	}()

	cleanup := func() {
		signal.Stop(sigCh)
		close(sigCh)
		cancel()
	}
	return ctx, cleanup
}
