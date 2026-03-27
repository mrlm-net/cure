package doctor_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pkgdoctor "github.com/mrlm-net/cure/pkg/doctor"
	"github.com/mrlm-net/cure/pkg/style"
)

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

// chdir changes the working directory to dir for the duration of the test.
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
// Run tally tests
// --------------------------------------------------------------------------

func TestRun_AllPass(t *testing.T) {
	style.Disable()
	t.Cleanup(style.Enable)

	checks := []pkgdoctor.CheckFunc{
		func() pkgdoctor.CheckResult {
			return pkgdoctor.CheckResult{Status: pkgdoctor.CheckPass, Message: "ok1"}
		},
		func() pkgdoctor.CheckResult {
			return pkgdoctor.CheckResult{Status: pkgdoctor.CheckPass, Message: "ok2"}
		},
		func() pkgdoctor.CheckResult {
			return pkgdoctor.CheckResult{Status: pkgdoctor.CheckPass, Message: "ok3"}
		},
	}

	var buf bytes.Buffer
	passed, warned, failed := pkgdoctor.Run(checks, &buf)

	if passed != 3 {
		t.Errorf("passed = %d, want 3", passed)
	}
	if warned != 0 {
		t.Errorf("warned = %d, want 0", warned)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}
}

func TestRun_AllWarn(t *testing.T) {
	style.Disable()
	t.Cleanup(style.Enable)

	checks := []pkgdoctor.CheckFunc{
		func() pkgdoctor.CheckResult {
			return pkgdoctor.CheckResult{Status: pkgdoctor.CheckWarn, Message: "warn1"}
		},
		func() pkgdoctor.CheckResult {
			return pkgdoctor.CheckResult{Status: pkgdoctor.CheckWarn, Message: "warn2"}
		},
	}

	var buf bytes.Buffer
	passed, warned, failed := pkgdoctor.Run(checks, &buf)

	if passed != 0 {
		t.Errorf("passed = %d, want 0", passed)
	}
	if warned != 2 {
		t.Errorf("warned = %d, want 2", warned)
	}
	if failed != 0 {
		t.Errorf("failed = %d, want 0", failed)
	}
}

func TestRun_AllFail(t *testing.T) {
	style.Disable()
	t.Cleanup(style.Enable)

	checks := []pkgdoctor.CheckFunc{
		func() pkgdoctor.CheckResult {
			return pkgdoctor.CheckResult{Status: pkgdoctor.CheckFail, Message: "fail1"}
		},
		func() pkgdoctor.CheckResult {
			return pkgdoctor.CheckResult{Status: pkgdoctor.CheckFail, Message: "fail2"}
		},
	}

	var buf bytes.Buffer
	passed, warned, failed := pkgdoctor.Run(checks, &buf)

	if passed != 0 {
		t.Errorf("passed = %d, want 0", passed)
	}
	if warned != 0 {
		t.Errorf("warned = %d, want 0", warned)
	}
	if failed != 2 {
		t.Errorf("failed = %d, want 2", failed)
	}
}

func TestRun_Mixed(t *testing.T) {
	style.Disable()
	t.Cleanup(style.Enable)

	checks := []pkgdoctor.CheckFunc{
		func() pkgdoctor.CheckResult {
			return pkgdoctor.CheckResult{Status: pkgdoctor.CheckPass, Message: "pass"}
		},
		func() pkgdoctor.CheckResult {
			return pkgdoctor.CheckResult{Status: pkgdoctor.CheckWarn, Message: "warn"}
		},
		func() pkgdoctor.CheckResult {
			return pkgdoctor.CheckResult{Status: pkgdoctor.CheckFail, Message: "fail"}
		},
	}

	var buf bytes.Buffer
	passed, warned, failed := pkgdoctor.Run(checks, &buf)

	if passed != 1 {
		t.Errorf("passed = %d, want 1", passed)
	}
	if warned != 1 {
		t.Errorf("warned = %d, want 1", warned)
	}
	if failed != 1 {
		t.Errorf("failed = %d, want 1", failed)
	}
}

func TestRun_EmptySlice(t *testing.T) {
	var buf bytes.Buffer
	passed, warned, failed := pkgdoctor.Run(nil, &buf)

	if passed != 0 || warned != 0 || failed != 0 {
		t.Errorf("Run(nil) = (%d,%d,%d), want (0,0,0)", passed, warned, failed)
	}

	var buf2 bytes.Buffer
	passed2, warned2, failed2 := pkgdoctor.Run([]pkgdoctor.CheckFunc{}, &buf2)
	if passed2 != 0 || warned2 != 0 || failed2 != 0 {
		t.Errorf("Run([]) = (%d,%d,%d), want (0,0,0)", passed2, warned2, failed2)
	}
}

func TestRun_PanicRecovery(t *testing.T) {
	style.Disable()
	t.Cleanup(style.Enable)

	afterPanic := false
	checks := []pkgdoctor.CheckFunc{
		func() pkgdoctor.CheckResult {
			return pkgdoctor.CheckResult{Status: pkgdoctor.CheckPass, Message: "before"}
		},
		func() pkgdoctor.CheckResult {
			panic("something went wrong")
		},
		func() pkgdoctor.CheckResult {
			afterPanic = true
			return pkgdoctor.CheckResult{Status: pkgdoctor.CheckPass, Message: "after"}
		},
	}

	var buf bytes.Buffer
	passed, warned, failed := pkgdoctor.Run(checks, &buf)

	if passed != 2 {
		t.Errorf("passed = %d, want 2", passed)
	}
	if warned != 0 {
		t.Errorf("warned = %d, want 0", warned)
	}
	if failed != 1 {
		t.Errorf("failed = %d, want 1", failed)
	}
	if !afterPanic {
		t.Error("check after panicking check was not executed — panic recovery halted iteration")
	}

	out := buf.String()
	if !strings.Contains(out, "check panicked:") {
		t.Errorf("output does not contain 'check panicked:': %s", out)
	}
	if !strings.Contains(out, "something went wrong") {
		t.Errorf("output does not contain panic value: %s", out)
	}
}

// --------------------------------------------------------------------------
// Built-in check tests
// --------------------------------------------------------------------------

func TestCheckREADME(t *testing.T) {
	tests := []struct {
		name    string
		files   []string
		wantSt  pkgdoctor.CheckStatus
		wantMsg string
	}{
		{
			name:    "README.md present",
			files:   []string{"README.md"},
			wantSt:  pkgdoctor.CheckPass,
			wantMsg: "README.md found",
		},
		{
			name:    "plain README present",
			files:   []string{"README"},
			wantSt:  pkgdoctor.CheckPass,
			wantMsg: "README found",
		},
		{
			name:    "no README",
			files:   []string{},
			wantSt:  pkgdoctor.CheckFail,
			wantMsg: "README not found",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir(t)
			for _, f := range tt.files {
				touch(t, f)
			}
			r := pkgdoctor.CheckREADME()
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
		wantSt pkgdoctor.CheckStatus
	}{
		{
			name:   ".gitignore present",
			files:  []string{".gitignore"},
			wantSt: pkgdoctor.CheckPass,
		},
		{
			name:   ".gitignore missing — warn, not fail",
			files:  []string{},
			wantSt: pkgdoctor.CheckWarn,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir(t)
			for _, f := range tt.files {
				touch(t, f)
			}
			r := pkgdoctor.CheckGitignore()
			if r.Status != tt.wantSt {
				t.Errorf("status = %v, want %v", r.Status, tt.wantSt)
			}
		})
	}
}

func TestBuiltinChecks(t *testing.T) {
	checks := pkgdoctor.BuiltinChecks()
	if len(checks) != 7 {
		t.Errorf("BuiltinChecks() returned %d checks, want 7", len(checks))
	}

	// Verify BuiltinChecks returns a new slice on each call (mutation safety).
	checks1 := pkgdoctor.BuiltinChecks()
	checks2 := pkgdoctor.BuiltinChecks()
	checks1[0] = nil
	if checks2[0] == nil {
		t.Error("BuiltinChecks() shares backing array — mutation of one slice affects another")
	}
}
