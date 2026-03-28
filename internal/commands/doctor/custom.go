package doctor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	pkgdoctor "github.com/mrlm-net/cure/pkg/doctor"
)

// customCheck holds the parsed JSON definition of a single custom check.
// The command field is split with strings.Fields and invoked directly
// (no sh -c). Quoted arguments are NOT supported; arguments containing
// spaces cannot be passed through this interface.
type customCheck struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	// PassOn specifies the success condition: "exit_0" (command exits with
	// code 0) or "stdout_contains:<pattern>" (stdout contains the literal
	// pattern). Any other value is rejected at load time.
	PassOn string `json:"pass_on"`
}

// customConfig is the subset of .cure.json read by this package.
type customConfig struct {
	Doctor struct {
		Checks []customCheck `json:"checks"`
	} `json:"doctor"`
}

// loadCustomChecks reads cfgPath (typically ".cure.json") and returns one
// CheckFunc per entry in the doctor.checks array. Entries with empty name
// or command are skipped silently. A missing file returns nil, nil (opt-in,
// not required). An unreadable or unparseable file returns an error.
func loadCustomChecks(cfgPath string) ([]pkgdoctor.CheckFunc, error) {
	data, err := os.ReadFile(cfgPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("custom checks: read %s: %w", cfgPath, err)
	}

	var cfg customConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("custom checks: parse %s: %w", cfgPath, err)
	}

	checks := make([]pkgdoctor.CheckFunc, 0, len(cfg.Doctor.Checks))
	for _, cc := range cfg.Doctor.Checks {
		if cc.Name == "" || cc.Command == "" {
			continue
		}
		// Validate pass_on at load time to surface typos immediately.
		if cc.PassOn != "exit_0" && !strings.HasPrefix(cc.PassOn, "stdout_contains:") {
			return nil, fmt.Errorf("custom checks: entry %q has unknown pass_on rule %q; valid: \"exit_0\", \"stdout_contains:<pattern>\"", cc.Name, cc.PassOn)
		}
		checks = append(checks, makeCustomCheckFunc(cc))
	}
	return checks, nil
}

// makeCustomCheckFunc builds a CheckFunc from a customCheck definition.
// The command is split with strings.Fields and invoked directly (not via
// sh -c) to eliminate shell injection risk. A 10-second timeout is applied
// per check; the context is rooted at context.Background() so it cannot be
// cancelled by an external caller — only the built-in timeout fires.
func makeCustomCheckFunc(cc customCheck) pkgdoctor.CheckFunc {
	return func() pkgdoctor.CheckResult {
		argv := strings.Fields(cc.Command)
		if len(argv) == 0 {
			return pkgdoctor.CheckResult{
				Name:    cc.Name,
				Status:  pkgdoctor.CheckFail,
				Message: cc.Name + ": empty command",
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		//nolint:gosec // argv[0] comes from .cure.json controlled by the developer
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
		out, err := cmd.Output()

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return pkgdoctor.CheckResult{
				Name:    cc.Name,
				Status:  pkgdoctor.CheckWarn,
				Message: cc.Name + ": check timed out",
			}
		}

		if err != nil {
			// Distinguish "command not found" from non-zero exit.
			if isCommandNotFound(err) {
				return pkgdoctor.CheckResult{
					Name:    cc.Name,
					Status:  pkgdoctor.CheckFail,
					Message: cc.Name + ": command not found",
				}
			}
			// Non-zero exit: evaluate pass_on before deciding status.
		}

		if passes(cc.PassOn, err, string(out)) {
			return pkgdoctor.CheckResult{
				Name:   cc.Name,
				Status: pkgdoctor.CheckPass,
				// Message carries the name so the output line is readable.
				// pkg/doctor.Run renders only r.Message; embedding the name
				// here ensures custom checks appear identically to built-ins.
				Message: cc.Name,
			}
		}

		msg := cc.Name + ": non-zero exit"
		if err != nil {
			msg = cc.Name + ": " + err.Error()
		}
		return pkgdoctor.CheckResult{
			Name:    cc.Name,
			Status:  pkgdoctor.CheckFail,
			Message: msg,
		}
	}
}

// passes reports whether the command outcome satisfies the pass_on rule.
// Supported rules (validated at load time by loadCustomChecks):
//   - "exit_0"                    — command must exit with code 0
//   - "stdout_contains:<pattern>" — stdout must contain the literal pattern
func passes(passOn string, execErr error, stdout string) bool {
	switch {
	case passOn == "exit_0":
		return execErr == nil
	case strings.HasPrefix(passOn, "stdout_contains:"):
		pattern := strings.TrimPrefix(passOn, "stdout_contains:")
		return strings.Contains(stdout, pattern)
	default:
		// Unreachable: loadCustomChecks validates pass_on before creating checks.
		return false
	}
}

// isCommandNotFound reports whether err indicates the binary was not found
// on $PATH. Uses errors.As to inspect the *exec.Error wrapper structurally
// rather than relying on string comparison.
func isCommandNotFound(err error) bool {
	var execErr *exec.Error
	return errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound)
}
