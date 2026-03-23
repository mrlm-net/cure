//go:build !linux && !darwin

package ctxcmd

// isatty always returns false on non-Unix platforms.
func isatty(fd uintptr) bool { return false }
