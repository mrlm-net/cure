package style

import (
	"os"
	"regexp"
)

// enabled controls whether ANSI escape codes are emitted. It is set to false
// when the NO_COLOR environment variable is non-empty, and can be toggled at
// runtime with [Disable] / [Enable].
//
// There is no mutex guarding this variable. The intended usage is:
//   - Set once at program startup via NO_COLOR or an explicit Disable call.
//   - Tests that mutate enabled must save and restore its value.
var enabled = true

// ansiRegexp matches ANSI SGR (Select Graphic Rendition) escape sequences of
// the form ESC [ <params> m. Compiled once at package init to avoid repeated
// allocations in [Reset].
var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func init() {
	if os.Getenv("NO_COLOR") != "" {
		enabled = false
	}
}

// apply wraps text with the given ANSI open code and the standard reset suffix.
// Returns text unchanged when styling is disabled.
func apply(code, text string) string {
	if !enabled {
		return text
	}
	return code + text + "\x1b[0m"
}

// --- Colors ---

// Red wraps text in the ANSI red foreground color code (31).
func Red(text string) string { return apply("\x1b[31m", text) }

// Green wraps text in the ANSI green foreground color code (32).
func Green(text string) string { return apply("\x1b[32m", text) }

// Yellow wraps text in the ANSI yellow foreground color code (33).
func Yellow(text string) string { return apply("\x1b[33m", text) }

// Blue wraps text in the ANSI blue foreground color code (34).
func Blue(text string) string { return apply("\x1b[34m", text) }

// Magenta wraps text in the ANSI magenta foreground color code (35).
func Magenta(text string) string { return apply("\x1b[35m", text) }

// Cyan wraps text in the ANSI cyan foreground color code (36).
func Cyan(text string) string { return apply("\x1b[36m", text) }

// White wraps text in the ANSI white foreground color code (37).
func White(text string) string { return apply("\x1b[37m", text) }

// Gray wraps text in the ANSI bright-black (gray) foreground color code (90).
func Gray(text string) string { return apply("\x1b[90m", text) }

// --- Text Styles ---

// Bold wraps text in the ANSI bold style code (1).
func Bold(text string) string { return apply("\x1b[1m", text) }

// Dim wraps text in the ANSI dim (faint) style code (2).
func Dim(text string) string { return apply("\x1b[2m", text) }

// Underline wraps text in the ANSI underline style code (4).
func Underline(text string) string { return apply("\x1b[4m", text) }

// --- Control ---

// Reset strips all ANSI SGR escape sequences from text. Useful for computing
// display widths or writing styled strings to non-terminal outputs such as
// log files.
func Reset(text string) string {
	return ansiRegexp.ReplaceAllString(text, "")
}

// Enabled reports whether ANSI styling is currently active.
func Enabled() bool { return enabled }

// Disable turns off ANSI styling. Subsequent calls to color and style
// functions will return their input unchanged until [Enable] is called.
func Disable() { enabled = false }

// Enable turns on ANSI styling. This overrides a prior [Disable] call or a
// NO_COLOR environment variable that was set at startup.
func Enable() { enabled = true }
