package project

import "testing"

func TestValidateBranch(t *testing.T) {
	pattern := `^(feat|fix|docs|refactor|test|chore)/\d+-.*$`

	tests := []struct {
		name    string
		branch  string
		pattern string
		wantErr bool
	}{
		{"valid feat", "feat/123-add-feature", pattern, false},
		{"valid fix", "fix/456-bug-fix", pattern, false},
		{"invalid no prefix", "my-branch", pattern, true},
		{"invalid no number", "feat/add-feature", pattern, true},
		{"empty pattern", "anything", "", false},
		{"invalid regex", "test", "[invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBranch(tt.branch, tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBranch(%q, %q) error = %v, wantErr %v",
					tt.branch, tt.pattern, err, tt.wantErr)
			}
		})
	}
}

func TestValidateCommit(t *testing.T) {
	pattern := `^(feat|fix|docs|test|refactor|chore)(\(.+\))?!?: .+`

	tests := []struct {
		name    string
		message string
		pattern string
		wantErr bool
	}{
		{"valid feat", "feat: add feature", pattern, false},
		{"valid scoped", "feat(gui): add editor", pattern, false},
		{"valid breaking", "feat!: breaking change", pattern, false},
		{"invalid no type", "add feature", pattern, true},
		{"invalid wrong type", "bugfix: something", pattern, true},
		{"empty pattern", "anything", "", false},
		{"invalid regex", "test", "[invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCommit(tt.message, tt.pattern)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommit(%q, %q) error = %v, wantErr %v",
					tt.message, tt.pattern, err, tt.wantErr)
			}
		})
	}
}

func TestIsProtected(t *testing.T) {
	protected := []string{"main", "release/*", "hotfix/*"}

	tests := []struct {
		name   string
		branch string
		want   bool
	}{
		{"main exact", "main", true},
		{"release glob", "release/v1.0", true},
		{"release nested", "release/v1.0.1", true},
		{"hotfix glob", "hotfix/urgent", true},
		{"feature branch", "feat/123-stuff", false},
		{"not main prefix", "main-copy", false},
		{"empty branch", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsProtected(tt.branch, protected)
			if got != tt.want {
				t.Errorf("IsProtected(%q) = %v, want %v", tt.branch, got, tt.want)
			}
		})
	}
}

func TestIsProtectedEmptyList(t *testing.T) {
	if IsProtected("main", nil) {
		t.Error("expected false for nil protected list")
	}
	if IsProtected("main", []string{}) {
		t.Error("expected false for empty protected list")
	}
}
