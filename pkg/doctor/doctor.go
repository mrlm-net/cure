// Package doctor provides a public framework for running project health checks.
//
// The core types (CheckFunc, CheckResult, CheckStatus) define a simple
// interface for check functions. Run executes a slice of CheckFunc values,
// recovers from panics, and returns tallied results.
//
// Built-in checks are available via BuiltinChecks. Callers can extend the
// suite by appending their own CheckFunc implementations.
//
// Design follows ADR-005: CheckFunc Type — checks are plain functions with a
// well-defined result type, making them easy to test and compose.
package doctor

import (
	"fmt"
	"io"

	"github.com/mrlm-net/cure/pkg/style"
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
	// Name is a short label identifying the check (e.g., "README").
	Name string
	// Status is one of CheckPass, CheckWarn, or CheckFail.
	Status CheckStatus
	// Message describes what was found or what is missing.
	Message string
}

// CheckFunc is a function that performs a single health check and returns
// the result. Checks are run against the current working directory.
type CheckFunc func() CheckResult

// BuiltinChecks returns the 7 default health checks in canonical order.
// A new slice is returned on each call to prevent mutation by callers.
func BuiltinChecks() []CheckFunc {
	return []CheckFunc{
		CheckREADME,
		CheckTests,
		CheckCI,
		CheckGitignore,
		CheckClaudeMD,
		CheckBuildTool,
		CheckDependencyManifest,
	}
}

// Run executes checks in order, writes a formatted result line to w for each
// check, and returns tallies by status. A check that panics is recovered and
// recorded as CheckFail with a "check panicked: ..." message; remaining checks
// continue to run.
func Run(checks []CheckFunc, w io.Writer) (passed, warned, failed int) {
	for _, check := range checks {
		r := runSafe(check)
		fmt.Fprintln(w, formatResult(r))
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

// runSafe calls fn and catches any panic, converting it to a CheckFail result.
func runSafe(fn CheckFunc) (result CheckResult) {
	defer func() {
		if r := recover(); r != nil {
			result = CheckResult{
				Status:  CheckFail,
				Message: fmt.Sprintf("check panicked: %v", r),
			}
		}
	}()
	return fn()
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
