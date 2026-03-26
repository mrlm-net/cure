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

func TestCursorRulesCommand_NonInteractive(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".cursor", "rules", "project.mdc")

	tests := []struct {
		name      string
		args      []string
		wantErr   bool
		checkFile func(t *testing.T, content string)
	}{
		{
			name: "all required flags provided",
			args: []string{
				"--non-interactive",
				"--name", "myapp",
				"--description", "A test app",
				"--language", "go",
				"--output", outputPath,
			},
			wantErr: false,
			checkFile: func(t *testing.T, content string) {
				if !strings.Contains(content, "myapp") {
					t.Error("Output missing project name")
				}
				if !strings.Contains(content, "A test app") {
					t.Error("Output missing description")
				}
			},
		},
		{
			name: "missing required name flag",
			args: []string{
				"--non-interactive",
				"--description", "A test app",
				"--language", "go",
				"--output", outputPath,
			},
			wantErr: true,
		},
		{
			name: "missing required description flag",
			args: []string{
				"--non-interactive",
				"--name", "myapp",
				"--language", "go",
				"--output", outputPath,
			},
			wantErr: true,
		},
		{
			name: "missing required language flag",
			args: []string{
				"--non-interactive",
				"--name", "myapp",
				"--description", "A test app",
				"--output", outputPath,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Remove(outputPath)

			cmd := &CursorRulesCommand{}
			fset := cmd.Flags()
			if err := fset.Parse(tt.args); err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			var stdout, stderr bytes.Buffer
			tc := &terminal.Context{
				Stdout: &stdout,
				Stderr: &stderr,
				Config: config.NewConfig(),
			}

			err := cmd.Run(context.Background(), tc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Fatalf("Failed to read output file: %v", err)
				}
				if tt.checkFile != nil {
					tt.checkFile(t, string(content))
				}
			}
		})
	}
}

func TestCursorRulesCommand_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".cursor", "rules", "project.mdc")

	cmd := &CursorRulesCommand{}
	fset := cmd.Flags()
	if err := fset.Parse([]string{
		"--non-interactive",
		"--dry-run",
		"--name", "myapp",
		"--description", "A test app",
		"--language", "go",
		"--output", outputPath,
	}); err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	var stdout, stderr bytes.Buffer
	tc := &terminal.Context{Stdout: &stdout, Stderr: &stderr, Config: nil}

	if err := cmd.Run(context.Background(), tc); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Error("Dry-run wrote a file to disk — it must not")
	}

	if !strings.Contains(stdout.String(), "# Dry run mode: would write to") {
		t.Error("Dry-run output missing header line")
	}
}

func TestCursorRulesCommand_CreatesParentDir(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".cursor", "rules", "project.mdc")

	cmd := &CursorRulesCommand{}
	fset := cmd.Flags()
	_ = fset.Parse([]string{
		"--non-interactive",
		"--name", "myapp",
		"--description", "A test app",
		"--language", "go",
		"--output", outputPath,
	})

	var stdout, stderr bytes.Buffer
	tc := &terminal.Context{Stdout: &stdout, Stderr: &stderr, Config: config.NewConfig()}

	if err := cmd.Run(context.Background(), tc); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if _, err := os.Stat(outputPath); err != nil {
		t.Errorf("Expected output file to exist: %v", err)
	}
}

func BenchmarkCursorRulesCommand_DryRun(b *testing.B) {
	var stdout, stderr bytes.Buffer
	tc := &terminal.Context{Stdout: &stdout, Stderr: &stderr, Config: nil}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		stdout.Reset()
		stderr.Reset()

		cmd := &CursorRulesCommand{}
		fset := cmd.Flags()
		_ = fset.Parse([]string{
			"--non-interactive",
			"--dry-run",
			"--name", "benchapp",
			"--description", "Benchmark application",
			"--language", "go",
		})

		if err := cmd.Run(context.Background(), tc); err != nil {
			b.Fatalf("Run() error: %v", err)
		}
	}
}
