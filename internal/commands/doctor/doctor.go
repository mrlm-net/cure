// Package doctor implements the "cure doctor" command, which runs a suite
// of project health checks against the current working directory and reports
// per-check results with a final summary.
//
// Design follows ADR-005: CheckFunc Type — checks are plain functions with a
// well-defined result type, making them easy to test and extend independently
// of the command itself.
package doctor

import (
	"context"
	"flag"
	"fmt"

	"github.com/mrlm-net/cure/pkg/fs"
	"github.com/mrlm-net/cure/pkg/style"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// CheckStatus represents the outcome of a single health check.
type CheckStatus int

const (
	// CheckPass indicates the check found everything it expected.
	CheckPass CheckStatus = iota
	// CheckWarn indicates a non-fatal condition worth noting.
	CheckWarn
	// CheckFail indicates a required item is missing or broken.
	CheckFail
)

// CheckResult holds the outcome of a single health check.
type CheckResult struct {
	// Name is a short label shown in the output (e.g., "README").
	Name string
	// Status is one of CheckPass, CheckWarn, or CheckFail.
	Status CheckStatus
	// Message describes what was found or what is missing.
	Message string
}

// CheckFunc is a function that performs a single health check and returns
// the result. Checks are run against the current working directory.
type CheckFunc func() CheckResult

// DoctorCommand implements "cure doctor".
type DoctorCommand struct{}

// NewDoctorCommand creates a new doctor command.
func NewDoctorCommand() terminal.Command {
	return &DoctorCommand{}
}

// Name returns "doctor".
func (c *DoctorCommand) Name() string { return "doctor" }

// Description returns a short description for help output.
func (c *DoctorCommand) Description() string {
	return "Run project health checks against the current directory"
}

// Usage returns detailed usage information.
func (c *DoctorCommand) Usage() string {
	return `Usage: cure doctor

Runs a suite of project health checks against the current working directory
and reports a pass/warn/fail status for each check.

Checks performed:
  README            README.md or README exists
  Tests             *_test.go files or tests/ directory
  CI Config         .github/workflows/, .gitlab-ci.yml, or .circleci/
  .gitignore        .gitignore file (warn if missing)
  CLAUDE.md         CLAUDE.md file
  Build Tool        Makefile, package.json, Cargo.toml, or build.gradle
  Dependency        go.mod, package.json, requirements.txt, or Cargo.toml

Exit code is non-zero if any check reports a failure.

Examples:
  cure doctor
`
}

// Flags returns nil — the doctor command accepts no flags.
func (c *DoctorCommand) Flags() *flag.FlagSet { return nil }

// Run executes all built-in health checks and prints results to tc.Stdout.
// Returns an error if any check has CheckFail status.
func (c *DoctorCommand) Run(_ context.Context, tc *terminal.Context) error {
	checks := []CheckFunc{
		CheckREADME,
		CheckTests,
		CheckCI,
		CheckGitignore,
		CheckClaudeMD,
		CheckBuildTool,
		CheckDependencyManifest,
	}

	fmt.Fprintln(tc.Stdout, "Running project health checks...")
	fmt.Fprintln(tc.Stdout)

	results := make([]CheckResult, 0, len(checks))
	for _, check := range checks {
		r := check()
		results = append(results, r)
		fmt.Fprintln(tc.Stdout, formatResult(r))
	}

	passed, warned, failed := tally(results)
	total := len(results)
	fmt.Fprintln(tc.Stdout)
	fmt.Fprintf(tc.Stdout, "Summary: %d/%d checks passed", passed, total)
	if warned > 0 {
		if warned == 1 {
			fmt.Fprintf(tc.Stdout, ", %d warning", warned)
		} else {
			fmt.Fprintf(tc.Stdout, ", %d warnings", warned)
		}
	}
	if failed > 0 {
		if failed == 1 {
			fmt.Fprintf(tc.Stdout, ", %d failure", failed)
		} else {
			fmt.Fprintf(tc.Stdout, ", %d failures", failed)
		}
	}
	fmt.Fprintln(tc.Stdout)

	if failed > 0 {
		return fmt.Errorf("doctor: %d check(s) failed", failed)
	}
	return nil
}

// formatResult formats a single CheckResult as a styled output line.
func formatResult(r CheckResult) string {
	var symbol string
	switch r.Status {
	case CheckPass:
		symbol = style.Green("✓")
	case CheckWarn:
		symbol = style.Yellow("⚠")
	case CheckFail:
		symbol = style.Red("✗")
	default:
		symbol = "?"
	}
	return fmt.Sprintf("%s %s", symbol, r.Message)
}

// tally counts results by status category.
func tally(results []CheckResult) (passed, warned, failed int) {
	for _, r := range results {
		switch r.Status {
		case CheckPass:
			passed++
		case CheckWarn:
			warned++
		case CheckFail:
			failed++
		}
	}
	return
}

// CheckREADME verifies that a README file exists in the current directory.
func CheckREADME() CheckResult {
	for _, name := range []string{"README.md", "README"} {
		ok, err := fs.Exists(name)
		if err == nil && ok {
			return CheckResult{
				Name:    "README",
				Status:  CheckPass,
				Message: fmt.Sprintf("%s found", name),
			}
		}
	}
	return CheckResult{
		Name:    "README",
		Status:  CheckFail,
		Message: "README not found (expected README.md or README)",
	}
}

// CheckTests verifies that test files or a tests directory exist.
func CheckTests() CheckResult {
	// Check for a tests/ directory first.
	ok, err := fs.Exists("tests")
	if err == nil && ok {
		return CheckResult{
			Name:    "Tests",
			Status:  CheckPass,
			Message: "Tests found (tests/)",
		}
	}

	// Walk the current directory (non-recursively) for *_test.go files.
	found, detail := findTestFiles(".")
	if found {
		return CheckResult{
			Name:    "Tests",
			Status:  CheckPass,
			Message: fmt.Sprintf("Tests found (%s)", detail),
		}
	}

	return CheckResult{
		Name:    "Tests",
		Status:  CheckFail,
		Message: "No tests found (expected *_test.go files or tests/ directory)",
	}
}

// CheckCI verifies that a CI configuration exists.
func CheckCI() CheckResult {
	candidates := []struct {
		path  string
		label string
	}{
		{".github/workflows", ".github/workflows/"},
		{".gitlab-ci.yml", ".gitlab-ci.yml"},
		{".circleci", ".circleci/"},
	}
	for _, c := range candidates {
		ok, err := fs.Exists(c.path)
		if err == nil && ok {
			return CheckResult{
				Name:    "CI Config",
				Status:  CheckPass,
				Message: fmt.Sprintf("CI configuration found (%s)", c.label),
			}
		}
	}
	return CheckResult{
		Name:    "CI Config",
		Status:  CheckFail,
		Message: "No CI configuration found (expected .github/workflows/, .gitlab-ci.yml, or .circleci/)",
	}
}

// CheckGitignore verifies that a .gitignore file exists.
// Missing .gitignore is a warning, not a failure.
func CheckGitignore() CheckResult {
	ok, err := fs.Exists(".gitignore")
	if err == nil && ok {
		return CheckResult{
			Name:    ".gitignore",
			Status:  CheckPass,
			Message: ".gitignore found",
		}
	}
	return CheckResult{
		Name:    ".gitignore",
		Status:  CheckWarn,
		Message: ".gitignore missing (optional but recommended)",
	}
}

// CheckClaudeMD verifies that a CLAUDE.md file exists.
func CheckClaudeMD() CheckResult {
	ok, err := fs.Exists("CLAUDE.md")
	if err == nil && ok {
		return CheckResult{
			Name:    "CLAUDE.md",
			Status:  CheckPass,
			Message: "CLAUDE.md found",
		}
	}
	return CheckResult{
		Name:    "CLAUDE.md",
		Status:  CheckFail,
		Message: "CLAUDE.md not found",
	}
}

// CheckBuildTool verifies that a recognized build tool configuration exists.
func CheckBuildTool() CheckResult {
	candidates := []struct {
		path  string
		label string
	}{
		{"Makefile", "Makefile"},
		{"package.json", "package.json"},
		{"Cargo.toml", "Cargo.toml"},
		{"build.gradle", "build.gradle"},
	}
	for _, c := range candidates {
		ok, err := fs.Exists(c.path)
		if err == nil && ok {
			return CheckResult{
				Name:    "Build Tool",
				Status:  CheckPass,
				Message: fmt.Sprintf("Build tool found (%s)", c.label),
			}
		}
	}
	return CheckResult{
		Name:    "Build Tool",
		Status:  CheckFail,
		Message: "No build tool found (expected Makefile, package.json, Cargo.toml, or build.gradle)",
	}
}

// CheckDependencyManifest verifies that a dependency manifest exists.
func CheckDependencyManifest() CheckResult {
	candidates := []struct {
		path  string
		label string
	}{
		{"go.mod", "go.mod"},
		{"package.json", "package.json"},
		{"requirements.txt", "requirements.txt"},
		{"Cargo.toml", "Cargo.toml"},
	}
	for _, c := range candidates {
		ok, err := fs.Exists(c.path)
		if err == nil && ok {
			return CheckResult{
				Name:    "Dependency Manifest",
				Status:  CheckPass,
				Message: fmt.Sprintf("Dependency manifest found (%s)", c.label),
			}
		}
	}
	return CheckResult{
		Name:    "Dependency Manifest",
		Status:  CheckFail,
		Message: "No dependency manifest found (expected go.mod, package.json, requirements.txt, or Cargo.toml)",
	}
}
