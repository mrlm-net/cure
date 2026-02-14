package terminal

import (
	"fmt"
	"strings"
)

// CommandNotFoundError is returned when a command name does not match any
// registered command. It includes suggestions for similar commands when
// available.
type CommandNotFoundError struct {
	// Name is the command name that was not found.
	Name string

	// Suggestions contains similar command names, ordered by relevance.
	// May be empty if no similar commands exist.
	Suggestions []string
}

// Error returns a human-readable message. If suggestions are available,
// they are appended as "Did you mean: X, Y?"
func (e *CommandNotFoundError) Error() string {
	msg := fmt.Sprintf("unknown command: %s", e.Name)
	if len(e.Suggestions) > 0 {
		msg += fmt.Sprintf(". Did you mean: %s?", strings.Join(e.Suggestions, ", "))
	}
	return msg
}

// CommandError wraps an error returned by a command's Run method,
// preserving the command name for structured error handling.
type CommandError struct {
	// Command is the name of the command that failed.
	Command string

	// Err is the underlying error from the command.
	Err error
}

// Error returns a human-readable message including the command name.
func (e *CommandError) Error() string {
	return fmt.Sprintf("command %s: %v", e.Command, e.Err)
}

// Unwrap returns the underlying error.
func (e *CommandError) Unwrap() error {
	return e.Err
}

// FlagParseError wraps a flag parsing failure, preserving the command
// name and the underlying parse error.
type FlagParseError struct {
	// Command is the name of the command whose flags failed to parse.
	Command string

	// Err is the underlying flag parsing error.
	Err error
}

// Error returns a human-readable message including the command name.
func (e *FlagParseError) Error() string {
	return fmt.Sprintf("flag parsing failed for %s: %v", e.Command, e.Err)
}

// Unwrap returns the underlying flag parsing error.
func (e *FlagParseError) Unwrap() error {
	return e.Err
}

// NoCommandError is returned when Run or RunContext is called with
// an empty argument list.
type NoCommandError struct{}

// Error returns "no command specified".
func (e *NoCommandError) Error() string {
	return "no command specified"
}

// levenshtein computes the edit distance between two strings.
// Uses the single-row optimization for O(min(m,n)) space.
func levenshtein(a, b string) int {
	if len(a) < len(b) {
		a, b = b, a
	}
	if len(b) == 0 {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		curr := make([]int, len(b)+1)
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			ins := curr[j-1] + 1
			del := prev[j] + 1
			sub := prev[j-1] + cost
			curr[j] = min(ins, min(del, sub))
		}
		prev = curr
	}
	return prev[len(b)]
}
