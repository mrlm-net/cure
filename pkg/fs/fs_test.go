package fs_test

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cfs "github.com/mrlm-net/cure/pkg/fs"
)

// ----- AtomicWrite -------------------------------------------------------

func TestAtomicWrite(t *testing.T) {
	t.Parallel()

	t.Run("creates new file with given content", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "new.txt")
		content := []byte("hello world")

		if err := cfs.AtomicWrite(path, content, 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read file: %v", err)
		}
		if !bytes.Equal(got, content) {
			t.Errorf("content mismatch: got %q, want %q", got, content)
		}
	})

	t.Run("overwrites existing file with new content", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "existing.txt")

		if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}

		newContent := []byte("updated content")
		if err := cfs.AtomicWrite(path, newContent, 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read file: %v", err)
		}
		if !bytes.Equal(got, newContent) {
			t.Errorf("content mismatch: got %q, want %q", got, newContent)
		}
	})

	t.Run("preserves existing file permissions", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "perms.txt")

		// Create file with a specific permission.
		originalPerm := fs.FileMode(0o600)
		if err := os.WriteFile(path, []byte("original"), originalPerm); err != nil {
			t.Fatalf("setup: %v", err)
		}

		// AtomicWrite with a different perm; existing perm should be preserved.
		if err := cfs.AtomicWrite(path, []byte("new"), 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if got := info.Mode().Perm(); got != originalPerm {
			t.Errorf("permission: got %o, want %o", got, originalPerm)
		}
	})

	t.Run("uses supplied perm for new file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "newperms.txt")
		perm := fs.FileMode(0o600)

		if err := cfs.AtomicWrite(path, []byte("data"), perm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if got := info.Mode().Perm(); got != perm {
			t.Errorf("permission: got %o, want %o", got, perm)
		}
	})

	t.Run("cleans up temp file on error", func(t *testing.T) {
		t.Parallel()
		// Use a non-existent parent directory to trigger a rename error.
		// The temp file must not be left behind.
		dir := t.TempDir()
		// Create a file where the target directory should be so that the
		// parent directory of path matches a real dir but the rename goes to
		// a path whose parent doesn't exist.
		nonExistentDir := filepath.Join(dir, "no-such-dir")
		path := filepath.Join(nonExistentDir, "file.txt")

		err := cfs.AtomicWrite(path, []byte("data"), 0o644)
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		// No temp files should remain in dir (the nonExistentDir was never created).
		entries, readErr := os.ReadDir(dir)
		if readErr != nil {
			t.Fatalf("read dir: %v", readErr)
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".cure-tmp-") {
				t.Errorf("temp file not cleaned up: %s", e.Name())
			}
		}
	})

	t.Run("writes empty content", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.txt")

		if err := cfs.AtomicWrite(path, []byte{}, 0o644); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if info.Size() != 0 {
			t.Errorf("expected empty file, got size %d", info.Size())
		}
	})
}

// ----- EnsureDir ---------------------------------------------------------

func TestEnsureDir(t *testing.T) {
	t.Parallel()

	t.Run("creates directory that does not exist", func(t *testing.T) {
		t.Parallel()
		base := t.TempDir()
		newDir := filepath.Join(base, "a", "b", "c")

		if err := cfs.EnsureDir(newDir, 0o755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := os.Stat(newDir)
		if err != nil {
			t.Fatalf("stat after EnsureDir: %v", err)
		}
		if !info.IsDir() {
			t.Error("expected directory, got non-directory")
		}
	})

	t.Run("returns nil for existing directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		if err := cfs.EnsureDir(dir, 0o755); err != nil {
			t.Fatalf("unexpected error for existing dir: %v", err)
		}
	})

	t.Run("returns error when path exists as a file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		file := filepath.Join(dir, "not-a-dir.txt")

		if err := os.WriteFile(file, []byte("content"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}

		if err := cfs.EnsureDir(file, 0o755); err == nil {
			t.Fatal("expected error for file path, got nil")
		}
	})

	t.Run("creates nested directories in one call", func(t *testing.T) {
		t.Parallel()
		base := t.TempDir()
		deep := filepath.Join(base, "x", "y", "z", "w")

		if err := cfs.EnsureDir(deep, 0o700); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(deep); err != nil {
			t.Fatalf("directory not created: %v", err)
		}
	})
}

// ----- Exists ------------------------------------------------------------

func TestExists(t *testing.T) {
	t.Parallel()

	t.Run("returns true for existing file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		file := filepath.Join(dir, "present.txt")

		if err := os.WriteFile(file, []byte("data"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}

		got, err := cfs.Exists(file)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Error("expected true for existing file, got false")
		}
	})

	t.Run("returns true for existing directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		got, err := cfs.Exists(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !got {
			t.Error("expected true for existing directory, got false")
		}
	})

	t.Run("returns false without error for absent path", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		absent := filepath.Join(dir, "no-such-file")

		got, err := cfs.Exists(absent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Error("expected false for absent path, got true")
		}
	})

	t.Run("returns false without error for deeply absent path", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		absent := filepath.Join(dir, "a", "b", "c", "d")

		got, err := cfs.Exists(absent)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got {
			t.Error("expected false for deeply absent path, got true")
		}
	})
}

// ----- TempDir -----------------------------------------------------------

func TestTempDir(t *testing.T) {
	t.Parallel()

	t.Run("creates directory with given prefix", func(t *testing.T) {
		t.Parallel()
		prefix := "cure-test-"

		dir, err := cfs.TempDir(prefix)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		t.Cleanup(func() { os.RemoveAll(dir) })

		if !strings.Contains(filepath.Base(dir), prefix) {
			t.Errorf("directory name %q does not contain prefix %q", dir, prefix)
		}

		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if !info.IsDir() {
			t.Error("TempDir did not create a directory")
		}
	})

	t.Run("creates unique directories on successive calls", func(t *testing.T) {
		t.Parallel()
		prefix := "cure-uniq-"

		dir1, err := cfs.TempDir(prefix)
		if err != nil {
			t.Fatalf("first call: %v", err)
		}
		t.Cleanup(func() { os.RemoveAll(dir1) })

		dir2, err := cfs.TempDir(prefix)
		if err != nil {
			t.Fatalf("second call: %v", err)
		}
		t.Cleanup(func() { os.RemoveAll(dir2) })

		if dir1 == dir2 {
			t.Errorf("expected unique directories, got %q twice", dir1)
		}
	})

	t.Run("empty prefix is accepted", func(t *testing.T) {
		t.Parallel()
		dir, err := cfs.TempDir("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		t.Cleanup(func() { os.RemoveAll(dir) })

		if dir == "" {
			t.Error("expected non-empty path")
		}
	})
}

// ----- SetPermissions ----------------------------------------------------

func TestSetPermissions(t *testing.T) {
	t.Parallel()

	t.Run("sets permissions on a file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		file := filepath.Join(dir, "target.txt")

		if err := os.WriteFile(file, []byte("data"), 0o644); err != nil {
			t.Fatalf("setup: %v", err)
		}

		wantPerm := fs.FileMode(0o600)
		if err := cfs.SetPermissions(file, wantPerm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := os.Stat(file)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if got := info.Mode().Perm(); got != wantPerm {
			t.Errorf("permission: got %o, want %o", got, wantPerm)
		}
	})

	t.Run("sets permissions on a directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		subdir := filepath.Join(dir, "sub")

		if err := os.Mkdir(subdir, 0o755); err != nil {
			t.Fatalf("setup: %v", err)
		}

		wantPerm := fs.FileMode(0o700)
		if err := cfs.SetPermissions(subdir, wantPerm); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		info, err := os.Stat(subdir)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if got := info.Mode().Perm(); got != wantPerm {
			t.Errorf("permission: got %o, want %o", got, wantPerm)
		}
	})

	t.Run("returns error for non-existent path", func(t *testing.T) {
		t.Parallel()
		absent := filepath.Join(t.TempDir(), "no-such-file")

		if err := cfs.SetPermissions(absent, 0o644); err == nil {
			t.Fatal("expected error for non-existent path, got nil")
		}
	})
}
