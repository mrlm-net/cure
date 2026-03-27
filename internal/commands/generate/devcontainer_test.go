package generate

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/terminal"
)

// runDevcontainerCmd is a helper that parses args and runs DevcontainerCommand with
// in-memory stdout/stderr, returning the writers for assertions.
func runDevcontainerCmd(t *testing.T, args []string) (stdout, stderr bytes.Buffer, err error) {
	t.Helper()
	cmd := &DevcontainerCommand{}
	fset := cmd.Flags()
	if parseErr := fset.Parse(args); parseErr != nil {
		t.Fatalf("failed to parse flags: %v", parseErr)
	}
	tc := &terminal.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	err = cmd.Run(context.Background(), tc)
	return
}

func TestDevcontainerCommand_NonInteractive(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantErr   bool
		checkFile func(t *testing.T, dir string)
	}{
		{
			name: "name flag produces valid JSON with correct name",
			args: []string{
				"--non-interactive",
				"--name", "myproject",
			},
			wantErr: false,
			checkFile: func(t *testing.T, dir string) {
				content := readFileContents(t, filepath.Join(dir, "devcontainer.json"))
				if !strings.Contains(content, `"name": "myproject"`) {
					t.Errorf("devcontainer.json missing expected name field; got:\n%s", content)
				}
				assertValidJSON(t, content)
			},
		},
		{
			name: "dockerfile flag generates two files",
			args: []string{
				"--non-interactive",
				"--name", "myapp",
				"--dockerfile",
			},
			wantErr: false,
			checkFile: func(t *testing.T, dir string) {
				dcContent := readFileContents(t, filepath.Join(dir, "devcontainer.json"))
				assertValidJSON(t, dcContent)
				if !strings.Contains(dcContent, `"dockerfile": "Dockerfile"`) {
					t.Errorf("devcontainer.json missing dockerfile reference; got:\n%s", dcContent)
				}
				dfContent := readFileContents(t, filepath.Join(dir, "Dockerfile"))
				if !strings.Contains(dfContent, "FROM ") {
					t.Errorf("Dockerfile missing FROM instruction; got:\n%s", dfContent)
				}
			},
		},
		{
			name: "extensions flag appears in JSON",
			args: []string{
				"--non-interactive",
				"--name", "myapp",
				"--extensions", "golang.go,eamodio.gitlens",
			},
			wantErr: false,
			checkFile: func(t *testing.T, dir string) {
				content := readFileContents(t, filepath.Join(dir, "devcontainer.json"))
				assertValidJSON(t, content)
				if !strings.Contains(content, `"golang.go"`) {
					t.Errorf("extensions missing golang.go; got:\n%s", content)
				}
				if !strings.Contains(content, `"eamodio.gitlens"`) {
					t.Errorf("extensions missing eamodio.gitlens; got:\n%s", content)
				}
			},
		},
		{
			name: "post-create-command appears in JSON",
			args: []string{
				"--non-interactive",
				"--name", "myapp",
				"--post-create-command", "make install",
			},
			wantErr: false,
			checkFile: func(t *testing.T, dir string) {
				content := readFileContents(t, filepath.Join(dir, "devcontainer.json"))
				assertValidJSON(t, content)
				if !strings.Contains(content, `"postCreateCommand": "make install"`) {
					t.Errorf("postCreateCommand missing; got:\n%s", content)
				}
			},
		},
		{
			name: "base-image flag appears in JSON",
			args: []string{
				"--non-interactive",
				"--name", "goapp",
				"--base-image", "mcr.microsoft.com/devcontainers/go:1",
			},
			wantErr: false,
			checkFile: func(t *testing.T, dir string) {
				content := readFileContents(t, filepath.Join(dir, "devcontainer.json"))
				assertValidJSON(t, content)
				if !strings.Contains(content, `"mcr.microsoft.com/devcontainers/go:1"`) {
					t.Errorf("base image missing; got:\n%s", content)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			args := append(tt.args, "--output-dir", tmpDir)

			_, _, err := runDevcontainerCmd(t, args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkFile != nil {
				tt.checkFile(t, tmpDir)
			}
		})
	}
}

func TestDevcontainerCommand_JSONValidity(t *testing.T) {
	// Comprehensive round-trip test for all optional fields combined.
	tmpDir := t.TempDir()
	args := []string{
		"--non-interactive",
		"--name", "roundtrip",
		"--base-image", "mcr.microsoft.com/devcontainers/base:ubuntu",
		"--extensions", "golang.go, eamodio.gitlens , ms-vsliveshare.vsliveshare",
		"--post-create-command", "npm install && go mod download",
		"--output-dir", tmpDir,
	}

	_, _, err := runDevcontainerCmd(t, args)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	content := readFileContents(t, filepath.Join(tmpDir, "devcontainer.json"))
	assertValidJSON(t, content)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		t.Fatalf("json.Unmarshal failed: %v\ncontent:\n%s", err, content)
	}

	if parsed["name"] != "roundtrip" {
		t.Errorf("name = %v, want roundtrip", parsed["name"])
	}
	if parsed["postCreateCommand"] != "npm install && go mod download" {
		t.Errorf("postCreateCommand = %v", parsed["postCreateCommand"])
	}
}

func TestDevcontainerCommand_DryRun(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		checkNoFile bool
		checkOutput func(t *testing.T, stdout string)
	}{
		{
			name: "dry-run prints content and path, no file written",
			args: []string{
				"--non-interactive",
				"--dry-run",
				"--name", "dryapp",
			},
			checkNoFile: true,
			checkOutput: func(t *testing.T, out string) {
				if !strings.Contains(out, "# Dry run mode: would write to") {
					t.Error("dry-run header missing")
				}
				if !strings.Contains(out, `"name": "dryapp"`) {
					t.Error("dry-run output missing name field")
				}
			},
		},
		{
			name: "dry-run with dockerfile prints both blocks",
			args: []string{
				"--non-interactive",
				"--dry-run",
				"--name", "dryapp",
				"--dockerfile",
			},
			checkNoFile: true,
			checkOutput: func(t *testing.T, out string) {
				count := strings.Count(out, "# Dry run mode: would write to")
				if count != 2 {
					t.Errorf("expected 2 dry-run headers, got %d", count)
				}
				if !strings.Contains(out, "FROM ") {
					t.Error("Dockerfile stub missing FROM instruction in dry-run output")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			args := append(tt.args, "--output-dir", tmpDir)

			stdout, _, err := runDevcontainerCmd(t, args)
			if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}

			if tt.checkNoFile {
				if _, statErr := os.Stat(filepath.Join(tmpDir, "devcontainer.json")); !os.IsNotExist(statErr) {
					t.Error("dry-run wrote devcontainer.json — it must not")
				}
				if _, statErr := os.Stat(filepath.Join(tmpDir, "Dockerfile")); !os.IsNotExist(statErr) {
					t.Error("dry-run wrote Dockerfile — it must not")
				}
			}

			if tt.checkOutput != nil {
				tt.checkOutput(t, stdout.String())
			}
		})
	}
}

func TestDevcontainerCommand_OverwriteProtection(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name: "existing file without force returns error",
			args: []string{
				"--non-interactive",
				"--name", "myapp",
			},
			wantErr: true,
		},
		{
			name: "existing file with force succeeds",
			args: []string{
				"--non-interactive",
				"--name", "myapp",
				"--force",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			// Pre-create the devcontainer.json to trigger overwrite protection.
			existing := filepath.Join(tmpDir, "devcontainer.json")
			if err := os.WriteFile(existing, []byte("{}"), 0644); err != nil {
				t.Fatalf("failed to create sentinel file: %v", err)
			}

			args := append(tt.args, "--output-dir", tmpDir)
			_, _, err := runDevcontainerCmd(t, args)

			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDevcontainerCommand_MissingNameNonInteractive(t *testing.T) {
	// The --name flag defaults to "dev", so an explicit empty value must be
	// tested by bypassing the flag default through DevcontainerOpts directly.
	// Flags always carry the default so we test the validateFlags path via opts.
	cmd := &DevcontainerCommand{}
	opts := &DevcontainerOpts{
		Name:           "", // explicitly empty — no default applied yet
		NonInteractive: true,
	}
	if err := cmd.validateFlags(opts); err == nil {
		t.Error("validateFlags() expected error for empty name in non-interactive mode, got nil")
	}
}

func TestDevcontainerCommand_EnsureDirCreated(t *testing.T) {
	// Verify that a non-existent output directory is created automatically.
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "sub", "devcontainer")

	args := []string{
		"--non-interactive",
		"--name", "myapp",
		"--output-dir", nestedDir,
	}

	_, _, err := runDevcontainerCmd(t, args)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	if _, statErr := os.Stat(filepath.Join(nestedDir, "devcontainer.json")); os.IsNotExist(statErr) {
		t.Error("devcontainer.json was not created in nested output directory")
	}
}

func TestParseCSV(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"", []string{}},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{",a,,b,", []string{"a", "b"}},
		{"golang.go,eamodio.gitlens", []string{"golang.go", "eamodio.gitlens"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseCSV(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseCSV(%q) = %v (len %d), want %v (len %d)",
					tt.input, got, len(got), tt.want, len(tt.want))
				return
			}
			for i, v := range got {
				if v != tt.want[i] {
					t.Errorf("parseCSV(%q)[%d] = %q, want %q", tt.input, i, v, tt.want[i])
				}
			}
		})
	}
}

func BenchmarkDevcontainerCommand_DryRun(b *testing.B) {
	var stdout, stderr bytes.Buffer
	tc := &terminal.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		stdout.Reset()
		stderr.Reset()

		cmd := &DevcontainerCommand{}
		fset := cmd.Flags()
		_ = fset.Parse([]string{
			"--non-interactive",
			"--dry-run",
			"--name", "benchapp",
			"--base-image", "mcr.microsoft.com/devcontainers/go:1",
			"--extensions", "golang.go,eamodio.gitlens",
			"--post-create-command", "make install",
		})

		if err := cmd.Run(context.Background(), tc); err != nil {
			b.Fatalf("Run() error: %v", err)
		}
	}
}

// --- helpers ---

func readFileContents(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(data)
}

func assertValidJSON(t *testing.T, content string) {
	t.Helper()
	var v interface{}
	if err := json.Unmarshal([]byte(content), &v); err != nil {
		t.Errorf("invalid JSON: %v\ncontent:\n%s", err, content)
	}
}
