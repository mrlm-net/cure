package terminal

import "log/slog"

// WithLogger sets the structured logger for the Router.
// When set, the Router logs command dispatch, flag parsing, and execution
// duration at debug and info levels. The logger is also passed to commands
// via [Context].Logger.
//
// When not set (the default), no logging occurs and there is zero
// overhead from log argument evaluation.
func WithLogger(logger *slog.Logger) Option {
	return func(r *Router) {
		r.logger = logger
	}
}
