package env

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Environment holds runtime environment information detected at startup.
// All fields are populated by [Detect] and are read-only after construction.
type Environment struct {
	// OS is the current operating system identifier (e.g. "darwin", "linux", "windows").
	// Sourced from runtime.GOOS.
	OS string

	// Arch is the current CPU architecture (e.g. "amd64", "arm64").
	// Sourced from runtime.GOARCH.
	Arch string

	// Shell is the path to the user's shell as reported by the SHELL environment
	// variable. Empty string when the variable is not set.
	Shell string

	// GoVersion is the Go toolchain version string (e.g. "go1.25.0") extracted
	// from `go version` output. Empty string when Go is not found on PATH.
	GoVersion string

	// GitVersion is the git version string (e.g. "git version 2.39.0") as
	// reported by `git --version`. Empty string when git is not found on PATH.
	GitVersion string

	// WorkDir is the absolute path of the process working directory at detection
	// time. Empty string when os.Getwd() fails.
	WorkDir string
}

var (
	cachedEnv *Environment
	once      sync.Once
)

// Detect returns the current runtime environment. The result is computed once
// on first call and cached; all subsequent calls return the cached value
// without re-executing any subprocesses. The returned struct is a copy and is
// safe for concurrent use.
func Detect() Environment {
	once.Do(func() {
		cachedEnv = &Environment{
			OS:         runtime.GOOS,
			Arch:       runtime.GOARCH,
			Shell:      os.Getenv("SHELL"),
			GoVersion:  detectGoVersion(),
			GitVersion: detectGitVersion(),
			WorkDir:    detectWorkDir(),
		}
	})
	return *cachedEnv
}

// HasTool reports whether the named executable is available on PATH. The check
// uses exec.LookPath and does not cache results — call it once per tool and
// store the result if repeated lookups matter for performance.
func HasTool(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// IsGitRepo reports whether the current working directory is inside a git
// repository. It walks up the directory tree from os.Getwd() looking for a
// .git entry (file or directory). Returns false when os.Getwd() fails or no
// .git entry is found up to the filesystem root.
func IsGitRepo() bool {
	dir, err := os.Getwd()
	if err != nil {
		return false
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// reached filesystem root
			break
		}
		dir = parent
	}
	return false
}

// detectGoVersion runs `go version` and extracts the version token (e.g.
// "go1.25.0"). Returns an empty string when go is not on PATH or the output
// cannot be parsed.
func detectGoVersion() string {
	out, err := exec.Command("go", "version").Output()
	if err != nil {
		return ""
	}
	// Output format: "go version go1.25.0 darwin/arm64"
	parts := strings.Fields(string(out))
	for _, p := range parts {
		if strings.HasPrefix(p, "go") && len(p) > 2 {
			// second "go" prefixed token is the version, e.g. "go1.25.0"
			if p != "go" && p != "version" {
				return p
			}
		}
	}
	return ""
}

// detectGitVersion runs `git --version` and returns the raw first line of
// output (e.g. "git version 2.39.0"). Returns an empty string when git is not
// on PATH or the command fails.
func detectGitVersion() string {
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	return line
}

// detectWorkDir returns the absolute path of the current working directory.
// Returns an empty string when os.Getwd() fails.
func detectWorkDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}
