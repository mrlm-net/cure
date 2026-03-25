package template

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/config"
)

// resetRegistry clears the registry and config so each test starts clean.
func resetRegistry() {
	mu.Lock()
	registry = nil
	globalConfig = nil
	mu.Unlock()
}

func TestRender(t *testing.T) {
	resetRegistry()

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
	resetRegistry()

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
	resetRegistry()

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
	resetRegistry()

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustRender() did not panic on invalid template")
		}
	}()

	MustRender("nonexistent", map[string]interface{}{})
}

func TestList(t *testing.T) {
	resetRegistry()

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
	resetRegistry()

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
	resetRegistry()

	var buf bytes.Buffer
	n, err := RenderTo(&buf, "nonexistent", map[string]interface{}{})
	if err == nil {
		t.Error("RenderTo() expected error for nonexistent template")
	}
	if n != 0 {
		t.Errorf("RenderTo() returned n=%d, want 0 on error", n)
	}
}

// TestSetConfigNil verifies nil config is valid (no custom dirs, no panic).
func TestSetConfigNil(t *testing.T) {
	resetRegistry()

	SetConfig(nil)

	names := List()
	if len(names) == 0 {
		t.Error("List() should return embedded templates even with nil config")
	}
}

// TestSetConfigForcesRebuild verifies SetConfig invalidates the cached registry.
func TestSetConfigForcesRebuild(t *testing.T) {
	resetRegistry()

	// Prime the registry
	_, err := getRegistry()
	if err != nil {
		t.Fatalf("getRegistry() error = %v", err)
	}

	// SetConfig should nil out the registry
	SetConfig(nil)

	mu.Lock()
	isNil := registry == nil
	mu.Unlock()

	if !isNil {
		t.Error("SetConfig() should have set registry to nil to force rebuild")
	}

	// Registry should be rebuilt on next use
	reg, err := getRegistry()
	if err != nil {
		t.Fatalf("getRegistry() after SetConfig error = %v", err)
	}
	if reg == nil {
		t.Error("getRegistry() returned nil after rebuild")
	}
}

// TestCustomTemplateOverridesEmbedded verifies a project-local template
// overrides the embedded template of the same name.
func TestCustomTemplateOverridesEmbedded(t *testing.T) {
	// Create a fake .cure/templates directory relative to the test's working dir.
	// We change to a temp dir so the project-local path ".cure/templates" is isolated.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		resetRegistry()
	})

	// Create .cure/templates/claude-md.tmpl with custom content
	templateDir := filepath.Join(tmpDir, ".cure", "templates")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	customContent := "CUSTOM_OVERRIDE_{{.Name}}"
	if err := os.WriteFile(filepath.Join(templateDir, "claude-md.tmpl"), []byte(customContent), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	resetRegistry()

	output, err := Render("claude-md", map[string]interface{}{"Name": "myapp"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(output, "CUSTOM_OVERRIDE_myapp") {
		t.Errorf("expected custom template override, got: %s", output)
	}
}

// TestCustomTemplateTplExtension verifies .tpl extension is also recognized.
func TestCustomTemplateTplExtension(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		resetRegistry()
	})

	templateDir := filepath.Join(tmpDir, ".cure", "templates")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Use .tpl extension
	customContent := "TPL_TEMPLATE_{{.Value}}"
	if err := os.WriteFile(filepath.Join(templateDir, "my-custom.tpl"), []byte(customContent), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	resetRegistry()

	output, err := Render("my-custom", map[string]interface{}{"Value": "hello"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	if !strings.Contains(output, "TPL_TEMPLATE_hello") {
		t.Errorf("expected .tpl template to be loaded, got: %s", output)
	}
}

// TestNewTemplateFromCustomDir verifies a new template added in a custom dir
// is available via List and Render.
func TestNewTemplateFromCustomDir(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		resetRegistry()
	})

	templateDir := filepath.Join(tmpDir, ".cure", "templates")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	customContent := "HELLO_{{.Name}}"
	if err := os.WriteFile(filepath.Join(templateDir, "greeting.tmpl"), []byte(customContent), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	resetRegistry()

	// Verify List includes the new template
	names := List()
	found := false
	for _, n := range names {
		if n == "greeting" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("List() did not include custom 'greeting' template. Got: %v", names)
	}

	// Verify Render works for the new template
	output, err := Render("greeting", map[string]interface{}{"Name": "world"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(output, "HELLO_world") {
		t.Errorf("expected HELLO_world, got: %s", output)
	}
}

// TestMissingDirectorySkippedSilently verifies no error when custom dirs don't exist.
func TestMissingDirectorySkippedSilently(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		resetRegistry()
	})

	// .cure/templates does NOT exist — should be silently skipped
	resetRegistry()

	names := List()
	if len(names) == 0 {
		t.Error("List() should return embedded templates when custom dirs don't exist")
	}
}

// TestConfigDirsIntegration verifies template.dirs from config adds extra search paths.
func TestConfigDirsIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a config-specified extra directory
	extraDir := filepath.Join(tmpDir, "extra-templates")
	if err := os.MkdirAll(extraDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	customContent := "FROM_CONFIG_DIR_{{.Val}}"
	if err := os.WriteFile(filepath.Join(extraDir, "config-tmpl.tmpl"), []byte(customContent), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Move to a temp working dir so .cure/templates doesn't exist
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	workDir := t.TempDir()
	if err := os.Chdir(workDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		resetRegistry()
	})

	cfg := config.NewConfig(config.ConfigObject{
		"template.dirs": []interface{}{extraDir},
	})
	SetConfig(cfg)

	output, err := Render("config-tmpl", map[string]interface{}{"Val": "ok"})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(output, "FROM_CONFIG_DIR_ok") {
		t.Errorf("expected FROM_CONFIG_DIR_ok, got: %s", output)
	}
}

// TestTemplateSyntaxErrorWarns verifies that a syntactically invalid custom
// template does not abort the load — it is skipped with a stderr warning.
func TestTemplateSyntaxErrorWarns(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		resetRegistry()
	})

	templateDir := filepath.Join(tmpDir, ".cure", "templates")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Write an invalid template (unclosed action)
	invalidContent := "{{.Name"
	if err := os.WriteFile(filepath.Join(templateDir, "bad.tmpl"), []byte(invalidContent), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Also write a valid template to verify load continues
	validContent := "VALID_{{.Name}}"
	if err := os.WriteFile(filepath.Join(templateDir, "good.tmpl"), []byte(validContent), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	resetRegistry()

	// The "bad" template should be skipped; "good" should be available
	output, err := Render("good", map[string]interface{}{"Name": "test"})
	if err != nil {
		t.Fatalf("Render('good') error = %v — load should have continued past bad template", err)
	}
	if !strings.Contains(output, "VALID_test") {
		t.Errorf("expected VALID_test, got: %s", output)
	}

	// The bad template should NOT be available (it was skipped)
	_, err = Render("bad", map[string]interface{}{"Name": "test"})
	if err == nil {
		t.Error("Render('bad') should fail because the template had a syntax error")
	}
}

// TestEmbeddedTemplateFallback verifies that without any custom dirs,
// the embedded templates are always available.
func TestEmbeddedTemplateFallback(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		resetRegistry()
	})

	resetRegistry()

	// claude-md is always embedded — must be available without any custom dirs
	output, err := Render("claude-md", map[string]interface{}{
		"Name":          "fallback-test",
		"Description":   "test",
		"Language":      "Go",
		"BuildTool":     "make",
		"TestFramework": "testing",
	})
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	if !strings.Contains(output, "# fallback-test") {
		t.Errorf("expected embedded fallback template, got: %s", output)
	}
}

// TestNonTemplateFilesIgnored verifies files without .tmpl/.tpl extension
// in custom dirs are not loaded.
func TestNonTemplateFilesIgnored(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		resetRegistry()
	})

	templateDir := filepath.Join(tmpDir, ".cure", "templates")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Write a file with wrong extension
	if err := os.WriteFile(filepath.Join(templateDir, "ignored.txt"), []byte("should not load"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	resetRegistry()

	_, err = Render("ignored", map[string]interface{}{})
	if err == nil {
		t.Error("Render('ignored') should fail — .txt files should be ignored")
	}
}

func BenchmarkRender(b *testing.B) {
	resetRegistry()

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

func BenchmarkGetRegistry(b *testing.B) {
	resetRegistry()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = getRegistry()
	}
}
