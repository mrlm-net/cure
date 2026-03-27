package doctor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	pkgdoctor "github.com/mrlm-net/cure/pkg/doctor"
)

// customCheck holds the parsed JSON definition of a single custom check.
type customCheck struct {
	Name    string `json:"name"`
	Command string `json:"command"`
	PassOn  string `json:"pass_on"`
}

// customConfig is the subset of .cure.json read by this package.
type customConfig struct {
	Doctor struct {
		Checks []customCheck `json:"checks"`
	} `json:"doctor"`
}

// loadCustomChecks reads cfgPath (typically ".cure.json") and returns one
// CheckFunc per entry in the doctor.checks array. Unknown or empty cfgPath
// fields produce no checks without error. A missing or unreadable file is
// treated as "no custom checks" (not an error).
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
		cc := cc // capture for closure
		if cc.Name == "" || cc.Command == "" {
			continue
		}
		checks = append(checks, makeCustomCheckFunc(cc))
	}
	return checks, nil
}

// makeCustomCheckFunc builds a CheckFunc from a customCheck definition.
// The command is split with strings.Fields and invoked directly (not via
// sh -c) to eliminate shell injection risk. A 10-second context timeout
// is applied per check.
func makeCustomCheckFunc(cc customCheck) pkgdoctor.CheckFunc {
	return func() pkgdoctor.CheckResult {
		argv := strings.Fields(cc.Command)
		if len(argv) == 0 {
			return pkgdoctor.CheckResult{
				Name:    cc.Name,
				Status:  pkgdoctor.CheckFail,
				Message: "empty command",
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		//nolint:gosec // argv[0] comes from .cure.json controlled by the developer
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
		out, err := cmd.Output()

		if ctx.Err() == context.DeadlineExceeded {
			return pkgdoctor.CheckResult{
				Name:    cc.Name,
				Status:  pkgdoctor.CheckWarn,
				Message: "check timed out",
			}
		}

		if err != nil {
			// Distinguish "command not found" from non-zero exit.
			if isCommandNotFound(err) {
				return pkgdoctor.CheckResult{
					Name:    cc.Name,
					Status:  pkgdoctor.CheckFail,
					Message: "command not found",
				}
			}
			// Non-zero exit: evaluate pass_on before deciding status.
		}

		if passes(cc.PassOn, err, string(out)) {
			return pkgdoctor.CheckResult{
				Name:    cc.Name,
				Status:  pkgdoctor.CheckPass,
				Message: "",
			}
		}

		msg := "non-zero exit"
		if err != nil {
			msg = err.Error()
		}
		return pkgdoctor.CheckResult{
			Name:    cc.Name,
			Status:  pkgdoctor.CheckFail,
			Message: msg,
		}
	}
}

// passes reports whether the command outcome satisfies the pass_on rule.
// Supported rules:
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
		// Unknown rule: fall back to exit_0 semantics.
		return execErr == nil
	}
}

// isCommandNotFound reports whether err indicates the binary was not found
// on $PATH. exec.ErrNotFound is returned by exec.LookPath when the binary
// does not exist; an *exec.ExitError is NOT returned in that case.
func isCommandNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), exec.ErrNotFound.Error())
}
