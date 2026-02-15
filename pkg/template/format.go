package template

import (
	"regexp"
	"strings"
)

var (
	// trailingWhitespace matches whitespace at end of lines
	trailingWhitespace = regexp.MustCompile(`[ \t]+$`)

	// multipleBlankLines matches 3+ consecutive blank lines
	multipleBlankLines = regexp.MustCompile(`\n{3,}`)
)

// Format cleans up rendered template output by:
//   - Normalizing line endings to \n
//   - Removing trailing whitespace from each line
//   - Reducing 3+ consecutive blank lines to 2 blank lines
//   - Ensuring output ends with exactly one newline
//
// This ensures generated files are clean and pass linting.
func Format(content string) string {
	// Normalize line endings (Windows \r\n → Unix \n)
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// Remove trailing whitespace per line
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	content = strings.Join(lines, "\n")

	// Reduce 3+ blank lines to 2 blank lines
	content = multipleBlankLines.ReplaceAllString(content, "\n\n")

	// Ensure final newline
	content = strings.TrimRight(content, "\n") + "\n"

	return content
}
