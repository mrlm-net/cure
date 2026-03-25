package prompt

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// makePrompter returns a Prompter backed by the given input lines and a buffer
// for capturing output.
func makePrompter(input string) (*Prompter, *bytes.Buffer) {
	out := &bytes.Buffer{}
	p := NewPrompter(out, strings.NewReader(input))
	return p, out
}

// TestRequired covers the Required method: default handling, non-empty input,
// repeated prompt on empty, and EOF.
func TestRequired(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		prompt     string
		defaultVal string
		wantVal    string
		wantErr    bool
	}{
		{
			name:       "returns default on empty enter",
			input:      "\n",
			prompt:     "Name",
			defaultVal: "world",
			wantVal:    "world",
		},
		{
			name:       "returns explicit input",
			input:      "alice\n",
			prompt:     "Name",
			defaultVal: "world",
			wantVal:    "alice",
		},
		{
			name:       "no default, user provides value",
			input:      "bob\n",
			prompt:     "Name",
			defaultVal: "",
			wantVal:    "bob",
		},
		{
			name:       "trims whitespace around input",
			input:      "  carol  \n",
			prompt:     "Name",
			defaultVal: "",
			wantVal:    "carol",
		},
		{
			name:       "retries on empty then accepts value",
			input:      "\n\ndan\n",
			prompt:     "Name",
			defaultVal: "",
			wantVal:    "dan",
		},
		{
			name:    "EOF without default returns error",
			input:   "",
			prompt:  "Name",
			wantErr: true,
		},
		{
			name:       "EOF with default returns error",
			input:      "",
			prompt:     "Name",
			defaultVal: "fallback",
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, _ := makePrompter(tc.input)
			got, err := p.Required(tc.prompt, tc.defaultVal)
			if tc.wantErr {
				if err == nil {
					t.Errorf("want error, got nil (value=%q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantVal {
				t.Errorf("got %q, want %q", got, tc.wantVal)
			}
		})
	}
}

// TestRequired_OutputContainsDefault verifies the default value appears in output.
func TestRequired_OutputContainsDefault(t *testing.T) {
	t.Parallel()

	p, out := makePrompter("myvalue\n")
	_, _ = p.Required("Label", "default-val")

	if !strings.Contains(out.String(), "[default-val]") {
		t.Errorf("output %q does not contain [default-val]", out.String())
	}
}

// TestRequired_NoDefaultInOutput verifies no brackets appear when default is empty.
func TestRequired_NoDefaultInOutput(t *testing.T) {
	t.Parallel()

	p, out := makePrompter("myvalue\n")
	_, _ = p.Required("Label", "")

	if strings.Contains(out.String(), "[") {
		t.Errorf("output %q contains unexpected brackets", out.String())
	}
}

// TestRequired_ErrorMessageOnEmpty verifies the re-prompt error message.
func TestRequired_ErrorMessageOnEmpty(t *testing.T) {
	t.Parallel()

	p, out := makePrompter("\nanswer\n")
	_, _ = p.Required("Label", "")

	if !strings.Contains(out.String(), "required") {
		t.Errorf("output %q missing 'required' in error message", out.String())
	}
}

// TestOptional covers the Optional method: default on empty, explicit input,
// whitespace trim, and EOF returning default.
func TestOptional(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		prompt     string
		defaultVal string
		wantVal    string
		wantErr    bool
	}{
		{
			name:       "returns default on empty enter",
			input:      "\n",
			prompt:     "Description",
			defaultVal: "none",
			wantVal:    "none",
		},
		{
			name:       "returns typed value",
			input:      "my description\n",
			prompt:     "Description",
			defaultVal: "none",
			wantVal:    "my description",
		},
		{
			name:       "trims whitespace",
			input:      "  trimmed  \n",
			prompt:     "Description",
			defaultVal: "",
			wantVal:    "trimmed",
		},
		{
			name:       "EOF returns default not error",
			input:      "",
			prompt:     "Description",
			defaultVal: "fallback",
			wantVal:    "fallback",
		},
		{
			name:       "EOF with empty default returns empty string",
			input:      "",
			prompt:     "Description",
			defaultVal: "",
			wantVal:    "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, _ := makePrompter(tc.input)
			got, err := p.Optional(tc.prompt, tc.defaultVal)
			if tc.wantErr {
				if err == nil {
					t.Errorf("want error, got nil (value=%q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantVal {
				t.Errorf("got %q, want %q", got, tc.wantVal)
			}
		})
	}
}

// TestConfirm covers the Confirm method: y/n variants, case-insensitivity,
// re-prompt on invalid, and EOF.
func TestConfirm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		prompt  string
		wantVal bool
		wantErr bool
	}{
		{name: "y returns true", input: "y\n", prompt: "Continue?", wantVal: true},
		{name: "yes returns true", input: "yes\n", prompt: "Continue?", wantVal: true},
		{name: "Y returns true", input: "Y\n", prompt: "Continue?", wantVal: true},
		{name: "YES returns true", input: "YES\n", prompt: "Continue?", wantVal: true},
		{name: "n returns false", input: "n\n", prompt: "Continue?", wantVal: false},
		{name: "no returns false", input: "no\n", prompt: "Continue?", wantVal: false},
		{name: "N returns false", input: "N\n", prompt: "Continue?", wantVal: false},
		{name: "NO returns false", input: "NO\n", prompt: "Continue?", wantVal: false},
		{
			name:    "invalid then y returns true",
			input:   "maybe\ny\n",
			prompt:  "Continue?",
			wantVal: true,
		},
		{
			name:    "invalid then no returns false",
			input:   "aye\nno\n",
			prompt:  "Continue?",
			wantVal: false,
		},
		{
			name:    "EOF returns error",
			input:   "",
			prompt:  "Continue?",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, _ := makePrompter(tc.input)
			got, err := p.Confirm(tc.prompt)
			if tc.wantErr {
				if err == nil {
					t.Errorf("want error, got nil (value=%v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantVal {
				t.Errorf("got %v, want %v", got, tc.wantVal)
			}
		})
	}
}

// TestConfirm_OutputContainsYN ensures the prompt shows (y/n).
func TestConfirm_OutputContainsYN(t *testing.T) {
	t.Parallel()

	p, out := makePrompter("y\n")
	_, _ = p.Confirm("Delete file?")

	if !strings.Contains(out.String(), "(y/n)") {
		t.Errorf("output %q does not contain (y/n)", out.String())
	}
}

var threeOptions = []Option{
	{Label: "Alpha", Value: "a"},
	{Label: "Beta", Value: "b", Description: "second option"},
	{Label: "Gamma", Value: "g"},
}

// TestSingleSelect covers valid picks, out-of-range, non-numeric, and EOF.
func TestSingleSelect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		options []Option
		wantVal string
		wantErr bool
	}{
		{
			name:    "pick first",
			input:   "1\n",
			options: threeOptions,
			wantVal: "a",
		},
		{
			name:    "pick last",
			input:   "3\n",
			options: threeOptions,
			wantVal: "g",
		},
		{
			name:    "invalid then valid",
			input:   "0\n5\n2\n",
			options: threeOptions,
			wantVal: "b",
		},
		{
			name:    "non-numeric then valid",
			input:   "abc\n1\n",
			options: threeOptions,
			wantVal: "a",
		},
		{
			name:    "EOF returns error",
			input:   "",
			options: threeOptions,
			wantErr: true,
		},
		{
			name:    "empty options returns error immediately",
			input:   "1\n",
			options: []Option{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, _ := makePrompter(tc.input)
			got, err := p.SingleSelect("Pick one", tc.options)
			if tc.wantErr {
				if err == nil {
					t.Errorf("want error, got nil (value=%+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Value != tc.wantVal {
				t.Errorf("got %q, want %q", got.Value, tc.wantVal)
			}
		})
	}
}

// TestSingleSelect_ListInOutput verifies each option label appears in the output.
func TestSingleSelect_ListInOutput(t *testing.T) {
	t.Parallel()

	p, out := makePrompter("1\n")
	_, _ = p.SingleSelect("Pick one", threeOptions)

	output := out.String()
	for _, opt := range threeOptions {
		if !strings.Contains(output, opt.Label) {
			t.Errorf("output missing label %q: %s", opt.Label, output)
		}
	}
}

// TestSingleSelect_DescriptionInOutput verifies description appears when set.
func TestSingleSelect_DescriptionInOutput(t *testing.T) {
	t.Parallel()

	p, out := makePrompter("1\n")
	_, _ = p.SingleSelect("Pick one", threeOptions)

	if !strings.Contains(out.String(), "second option") {
		t.Errorf("output missing description: %s", out.String())
	}
}

// TestMultiSelect covers "all", "none", comma lists, duplicates, invalid, and EOF.
func TestMultiSelect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		options   []Option
		wantCount int
		wantVals  []string
		wantErr   bool
	}{
		{
			name:      "all returns every option",
			input:     "all\n",
			options:   threeOptions,
			wantCount: 3,
			wantVals:  []string{"a", "b", "g"},
		},
		{
			name:      "ALL case-insensitive",
			input:     "ALL\n",
			options:   threeOptions,
			wantCount: 3,
			wantVals:  []string{"a", "b", "g"},
		},
		{
			name:      "none returns empty slice",
			input:     "none\n",
			options:   threeOptions,
			wantCount: 0,
			wantVals:  []string{},
		},
		{
			name:      "NONE case-insensitive",
			input:     "NONE\n",
			options:   threeOptions,
			wantCount: 0,
			wantVals:  []string{},
		},
		{
			name:      "single number",
			input:     "2\n",
			options:   threeOptions,
			wantCount: 1,
			wantVals:  []string{"b"},
		},
		{
			name:      "comma-separated numbers",
			input:     "1,3\n",
			options:   threeOptions,
			wantCount: 2,
			wantVals:  []string{"a", "g"},
		},
		{
			name:      "duplicates deduplicated",
			input:     "1,1,2\n",
			options:   threeOptions,
			wantCount: 2,
			wantVals:  []string{"a", "b"},
		},
		{
			name:      "order follows original options not entry order",
			input:     "3,1\n",
			options:   threeOptions,
			wantCount: 2,
			wantVals:  []string{"a", "g"},
		},
		{
			name:      "spaces around numbers trimmed",
			input:     " 1 , 2 \n",
			options:   threeOptions,
			wantCount: 2,
			wantVals:  []string{"a", "b"},
		},
		{
			name:      "invalid number re-prompts then accepts all",
			input:     "0\nall\n",
			options:   threeOptions,
			wantCount: 3,
			wantVals:  []string{"a", "b", "g"},
		},
		{
			name:      "out-of-range re-prompts then accepts none",
			input:     "99\nnone\n",
			options:   threeOptions,
			wantCount: 0,
			wantVals:  []string{},
		},
		{
			name:    "EOF returns error",
			input:   "",
			options: threeOptions,
			wantErr: true,
		},
		{
			name:    "empty options returns error",
			input:   "1\n",
			options: []Option{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p, _ := makePrompter(tc.input)
			got, err := p.MultiSelect("Select", tc.options)
			if tc.wantErr {
				if err == nil {
					t.Errorf("want error, got nil (value=%+v)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != tc.wantCount {
				t.Errorf("got %d items, want %d: %+v", len(got), tc.wantCount, got)
			}
			gotVals := make([]string, len(got))
			for i, o := range got {
				gotVals[i] = o.Value
			}
			for i, wv := range tc.wantVals {
				if i >= len(gotVals) || gotVals[i] != wv {
					t.Errorf("position %d: got %q, want %q (all: %v)", i, gotVals, wv, gotVals)
					break
				}
			}
		})
	}
}

// TestIsInteractive checks that non-file readers return false.
func TestIsInteractive(t *testing.T) {
	t.Parallel()

	t.Run("strings.Reader is not interactive", func(t *testing.T) {
		t.Parallel()
		r := strings.NewReader("hello")
		if IsInteractive(r) {
			t.Error("expected false for strings.Reader")
		}
	})

	t.Run("bytes.Buffer is not interactive", func(t *testing.T) {
		t.Parallel()
		r := &bytes.Buffer{}
		if IsInteractive(r) {
			t.Error("expected false for bytes.Buffer")
		}
	})

	t.Run("io.Reader implementation is not interactive", func(t *testing.T) {
		t.Parallel()
		r := io.NopCloser(strings.NewReader(""))
		if IsInteractive(r) {
			t.Error("expected false for io.NopCloser")
		}
	})

	// os.Stdin is detected correctly only when running in a real TTY, which
	// CI environments typically do not have. We test the file path without
	// asserting interactive = true, since the test runner pipes stdin.
	t.Run("os.Stdin does not panic", func(t *testing.T) {
		t.Parallel()
		// Just verifying no panic — actual result depends on the environment.
		_ = IsInteractive(os.Stdin)
	})

	t.Run("regular file is not interactive", func(t *testing.T) {
		t.Parallel()
		f, err := os.CreateTemp(t.TempDir(), "prompt-test-*")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { f.Close() })

		if IsInteractive(f) {
			t.Error("expected false for regular file")
		}
	})
}

// --- Benchmarks ---

// BenchmarkRequired measures the hot path for a single Required call with
// a pre-filled default value.
func BenchmarkRequired(b *testing.B) {
	out := io.Discard
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := NewPrompter(out, strings.NewReader("\n"))
		_, _ = p.Required("Name", "default")
	}
}

// BenchmarkOptional measures the hot path for a single Optional call.
func BenchmarkOptional(b *testing.B) {
	out := io.Discard
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := NewPrompter(out, strings.NewReader("\n"))
		_, _ = p.Optional("Name", "default")
	}
}

// BenchmarkConfirm measures the hot path for a single Confirm call.
func BenchmarkConfirm(b *testing.B) {
	out := io.Discard
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := NewPrompter(out, strings.NewReader("y\n"))
		_, _ = p.Confirm("Continue?")
	}
}

// BenchmarkSingleSelect measures the hot path for a single SingleSelect call.
func BenchmarkSingleSelect(b *testing.B) {
	out := io.Discard
	opts := []Option{
		{Label: "Alpha", Value: "a"},
		{Label: "Beta", Value: "b"},
		{Label: "Gamma", Value: "g"},
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := NewPrompter(out, strings.NewReader("2\n"))
		_, _ = p.SingleSelect("Pick", opts)
	}
}

// BenchmarkMultiSelect measures the hot path for a single MultiSelect call
// using the "all" keyword.
func BenchmarkMultiSelect(b *testing.B) {
	out := io.Discard
	opts := []Option{
		{Label: "Alpha", Value: "a"},
		{Label: "Beta", Value: "b"},
		{Label: "Gamma", Value: "g"},
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := NewPrompter(out, strings.NewReader("all\n"))
		_, _ = p.MultiSelect("Select", opts)
	}
}

// BenchmarkMultiSelectCSV measures the comma-separated number parsing path.
func BenchmarkMultiSelectCSV(b *testing.B) {
	out := io.Discard
	opts := []Option{
		{Label: "Alpha", Value: "a"},
		{Label: "Beta", Value: "b"},
		{Label: "Gamma", Value: "g"},
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := NewPrompter(out, strings.NewReader("1,3\n"))
		_, _ = p.MultiSelect("Select", opts)
	}
}

// BenchmarkIsInteractive measures the type-assertion path for a strings.Reader.
func BenchmarkIsInteractive(b *testing.B) {
	r := strings.NewReader("")
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = IsInteractive(r)
	}
}
