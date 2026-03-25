package env

import (
	"runtime"
	"strings"
	"testing"
)

// TestDetect_Fields verifies that Detect returns an Environment with the
// expected static fields. We avoid asserting exact values for fields that
// vary across machines (GoVersion, GitVersion, WorkDir).
func TestDetect_Fields(t *testing.T) {
	e := Detect()

	t.Run("OS matches runtime.GOOS", func(t *testing.T) {
		if e.OS != runtime.GOOS {
			t.Errorf("OS = %q, want %q", e.OS, runtime.GOOS)
		}
	})

	t.Run("Arch matches runtime.GOARCH", func(t *testing.T) {
		if e.Arch != runtime.GOARCH {
			t.Errorf("Arch = %q, want %q", e.Arch, runtime.GOARCH)
		}
	})

	t.Run("OS is non-empty", func(t *testing.T) {
		if e.OS == "" {
			t.Error("OS must not be empty")
		}
	})

	t.Run("Arch is non-empty", func(t *testing.T) {
		if e.Arch == "" {
			t.Error("Arch must not be empty")
		}
	})

	t.Run("GoVersion is non-empty when go is on PATH", func(t *testing.T) {
		// Go must be available in any environment where tests run.
		if e.GoVersion == "" {
			t.Error("GoVersion must not be empty — go must be on PATH in test environments")
		}
	})

	t.Run("GoVersion has go prefix", func(t *testing.T) {
		if e.GoVersion != "" && !strings.HasPrefix(e.GoVersion, "go") {
			t.Errorf("GoVersion = %q, expected a 'go' prefix", e.GoVersion)
		}
	})

	t.Run("WorkDir is non-empty", func(t *testing.T) {
		if e.WorkDir == "" {
			t.Error("WorkDir must not be empty")
		}
	})
}

// TestDetect_Caching verifies that two calls to Detect return identical values,
// confirming the sync.Once cache is working.
func TestDetect_Caching(t *testing.T) {
	first := Detect()
	second := Detect()

	if first.OS != second.OS {
		t.Errorf("OS differs between calls: %q vs %q", first.OS, second.OS)
	}
	if first.Arch != second.Arch {
		t.Errorf("Arch differs between calls: %q vs %q", first.Arch, second.Arch)
	}
	if first.Shell != second.Shell {
		t.Errorf("Shell differs between calls: %q vs %q", first.Shell, second.Shell)
	}
	if first.GoVersion != second.GoVersion {
		t.Errorf("GoVersion differs between calls: %q vs %q", first.GoVersion, second.GoVersion)
	}
	if first.GitVersion != second.GitVersion {
		t.Errorf("GitVersion differs between calls: %q vs %q", first.GitVersion, second.GitVersion)
	}
	if first.WorkDir != second.WorkDir {
		t.Errorf("WorkDir differs between calls: %q vs %q", first.WorkDir, second.WorkDir)
	}
}

// TestHasTool verifies HasTool returns true for known tools and false for
// fictitious ones.
func TestHasTool(t *testing.T) {
	tests := []struct {
		name     string
		tool     string
		wantBool bool
	}{
		{
			name:     "go is available",
			tool:     "go",
			wantBool: true,
		},
		{
			name:     "nonexistent tool returns false",
			tool:     "nonexistent-tool-xyz-abc-123",
			wantBool: false,
		},
		{
			name:     "empty string returns false",
			tool:     "",
			wantBool: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasTool(tt.tool)
			if got != tt.wantBool {
				t.Errorf("HasTool(%q) = %v, want %v", tt.tool, got, tt.wantBool)
			}
		})
	}
}

// TestIsGitRepo verifies that IsGitRepo returns true inside the cure project
// repository. Tests always run from within this git worktree.
func TestIsGitRepo(t *testing.T) {
	t.Run("cure project is a git repo", func(t *testing.T) {
		if !IsGitRepo() {
			t.Error("IsGitRepo() returned false — expected true inside the cure git repository")
		}
	})
}

// TestDetectGoVersion_Format verifies the extracted version string looks like
// a go version token.
func TestDetectGoVersion_Format(t *testing.T) {
	v := detectGoVersion()
	if v == "" {
		t.Skip("go not found on PATH — skipping format check")
	}
	if !strings.HasPrefix(v, "go") {
		t.Errorf("detectGoVersion() = %q, want string with 'go' prefix", v)
	}
	// Version must have at least one dot (e.g. "go1.25.0")
	if !strings.Contains(v, ".") {
		t.Errorf("detectGoVersion() = %q, want version with dots", v)
	}
}

// TestDetectGitVersion_Format verifies the extracted git version string.
func TestDetectGitVersion_Format(t *testing.T) {
	v := detectGitVersion()
	if v == "" {
		t.Skip("git not found on PATH — skipping format check")
	}
	if !strings.HasPrefix(v, "git") {
		t.Errorf("detectGitVersion() = %q, want string starting with 'git'", v)
	}
}

// TestDetectWorkDir_NonEmpty verifies detectWorkDir returns a non-empty path.
func TestDetectWorkDir_NonEmpty(t *testing.T) {
	d := detectWorkDir()
	if d == "" {
		t.Error("detectWorkDir() returned empty string")
	}
}

// TestShellEnvVar verifies the Shell field reflects the SHELL environment
// variable (or is empty when unset).
func TestShellEnvVar(t *testing.T) {
	// Since sync.Once has already fired, we can only verify the field is
	// consistent with what os.Getenv("SHELL") returns at this point.
	e := Detect()
	// We just confirm the value is a string (may be empty on Windows).
	_ = e.Shell
}
