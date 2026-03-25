---
title: "pkg/agent/store"
description: "File-backed JSON session store with atomic writes and ID validation"
order: 3
section: "libraries"
---

# pkg/agent/store

`pkg/agent/store` provides `JSONStore`, a file-backed `SessionStore` implementation that is ready to use without any external dependencies.

**Import path:** `github.com/mrlm-net/cure/pkg/agent/store`

## Creating a store

`NewJSONStore` accepts a directory path with optional tilde expansion. The directory is created lazily on the first `Save` — you do not need to create it in advance.

```go
import (
    "context"
    "errors"
    "fmt"

    "github.com/mrlm-net/cure/pkg/agent"
    "github.com/mrlm-net/cure/pkg/agent/store"
)

// Create a store backed by ~/.local/share/cure/sessions.
// The directory is created on first Save with mode 0700.
s, err := store.NewJSONStore("~/.local/share/cure/sessions")
if err != nil {
    log.Fatal(err)
}
```

## Saving a session

```go
// Writes are atomic (os.CreateTemp + os.Rename).
// Session files receive mode 0600.
if err := s.Save(ctx, session); err != nil {
    log.Fatal(err)
}
```

`JSONStore.Save` is protected by a `sync.Mutex` and is safe for concurrent use from multiple goroutines.

## Loading a session

```go
loaded, err := s.Load(ctx, session.ID)
if errors.Is(err, agent.ErrSessionNotFound) {
    fmt.Println("session not found")
} else if err != nil {
    log.Fatal(err)
}
```

## Listing sessions

```go
// Returns all sessions sorted by UpdatedAt descending.
// Corrupt or unreadable files are silently skipped.
sessions, err := s.List(ctx)
if err != nil {
    log.Fatal(err)
}
for _, sess := range sessions {
    fmt.Printf("%s  %s  %s\n", sess.ID, sess.Provider, sess.UpdatedAt.Format(time.RFC3339))
}
```

## Forking a session

```go
// Fork creates an independent copy with a new ID.
branch, err := s.Fork(ctx, session.ID)
```

## Deleting a session

```go
if err := s.Delete(ctx, session.ID); err != nil {
    if errors.Is(err, agent.ErrSessionNotFound) {
        fmt.Println("session already gone")
    } else {
        log.Fatal(err)
    }
}
```

## Security properties

**Atomic writes** — `Save` writes to a temporary file via `os.CreateTemp`, then renames it into place with `os.Rename`. This ensures sessions cannot be partially written. A crash mid-write leaves the temporary file orphaned, not a corrupt session file.

**File permissions** — session files receive mode `0600` (owner read/write only). The sessions directory receives mode `0700`.

**ID validation** — session IDs must match `^[0-9a-f]{1,64}$` (1–64 lowercase hex characters). Any other value is rejected before touching the filesystem, eliminating path-traversal surface entirely. Characters outside lowercase hex — including `/`, `\`, `..`, null bytes, and Unicode — are all rejected.

**Corrupt file handling** — `List` silently skips unreadable or corrupt JSON files. This ensures a single bad file does not prevent listing other sessions.

## Concurrency

`Save` is protected by `sync.Mutex`. `Load`, `List`, `Delete`, and `Fork` use direct filesystem reads and are inherently safe for concurrent access.

## Compile-time interface check

`JSONStore` satisfies `agent.SessionStore` with a compile-time assertion:

```go
var _ agent.SessionStore = (*JSONStore)(nil)
```

If `JSONStore` ever fails to implement the interface, the package fails to compile immediately rather than at runtime.
