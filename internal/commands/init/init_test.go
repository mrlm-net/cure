package initcmd_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	initcmd "github.com/mrlm-net/cure/internal/commands/init"
	"github.com/mrlm-net/cure/pkg/config"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// runInit is a test helper that parses flags, wires up a terminal.Context with
// isolated output buffers, and invokes InitCommand.Run. All file writes are
// directed at tmpDir by passing --output flags is not possible for all
// generators, so instead we chdir into tmpDir before running and restore
// afterwards. This ensures relative default output paths (e.g. "./CLAUDE.md")
// resolve inside tmpDir.
func runInit(t *testing.T, tmpDir string, args []string) (stdout, stderr bytes.Buffer, err error) {
	t.Helper()

	// Change directory to tmpDir so relative paths go there.
	orig, cwdErr := os.Getwd()
	if cwdErr != nil {
		t.Fatalf("getwd: %v", cwdErr)
	}
	if chdirErr := os.Chdir(tmpDir); chdirErr != nil {
		t.Fatalf("chdir to tmpDir: %v", chdirErr)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	cmd := initcmd.NewInitCommand()
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

// TestInitCommand_NonInteractiveAllDefaults runs with --non-interactive,
// --name, --language and expects all components (6 AI files + devcontainer +
// ci + editorconfig + gitignore) to be generated.
func TestInitCommand_NonInteractiveAllDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	stdout, _, err := runInit(t, tmpDir, []string{
		"--non-interactive",
		"--name", "myapp",
		"--language", "go",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Verify core AI files.
	for _, rel := range []string{
		"CLAUDE.md",
		"AGENTS.md",
		"GEMINI.md",
		".windsurfrules",
		filepath.Join(".github", "copilot-instructions.md"),
		filepath.Join(".cursor", "rules", "project.mdc"),
	} {
		p := filepath.Join(tmpDir, rel)
		if _, statErr := os.Stat(p); statErr != nil {
			t.Errorf("expected %s to exist: %v", rel, statErr)
		}
	}

	// Verify devcontainer.
	dcPath := filepath.Join(tmpDir, ".devcontainer", "devcontainer.json")
	if _, statErr := os.Stat(dcPath); statErr != nil {
		t.Errorf("expected .devcontainer/devcontainer.json to exist: %v", statErr)
	}

	// Verify CI workflow.
	ciPath := filepath.Join(tmpDir, ".github", "workflows", "ci.yml")
	if _, statErr := os.Stat(ciPath); statErr != nil {
		t.Errorf("expected .github/workflows/ci.yml to exist: %v", statErr)
	}

	// Verify editorconfig and gitignore.
	for _, rel := range []string{".editorconfig", ".gitignore"} {
		p := filepath.Join(tmpDir, rel)
		if _, statErr := os.Stat(p); statErr != nil {
			t.Errorf("expected %s to exist: %v", rel, statErr)
		}
	}

	// Summary must be printed.
	if !strings.Contains(stdout.String(), "cure init summary:") {
		t.Errorf("expected summary in stdout; got:\n%s", stdout.String())
	}
}

// TestInitCommand_MissingName expects an error when --name is absent in
// non-interactive mode.
func TestInitCommand_MissingName(t *testing.T) {
	tmpDir := t.TempDir()
	_, _, err := runInit(t, tmpDir, []string{
		"--non-interactive",
		"--language", "go",
	})
	if err == nil {
		t.Fatal("expected error for missing --name, got nil")
	}
	if !strings.Contains(err.Error(), "--name") {
		t.Errorf("error should mention --name; got: %v", err)
	}
}

// TestInitCommand_MissingLanguage expects an error when --language is absent
// in non-interactive mode.
func TestInitCommand_MissingLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	_, _, err := runInit(t, tmpDir, []string{
		"--non-interactive",
		"--name", "myapp",
	})
	if err == nil {
		t.Fatal("expected error for missing --language, got nil")
	}
	if !strings.Contains(err.Error(), "--language") {
		t.Errorf("error should mention --language; got: %v", err)
	}
}

// TestInitCommand_DryRunNoFiles verifies that --dry-run produces output on
// stdout but writes no files to disk.
func TestInitCommand_DryRunNoFiles(t *testing.T) {
	tmpDir := t.TempDir()
	stdout, _, err := runInit(t, tmpDir, []string{
		"--non-interactive",
		"--dry-run",
		"--name", "myapp",
		"--language", "go",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// stdout must contain dry-run markers.
	if !strings.Contains(stdout.String(), "# Dry run mode: would write to") {
		t.Errorf("expected dry-run header in stdout; got:\n%s", stdout.String())
	}

	// No files should have been created (only .devcontainer dir may be absent).
	for _, rel := range []string{"CLAUDE.md", "AGENTS.md", ".editorconfig", ".gitignore"} {
		p := filepath.Join(tmpDir, rel)
		if _, statErr := os.Stat(p); !os.IsNotExist(statErr) {
			t.Errorf("dry-run: file %s should not exist on disk", rel)
		}
	}
}

// TestInitCommand_AIToolsSubset verifies that only the explicitly listed AI
// tools are generated when --ai-tools is provided.
func TestInitCommand_AIToolsSubset(t *testing.T) {
	tmpDir := t.TempDir()
	_, _, err := runInit(t, tmpDir, []string{
		"--non-interactive",
		"--name", "myapp",
		"--language", "go",
		"--ai-tools", "claude-md,cursor-rules",
		"--devcontainer=false",
		"--ci=false",
		"--editorconfig=false",
		"--gitignore=false",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Only these two AI files should exist.
	if _, statErr := os.Stat(filepath.Join(tmpDir, "CLAUDE.md")); statErr != nil {
		t.Errorf("CLAUDE.md should exist: %v", statErr)
	}
	if _, statErr := os.Stat(filepath.Join(tmpDir, ".cursor", "rules", "project.mdc")); statErr != nil {
		t.Errorf(".cursor/rules/project.mdc should exist: %v", statErr)
	}

	// Other AI files must not be written.
	for _, rel := range []string{"AGENTS.md", "GEMINI.md", ".windsurfrules"} {
		p := filepath.Join(tmpDir, rel)
		if _, statErr := os.Stat(p); !os.IsNotExist(statErr) {
			t.Errorf("%s should not exist when not in --ai-tools", rel)
		}
	}
}

// TestInitCommand_UnknownAITool verifies that an unrecognised AI tool ID
// returns an error before any files are written.
func TestInitCommand_UnknownAITool(t *testing.T) {
	tmpDir := t.TempDir()
	_, _, err := runInit(t, tmpDir, []string{
		"--non-interactive",
		"--name", "myapp",
		"--language", "go",
		"--ai-tools", "claude-md,nonexistent-tool",
	})
	if err == nil {
		t.Fatal("expected error for unknown AI tool ID, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent-tool") {
		t.Errorf("error should mention the unknown tool; got: %v", err)
	}
}

// TestInitCommand_SummarySuccessOutput verifies the stdout summary when all
// generators succeed.
func TestInitCommand_SummarySuccessOutput(t *testing.T) {
	tmpDir := t.TempDir()
	stdout, _, err := runInit(t, tmpDir, []string{
		"--non-interactive",
		"--name", "myapp",
		"--language", "go",
		"--ai-tools", "claude-md",
		"--devcontainer=false",
		"--ci=false",
		"--editorconfig=false",
		"--gitignore=false",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "cure init summary:") {
		t.Errorf("expected summary header; got:\n%s", out)
	}
	if !strings.Contains(out, "ok claude-md") {
		t.Errorf("expected success line for claude-md; got:\n%s", out)
	}
}

// TestInitCommand_SummaryFailureOutput verifies that the summary reports
// failures and the command returns a non-nil error when a generator fails.
func TestInitCommand_SummaryFailureOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// Pre-create CLAUDE.md so it triggers an "already exists" error.
	if err := os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte("existing"), 0644); err != nil {
		t.Fatalf("setup: write sentinel: %v", err)
	}

	stdout, _, err := runInit(t, tmpDir, []string{
		"--non-interactive",
		"--name", "myapp",
		"--language", "go",
		"--ai-tools", "claude-md,agents-md",
		"--devcontainer=false",
		"--ci=false",
		"--editorconfig=false",
		"--gitignore=false",
	})

	if err == nil {
		t.Fatal("expected non-nil error when a generator fails")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Errorf("error message should mention failure count; got: %v", err)
	}

	out := stdout.String()
	// Failure for claude-md should appear in summary.
	if !strings.Contains(out, "x claude-md") {
		t.Errorf("expected failure marker for claude-md in summary; got:\n%s", out)
	}
	// agents-md should still succeed (continue-on-failure).
	if !strings.Contains(out, "ok agents-md") {
		t.Errorf("expected success for agents-md despite claude-md failure; got:\n%s", out)
	}
}

// TestInitCommand_NoInfrastructureComponents runs with only AI tools (all
// infra flags false) and verifies no infra files are generated.
func TestInitCommand_NoInfrastructureComponents(t *testing.T) {
	tmpDir := t.TempDir()
	_, _, err := runInit(t, tmpDir, []string{
		"--non-interactive",
		"--name", "myapp",
		"--language", "go",
		"--ai-tools", "claude-md",
		"--devcontainer=false",
		"--ci=false",
		"--editorconfig=false",
		"--gitignore=false",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// Infrastructure files must be absent.
	for _, rel := range []string{
		filepath.Join(".devcontainer", "devcontainer.json"),
		filepath.Join(".github", "workflows", "ci.yml"),
		".editorconfig",
		".gitignore",
	} {
		p := filepath.Join(tmpDir, rel)
		if _, statErr := os.Stat(p); !os.IsNotExist(statErr) {
			t.Errorf("%s should not exist; was generated despite flag=false", rel)
		}
	}
}

// TestInitCommand_ForceOverwrite verifies that --force allows overwriting
// existing files without returning an error.
func TestInitCommand_ForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	// Pre-create CLAUDE.md.
	if err := os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte("old"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, _, err := runInit(t, tmpDir, []string{
		"--non-interactive",
		"--force",
		"--name", "myapp",
		"--language", "go",
		"--ai-tools", "claude-md",
		"--devcontainer=false",
		"--ci=false",
		"--editorconfig=false",
		"--gitignore=false",
	})
	if err != nil {
		t.Fatalf("Run() error with --force = %v", err)
	}

	// The file should have been overwritten with new content.
	content, readErr := os.ReadFile(filepath.Join(tmpDir, "CLAUDE.md"))
	if readErr != nil {
		t.Fatalf("read CLAUDE.md: %v", readErr)
	}
	if string(content) == "old" {
		t.Error("CLAUDE.md was not overwritten despite --force")
	}
}
