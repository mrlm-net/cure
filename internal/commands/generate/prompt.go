package generate

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Prompter handles interactive user input with validation.
type Prompter struct {
	stdout io.Writer
	stdin  io.Reader
	reader *bufio.Scanner
}

// NewPrompter creates a Prompter with the given output and input streams.
func NewPrompter(stdout io.Writer, stdin io.Reader) *Prompter {
	return &Prompter{
		stdout: stdout,
		stdin:  stdin,
		reader: bufio.NewScanner(stdin),
	}
}

// Required prompts for a value and validates it is non-empty.
// Repeats prompt until valid input is received.
// defaultVal is shown if non-empty and used if user presses Enter.
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

// Optional prompts for a value and returns defaultVal if user presses Enter.
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

// Confirm prompts for yes/no confirmation. Returns true for "y", false for "n".
// Repeats prompt until valid input is received.
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
