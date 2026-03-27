package doctor

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/style"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

// chdir changes the working directory to dir for the duration of the test.
// The original directory is restored via t.Cleanup.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck
}

// tempDir creates a fresh temp directory, changes into it, and returns the path.
func tempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	chdir(t, dir)
	return dir
}

// touch creates an empty file at path (relative to the current directory).
func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	f.Close()
}

// --------------------------------------------------------------------------
// Unit tests — per CheckFunc
// --------------------------------------------------------------------------

func TestCheckREADME(t *testing.T) {
	tests := []struct {
		name    string
		files   []string
		wantSt  CheckStatus
		wantMsg string
	}{
		{
			name:    "README.md present",
			files:   []string{"README.md"},
			wantSt:  CheckPass,
			wantMsg: "README.md found",
		},
		{
			name:    "plain README present",
			files:   []string{"README"},
			wantSt:  CheckPass,
			wantMsg: "README found",
		},
		{
			name:    "no README",
			files:   []string{},
			wantSt:  CheckFail,
			wantMsg: "README not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir(t)
			for _, f := range tt.files {
				touch(t, f)
			}
			r := CheckREADME()
			if r.Status != tt.wantSt {
				t.Errorf("status = %v, want %v", r.Status, tt.wantSt)
			}
			if !strings.Contains(r.Message, tt.wantMsg) {
				t.Errorf("message = %q, want to contain %q", r.Message, tt.wantMsg)
			}
		})
	}
}

func TestCheckTests(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T)
		wantSt  CheckStatus
		wantMsg string
	}{
		{
			name: "_test.go file present",
			setup: func(t *testing.T) {
				touch(t, "foo_test.go")
			},
			wantSt:  CheckPass,
			wantMsg: "foo_test.go",
		},
		{
			name: "tests/ directory present",
			setup: func(t *testing.T) {
				if err := os.Mkdir("tests", 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantSt:  CheckPass,
			wantMsg: "tests/",
		},
		{
			name:    "no tests",
			setup:   func(_ *testing.T) {},
			wantSt:  CheckFail,
			wantMsg: "No tests found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir(t)
			tt.setup(t)
			r := CheckTests()
			if r.Status != tt.wantSt {
				t.Errorf("status = %v, want %v", r.Status, tt.wantSt)
			}
			if !strings.Contains(r.Message, tt.wantMsg) {
				t.Errorf("message = %q, want to contain %q", r.Message, tt.wantMsg)
			}
		})
	}
}

func TestCheckCI(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T)
		wantSt  CheckStatus
		wantMsg string
	}{
		{
			name: ".github/workflows present",
			setup: func(t *testing.T) {
				if err := os.MkdirAll(".github/workflows", 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantSt:  CheckPass,
			wantMsg: ".github/workflows/",
		},
		{
			name: ".gitlab-ci.yml present",
			setup: func(t *testing.T) {
				touch(t, ".gitlab-ci.yml")
			},
			wantSt:  CheckPass,
			wantMsg: ".gitlab-ci.yml",
		},
		{
			name: ".circleci present",
			setup: func(t *testing.T) {
				if err := os.Mkdir(".circleci", 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantSt:  CheckPass,
			wantMsg: ".circleci/",
		},
		{
			name:    "no CI config",
			setup:   func(_ *testing.T) {},
			wantSt:  CheckFail,
			wantMsg: "No CI configuration found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir(t)
			tt.setup(t)
			r := CheckCI()
			if r.Status != tt.wantSt {
				t.Errorf("status = %v, want %v", r.Status, tt.wantSt)
			}
			if !strings.Contains(r.Message, tt.wantMsg) {
				t.Errorf("message = %q, want to contain %q", r.Message, tt.wantMsg)
			}
		})
	}
}

func TestCheckGitignore(t *testing.T) {
	tests := []struct {
		name   string
		files  []string
		wantSt CheckStatus
	}{
		{
			name:   ".gitignore present",
			files:  []string{".gitignore"},
			wantSt: CheckPass,
		},
		{
			name:   ".gitignore missing — warn, not fail",
			files:  []string{},
			wantSt: CheckWarn,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir(t)
			for _, f := range tt.files {
				touch(t, f)
			}
			r := CheckGitignore()
			if r.Status != tt.wantSt {
				t.Errorf("status = %v, want %v", r.Status, tt.wantSt)
			}
		})
	}
}

func TestCheckClaudeMD(t *testing.T) {
	tests := []struct {
		name   string
		files  []string
		wantSt CheckStatus
	}{
		{
			name:   "CLAUDE.md present",
			files:  []string{"CLAUDE.md"},
			wantSt: CheckPass,
		},
		{
			name:   "CLAUDE.md missing",
			files:  []string{},
			wantSt: CheckFail,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir(t)
			for _, f := range tt.files {
				touch(t, f)
			}
			r := CheckClaudeMD()
			if r.Status != tt.wantSt {
				t.Errorf("status = %v, want %v", r.Status, tt.wantSt)
			}
		})
	}
}

func TestCheckBuildTool(t *testing.T) {
	tests := []struct {
		name    string
		files   []string
		wantSt  CheckStatus
		wantMsg string
	}{
		{
			name:    "Makefile present",
			files:   []string{"Makefile"},
			wantSt:  CheckPass,
			wantMsg: "Makefile",
		},
		{
			name:    "package.json present",
			files:   []string{"package.json"},
			wantSt:  CheckPass,
			wantMsg: "package.json",
		},
		{
			name:    "Cargo.toml present",
			files:   []string{"Cargo.toml"},
			wantSt:  CheckPass,
			wantMsg: "Cargo.toml",
		},
		{
			name:    "build.gradle present",
			files:   []string{"build.gradle"},
			wantSt:  CheckPass,
			wantMsg: "build.gradle",
		},
		{
			name:    "no build tool",
			files:   []string{},
			wantSt:  CheckFail,
			wantMsg: "No build tool found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir(t)
			for _, f := range tt.files {
				touch(t, f)
			}
			r := CheckBuildTool()
			if r.Status != tt.wantSt {
				t.Errorf("status = %v, want %v", r.Status, tt.wantSt)
			}
			if !strings.Contains(r.Message, tt.wantMsg) {
				t.Errorf("message = %q, want to contain %q", r.Message, tt.wantMsg)
			}
		})
	}
}

func TestCheckDependencyManifest(t *testing.T) {
	tests := []struct {
		name    string
		files   []string
		wantSt  CheckStatus
		wantMsg string
	}{
		{
			name:    "go.mod present",
			files:   []string{"go.mod"},
			wantSt:  CheckPass,
			wantMsg: "go.mod",
		},
		{
			name:    "requirements.txt present",
			files:   []string{"requirements.txt"},
			wantSt:  CheckPass,
			wantMsg: "requirements.txt",
		},
		{
			name:    "no manifest",
			files:   []string{},
			wantSt:  CheckFail,
			wantMsg: "No dependency manifest found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir(t)
			for _, f := range tt.files {
				touch(t, f)
			}
			r := CheckDependencyManifest()
			if r.Status != tt.wantSt {
				t.Errorf("status = %v, want %v", r.Status, tt.wantSt)
			}
			if !strings.Contains(r.Message, tt.wantMsg) {
				t.Errorf("message = %q, want to contain %q", r.Message, tt.wantMsg)
			}
		})
	}
}

// --------------------------------------------------------------------------
// Integration test — Run()
// --------------------------------------------------------------------------

func TestDoctorCommand_Run_AllPass(t *testing.T) {
	// Disable ANSI styling so we can check plain strings.
	style.Disable()
	t.Cleanup(style.Enable)

	tempDir(t)

	// Create all expected files so every check passes.
	touch(t, "README.md")
	touch(t, "foo_test.go")
	if err := os.MkdirAll(".github/workflows", 0o755); err != nil {
		t.Fatal(err)
	}
	touch(t, ".gitignore")
	touch(t, "CLAUDE.md")
	touch(t, "Makefile")
	touch(t, "go.mod")

	var buf bytes.Buffer
	tc := &terminal.Context{Stdout: &buf, Stderr: &buf}
	cmd := NewDoctorCommand()

	if err := cmd.Run(context.Background(), tc); err != nil {
		t.Errorf("expected nil error, got: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "7/7 checks passed") {
		t.Errorf("expected '7/7 checks passed', got:\n%s", out)
	}
}

func TestDoctorCommand_Run_SomeFailures(t *testing.T) {
	style.Disable()
	t.Cleanup(style.Enable)

	tempDir(t)
	// Provide only README — all other checks will fail or warn.
	touch(t, "README.md")

	var buf bytes.Buffer
	tc := &terminal.Context{Stdout: &buf, Stderr: &buf}
	cmd := NewDoctorCommand()

	err := cmd.Run(context.Background(), tc)
	if err == nil {
		t.Error("expected non-nil error when checks fail")
	}
	if !strings.Contains(err.Error(), "check(s) failed") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestDoctorCommand_Run_OnlyWarnings(t *testing.T) {
	style.Disable()
	t.Cleanup(style.Enable)

	tempDir(t)

	// Provide all required files EXCEPT .gitignore (which is only a warn).
	touch(t, "README.md")
	touch(t, "foo_test.go")
	if err := os.MkdirAll(".github/workflows", 0o755); err != nil {
		t.Fatal(err)
	}
	// .gitignore intentionally absent
	touch(t, "CLAUDE.md")
	touch(t, "Makefile")
	touch(t, "go.mod")

	var buf bytes.Buffer
	tc := &terminal.Context{Stdout: &buf, Stderr: &buf}
	cmd := NewDoctorCommand()

	// Only warnings — should return nil.
	if err := cmd.Run(context.Background(), tc); err != nil {
		t.Errorf("expected nil error for warn-only outcome, got: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "1 warning") {
		t.Errorf("expected '1 warning' in output, got:\n%s", out)
	}
}

// --------------------------------------------------------------------------
// Metadata tests
// --------------------------------------------------------------------------

func TestDoctorCommand_Metadata(t *testing.T) {
	cmd := NewDoctorCommand()

	if cmd.Name() != "doctor" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "doctor")
	}
	if cmd.Description() == "" {
		t.Error("Description() must not be empty")
	}
	if cmd.Usage() == "" {
		t.Error("Usage() must not be empty")
	}
	if cmd.Flags() == nil {
		t.Error("Flags() must return a non-nil FlagSet — doctor accepts --no-custom")
	}
}

// --------------------------------------------------------------------------
// Benchmarks
// --------------------------------------------------------------------------

// BenchmarkCheckREADME measures the cost of the README check against a
// directory that has a README.md (the common, fast-path case).
func BenchmarkCheckREADME(b *testing.B) {
	dir := b.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)        //nolint:errcheck
	defer os.Chdir(orig) //nolint:errcheck

	f, _ := os.Create(filepath.Join(dir, "README.md"))
	f.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CheckREADME()
	}
}

// BenchmarkDoctorRun measures a full doctor run against a well-formed project.
func BenchmarkDoctorRun(b *testing.B) {
	dir := b.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)        //nolint:errcheck
	defer os.Chdir(orig) //nolint:errcheck

	// Seed all expected files.
	for _, name := range []string{"README.md", "foo_test.go", ".gitignore", "CLAUDE.md", "Makefile", "go.mod"} {
		f, _ := os.Create(filepath.Join(dir, name))
		f.Close()
	}
	os.MkdirAll(filepath.Join(dir, ".github", "workflows"), 0o755) //nolint:errcheck

	style.Disable()
	defer style.Enable()

	cmd := NewDoctorCommand()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out, errOut bytes.Buffer
		tc := &terminal.Context{Stdout: &out, Stderr: &errOut}
		cmd.Run(context.Background(), tc) //nolint:errcheck
	}
}
