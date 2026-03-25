// Package fs provides safe and atomic filesystem operations for CLI tools.
//
// This package wraps common filesystem operations with correct error handling,
// atomic write semantics, and cross-platform compatibility (POSIX systems).
//
// # Atomic Writes
//
// AtomicWrite performs a write-then-rename sequence to prevent partial writes
// from leaving files in a corrupt state. The content is written to a temporary
// file in the same directory as the target, fsynced to disk, and then renamed
// over the target. On POSIX systems, rename is atomic with respect to the
// directory entry — readers always see either the old content or the new
// content, never a partial write.
//
// Limitation: rename is atomic only when the source and destination are on the
// same filesystem. AtomicWrite creates the temporary file in filepath.Dir(path)
// to ensure this, but crossing a mount point boundary is not supported and will
// return an error.
//
// # Permissions
//
// AtomicWrite preserves existing file permissions when the target already exists.
// The caller-supplied perm is used only when creating a new file. To change an
// existing file's permissions, use SetPermissions explicitly.
//
// # Usage
//
//	// Write a config file atomically.
//	err := fs.AtomicWrite("/etc/app/config.json", data, 0o600)
//
//	// Ensure a directory exists before writing.
//	if err := fs.EnsureDir("/var/lib/app/cache", 0o700); err != nil {
//	    return err
//	}
//
//	// Check whether a path exists without treating absence as an error.
//	exists, err := fs.Exists("/var/lib/app/pid")
//	if err != nil {
//	    return err // real I/O or permission error
//	}
//	if !exists {
//	    // create or skip
//	}
//
//	// Create an isolated temp directory for a build step.
//	dir, err := fs.TempDir("cure-build-")
//	if err != nil {
//	    return err
//	}
//	defer os.RemoveAll(dir)
package fs
