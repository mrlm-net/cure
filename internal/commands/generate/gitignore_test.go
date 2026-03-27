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

// runGitignoreDryRun is a helper that runs GenerateGitignore in dry-run mode and
// returns the output written to the writer.
func runGitignoreDryRun(t *testing.T, profiles []string) string {
	t.Helper()
	var buf bytes.Buffer
	opts := GitignoreOpts{
		Profiles:       profiles,
		OutputPath:     "./.gitignore",
		DryRun:         true,
		NonInteractive: true,
	}
	if err := GenerateGitignore(context.Background(), &buf, opts); err != nil {
		t.Fatalf("GenerateGitignore() error = %v", err)
	}
	return buf.String()
}

func TestGenerateGitignore_EmptyProfiles_UniversalOnly(t *testing.T) {
	output := runGitignoreDryRun(t, []string{})

	for _, want := range []string{".env", "*.log", "*.tmp", "*.temp", ".cache/", "tmp/", "temp/"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing universal pattern %q", want)
		}
	}

	// No language-specific patterns should appear.
	for _, notWant := range []string{"*.test", "vendor/", "node_modules/"} {
		if strings.Contains(output, notWant) {
			t.Errorf("output should not contain %q in universal-only mode", notWant)
		}
	}
}

func TestGenerateGitignore_SingleProfile_Go(t *testing.T) {
	output := runGitignoreDryRun(t, []string{"go"})

	for _, want := range []string{"*.test", "vendor/"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing Go pattern %q", want)
		}
	}

	// Universal patterns must still be present.
	if !strings.Contains(output, ".env") {
		t.Error("output missing universal pattern .env")
	}
}

func TestGenerateGitignore_MultipleProfiles_BothSectionsPresent(t *testing.T) {
	output := runGitignoreDryRun(t, []string{"go", "node"})

	goPatterns := []string{"*.test", "vendor/"}
	for _, want := range goPatterns {
		if !strings.Contains(output, want) {
			t.Errorf("output missing Go pattern %q", want)
		}
	}

	nodePatterns := []string{"node_modules/", "npm-debug.log*"}
	for _, want := range nodePatterns {
		if !strings.Contains(output, want) {
			t.Errorf("output missing Node.js pattern %q", want)
		}
	}
}

func TestGenerateGitignore_Deduplication(t *testing.T) {
	// The patterns "*.log" is in universalPatterns AND "*.log" is in the java profile.
	// It must appear exactly once in the combined output.
	output := runGitignoreDryRun(t, []string{"java"})

	count := strings.Count(output, "*.log\n")
	if count != 1 {
		t.Errorf("expected *.log to appear exactly once, got %d occurrences", count)
	}

	// Python and Node both have "dist/" and "build/".
	// Count exact-line occurrences (pattern appears at start of a line).
	output2 := runGitignoreDryRun(t, []string{"node", "python"})
	for _, dup := range []string{"dist/", "build/"} {
		// Use "\n" + dup + "\n" to match exact full-line occurrences.
		// Prepend "\n" to avoid matching "sdist/" when searching for "dist/".
		count := strings.Count("\n"+output2, "\n"+dup+"\n")
		if count != 1 {
			t.Errorf("expected %q to appear as a distinct line exactly once after dedup, got %d", dup, count)
		}
	}
}

func TestGenerateGitignore_UnknownProfile_Error(t *testing.T) {
	var buf bytes.Buffer
	opts := GitignoreOpts{
		Profiles:       []string{"go", "nonexistent"},
		OutputPath:     "./.gitignore",
		DryRun:         false,
		NonInteractive: true,
	}
	err := GenerateGitignore(context.Background(), &buf, opts)
	if err == nil {
		t.Fatal("expected error for unknown profile, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error message should mention the unknown profile key, got: %v", err)
	}
}

func TestGenerateGitignore_DryRun_NoFileWritten(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".gitignore")

	var buf bytes.Buffer
	opts := GitignoreOpts{
		Profiles:       []string{"go"},
		OutputPath:     outputPath,
		DryRun:         true,
		NonInteractive: true,
	}
	if err := GenerateGitignore(context.Background(), &buf, opts); err != nil {
		t.Fatalf("GenerateGitignore() error = %v", err)
	}

	if _, err := os.Stat(outputPath); !os.IsNotExist(err) {
		t.Error("dry-run wrote a file to disk — it must not")
	}

	if !strings.Contains(buf.String(), "# Dry run mode: would write to") {
		t.Error("dry-run output missing header line")
	}
}

func TestGenerateGitignore_OverwriteProtection(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".gitignore")

	if err := os.WriteFile(outputPath, []byte("existing content"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Without --force: must return an error.
	var buf bytes.Buffer
	opts := GitignoreOpts{
		Profiles:       []string{"go"},
		OutputPath:     outputPath,
		Force:          false,
		NonInteractive: true,
	}
	if err := GenerateGitignore(context.Background(), &buf, opts); err == nil {
		t.Fatal("expected error when overwriting without --force, got nil")
	}

	// Verify file was not modified.
	content, _ := os.ReadFile(outputPath)
	if string(content) != "existing content" {
		t.Error("file was overwritten despite missing --force flag")
	}

	// With --force: must succeed.
	opts.Force = true
	var buf2 bytes.Buffer
	if err := GenerateGitignore(context.Background(), &buf2, opts); err != nil {
		t.Errorf("expected success with --force, got error: %v", err)
	}
}

func TestGenerateGitignore_NonInteractive_UniversalOnly(t *testing.T) {
	// Non-interactive without --profiles → universal section only.
	output := runGitignoreDryRun(t, []string{})

	if !strings.Contains(output, "# Universal") {
		t.Error("output missing universal section header")
	}
	if !strings.Contains(output, ".env") {
		t.Error("output missing universal pattern .env")
	}

	// No profile sections.
	if strings.Contains(output, "# Go") || strings.Contains(output, "# Node") {
		t.Error("expected no profile sections in universal-only output")
	}
}

// TestGitignoreCommand_NonInteractive tests the full command via flag parsing.
func TestGitignoreCommand_NonInteractive(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantErr  bool
		checkOut func(t *testing.T, content string)
	}{
		{
			name:    "universal only",
			args:    []string{"--non-interactive", "--dry-run"},
			wantErr: false,
			checkOut: func(t *testing.T, content string) {
				if !strings.Contains(content, ".env") {
					t.Error("missing universal pattern .env")
				}
			},
		},
		{
			name:    "go profile",
			args:    []string{"--non-interactive", "--dry-run", "--profiles", "go"},
			wantErr: false,
			checkOut: func(t *testing.T, content string) {
				if !strings.Contains(content, "*.test") {
					t.Error("missing Go pattern *.test")
				}
			},
		},
		{
			name:    "unknown profile flag",
			args:    []string{"--non-interactive", "--dry-run", "--profiles", "badprofile"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &GitignoreCommand{}
			fset := cmd.Flags()
			if err := fset.Parse(tt.args); err != nil {
				t.Fatalf("failed to parse flags: %v", err)
			}

			var stdout, stderr bytes.Buffer
			tc := &terminal.Context{
				Stdout: &stdout,
				Stderr: &stderr,
			}

			err := cmd.Run(context.Background(), tc)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.checkOut != nil {
				tt.checkOut(t, stdout.String())
			}
		})
	}
}

func TestGitignoreCommand_OverwriteProtection(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, ".gitignore")

	if err := os.WriteFile(outputPath, []byte("existing"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Without --force: error.
	cmd := &GitignoreCommand{}
	fset := cmd.Flags()
	_ = fset.Parse([]string{"--non-interactive", "--profiles", "go", "--output", outputPath})

	var stdout, stderr bytes.Buffer
	tc := &terminal.Context{Stdout: &stdout, Stderr: &stderr}
	if err := cmd.Run(context.Background(), tc); err == nil {
		t.Error("expected error when overwriting without --force")
	}

	// With --force: success.
	cmd2 := &GitignoreCommand{}
	fset2 := cmd2.Flags()
	_ = fset2.Parse([]string{"--non-interactive", "--profiles", "go", "--output", outputPath, "--force"})

	var stdout2, stderr2 bytes.Buffer
	tc2 := &terminal.Context{Stdout: &stdout2, Stderr: &stderr2}
	if err := cmd2.Run(context.Background(), tc2); err != nil {
		t.Errorf("expected success with --force, got: %v", err)
	}
}

func BenchmarkGitignoreCommand_DryRun(b *testing.B) {
	// Collect all profile keys for a worst-case benchmark.
	allProfiles := strings.Join(gitignoreProfileOrder, ",")

	var stdout, stderr bytes.Buffer
	tc := &terminal.Context{Stdout: &stdout, Stderr: &stderr}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		stdout.Reset()
		stderr.Reset()

		cmd := &GitignoreCommand{}
		fset := cmd.Flags()
		_ = fset.Parse([]string{
			"--non-interactive",
			"--dry-run",
			"--profiles", allProfiles,
		})

		if err := cmd.Run(context.Background(), tc); err != nil {
			b.Fatalf("Run() error: %v", err)
		}
	}
}
