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

// runGithubWorkflowCmd is a helper that parses args, runs the command, and
// returns stdout and the error.
func runGithubWorkflowCmd(t *testing.T, args []string) (string, error) {
	t.Helper()
	cmd := &GithubWorkflowCommand{}
	fset := cmd.Flags()
	if err := fset.Parse(args); err != nil {
		t.Fatalf("flag parse error: %v", err)
	}
	var stdout, stderr bytes.Buffer
	tc := &terminal.Context{Stdout: &stdout, Stderr: &stderr}
	err := cmd.Run(context.Background(), tc)
	return stdout.String(), err
}

func TestGithubWorkflowCommand_NonInteractiveDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "ci.yml")

	stdout, err := runGithubWorkflowCmd(t, []string{
		"--non-interactive",
		"--output", outputPath,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	content, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatalf("ReadFile() error = %v", readErr)
	}
	got := string(content)

	if !strings.Contains(got, "go-version: '1.25'") {
		t.Errorf("output missing default go-version; stdout=%q content=%q", stdout, got)
	}
	if !strings.Contains(got, "go test -race ./...") {
		t.Error("output missing 'go test -race ./...'")
	}
}

func TestGithubWorkflowCommand_CustomGoVersion(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "ci.yml")

	_, err := runGithubWorkflowCmd(t, []string{
		"--non-interactive",
		"--go-version", "1.22",
		"--output", outputPath,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	content, _ := os.ReadFile(outputPath)
	if !strings.Contains(string(content), "go-version: '1.22'") {
		t.Errorf("output missing go-version 1.22; content=%q", string(content))
	}
}

func TestGithubWorkflowCommand_WithLint(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "ci.yml")

	_, err := runGithubWorkflowCmd(t, []string{
		"--non-interactive",
		"--lint",
		"--output", outputPath,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	content, _ := os.ReadFile(outputPath)
	if !strings.Contains(string(content), "go vet ./...") {
		t.Errorf("output missing 'go vet ./...'; content=%q", string(content))
	}
}

func TestGithubWorkflowCommand_WithCoverage(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "ci.yml")

	_, err := runGithubWorkflowCmd(t, []string{
		"--non-interactive",
		"--coverage",
		"--output", outputPath,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	content, _ := os.ReadFile(outputPath)
	if !strings.Contains(string(content), "codecov/codecov-action") {
		t.Errorf("output missing codecov action; content=%q", string(content))
	}
}

func TestGithubWorkflowCommand_InvalidGoVersion(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "ci.yml")

	_, err := runGithubWorkflowCmd(t, []string{
		"--non-interactive",
		"--go-version", "invalid",
		"--output", outputPath,
	})
	if err == nil {
		t.Error("expected error for invalid go-version, got nil")
	}
	if !strings.Contains(err.Error(), "invalid --go-version") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGithubWorkflowCommand_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "ci.yml")

	stdout, err := runGithubWorkflowCmd(t, []string{
		"--non-interactive",
		"--dry-run",
		"--output", outputPath,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	// File must NOT be written.
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Error("dry-run wrote a file to disk — it must not")
	}

	// Content must appear on stdout.
	if !strings.Contains(stdout, "# Dry run mode: would write to") {
		t.Error("dry-run output missing header line")
	}
	if !strings.Contains(stdout, "go test -race ./...") {
		t.Error("dry-run stdout missing workflow content")
	}
}

func TestGithubWorkflowCommand_OverwriteProtection(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "ci.yml")

	// Pre-create the target file.
	if err := os.WriteFile(outputPath, []byte("sentinel"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name: "error without --force",
			args: []string{
				"--non-interactive",
				"--output", outputPath,
			},
			wantErr: true,
		},
		{
			name: "success with --force",
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
			// Restore sentinel before each sub-test.
			os.WriteFile(outputPath, []byte("sentinel"), 0644)

			_, err := runGithubWorkflowCmd(t, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v got err=%v", tt.wantErr, err)
			}
		})
	}
}

func TestGithubWorkflowCommand_EnsureDir(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a nested path that does not yet exist.
	outputPath := filepath.Join(tmpDir, "deep", "nested", "ci.yml")

	_, err := runGithubWorkflowCmd(t, []string{
		"--non-interactive",
		"--output", outputPath,
	})
	if err != nil {
		t.Fatalf("Run() error = %v (EnsureDir should have created parent dirs)", err)
	}

	if _, statErr := os.Stat(outputPath); statErr != nil {
		t.Errorf("output file not created: %v", statErr)
	}
}

func TestGithubWorkflowCommand_YAMLStructure(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "ci.yml")

	_, err := runGithubWorkflowCmd(t, []string{
		"--non-interactive",
		"--output", outputPath,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	content, _ := os.ReadFile(outputPath)
	got := string(content)

	requiredStrings := []string{
		"on:",
		"push:",
		"pull_request:",
		"jobs:",
		"steps:",
	}
	for _, s := range requiredStrings {
		if !strings.Contains(got, s) {
			t.Errorf("output missing required YAML key %q", s)
		}
	}
}

func TestGithubWorkflowCommand_NoTrailingSpaces(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "ci.yml")

	_, err := runGithubWorkflowCmd(t, []string{
		"--non-interactive",
		"--lint",
		"--coverage",
		"--output", outputPath,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	content, _ := os.ReadFile(outputPath)
	for i, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimRight(line, " \t")
		if trimmed != line {
			t.Errorf("line %d has trailing whitespace: %q", i+1, line)
		}
	}
}

func BenchmarkGithubWorkflowCommand_DryRun(b *testing.B) {
	var stdout, stderr bytes.Buffer
	tc := &terminal.Context{Stdout: &stdout, Stderr: &stderr}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		stdout.Reset()
		stderr.Reset()

		cmd := &GithubWorkflowCommand{}
		fset := cmd.Flags()
		_ = fset.Parse([]string{
			"--non-interactive",
			"--dry-run",
			"--go-version", "1.25",
			"--lint",
			"--coverage",
		})

		if err := cmd.Run(context.Background(), tc); err != nil {
			b.Fatalf("Run() error: %v", err)
		}
	}
}
