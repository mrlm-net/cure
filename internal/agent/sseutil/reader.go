// Package sseutil provides a minimal SSE (Server-Sent Events) line parser.
package sseutil

import (
	"bufio"
	"context"
	"io"
	"strings"
)

// Parse reads SSE-formatted data from r, calling onEvent for each "data:" line's payload.
// Returns when r is exhausted, ctx is cancelled, or onEvent returns false.
// The [DONE] sentinel is treated as EOF (returns nil, not an error).
// Skips blank lines and lines that do not start with "data:".
func Parse(ctx context.Context, r io.Reader, onEvent func(data []byte) bool) error {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}

		line := scanner.Text()

		if line == "data: [DONE]" {
			return nil
		}

		if strings.HasPrefix(line, "data: ") {
			payload := []byte(line[6:])
			if !onEvent(payload) {
				return nil
			}
		}
		// Blank lines and non-data lines are skipped.
	}

	return scanner.Err()
}
