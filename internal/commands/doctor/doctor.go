// Package doctor implements the "cure doctor" command, which runs a suite
// of project health checks against the current working directory and reports
// per-check results with a final summary.
//
// The check framework and built-in checks live in pkg/doctor. This package
// is a thin adapter that wires pkg/doctor into the cure CLI.
package doctor

import (
	"context"
	"flag"
	"fmt"

	pkgdoctor "github.com/mrlm-net/cure/pkg/doctor"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// Re-export pkg/doctor types so existing tests that reference these names
// within package doctor continue to compile without modification.
type CheckStatus = pkgdoctor.CheckStatus
type CheckResult = pkgdoctor.CheckResult
type CheckFunc = pkgdoctor.CheckFunc

const (
	CheckPass = pkgdoctor.CheckPass
	CheckWarn = pkgdoctor.CheckWarn
	CheckFail = pkgdoctor.CheckFail
)

// Re-export built-in check functions so internal tests can call them directly.
var (
	CheckREADME              = pkgdoctor.CheckREADME
	CheckTests               = pkgdoctor.CheckTests
	CheckCI                  = pkgdoctor.CheckCI
	CheckGitignore           = pkgdoctor.CheckGitignore
	CheckClaudeMD            = pkgdoctor.CheckClaudeMD
	CheckBuildTool           = pkgdoctor.CheckBuildTool
	CheckDependencyManifest  = pkgdoctor.CheckDependencyManifest
)

// DoctorCommand implements "cure doctor".
type DoctorCommand struct {
	noCustom bool
}

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
	return `Usage: cure doctor [--no-custom]

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

Flags:
  --no-custom   Skip custom checks from .cure.json

Examples:
  cure doctor
  cure doctor --no-custom
`
}

// Flags returns a FlagSet with the --no-custom flag.
func (c *DoctorCommand) Flags() *flag.FlagSet {
	fset := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fset.BoolVar(&c.noCustom, "no-custom", false, "Skip custom checks from .cure.json")
	return fset
}

// Run executes all built-in health checks and prints results to tc.Stdout.
// Returns an error if any check has CheckFail status.
func (c *DoctorCommand) Run(_ context.Context, tc *terminal.Context) error {
	checks := pkgdoctor.BuiltinChecks()

	// TODO(#96): load custom checks from .cure.json when --no-custom is false

	fmt.Fprintln(tc.Stdout, "Running project health checks...")
	fmt.Fprintln(tc.Stdout)

	passed, warned, failed := pkgdoctor.Run(checks, tc.Stdout)

	total := passed + warned + failed
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
