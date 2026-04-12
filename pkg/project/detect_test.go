package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)

	repoDir := t.TempDir()
	subDir := filepath.Join(repoDir, "sub", "dir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("create subdir: %v", err)
	}

	p := &Project{
		Name:  "detect-test",
		Repos: []Repo{{Path: repoDir}},
	}
	if err := st.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	det := NewDetector(st)

	tests := []struct {
		name    string
		cwd     string
		wantNil bool
	}{
		{"exact repo path", repoDir, false},
		{"subdirectory of repo", subDir, false},
		{"unrelated path", t.TempDir(), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found, err := det.Detect(tt.cwd)
			if err != nil {
				t.Fatalf("Detect(%q): %v", tt.cwd, err)
			}
			if tt.wantNil && found != nil {
				t.Errorf("expected nil, got project %q", found.Name)
			}
			if !tt.wantNil && found == nil {
				t.Error("expected project, got nil")
			}
			if !tt.wantNil && found != nil && found.Name != "detect-test" {
				t.Errorf("Name = %q, want %q", found.Name, "detect-test")
			}
		})
	}
}

func TestDetectNoProjects(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)
	det := NewDetector(st)

	found, err := det.Detect(t.TempDir())
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if found != nil {
		t.Errorf("expected nil, got %v", found)
	}
}

func TestDetectMultipleRepos(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)

	repo1 := t.TempDir()
	repo2 := t.TempDir()

	p := &Project{
		Name: "multi-repo",
		Repos: []Repo{
			{Path: repo1},
			{Path: repo2},
		},
	}
	if err := st.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	det := NewDetector(st)

	found, err := det.Detect(repo2)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if found == nil {
		t.Fatal("expected project for repo2")
	}
	if found.Name != "multi-repo" {
		t.Errorf("Name = %q, want %q", found.Name, "multi-repo")
	}
}

func TestIsSubdir(t *testing.T) {
	tests := []struct {
		name   string
		parent string
		child  string
		want   bool
	}{
		{"exact match", "/a/b", "/a/b", true},
		{"subdirectory", "/a/b", "/a/b/c", true},
		{"deep subdirectory", "/a/b", "/a/b/c/d/e", true},
		{"not subdir", "/a/b", "/a/c", false},
		{"prefix but not subdir", "/a/b", "/a/bc", false},
		{"parent is longer", "/a/b/c", "/a/b", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSubdir(tt.parent, tt.child)
			if got != tt.want {
				t.Errorf("isSubdir(%q, %q) = %v, want %v", tt.parent, tt.child, got, tt.want)
			}
		})
	}
}
