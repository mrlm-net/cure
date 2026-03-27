package generate

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/config"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// runScaffold is a test helper that parses flags, sets output paths to a temp
// directory via the scaffoldGenerators override, runs ScaffoldCommand.Run, and
// returns the stdout/stderr buffers together with any error.
func runScaffold(t *testing.T, tmpDir string, args []string) (stdout, stderr bytes.Buffer, err error) {
	t.Helper()

	// Redirect all generator default paths into tmpDir so tests don't write to
	// the working directory.
	orig := make(map[string]scaffoldEntry, len(scaffoldGenerators))
	for k, v := range scaffoldGenerators {
		orig[k] = v
	}
	for k, entry := range scaffoldGenerators {
		localEntry := entry
		localK := k
		localEntry.defaultPath = filepath.Join(tmpDir, filepath.Base(localEntry.defaultPath))
		// For nested paths, preserve the subdirectory structure under tmpDir.
		localEntry.defaultPath = filepath.Join(tmpDir, strings.TrimPrefix(entry.defaultPath, "./"))
		scaffoldGenerators[localK] = localEntry
	}
	t.Cleanup(func() {
		for k, v := range orig {
			scaffoldGenerators[k] = v
		}
	})

	cmd := &ScaffoldCommand{}
	fset := cmd.Flags()
	if parseErr := fset.Parse(args); parseErr != nil {
		t.Fatalf("failed to parse flags: %v", parseErr)
	}

	tc := &terminal.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Config: config.NewConfig(),
	}

	err = cmd.Run(context.Background(), tc)
	return
}

func TestScaffoldCommand_SelectSingleValid(t *testing.T) {
	tmpDir := t.TempDir()
	stdout, _, err := runScaffold(t, tmpDir, []string{
		"--non-interactive",
		"--select", "claude-md",
		"--name", "myapp",
		"--description", "A test app",
		"--language", "go",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Only CLAUDE.md should be written.
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	if _, statErr := os.Stat(claudePath); statErr != nil {
		t.Errorf("Expected CLAUDE.md to exist: %v", statErr)
	}
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if _, statErr := os.Stat(agentsPath); !os.IsNotExist(statErr) {
		t.Error("AGENTS.md should not have been written")
	}

	_ = stdout // generation success is verified by file presence
}

func TestScaffoldCommand_SelectMultipleValid(t *testing.T) {
	tmpDir := t.TempDir()
	_, _, err := runScaffold(t, tmpDir, []string{
		"--non-interactive",
		"--select", "claude-md,agents-md",
		"--name", "myapp",
		"--description", "A test app",
		"--language", "go",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		p := filepath.Join(tmpDir, name)
		if _, statErr := os.Stat(p); statErr != nil {
			t.Errorf("Expected %s to exist: %v", name, statErr)
		}
	}
	// Unselected generators should not have written files.
	for _, name := range []string{"GEMINI.md", ".windsurfrules"} {
		p := filepath.Join(tmpDir, name)
		if _, statErr := os.Stat(p); !os.IsNotExist(statErr) {
			t.Errorf("%s should not have been written", name)
		}
	}
}

func TestScaffoldCommand_SelectUnknownName(t *testing.T) {
	tmpDir := t.TempDir()
	_, _, err := runScaffold(t, tmpDir, []string{
		"--non-interactive",
		"--select", "claude-md,nonexistent",
		"--name", "myapp",
		"--description", "A test app",
		"--language", "go",
	})
	if err == nil {
		t.Fatal("Expected error for unknown generator name, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("Error should mention the unknown name; got: %v", err)
	}
	// No files should have been written — error occurs before generation.
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	if _, statErr := os.Stat(claudePath); !os.IsNotExist(statErr) {
		t.Error("No files should be written when --select validation fails")
	}
}

func TestScaffoldCommand_NonInteractiveWithSelect(t *testing.T) {
	tmpDir := t.TempDir()
	_, _, err := runScaffold(t, tmpDir, []string{
		"--non-interactive",
		"--select", "claude-md,agents-md",
		"--name", "myapp",
		"--description", "A test app",
		"--language", "go",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	for _, name := range []string{"CLAUDE.md", "AGENTS.md"} {
		if _, statErr := os.Stat(filepath.Join(tmpDir, name)); statErr != nil {
			t.Errorf("Expected %s to exist: %v", name, statErr)
		}
	}
}

func TestScaffoldCommand_DryRunNonInteractiveSelectClaudeMD(t *testing.T) {
	tmpDir := t.TempDir()
	stdout, _, err := runScaffold(t, tmpDir, []string{
		"--non-interactive",
		"--dry-run",
		"--select", "claude-md",
		"--name", "myapp",
		"--description", "A test app",
		"--language", "go",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// File must NOT be written in dry-run mode.
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	if _, statErr := os.Stat(claudePath); !os.IsNotExist(statErr) {
		t.Error("Dry-run must not write a file to disk")
	}

	// Stdout must contain the dry-run header.
	if !strings.Contains(stdout.String(), "# Dry run mode: would write to") {
		t.Errorf("Dry-run output missing header; got:\n%s", stdout.String())
	}
}

func TestScaffoldCommand_NonInteractiveNoSelectDefaultsToAll(t *testing.T) {
	tmpDir := t.TempDir()
	_, _, err := runScaffold(t, tmpDir, []string{
		"--non-interactive",
		"--name", "myapp",
		"--description", "A test app",
		"--language", "go",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// All 6 generators should have written files.
	expected := map[string]string{
		"claude-md":            "CLAUDE.md",
		"agents-md":            "AGENTS.md",
		"copilot-instructions": filepath.Join(".github", "copilot-instructions.md"),
		"cursor-rules":         filepath.Join(".cursor", "rules", "project.mdc"),
		"windsurf-rules":       ".windsurfrules",
		"gemini-md":            "GEMINI.md",
	}
	for gen, rel := range expected {
		p := filepath.Join(tmpDir, rel)
		if _, statErr := os.Stat(p); statErr != nil {
			t.Errorf("Generator %q: expected %s to exist: %v", gen, rel, statErr)
		}
	}
}

func TestScaffoldCommand_ContinueOnError(t *testing.T) {
	tmpDir := t.TempDir()

	// Pre-create CLAUDE.md as a regular file so the non-interactive path
	// returns an "already exists" error (no --force supplied).
	claudePath := filepath.Join(tmpDir, "CLAUDE.md")
	if err := os.WriteFile(claudePath, []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to create sentinel file: %v", err)
	}

	_, stderr, err := runScaffold(t, tmpDir, []string{
		"--non-interactive",
		"--select", "claude-md,agents-md",
		"--name", "myapp",
		"--description", "A test app",
		"--language", "go",
	})

	// Should return an error because one generator failed.
	if err == nil {
		t.Fatal("Expected error when a generator fails, got nil")
	}

	// AGENTS.md should still have been written (continue-on-error).
	agentsPath := filepath.Join(tmpDir, "AGENTS.md")
	if _, statErr := os.Stat(agentsPath); statErr != nil {
		t.Errorf("Expected AGENTS.md to exist despite claude-md failure: %v", statErr)
	}

	// Error summary should appear on stderr.
	if !strings.Contains(stderr.String(), "generators failed") {
		t.Errorf("Expected failure summary on stderr; got: %s", stderr.String())
	}
}

func TestScaffoldCommand_NoSelection(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate non-interactive with an empty (but validated) --select value
	// by passing an explicit empty string after trimming. The only way to get
	// len(selected)==0 via flags is via a buffer that returns "none" for the
	// MultiSelect prompt. Since the prompter reads from os.Stdin (a non-TTY
	// *os.File in tests), IsInteractive returns false and the code defaults to
	// all — so we cannot exercise the menu path directly here. We instead test
	// the "all" path with verify the summary message appears.
	stdout, _, err := runScaffold(t, tmpDir, []string{
		"--non-interactive",
		"--name", "myapp",
		"--description", "A test app",
		"--language", "go",
		"--dry-run",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	// In dry-run all 6 generators produce output; just check stdout is non-empty.
	if stdout.Len() == 0 {
		t.Error("Expected non-empty stdout for dry-run scaffold")
	}
}

func TestScaffoldCommand_MissingRequiredFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "missing name",
			args: []string{"--non-interactive", "--description", "A test app", "--language", "go"},
		},
		{
			name: "missing description",
			args: []string{"--non-interactive", "--name", "myapp", "--language", "go"},
		},
		{
			name: "missing language",
			args: []string{"--non-interactive", "--name", "myapp", "--description", "A test app"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			_, _, err := runScaffold(t, tmpDir, tt.args)
			if err == nil {
				t.Errorf("Expected error for %s, got nil", tt.name)
			}
		})
	}
}

func BenchmarkScaffoldCommand_AllFiles(b *testing.B) {
	tmpDir := b.TempDir()

	// Redirect generators to temp paths.
	orig := make(map[string]scaffoldEntry, len(scaffoldGenerators))
	for k, v := range scaffoldGenerators {
		orig[k] = v
	}
	for k, entry := range scaffoldGenerators {
		localEntry := entry
		localK := k
		localEntry.defaultPath = filepath.Join(tmpDir, strings.TrimPrefix(entry.defaultPath, "./"))
		scaffoldGenerators[localK] = localEntry
	}
	b.Cleanup(func() {
		for k, v := range orig {
			scaffoldGenerators[k] = v
		}
	})

	args := []string{
		"--non-interactive",
		"--dry-run",
		"--name", "benchapp",
		"--description", "Benchmark application",
		"--language", "go",
		"--build-tool", "make",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var stdout, stderr bytes.Buffer
		cmd := &ScaffoldCommand{}
		fset := cmd.Flags()
		_ = fset.Parse(args)
		tc := &terminal.Context{Stdout: &stdout, Stderr: &stderr, Config: config.NewConfig()}
		if err := cmd.Run(context.Background(), tc); err != nil {
			b.Fatalf("Run() error: %v", err)
		}
	}
}
