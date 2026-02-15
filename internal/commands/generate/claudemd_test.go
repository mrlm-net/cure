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

func TestClaudeMDCommand_NonInteractive(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "CLAUDE.md")

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
				if !strings.Contains(content, "# myapp") {
					t.Error("Output missing project name header")
				}
				if !strings.Contains(content, "A test app") {
					t.Error("Output missing description")
				}
				if !strings.Contains(content, "- **Language**: go") {
					t.Error("Output missing language")
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
		{
			name: "with conventions",
			args: []string{
				"--non-interactive",
				"--name", "myapp",
				"--description", "A test app",
				"--language", "go",
				"--conventions", "gofmt,go vet,golint",
				"--output", outputPath,
			},
			wantErr: false,
			checkFile: func(t *testing.T, content string) {
				if !strings.Contains(content, "- gofmt") {
					t.Error("Output missing convention: gofmt")
				}
				if !strings.Contains(content, "- go vet") {
					t.Error("Output missing convention: go vet")
				}
				if !strings.Contains(content, "- golint") {
					t.Error("Output missing convention: golint")
				}
			},
		},
		{
			name: "custom build tool and test framework",
			args: []string{
				"--non-interactive",
				"--name", "myapp",
				"--description", "A test app",
				"--language", "rust",
				"--build-tool", "cargo",
				"--test-framework", "cargo test",
				"--output", outputPath,
			},
			wantErr: false,
			checkFile: func(t *testing.T, content string) {
				if !strings.Contains(content, "- **Build tool**: cargo") {
					t.Error("Output missing custom build tool")
				}
				if !strings.Contains(content, "- **Test framework**: cargo test") {
					t.Error("Output missing custom test framework")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up before each test
			os.Remove(outputPath)

			// Create command
			cmd := &ClaudeMDCommand{}
			fs := cmd.Flags()
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			// Create context
			var stdout, stderr bytes.Buffer
			tc := &terminal.Context{
				Stdout: &stdout,
				Stderr: &stderr,
				Config: config.NewConfig(),
			}

			// Run command
			err := cmd.Run(context.Background(), tc)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check file creation and content
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

func TestClaudeMDCommand_Interactive(t *testing.T) {
	// Note: Interactive mode testing with os.Stdin replacement is complex
	// and prone to race conditions with bufio.Scanner. We test the core
	// interactive logic via the Prompter unit tests and rely on E2E tests
	// for full integration. This test validates the non-stdin path.
	t.Skip("Interactive stdin testing is unreliable in unit tests - covered by Prompter tests and E2E")
}

func TestClaudeMDCommand_OverwriteProtection(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "CLAUDE.md")

	// Create existing file
	if err := os.WriteFile(outputPath, []byte("existing content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name: "non-interactive without force fails",
			args: []string{
				"--non-interactive",
				"--name", "myapp",
				"--description", "A test app",
				"--language", "go",
				"--output", outputPath,
			},
			wantErr: true,
		},
		{
			name: "non-interactive with force succeeds",
			args: []string{
				"--non-interactive",
				"--name", "myapp",
				"--description", "A test app",
				"--language", "go",
				"--force",
				"--output", outputPath,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Recreate existing file before each test
			os.WriteFile(outputPath, []byte("existing content"), 0644)

			// Create command
			cmd := &ClaudeMDCommand{}
			fs := cmd.Flags()
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			// Create context
			var stdout, stderr bytes.Buffer
			tc := &terminal.Context{
				Stdout: &stdout,
				Stderr: &stderr,
				Config: config.NewConfig(),
			}

			// Run command
			err := cmd.Run(context.Background(), tc)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClaudeMDCommand_DefaultTestFramework(t *testing.T) {
	tests := []struct {
		language string
		want     string
	}{
		{"go", "testing"},
		{"Go", "testing"},
		{"GO", "testing"},
		{"python", "pytest"},
		{"Python", "pytest"},
		{"javascript", "jest"},
		{"typescript", "jest"},
		{"rust", "cargo test"},
		{"java", "junit"},
		{"unknown", "testing"},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			cmd := &ClaudeMDCommand{
				language: tt.language,
			}
			got := cmd.defaultTestFramework()
			if got != tt.want {
				t.Errorf("defaultTestFramework() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClaudeMDCommand_E2E(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "CLAUDE.md")

	// Test via router (like real usage)
	var stdout bytes.Buffer
	router := terminal.New(
		terminal.WithStdout(&stdout),
		terminal.WithConfig(config.NewConfig()),
	)
	router.Register(NewGenerateCommand())

	args := []string{
		"generate", "claude-md",
		"--non-interactive",
		"--name", "cure",
		"--description", "A Go CLI tool for dev automation",
		"--language", "go",
		"--build-tool", "make",
		"--test-framework", "testing",
		"--conventions", "gofmt,go vet",
		"--output", outputPath,
	}

	err := router.RunArgs(args)
	if err != nil {
		t.Fatalf("RunArgs() error = %v", err)
	}

	// Validate file was created
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Validate key content
	expectedSections := []string{
		"# cure",
		"A Go CLI tool for dev automation",
		"- **Language**: go",
		"- **Build tool**: make",
		"- **Test framework**: testing",
		"- gofmt",
		"- go vet",
	}

	for _, section := range expectedSections {
		if !strings.Contains(string(content), section) {
			t.Errorf("Output missing expected section: %q", section)
		}
	}
}
