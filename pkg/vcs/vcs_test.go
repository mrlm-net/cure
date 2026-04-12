package vcs

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run(dir, "init")
	run(dir, "config", "user.email", "test@test.com")
	run(dir, "config", "user.name", "Test")
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test"), 0644)
	run(dir, "add", ".")
	run(dir, "commit", "-m", "initial")
	return dir
}

func TestStatus(t *testing.T) {
	dir := initRepo(t)

	s, err := Status(dir)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !s.Clean {
		t.Error("expected clean repo")
	}

	// Create an untracked file
	os.WriteFile(filepath.Join(dir, "new.txt"), []byte("x"), 0644)
	s, _ = Status(dir)
	if s.Clean {
		t.Error("expected dirty repo")
	}
	if len(s.Untracked) != 1 {
		t.Errorf("untracked = %d, want 1", len(s.Untracked))
	}
}

func TestBranch(t *testing.T) {
	dir := initRepo(t)

	if err := Branch(dir, "feat/test"); err != nil {
		t.Fatalf("Branch: %v", err)
	}

	branch, _ := CurrentBranch(dir)
	if branch != "feat/test" {
		t.Errorf("branch = %q, want %q", branch, "feat/test")
	}
}

func TestCommit(t *testing.T) {
	dir := initRepo(t)

	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
	run(dir, "add", ".")

	err := Commit(dir, "feat: add file")
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}
}

func TestCommitWithValidation(t *testing.T) {
	dir := initRepo(t)

	os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644)
	run(dir, "add", ".")

	pattern := `^(feat|fix): .+`

	err := Commit(dir, "feat: valid", WithValidatePattern(pattern))
	if err != nil {
		t.Fatalf("valid commit: %v", err)
	}

	os.WriteFile(filepath.Join(dir, "file2.txt"), []byte("x"), 0644)
	run(dir, "add", ".")

	err = Commit(dir, "bad message", WithValidatePattern(pattern))
	if err == nil {
		t.Fatal("invalid commit should fail")
	}
}

func TestDiff(t *testing.T) {
	dir := initRepo(t)

	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# modified"), 0644)

	d, err := Diff(dir)
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if len(d.Files) == 0 {
		t.Error("expected diff files")
	}
}

func TestLog(t *testing.T) {
	dir := initRepo(t)

	entries, err := Log(dir, 5)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("got %d entries, want 1", len(entries))
	}
	if entries[0].Subject != "initial" {
		t.Errorf("subject = %q, want %q", entries[0].Subject, "initial")
	}
}

func TestIsDirty(t *testing.T) {
	dir := initRepo(t)

	dirty, _ := IsDirty(dir)
	if dirty {
		t.Error("clean repo should not be dirty")
	}

	os.WriteFile(filepath.Join(dir, "x.txt"), []byte("x"), 0644)
	dirty, _ = IsDirty(dir)
	if !dirty {
		t.Error("modified repo should be dirty")
	}
}

func TestRunNotGitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}
	_, err := Status(t.TempDir())
	if err == nil {
		t.Error("expected error for non-git directory")
	}
}
