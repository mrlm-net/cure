package style

import (
	"strings"
	"testing"
)

// saveState returns a function that restores the package-level enabled var to
// its value at the time of the call. Use with defer in any test that mutates
// the enabled state.
func saveState() func() {
	orig := enabled
	return func() { enabled = orig }
}

// TestColorFunctions verifies that each color function wraps text in the
// correct ANSI escape sequence when styling is enabled.
func TestColorFunctions(t *testing.T) {
	defer saveState()()
	Enable()

	tests := []struct {
		name     string
		fn       func(string) string
		wantOpen string
	}{
		{"Red", Red, "\x1b[31m"},
		{"Green", Green, "\x1b[32m"},
		{"Yellow", Yellow, "\x1b[33m"},
		{"Blue", Blue, "\x1b[34m"},
		{"Magenta", Magenta, "\x1b[35m"},
		{"Cyan", Cyan, "\x1b[36m"},
		{"White", White, "\x1b[37m"},
		{"Gray", Gray, "\x1b[90m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "hello"
			got := tt.fn(input)
			want := tt.wantOpen + input + "\x1b[0m"
			if got != want {
				t.Errorf("%s(%q) = %q, want %q", tt.name, input, got, want)
			}
		})
	}
}

// TestStyleFunctions verifies that Bold, Dim, and Underline wrap text with the
// correct ANSI codes when styling is enabled.
func TestStyleFunctions(t *testing.T) {
	defer saveState()()
	Enable()

	tests := []struct {
		name     string
		fn       func(string) string
		wantOpen string
	}{
		{"Bold", Bold, "\x1b[1m"},
		{"Dim", Dim, "\x1b[2m"},
		{"Underline", Underline, "\x1b[4m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "text"
			got := tt.fn(input)
			want := tt.wantOpen + input + "\x1b[0m"
			if got != want {
				t.Errorf("%s(%q) = %q, want %q", tt.name, input, got, want)
			}
		})
	}
}

// TestDisabledReturnsPlainText verifies that all styling functions return their
// input unchanged when styling is disabled, simulating NO_COLOR behaviour.
func TestDisabledReturnsPlainText(t *testing.T) {
	defer saveState()()
	Disable()

	fns := []struct {
		name string
		fn   func(string) string
	}{
		{"Red", Red},
		{"Green", Green},
		{"Yellow", Yellow},
		{"Blue", Blue},
		{"Magenta", Magenta},
		{"Cyan", Cyan},
		{"White", White},
		{"Gray", Gray},
		{"Bold", Bold},
		{"Dim", Dim},
		{"Underline", Underline},
	}

	for _, tt := range fns {
		t.Run(tt.name, func(t *testing.T) {
			input := "plain"
			got := tt.fn(input)
			if got != input {
				t.Errorf("disabled %s(%q) = %q, want %q", tt.name, input, got, input)
			}
		})
	}
}

// TestEnabled verifies that Enable and Disable toggle the Enabled() predicate.
func TestEnabled(t *testing.T) {
	defer saveState()()

	Enable()
	if !Enabled() {
		t.Error("Enabled() = false after Enable(), want true")
	}

	Disable()
	if Enabled() {
		t.Error("Enabled() = true after Disable(), want false")
	}

	Enable()
	if !Enabled() {
		t.Error("Enabled() = false after second Enable(), want true")
	}
}

// TestReset verifies that Reset strips all ANSI SGR codes from a string.
func TestReset(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain string unchanged",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "single color code stripped",
			input: "\x1b[31mhello\x1b[0m",
			want:  "hello",
		},
		{
			name:  "bold code stripped",
			input: "\x1b[1mhello\x1b[0m",
			want:  "hello",
		},
		{
			name:  "nested codes stripped",
			input: "\x1b[1m\x1b[31mhello\x1b[0m\x1b[0m",
			want:  "hello",
		},
		{
			name:  "multiple segments stripped",
			input: "\x1b[31mred\x1b[0m and \x1b[32mgreen\x1b[0m",
			want:  "red and green",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "multi-param code stripped",
			input: "\x1b[1;31mhello\x1b[0m",
			want:  "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Reset(tt.input)
			if got != tt.want {
				t.Errorf("Reset(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestResetOnStyledOutput verifies that Reset undoes the output of styling
// functions, regardless of the enabled state.
func TestResetOnStyledOutput(t *testing.T) {
	defer saveState()()
	Enable()

	input := "text"
	styled := Bold(Red(input))
	got := Reset(styled)
	if got != input {
		t.Errorf("Reset(Bold(Red(%q))) = %q, want %q", input, got, input)
	}
}

// TestNestedComposition verifies that nesting style functions produces a string
// that contains both ANSI codes and correctly resolves to plain text after Reset.
func TestNestedComposition(t *testing.T) {
	defer saveState()()
	Enable()

	got := Bold(Red("text"))
	// Must contain both codes.
	if !strings.Contains(got, "\x1b[1m") {
		t.Errorf("Bold(Red()) result missing bold code: %q", got)
	}
	if !strings.Contains(got, "\x1b[31m") {
		t.Errorf("Bold(Red()) result missing red code: %q", got)
	}
	// Stripping must yield the plain text.
	if plain := Reset(got); plain != "text" {
		t.Errorf("Reset(Bold(Red(\"text\"))) = %q, want %q", plain, "text")
	}
}

// TestEmptyString verifies that all functions handle an empty string without
// panicking and return the empty string (styled or plain depending on state).
func TestEmptyString(t *testing.T) {
	defer saveState()()
	Enable()

	fns := []struct {
		name string
		fn   func(string) string
	}{
		{"Red", Red},
		{"Green", Green},
		{"Yellow", Yellow},
		{"Blue", Blue},
		{"Magenta", Magenta},
		{"Cyan", Cyan},
		{"White", White},
		{"Gray", Gray},
		{"Bold", Bold},
		{"Dim", Dim},
		{"Underline", Underline},
	}

	for _, tt := range fns {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn("")
			// When enabled, output must start with an escape and end with reset.
			if !strings.HasPrefix(got, "\x1b[") {
				t.Errorf("%s(\"\") = %q, want ANSI prefix", tt.name, got)
			}
			if !strings.HasSuffix(got, "\x1b[0m") {
				t.Errorf("%s(\"\") = %q, want ANSI reset suffix", tt.name, got)
			}
			// Stripping must yield empty string.
			if plain := Reset(got); plain != "" {
				t.Errorf("Reset(%s(\"\")) = %q, want \"\"", tt.name, plain)
			}
		})
	}
}

// --- Benchmarks ---

// BenchmarkRed measures the cost of applying a color code to a short string.
func BenchmarkRed(b *testing.B) {
	defer saveState()()
	Enable()
	text := "benchmark text"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Red(text)
	}
}

// BenchmarkReset measures the cost of stripping ANSI codes from a styled string.
func BenchmarkReset(b *testing.B) {
	styled := Bold(Red("benchmark text"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Reset(styled)
	}
}

// BenchmarkNestedComposition measures the cost of composing multiple style
// functions (the common case for styled CLI output).
func BenchmarkNestedComposition(b *testing.B) {
	defer saveState()()
	Enable()
	text := "benchmark"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Bold(Red(text))
	}
}
