// Package vcs provides typed wrappers over the git CLI for version control
// operations. All functions shell out to the git binary.
package vcs

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// StatusResult holds parsed git status output.
type StatusResult struct {
	Branch    string
	Staged    []FileStatus
	Unstaged  []FileStatus
	Untracked []string
	Clean     bool
}

// FileStatus represents a file's status.
type FileStatus struct {
	Path   string
	Status string // "M", "A", "D", "R", etc.
}

// LogEntry represents a single commit.
type LogEntry struct {
	Hash      string    `json:"hash"`
	Author    string    `json:"author"`
	Date      time.Time `json:"date"`
	Subject   string    `json:"subject"`
}

// DiffResult holds diff output.
type DiffResult struct {
	Files   []string
	Patch   string
	Summary string
}

// CommitOption configures Commit behavior.
type CommitOption func(*commitOpts)

type commitOpts struct {
	validatePattern string
}

// WithValidatePattern validates the commit message against a regex before committing.
func WithValidatePattern(pattern string) CommitOption {
	return func(o *commitOpts) { o.validatePattern = pattern }
}

// Status returns the working tree status.
func Status(dir string) (*StatusResult, error) {
	out, err := run(dir, "status", "--porcelain", "-b")
	if err != nil {
		return nil, err
	}

	result := &StatusResult{Clean: true}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			branch := strings.TrimPrefix(line, "## ")
			if idx := strings.Index(branch, "..."); idx > 0 {
				branch = branch[:idx]
			}
			result.Branch = branch
			continue
		}

		result.Clean = false
		x, y := line[0], line[1]
		path := strings.TrimSpace(line[3:])

		if x != ' ' && x != '?' {
			result.Staged = append(result.Staged, FileStatus{Path: path, Status: string(x)})
		}
		if y != ' ' && y != '?' {
			result.Unstaged = append(result.Unstaged, FileStatus{Path: path, Status: string(y)})
		}
		if x == '?' {
			result.Untracked = append(result.Untracked, path)
		}
	}
	return result, nil
}

// Branch creates and checks out a new branch.
func Branch(dir, name string) error {
	_, err := run(dir, "checkout", "-b", name)
	return err
}

// Commit creates a commit with the given message.
func Commit(dir, message string, opts ...CommitOption) error {
	var o commitOpts
	for _, opt := range opts {
		opt(&o)
	}

	if o.validatePattern != "" {
		re, err := regexp.Compile(o.validatePattern)
		if err != nil {
			return fmt.Errorf("vcs: invalid commit pattern: %w", err)
		}
		if !re.MatchString(message) {
			return fmt.Errorf("vcs: commit message does not match pattern %q", o.validatePattern)
		}
	}

	_, err := run(dir, "commit", "-m", message)
	return err
}

// Push pushes the current branch to origin.
func Push(dir string) error {
	_, err := run(dir, "push")
	return err
}

// PushUpstream pushes and sets upstream tracking.
func PushUpstream(dir, remote, branch string) error {
	_, err := run(dir, "push", "-u", remote, branch)
	return err
}

// Pull pulls the current branch from origin.
func Pull(dir string) error {
	_, err := run(dir, "pull")
	return err
}

// Diff returns the diff for uncommitted changes.
func Diff(dir string) (*DiffResult, error) {
	patch, err := run(dir, "diff")
	if err != nil {
		return nil, err
	}

	nameOnly, _ := run(dir, "diff", "--name-only")
	files := splitNonEmpty(nameOnly)

	stat, _ := run(dir, "diff", "--stat")

	return &DiffResult{Files: files, Patch: patch, Summary: stat}, nil
}

// DiffStaged returns the diff for staged changes.
func DiffStaged(dir string) (*DiffResult, error) {
	patch, err := run(dir, "diff", "--cached")
	if err != nil {
		return nil, err
	}

	nameOnly, _ := run(dir, "diff", "--cached", "--name-only")
	files := splitNonEmpty(nameOnly)

	return &DiffResult{Files: files, Patch: patch}, nil
}

// Log returns recent commit history.
func Log(dir string, count int) ([]LogEntry, error) {
	if count <= 0 {
		count = 10
	}
	out, err := run(dir, "log", fmt.Sprintf("-%d", count),
		"--format={\"hash\":\"%H\",\"author\":\"%an\",\"date\":\"%aI\",\"subject\":\"%s\"}")
	if err != nil {
		return nil, err
	}

	var entries []LogEntry
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		var e LogEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// CurrentBranch returns the current branch name.
func CurrentBranch(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// IsDirty reports whether the working tree has uncommitted changes.
func IsDirty(dir string) (bool, error) {
	out, err := run(dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func splitNonEmpty(s string) []string {
	var result []string
	for _, line := range strings.Split(strings.TrimSpace(s), "\n") {
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
