package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pkgdoctor "github.com/mrlm-net/cure/pkg/doctor"
)

// writeTempConfig writes content to a temporary .cure.json and returns its path.
func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ".cure.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeTempConfig: %v", err)
	}
	return path
}

func TestLoadCustomChecks(t *testing.T) {
	t.Run("missing file returns no checks and no error", func(t *testing.T) {
		checks, err := loadCustomChecks("/nonexistent/path/.cure.json")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(checks) != 0 {
			t.Errorf("want 0 checks, got %d", len(checks))
		}
	})

	t.Run("empty doctor.checks array returns no checks", func(t *testing.T) {
		path := writeTempConfig(t, `{"doctor":{"checks":[]}}`)
		checks, err := loadCustomChecks(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(checks) != 0 {
			t.Errorf("want 0 checks, got %d", len(checks))
		}
	})

	t.Run("check with empty name is skipped", func(t *testing.T) {
		path := writeTempConfig(t, `{"doctor":{"checks":[{"name":"","command":"true","pass_on":"exit_0"}]}}`)
		checks, err := loadCustomChecks(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(checks) != 0 {
			t.Errorf("want 0 checks (empty name skipped), got %d", len(checks))
		}
	})

	t.Run("check with empty command is skipped", func(t *testing.T) {
		path := writeTempConfig(t, `{"doctor":{"checks":[{"name":"My check","command":"","pass_on":"exit_0"}]}}`)
		checks, err := loadCustomChecks(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(checks) != 0 {
			t.Errorf("want 0 checks (empty command skipped), got %d", len(checks))
		}
	})

	t.Run("two valid checks returns two CheckFuncs", func(t *testing.T) {
		path := writeTempConfig(t, `{"doctor":{"checks":[
			{"name":"A","command":"true","pass_on":"exit_0"},
			{"name":"B","command":"true","pass_on":"exit_0"}
		]}}`)
		checks, err := loadCustomChecks(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(checks) != 2 {
			t.Errorf("want 2 checks, got %d", len(checks))
		}
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		path := writeTempConfig(t, `{not valid json`)
		_, err := loadCustomChecks(path)
		if err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})

	t.Run("unknown pass_on rule returns error", func(t *testing.T) {
		path := writeTempConfig(t, `{"doctor":{"checks":[{"name":"bad","command":"true","pass_on":"stdout_Contains:hello"}]}}`)
		_, err := loadCustomChecks(path)
		if err == nil {
			t.Fatal("expected error for unknown pass_on rule, got nil")
		}
		if !strings.Contains(err.Error(), "unknown pass_on rule") {
			t.Errorf("error %q should mention 'unknown pass_on rule'", err.Error())
		}
	})
}

func TestMakeCustomCheckFunc_ExitZero(t *testing.T) {
	t.Run("exit_0 — true command passes and message contains name", func(t *testing.T) {
		cc := customCheck{Name: "always pass", Command: "true", PassOn: "exit_0"}
		result := makeCustomCheckFunc(cc)()
		if result.Status != pkgdoctor.CheckPass {
			t.Errorf("status = %v, want CheckPass", result.Status)
		}
		if !strings.Contains(result.Message, cc.Name) {
			t.Errorf("message %q should contain check name %q", result.Message, cc.Name)
		}
	})

	t.Run("exit_0 — false command fails", func(t *testing.T) {
		cc := customCheck{Name: "always fail", Command: "false", PassOn: "exit_0"}
		result := makeCustomCheckFunc(cc)()
		if result.Status != pkgdoctor.CheckFail {
			t.Errorf("status = %v, want CheckFail", result.Status)
		}
	})
}

func TestMakeCustomCheckFunc_StdoutContains(t *testing.T) {
	t.Run("stdout_contains — match passes", func(t *testing.T) {
		cc := customCheck{
			Name:    "echo check",
			Command: "echo hello world",
			PassOn:  "stdout_contains:hello",
		}
		result := makeCustomCheckFunc(cc)()
		if result.Status != pkgdoctor.CheckPass {
			t.Errorf("status = %v, want CheckPass; message: %s", result.Status, result.Message)
		}
	})

	t.Run("stdout_contains — no match fails", func(t *testing.T) {
		cc := customCheck{
			Name:    "echo check",
			Command: "echo hello world",
			PassOn:  "stdout_contains:notpresent",
		}
		result := makeCustomCheckFunc(cc)()
		if result.Status != pkgdoctor.CheckFail {
			t.Errorf("status = %v, want CheckFail", result.Status)
		}
	})
}

func TestMakeCustomCheckFunc_CommandNotFound(t *testing.T) {
	cc := customCheck{
		Name:    "no such binary",
		Command: "cure_test_binary_that_does_not_exist_xyz",
		PassOn:  "exit_0",
	}
	result := makeCustomCheckFunc(cc)()
	if result.Status != pkgdoctor.CheckFail {
		t.Errorf("status = %v, want CheckFail", result.Status)
	}
	if !strings.Contains(result.Message, "command not found") {
		t.Errorf("message %q should contain 'command not found'", result.Message)
	}
}

func TestPasses(t *testing.T) {
	tests := []struct {
		name    string
		passOn  string
		execErr error
		stdout  string
		want    bool
	}{
		{"exit_0 success", "exit_0", nil, "", true},
		{"exit_0 failure", "exit_0", fmt.Errorf("exit 1"), "", false},
		{"stdout_contains match", "stdout_contains:hello", nil, "hello world", true},
		{"stdout_contains no match", "stdout_contains:nope", nil, "hello world", false},
		// unknown rule unreachable in normal flow (loadCustomChecks validates),
		// but passes returns false defensively.
		{"unknown rule returns false", "unknown_rule", nil, "anything", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := passes(tt.passOn, tt.execErr, tt.stdout)
			if got != tt.want {
				t.Errorf("passes(%q, %v, %q) = %v, want %v", tt.passOn, tt.execErr, tt.stdout, got, tt.want)
			}
		})
	}
}

// TestTimeoutIs10Seconds verifies the 10-second per-check timeout produces
// CheckWarn. Skipped under -short because it takes approximately 10 seconds.
func TestTimeoutIs10Seconds(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: requires ~10s for timeout to fire; run without -short to include")
	}
	if _, err := os.Stat("/bin/sleep"); err != nil {
		t.Skip("sleep not available")
	}

	const expected = 10 * time.Second
	start := time.Now()
	cc := customCheck{Name: "sleep 11", Command: "sleep 11", PassOn: "exit_0"}
	result := makeCustomCheckFunc(cc)()
	elapsed := time.Since(start)

	if result.Status != pkgdoctor.CheckWarn {
		t.Errorf("status = %v, want CheckWarn (timed out)", result.Status)
	}
	if !strings.Contains(result.Message, "timed out") {
		t.Errorf("message %q should contain 'timed out'", result.Message)
	}
	// Should complete in roughly 10s (allow 5s margin for slow CI).
	if elapsed > expected+5*time.Second {
		t.Errorf("timeout took %v, expected ~%v", elapsed, expected)
	}
}
