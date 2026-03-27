package generate

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/terminal"
)

// --- GenerateEditorconfig unit tests ---

func TestGenerateEditorconfig_EmptyLanguages(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".editorconfig")

	var w bytes.Buffer
	err := GenerateEditorconfig(context.Background(), &w, EditorconfigOpts{
		OutputPath: outputPath,
	})
	if err != nil {
		t.Fatalf("GenerateEditorconfig() error = %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	got := string(content)
	if !strings.Contains(got, "root = true") {
		t.Error("Output missing 'root = true'")
	}
	if !strings.Contains(got, "[*]") {
		t.Error("Output missing '[*]' section")
	}
	// No language-specific sections should be present.
	if strings.Contains(got, "[*.go]") {
		t.Error("Output must not contain '[*.go]' when no languages selected")
	}
}

func TestGenerateEditorconfig_SingleLanguageGo(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".editorconfig")

	var w bytes.Buffer
	err := GenerateEditorconfig(context.Background(), &w, EditorconfigOpts{
		OutputPath: outputPath,
		Languages:  []string{"go"},
	})
	if err != nil {
		t.Fatalf("GenerateEditorconfig() error = %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	got := string(content)
	if !strings.Contains(got, "[*.go]") {
		t.Error("Output missing '[*.go]' section")
	}
	if !strings.Contains(got, "indent_style = tab") {
		t.Error("Output missing 'indent_style = tab' for Go")
	}
}

func TestGenerateEditorconfig_MultipleLanguages(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".editorconfig")

	var w bytes.Buffer
	err := GenerateEditorconfig(context.Background(), &w, EditorconfigOpts{
		OutputPath: outputPath,
		Languages:  []string{"go", "python", "yaml"},
	})
	if err != nil {
		t.Fatalf("GenerateEditorconfig() error = %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	got := string(content)
	for _, want := range []string{"[*.go]", "[*.py]", "[*.{yml,yaml}]"} {
		if !strings.Contains(got, want) {
			t.Errorf("Output missing section %q", want)
		}
	}
}

func TestGenerateEditorconfig_OutputStartsWithRootTrue(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".editorconfig")

	var w bytes.Buffer
	err := GenerateEditorconfig(context.Background(), &w, EditorconfigOpts{
		OutputPath: outputPath,
		Languages:  []string{"go"},
	})
	if err != nil {
		t.Fatalf("GenerateEditorconfig() error = %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	got := string(content)
	if !strings.HasPrefix(got, "# EditorConfig") {
		t.Errorf("Output should start with '# EditorConfig', got: %q", got[:min(40, len(got))])
	}
	if !strings.Contains(got, "root = true") {
		t.Error("Output missing 'root = true'")
	}
}

func TestGenerateEditorconfig_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".editorconfig")

	var w bytes.Buffer
	err := GenerateEditorconfig(context.Background(), &w, EditorconfigOpts{
		OutputPath: outputPath,
		Languages:  []string{"go"},
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("GenerateEditorconfig() error = %v", err)
	}

	// File must NOT be written in dry-run mode.
	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Error("Dry-run must not write a file to disk")
	}

	// Content should appear on stdout (w).
	stdout := w.String()
	if !strings.Contains(stdout, "# Dry run mode: would write to") {
		t.Error("Dry-run output missing header line")
	}
	if !strings.Contains(stdout, "root = true") {
		t.Error("Dry-run output missing 'root = true'")
	}
}

func TestGenerateEditorconfig_OverwriteProtection(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".editorconfig")

	// Pre-create file.
	if err := os.WriteFile(outputPath, []byte("existing"), 0644); err != nil {
		t.Fatalf("Failed to create sentinel file: %v", err)
	}

	var w bytes.Buffer
	err := GenerateEditorconfig(context.Background(), &w, EditorconfigOpts{
		OutputPath: outputPath,
	})
	if err == nil {
		t.Error("Expected error when file exists without --force, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected 'already exists' in error message, got: %v", err)
	}

	// With Force, it should succeed.
	w.Reset()
	err = GenerateEditorconfig(context.Background(), &w, EditorconfigOpts{
		OutputPath: outputPath,
		Force:      true,
	})
	if err != nil {
		t.Errorf("Expected success with Force=true, got error: %v", err)
	}
}

func TestGenerateEditorconfig_DefaultOutputPath(t *testing.T) {
	// When OutputPath is empty it should default to "./.editorconfig".
	// We test this via dry-run to avoid writing to the working directory.
	var w bytes.Buffer
	err := GenerateEditorconfig(context.Background(), &w, EditorconfigOpts{
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("GenerateEditorconfig() error = %v", err)
	}

	if !strings.Contains(w.String(), "./.editorconfig") {
		t.Errorf("Dry-run header should show default path './.editorconfig'; got:\n%s", w.String())
	}
}

// --- EditorconfigCommand integration tests ---

func TestEditorconfigCommand_NonInteractive_NoLanguages(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".editorconfig")

	cmd := &EditorconfigCommand{}
	fset := cmd.Flags()
	if err := fset.Parse([]string{
		"--non-interactive",
		"--output", outputPath,
	}); err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	var stdout, stderr bytes.Buffer
	tc := &terminal.Context{Stdout: &stdout, Stderr: &stderr}

	if err := cmd.Run(context.Background(), tc); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}
	got := string(content)

	if !strings.Contains(got, "root = true") {
		t.Error("Output missing 'root = true'")
	}
	if !strings.Contains(got, "[*]") {
		t.Error("Output missing '[*]' section")
	}
	// No language sections should be present.
	if strings.Contains(got, "[*.go]") {
		t.Error("Output must not contain '[*.go]' when no languages specified")
	}
}

func TestEditorconfigCommand_NonInteractive_WithLanguages(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".editorconfig")

	cmd := &EditorconfigCommand{}
	fset := cmd.Flags()
	if err := fset.Parse([]string{
		"--non-interactive",
		"--languages", "go,python",
		"--output", outputPath,
	}); err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	var stdout, stderr bytes.Buffer
	tc := &terminal.Context{Stdout: &stdout, Stderr: &stderr}

	if err := cmd.Run(context.Background(), tc); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}
	got := string(content)

	if !strings.Contains(got, "[*.go]") {
		t.Error("Output missing '[*.go]' section")
	}
	if !strings.Contains(got, "[*.py]") {
		t.Error("Output missing '[*.py]' section")
	}
}

func TestEditorconfigCommand_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".editorconfig")

	cmd := &EditorconfigCommand{}
	fset := cmd.Flags()
	if err := fset.Parse([]string{
		"--non-interactive",
		"--dry-run",
		"--languages", "go",
		"--output", outputPath,
	}); err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	var stdout, stderr bytes.Buffer
	tc := &terminal.Context{Stdout: &stdout, Stderr: &stderr}

	if err := cmd.Run(context.Background(), tc); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Error("Dry-run must not write file to disk")
	}

	out := stdout.String()
	if !strings.Contains(out, "# Dry run mode: would write to") {
		t.Error("Dry-run output missing header")
	}
	if !strings.Contains(out, "root = true") {
		t.Error("Dry-run output missing 'root = true'")
	}
}

func TestEditorconfigCommand_OverwriteProtection(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".editorconfig")

	if err := os.WriteFile(outputPath, []byte("existing"), 0644); err != nil {
		t.Fatalf("Failed to create sentinel file: %v", err)
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name: "existing file without --force fails",
			args: []string{
				"--non-interactive",
				"--output", outputPath,
			},
			wantErr: true,
		},
		{
			name: "existing file with --force succeeds",
			args: []string{
				"--non-interactive",
				"--force",
				"--output", outputPath,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.WriteFile(outputPath, []byte("existing"), 0644)

			cmd := &EditorconfigCommand{}
			fset := cmd.Flags()
			if err := fset.Parse(tt.args); err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			var stdout, stderr bytes.Buffer
			tc := &terminal.Context{Stdout: &stdout, Stderr: &stderr}

			err := cmd.Run(context.Background(), tc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// --- buildEditorSections unit tests ---

func TestBuildEditorSections(t *testing.T) {
	tests := []struct {
		name      string
		languages []string
		wantGlobs []string
		wantCount int
	}{
		{
			name:      "empty languages returns nil",
			languages: nil,
			wantCount: 0,
		},
		{
			name:      "single go",
			languages: []string{"go"},
			wantGlobs: []string{"*.go"},
			wantCount: 1,
		},
		{
			name:      "go and python in canonical order",
			languages: []string{"python", "go"}, // intentionally reversed
			wantGlobs: []string{"*.go", "*.py"}, // expect canonical order: go first
			wantCount: 2,
		},
		// Note: unknown language keys now return an error (tested separately below).
		{
			name:      "only known languages",
			languages: []string{"go", "python"},
			wantGlobs: []string{"*.go", "*.py"},
			wantCount: 2,
		},
		{
			name:      "all supported languages",
			languages: editorConfigLanguageOrder,
			wantCount: len(editorConfigLanguageOrder),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildEditorSections(tt.languages)
			if err != nil {
				t.Fatalf("buildEditorSections() unexpected error: %v", err)
			}
			if len(got) != tt.wantCount {
				t.Errorf("buildEditorSections() count = %d, want %d", len(got), tt.wantCount)
			}
			for i, wantGlob := range tt.wantGlobs {
				if i >= len(got) {
					t.Errorf("Missing section at index %d (want glob %q)", i, wantGlob)
					continue
				}
				if got[i].Glob != wantGlob {
					t.Errorf("Section[%d].Glob = %q, want %q", i, got[i].Glob, wantGlob)
				}
			}
		})
	}
}

func TestBuildEditorSections_UnknownLanguageErrors(t *testing.T) {
	_, err := buildEditorSections([]string{"cobol", "go"})
	if err == nil {
		t.Fatal("buildEditorSections() expected error for unknown language, got nil")
	}
	if !strings.Contains(err.Error(), "cobol") {
		t.Errorf("error %q should mention the unknown language key", err.Error())
	}
}

// --- Benchmark ---

func BenchmarkEditorconfigCommand_DryRun(b *testing.B) {
	var stdout, stderr bytes.Buffer
	tc := &terminal.Context{Stdout: &stdout, Stderr: &stderr}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		stdout.Reset()
		stderr.Reset()

		cmd := &EditorconfigCommand{}
		fset := cmd.Flags()
		_ = fset.Parse([]string{
			"--non-interactive",
			"--dry-run",
			"--languages", "go,javascript,python,rust",
		})

		if err := cmd.Run(context.Background(), tc); err != nil {
			b.Fatalf("Run() error: %v", err)
		}
	}
}

// min returns the smaller of a and b.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
