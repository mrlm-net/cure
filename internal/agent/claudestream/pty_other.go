//go:build !darwin && !linux

package claudestream

import (
	"errors"
	"os"
)

// openPTY is not implemented on this platform.  The adapter falls back to
// pipe-based subprocess communication, which works but does not give
// true character-level streaming (Node.js may buffer stdout on pipes).
func openPTY() (*os.File, string, error) {
	return nil, "", errors.New("PTY not supported on this platform")
}
