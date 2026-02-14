package terminal

import (
	"errors"
	"fmt"
	"testing"
)

func TestCommandNotFoundError(t *testing.T) {
	tests := []struct {
		name        string
		err         *CommandNotFoundError
		wantMessage string
	}{
		{
			name:        "no suggestions",
			err:         &CommandNotFoundError{Name: "unknown"},
			wantMessage: "unknown command: unknown",
		},
		{
			name:        "one suggestion",
			err:         &CommandNotFoundError{Name: "verion", Suggestions: []string{"version"}},
			wantMessage: "unknown command: verion. Did you mean: version?",
		},
		{
			name:        "multiple suggestions",
			err:         &CommandNotFoundError{Name: "con", Suggestions: []string{"config", "confirm", "connect"}},
			wantMessage: "unknown command: con. Did you mean: config, confirm, connect?",
		},
		{
			name:        "empty name",
			err:         &CommandNotFoundError{Name: ""},
			wantMessage: "unknown command: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.wantMessage {
				t.Errorf("Error() = %q, want %q", got, tt.wantMessage)
			}
		})
	}
}

func TestCommandError(t *testing.T) {
	inner := errors.New("disk full")
	err := &CommandError{Command: "generate", Err: inner}

	want := "command generate: disk full"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	if !errors.Is(err, inner) {
		t.Error("errors.Is should find the wrapped error")
	}

	var target *CommandError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match *CommandError")
	}
	if target.Command != "generate" {
		t.Errorf("Command = %q, want %q", target.Command, "generate")
	}
}

func TestCommandError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("wrapped: %w", errors.New("root cause"))
	err := &CommandError{Command: "test", Err: inner}

	if err.Unwrap() != inner {
		t.Error("Unwrap() should return the inner error")
	}

	rootCause := errors.New("root cause")
	err2 := &CommandError{Command: "test", Err: fmt.Errorf("wrapped: %w", rootCause)}
	if !errors.Is(err2, rootCause) {
		t.Error("errors.Is should traverse the Unwrap chain")
	}
}

func TestFlagParseError(t *testing.T) {
	inner := errors.New("invalid value")
	err := &FlagParseError{Command: "generate", Err: inner}

	want := "flag parsing failed for generate: invalid value"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	if !errors.Is(err, inner) {
		t.Error("errors.Is should find the wrapped error")
	}

	var target *FlagParseError
	if !errors.As(err, &target) {
		t.Fatal("errors.As should match *FlagParseError")
	}
	if target.Command != "generate" {
		t.Errorf("Command = %q, want %q", target.Command, "generate")
	}
}

func TestFlagParseError_Unwrap(t *testing.T) {
	inner := errors.New("bad flag")
	err := &FlagParseError{Command: "test", Err: inner}

	if err.Unwrap() != inner {
		t.Error("Unwrap() should return the inner error")
	}
}

func TestNoCommandError(t *testing.T) {
	err := &NoCommandError{}
	want := "no command specified"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"kitten", "sitting", 3},
		{"saturday", "sunday", 3},
		{"ab", "ba", 2},
		{"a", "b", 1},
		{"version", "verion", 1},
		{"config", "confirg", 1},
		{"help", "help", 0},
		{"generate", "generaet", 2},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.a, tt.b), func(t *testing.T) {
			got := levenshtein(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
			// Verify symmetry
			got2 := levenshtein(tt.b, tt.a)
			if got2 != tt.want {
				t.Errorf("levenshtein(%q, %q) = %d, want %d (symmetry)", tt.b, tt.a, got2, tt.want)
			}
		})
	}
}

func TestFindSimilar(t *testing.T) {
	root := &node{children: make(map[byte]*node)}
	commands := []string{"version", "verify", "help", "generate", "config", "confirm"}
	for _, name := range commands {
		root.insert(name, &mockCommand{name: name})
	}

	tests := []struct {
		name       string
		input      string
		maxResults int
		wantNames  []string
	}{
		{
			name:       "typo in version",
			input:      "verion",
			maxResults: 3,
			wantNames:  []string{"version", "verify"},
		},
		{
			name:       "close to verify and version",
			input:      "versio",
			maxResults: 3,
			wantNames:  []string{"version"},
		},
		{
			name:       "empty input",
			input:      "",
			maxResults: 3,
			wantNames:  nil,
		},
		{
			name:       "zero max results",
			input:      "version",
			maxResults: 0,
			wantNames:  nil,
		},
		{
			name:       "completely different",
			input:      "zzzzzzzzzzz",
			maxResults: 3,
			wantNames:  nil,
		},
		{
			name:       "exact match included",
			input:      "help",
			maxResults: 3,
			wantNames:  []string{"help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := root.findSimilar(tt.input, tt.maxResults)
			if tt.wantNames == nil {
				if got != nil {
					names := make([]string, len(got))
					for i, c := range got {
						names[i] = c.Name()
					}
					t.Errorf("findSimilar(%q, %d) = %v, want nil", tt.input, tt.maxResults, names)
				}
				return
			}
			if len(got) != len(tt.wantNames) {
				names := make([]string, len(got))
				for i, c := range got {
					names[i] = c.Name()
				}
				t.Errorf("findSimilar(%q, %d) len = %d (%v), want %d (%v)",
					tt.input, tt.maxResults, len(got), names, len(tt.wantNames), tt.wantNames)
				return
			}
			for i, wantName := range tt.wantNames {
				if got[i].Name() != wantName {
					t.Errorf("findSimilar[%d] = %q, want %q", i, got[i].Name(), wantName)
				}
			}
		})
	}
}

func BenchmarkLevenshtein(b *testing.B) {
	benchmarks := []struct {
		name string
		a, s string
	}{
		{"short", "help", "halp"},
		{"medium", "generate", "generaet"},
		{"long", "configuration", "configuraiton"},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				levenshtein(bm.a, bm.s)
			}
		})
	}
}

func BenchmarkFindSimilar(b *testing.B) {
	root := &node{children: make(map[byte]*node)}
	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("command-%d", i)
		root.insert(name, &mockCommand{name: name})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root.findSimilar("comand-25", 3)
	}
}
