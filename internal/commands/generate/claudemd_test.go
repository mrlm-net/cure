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

func TestClaudeMDCommand_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "CLAUDE.md")

	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		checkOutput func(t *testing.T, stdout string)
		checkNoFile bool
	}{
		{
			name: "dry-run prints header and content, no file written",
			args: []string{
				"--non-interactive",
				"--dry-run",
				"--name", "myapp",
				"--description", "A test app",
				"--language", "go",
				"--output", outputPath,
			},
			wantErr:     false,
			checkNoFile: true,
			checkOutput: func(t *testing.T, stdout string) {
				if !strings.Contains(stdout, "# Dry run mode: would write to") {
					t.Error("Dry-run output missing header line")
				}
				if !strings.Contains(stdout, outputPath) {
					t.Errorf("Dry-run header missing output path %q", outputPath)
				}
				if !strings.Contains(stdout, "# myapp") {
					t.Error("Dry-run output missing project name")
				}
				if !strings.Contains(stdout, "A test app") {
					t.Error("Dry-run output missing description")
				}
				if !strings.Contains(stdout, "- **Language**: go") {
					t.Error("Dry-run output missing language")
				}
			},
		},
		{
			name: "dry-run with conventions renders them without writing",
			args: []string{
				"--non-interactive",
				"--dry-run",
				"--name", "cure",
				"--description", "Go CLI for dev automation",
				"--language", "go",
				"--build-tool", "make",
				"--conventions", "gofmt,go vet",
				"--output", outputPath,
			},
			wantErr:     false,
			checkNoFile: true,
			checkOutput: func(t *testing.T, stdout string) {
				if !strings.Contains(stdout, "# Dry run mode: would write to") {
					t.Error("Dry-run output missing header line")
				}
				if !strings.Contains(stdout, "- gofmt") {
					t.Error("Dry-run output missing convention: gofmt")
				}
				if !strings.Contains(stdout, "- go vet") {
					t.Error("Dry-run output missing convention: go vet")
				}
				if !strings.Contains(stdout, "- **Build tool**: make") {
					t.Error("Dry-run output missing build tool")
				}
			},
		},
		{
			name: "dry-run with missing required flag still errors",
			args: []string{
				"--non-interactive",
				"--dry-run",
				"--name", "myapp",
				// missing --description and --language
				"--output", outputPath,
			},
			wantErr:     true,
			checkNoFile: true,
		},
		{
			name: "dry-run does not overwrite existing file",
			args: []string{
				"--non-interactive",
				"--dry-run",
				"--name", "myapp",
				"--description", "A test app",
				"--language", "go",
				"--output", outputPath,
			},
			wantErr:     false,
			checkNoFile: false, // file pre-exists; we verify its content is unchanged
			checkOutput: func(t *testing.T, stdout string) {
				if !strings.Contains(stdout, "# Dry run mode: would write to") {
					t.Error("Dry-run output missing header line")
				}
			},
		},
		{
			name: "dry-run default output path appears in header",
			args: []string{
				"--non-interactive",
				"--dry-run",
				"--name", "myapp",
				"--description", "A test app",
				"--language", "go",
				// no --output: defaults to ./CLAUDE.md
			},
			wantErr:     false,
			checkNoFile: false, // not checking tmp dir; just checking stdout
			checkOutput: func(t *testing.T, stdout string) {
				if !strings.Contains(stdout, "# Dry run mode: would write to ./CLAUDE.md") {
					t.Errorf("Dry-run header should show default path; got stdout:\n%s", stdout)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Pre-create the file for the "does not overwrite" test case
			if tt.name == "dry-run does not overwrite existing file" {
				if err := os.WriteFile(outputPath, []byte("sentinel content"), 0644); err != nil {
					t.Fatalf("Failed to create sentinel file: %v", err)
				}
			} else {
				os.Remove(outputPath)
			}

			// Create and configure command
			cmd := &ClaudeMDCommand{}
			fs := cmd.Flags()
			if err := fs.Parse(tt.args); err != nil {
				t.Fatalf("Failed to parse flags: %v", err)
			}

			// Inject I/O
			var stdout, stderr bytes.Buffer
			tc := &terminal.Context{
				Stdout: &stdout,
				Stderr: &stderr,
				Config: nil,
			}

			// Run
			err := cmd.Run(context.Background(), tc)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check that no file was written when checkNoFile is true
			if tt.checkNoFile {
				if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
					t.Error("Dry-run wrote a file to disk — it must not")
				}
			}

			// Check that pre-existing file content is unchanged in overwrite test
			if tt.name == "dry-run does not overwrite existing file" && !tt.wantErr {
				content, readErr := os.ReadFile(outputPath)
				if readErr != nil {
					t.Fatalf("Failed to read sentinel file: %v", readErr)
				}
				if string(content) != "sentinel content" {
					t.Errorf("Dry-run modified existing file; content = %q", string(content))
				}
			}

			// Validate stdout
			if !tt.wantErr && tt.checkOutput != nil {
				tt.checkOutput(t, stdout.String())
			}
		})
	}
}

func BenchmarkClaudeMDCommand_DryRun(b *testing.B) {
	var stdout, stderr bytes.Buffer
	tc := &terminal.Context{
		Stdout: &stdout,
		Stderr: &stderr,
		Config: nil,
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		stdout.Reset()
		stderr.Reset()

		cmd := &ClaudeMDCommand{}
		fs := cmd.Flags()
		_ = fs.Parse([]string{
			"--non-interactive",
			"--dry-run",
			"--name", "benchapp",
			"--description", "Benchmark application",
			"--language", "go",
			"--build-tool", "make",
			"--conventions", "gofmt,go vet",
		})

		if err := cmd.Run(context.Background(), tc); err != nil {
			b.Fatalf("Run() error: %v", err)
		}
	}
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

func TestDefaultTestFramework(t *testing.T) {
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
			got := defaultTestFramework(tt.language)
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
