---
title: "pkg/fs"
description: "Atomic file writes, safe directory creation, and portable filesystem helpers"
order: 7
section: "libraries"
---

# pkg/fs

`pkg/fs` provides crash-safe file operations. The central primitive is `AtomicWrite`, which writes to a temporary file on the same filesystem, syncs to disk, and renames into place — ensuring the target either has the old content or the new content, never a partial write.

**Import path:** `github.com/mrlm-net/cure/pkg/fs`

## AtomicWrite

```go
import "github.com/mrlm-net/cure/pkg/fs"

err := fs.AtomicWrite("config.json", data, 0644)
```

The algorithm:

1. Create a temp file in `filepath.Dir(path)` (same filesystem as target, ensuring rename is atomic on POSIX)
2. Write content and call `fsync`
3. If the target file already exists, preserve its current permissions (the `perm` argument is used only for new files)
4. Rename temp → target

On any error the temp file is deleted. The rename is **not** atomic on Windows (`MoveFileEx` requires a separate call to replace an existing file).

## EnsureDir

```go
// Creates the directory and all parents. No-op if it already exists.
// Returns an error if the path exists but is not a directory.
err := fs.EnsureDir("~/.local/share/cure/sessions", 0700)
```

## Exists

```go
exists, err := fs.Exists("config.json")
// err is non-nil only for permission or I/O failures, not for "file not found".
```

## TempDir

```go
// Creates a unique temporary directory under os.TempDir().
dir, err := fs.TempDir("cure-")
defer os.RemoveAll(dir)
```

## SetPermissions

```go
err := fs.SetPermissions("private.key", 0600)
```

## Notes

- `AtomicWrite` requires the target directory to exist; create it first with `EnsureDir` if needed.
- Cross-filesystem writes (e.g., temp dir on `/tmp`, target on `/home`) will fail at rename time; place both on the same mount.
