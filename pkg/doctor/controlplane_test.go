package doctor

import "testing"

func TestControlPlaneChecks(t *testing.T) {
	checks := ControlPlaneChecks()
	if len(checks) != 5 {
		t.Errorf("got %d checks, want 5", len(checks))
	}

	// Run each check — should not panic
	for _, check := range checks {
		result := check()
		if result.Name == "" {
			t.Error("check returned empty name")
		}
		if result.Status != CheckPass && result.Status != CheckWarn && result.Status != CheckFail {
			t.Errorf("check %q returned invalid status %d", result.Name, result.Status)
		}
	}
}

func TestCheckGit(t *testing.T) {
	result := CheckGit()
	if result.Name != "Git" {
		t.Errorf("name = %q, want Git", result.Name)
	}
	// Git should be available in any dev environment
	if result.Status != CheckPass {
		t.Logf("git not found: %s", result.Message)
	}
}

func TestCheckCureConfig(t *testing.T) {
	result := CheckCureConfig()
	if result.Name != "Cure Config" {
		t.Errorf("name = %q, want Cure Config", result.Name)
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		n    int
		word string
		want string
	}{
		{0, "project", "0 projects"},
		{1, "project", "1 project"},
		{5, "repo", "5 repos"},
	}
	for _, tt := range tests {
		got := pluralize(tt.n, tt.word)
		if got != tt.want {
			t.Errorf("pluralize(%d, %q) = %q, want %q", tt.n, tt.word, got, tt.want)
		}
	}
}
