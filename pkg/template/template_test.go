package template

import (
	"bytes"
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	tests := []struct {
		name     string
		template string
		data     interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "valid template with data",
			template: "claude-md",
			data: map[string]interface{}{
				"Name":          "myapp",
				"Description":   "A CLI tool",
				"Language":      "Go",
				"BuildTool":     "make",
				"TestFramework": "testing",
				"Conventions":   []string{"gofmt", "go vet"},
			},
			want:    "# myapp",
			wantErr: false,
		},
		{
			name:     "template not found",
			template: "nonexistent",
			data:     map[string]interface{}{},
			wantErr:  true,
		},
		{
			name:     "missing fields render as no value",
			template: "claude-md",
			data: map[string]interface{}{
				"Name":          "myapp",
				"Description":   "A CLI tool",
				"Language":      "Go",
				"BuildTool":     "make",
				"TestFramework": "testing",
				// Conventions omitted
			},
			want:    "# myapp",
			wantErr: false,
		},
		{
			name:     "empty conventions list",
			template: "claude-md",
			data: map[string]interface{}{
				"Name":          "myapp",
				"Description":   "A CLI tool",
				"Language":      "Go",
				"BuildTool":     "make",
				"TestFramework": "testing",
				"Conventions":   []string{},
			},
			want:    "# myapp",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Render(tt.template, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("Render() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(got, tt.want) {
				t.Errorf("Render() output does not contain expected substring.\nGot:\n%s\nWant substring:\n%s", got, tt.want)
			}
		})
	}
}

func TestRenderFullOutput(t *testing.T) {
	data := map[string]interface{}{
		"Name":          "cure",
		"Description":   "A Go CLI tool for automating development tasks",
		"Language":      "Go",
		"BuildTool":     "make",
		"TestFramework": "testing",
		"Conventions":   []string{"gofmt", "go vet", "golint"},
	}

	output, err := Render("claude-md", data)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	// Validate key sections are present
	expectedSections := []string{
		"# cure",
		"A Go CLI tool for automating development tasks",
		"## Tech Stack",
		"- **Language**: Go",
		"- **Build tool**: make",
		"- **Test framework**: testing",
		"## Architecture",
		"## Development",
		"## Conventions",
		"- gofmt",
		"- go vet",
		"- golint",
		"## Versioning",
		"## Contributing",
	}

	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("Output missing expected section: %q", section)
		}
	}

	// Validate formatting
	if !strings.HasSuffix(output, "\n") {
		t.Error("Output should end with newline")
	}

	// Check for excessive blank lines
	if strings.Contains(output, "\n\n\n\n") {
		t.Error("Output contains 3+ consecutive blank lines")
	}
}

func TestMustRender(t *testing.T) {
	data := map[string]interface{}{
		"Name":          "test",
		"Description":   "A test",
		"Language":      "Go",
		"BuildTool":     "make",
		"TestFramework": "testing",
	}

	// Should not panic with valid template
	output := MustRender("claude-md", data)
	if !strings.Contains(output, "# test") {
		t.Error("MustRender() output missing expected content")
	}
}

func TestMustRenderPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustRender() did not panic on invalid template")
		}
	}()

	MustRender("nonexistent", map[string]interface{}{})
}

func TestList(t *testing.T) {
	names := List()
	if len(names) == 0 {
		t.Error("List() returned empty list, expected at least one template")
	}

	found := false
	for _, name := range names {
		if name == "claude-md" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("List() did not include 'claude-md' template. Got: %v", names)
	}
}

func TestRenderTo(t *testing.T) {
	data := map[string]interface{}{
		"Name":          "test",
		"Description":   "A test",
		"Language":      "Go",
		"BuildTool":     "make",
		"TestFramework": "testing",
	}

	var buf bytes.Buffer
	n, err := RenderTo(&buf, "claude-md", data)
	if err != nil {
		t.Fatalf("RenderTo() error = %v", err)
	}

	if n == 0 {
		t.Error("RenderTo() wrote 0 bytes")
	}

	output := buf.String()
	if !strings.Contains(output, "# test") {
		t.Error("RenderTo() output missing expected content")
	}
}

func TestRenderToError(t *testing.T) {
	var buf bytes.Buffer
	n, err := RenderTo(&buf, "nonexistent", map[string]interface{}{})
	if err == nil {
		t.Error("RenderTo() expected error for nonexistent template")
	}
	if n != 0 {
		t.Errorf("RenderTo() returned n=%d, want 0 on error", n)
	}
}

func BenchmarkRender(b *testing.B) {
	data := map[string]interface{}{
		"Name":          "myapp",
		"Description":   "A CLI tool",
		"Language":      "Go",
		"BuildTool":     "make",
		"TestFramework": "testing",
		"Conventions":   []string{"gofmt", "go vet"},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Render("claude-md", data)
	}
}

func BenchmarkFormat(b *testing.B) {
	input := "line1   \r\n\n\n\n  line2\t\r\nline3  "
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Format(input)
	}
}
