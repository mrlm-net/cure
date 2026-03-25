package fs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// AtomicWrite writes content to path using a temp-file-then-rename sequence,
// ensuring that concurrent readers always observe either the complete old
// content or the complete new content — never a partial write.
//
// The algorithm:
//  1. Create a temp file in filepath.Dir(path) with prefix ".cure-tmp-".
//  2. Write content to the temp file.
//  3. Fsync the temp file to flush data from the OS buffer to disk.
//  4. If path already exists, stat it and inherit its permissions; otherwise
//     use perm for the new file.
//  5. Set the temp file's permissions to match.
//  6. Rename the temp file over path (atomic on POSIX).
//
// On any error, the temp file is removed (best effort) so it does not accumulate.
//
// Limitation: rename is atomic only within the same filesystem. AtomicWrite
// always creates the temp file in the same directory as path to guarantee
// this, but crossing a mount point boundary will return an error.
func AtomicWrite(path string, content []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".cure-tmp-*")
	if err != nil {
		return fmt.Errorf("atomic write %s: create temp file: %w", path, err)
	}
	tmpName := tmp.Name()

	// Track whether the write succeeded so the deferred cleanup knows whether
	// to remove the temp file.
	committed := false
	defer func() {
		if !committed {
			os.Remove(tmpName) // best-effort cleanup
		}
	}()

	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return fmt.Errorf("atomic write %s: write temp file: %w", path, err)
	}

	// Flush OS buffers to disk before the rename so that a crash after rename
	// does not leave the new file with zero bytes.
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("atomic write %s: sync temp file: %w", path, err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("atomic write %s: close temp file: %w", path, err)
	}

	// Resolve effective permissions: inherit from the existing target when present
	// so that a rewrite does not silently downgrade (or upgrade) access rights.
	effectivePerm := perm
	if info, err := os.Stat(path); err == nil {
		effectivePerm = info.Mode().Perm()
	}

	if err := os.Chmod(tmpName, effectivePerm); err != nil {
		return fmt.Errorf("atomic write %s: chmod temp file: %w", path, err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("atomic write %s: rename: %w", path, err)
	}

	committed = true
	return nil
}

// EnsureDir creates dir and all necessary parent directories with the given
// permissions. It returns nil if the directory already exists. It returns an
// error if path exists and is NOT a directory.
func EnsureDir(path string, perm fs.FileMode) error {
	info, err := os.Stat(path)
	if err == nil {
		// Path exists — verify it is a directory.
		if !info.IsDir() {
			return fmt.Errorf("ensure dir %s: path exists and is not a directory", path)
		}
		return nil
	}

	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("ensure dir %s: %w", path, err)
	}
	return nil
}

// Exists reports whether path exists in the filesystem.
//
// Return values:
//   - (true, nil)  — path exists (file, directory, or other entry).
//   - (false, nil) — path does not exist.
//   - (false, err) — an I/O or permission error occurred; the caller should
//     treat the existence as unknown.
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	// Permission denied or other real I/O error — propagate.
	return false, fmt.Errorf("exists %s: %w", path, err)
}

// TempDir creates a new temporary directory inside os.TempDir() with a name
// beginning with prefix and returns the directory's path. The caller is
// responsible for removing the directory when it is no longer needed (e.g.,
// via os.RemoveAll).
func TempDir(prefix string) (string, error) {
	dir, err := os.MkdirTemp(os.TempDir(), prefix)
	if err != nil {
		return "", fmt.Errorf("temp dir: %w", err)
	}
	return dir, nil
}

// SetPermissions sets the permission bits of path to mode.
// It is a thin wrapper around os.Chmod with a consistent error message.
func SetPermissions(path string, mode fs.FileMode) error {
	if err := os.Chmod(path, mode); err != nil {
		return fmt.Errorf("set permissions %s: %w", path, err)
	}
	return nil
}
