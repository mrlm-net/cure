package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Option represents a selectable item in a menu prompt.
type Option struct {
	// Label is the display text shown to the user.
	Label string
	// Value is the machine-readable return value.
	Value string
	// Description is optional help text shown alongside the label.
	Description string
}

// Prompter handles interactive user input with validation.
// Use [NewPrompter] to construct a Prompter — the zero value is not usable.
type Prompter struct {
	stdout io.Writer
	stdin  io.Reader
	reader *bufio.Scanner
}

// NewPrompter creates a Prompter with the given output and input streams.
// The scanner is initialized once and shared across all prompt calls, so
// callers must not read from stdin directly after creating a Prompter.
func NewPrompter(stdout io.Writer, stdin io.Reader) *Prompter {
	return &Prompter{
		stdout: stdout,
		stdin:  stdin,
		reader: bufio.NewScanner(stdin),
	}
}

// Required prompts for a value and validates it is non-empty.
// If defaultVal is non-empty it is shown in brackets and returned when the
// user presses Enter without typing anything. The prompt repeats until a
// non-empty value is provided. EOF without input returns an error.
func (p *Prompter) Required(prompt string, defaultVal string) (string, error) {
	for {
		fmt.Fprint(p.stdout, prompt)
		if defaultVal != "" {
			fmt.Fprintf(p.stdout, " [%s]", defaultVal)
		}
		fmt.Fprint(p.stdout, " ")

		if !p.reader.Scan() {
			if err := p.reader.Err(); err != nil {
				return "", err
			}
			return "", fmt.Errorf("unexpected EOF")
		}

		input := strings.TrimSpace(p.reader.Text())
		if input == "" && defaultVal != "" {
			return defaultVal, nil
		}
		if input != "" {
			return input, nil
		}

		fmt.Fprintln(p.stdout, "Error: This field is required. Please provide a value.")
	}
}

// Optional prompts for a value and returns defaultVal if the user presses
// Enter without typing anything. Unlike [Prompter.Required], the prompt is
// never repeated. EOF without input also returns defaultVal.
func (p *Prompter) Optional(prompt string, defaultVal string) (string, error) {
	fmt.Fprint(p.stdout, prompt, " ")

	if !p.reader.Scan() {
		if err := p.reader.Err(); err != nil {
			return "", err
		}
		return defaultVal, nil
	}

	input := strings.TrimSpace(p.reader.Text())
	if input == "" {
		return defaultVal, nil
	}
	return input, nil
}

// Confirm prompts for yes/no confirmation. Accepted values are y, yes, n, no
// (case-insensitive). The prompt repeats until valid input is received.
// EOF returns an error.
func (p *Prompter) Confirm(prompt string) (bool, error) {
	for {
		fmt.Fprintf(p.stdout, "%s (y/n): ", prompt)

		if !p.reader.Scan() {
			if err := p.reader.Err(); err != nil {
				return false, err
			}
			return false, fmt.Errorf("unexpected EOF")
		}

		input := strings.ToLower(strings.TrimSpace(p.reader.Text()))
		switch input {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Fprintln(p.stdout, "Please answer 'y' or 'n'.")
		}
	}
}

// SingleSelect displays a numbered list of options and returns the selected
// [Option]. Selection is 1-based. The prompt repeats on invalid input or
// out-of-range numbers. EOF returns an error.
//
// Example output:
//
//	Choose language:
//	  1) Go
//	  2) TypeScript (compiled to JS)
//	Enter number:
func (p *Prompter) SingleSelect(prompt string, options []Option) (Option, error) {
	if len(options) == 0 {
		return Option{}, fmt.Errorf("no options provided")
	}

	for {
		fmt.Fprintf(p.stdout, "%s:\n", prompt)
		for i, opt := range options {
			if opt.Description != "" {
				fmt.Fprintf(p.stdout, "  %d) %s (%s)\n", i+1, opt.Label, opt.Description)
			} else {
				fmt.Fprintf(p.stdout, "  %d) %s\n", i+1, opt.Label)
			}
		}
		fmt.Fprint(p.stdout, "Enter number: ")

		if !p.reader.Scan() {
			if err := p.reader.Err(); err != nil {
				return Option{}, err
			}
			return Option{}, fmt.Errorf("unexpected EOF")
		}

		input := strings.TrimSpace(p.reader.Text())
		n, err := strconv.Atoi(input)
		if err != nil || n < 1 || n > len(options) {
			fmt.Fprintf(p.stdout, "Please enter a number between 1 and %d.\n", len(options))
			continue
		}

		return options[n-1], nil
	}
}

// MultiSelect displays a numbered list of options and returns the subset
// selected by the user. The user may enter:
//   - Comma-separated numbers (e.g. "1,3,4")
//   - "all" to select every option
//   - "none" to select no options
//
// Duplicate numbers are deduplicated and order follows the original options
// slice. The prompt repeats on invalid input. EOF returns an error.
//
// Example output:
//
//	Select features (comma-separated numbers, "all", or "none"):
//	  1) Logging
//	  2) Metrics
//	  3) Tracing
//	Enter selection:
func (p *Prompter) MultiSelect(prompt string, options []Option) ([]Option, error) {
	if len(options) == 0 {
		return nil, fmt.Errorf("no options provided")
	}

	for {
		fmt.Fprintf(p.stdout, "%s (comma-separated numbers, \"all\", or \"none\"):\n", prompt)
		for i, opt := range options {
			if opt.Description != "" {
				fmt.Fprintf(p.stdout, "  %d) %s (%s)\n", i+1, opt.Label, opt.Description)
			} else {
				fmt.Fprintf(p.stdout, "  %d) %s\n", i+1, opt.Label)
			}
		}
		fmt.Fprint(p.stdout, "Enter selection: ")

		if !p.reader.Scan() {
			if err := p.reader.Err(); err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("unexpected EOF")
		}

		raw := strings.TrimSpace(p.reader.Text())
		lower := strings.ToLower(raw)

		switch lower {
		case "all":
			result := make([]Option, len(options))
			copy(result, options)
			return result, nil
		case "none":
			return []Option{}, nil
		}

		// Parse comma-separated numbers.
		parts := strings.Split(raw, ",")
		seen := make(map[int]bool, len(parts))
		valid := true
		for _, part := range parts {
			part = strings.TrimSpace(part)
			n, err := strconv.Atoi(part)
			if err != nil || n < 1 || n > len(options) {
				fmt.Fprintf(p.stdout, "Invalid selection %q. Enter numbers between 1 and %d, \"all\", or \"none\".\n", part, len(options))
				valid = false
				break
			}
			seen[n] = true
		}
		if !valid {
			continue
		}

		// Collect in original order to provide deterministic output.
		result := make([]Option, 0, len(seen))
		for i, opt := range options {
			if seen[i+1] {
				result = append(result, opt)
			}
		}
		return result, nil
	}
}

// IsInteractive reports whether stdin is a terminal (as opposed to a pipe,
// file redirect, or other non-interactive source). It uses an *os.File type
// assertion and ModeCharDevice so no syscall package is required.
//
// Returns false for any io.Reader that is not an *os.File, including test
// buffers and strings.Reader values. This makes non-interactive behavior the
// safe default in automated contexts.
func IsInteractive(stdin io.Reader) bool {
	f, ok := stdin.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
