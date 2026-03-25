---
title: "pkg/env"
description: "Runtime environment detection — OS, installed tools, and git repository presence"
order: 9
section: "libraries"
---

# pkg/env

`pkg/env` detects the runtime environment once and caches the result. It answers questions like "is Go installed?", "are we inside a git repository?", and "what shell is the user running?" — useful for contextual help messages and health checks.

**Import path:** `github.com/mrlm-net/cure/pkg/env`

## Detect

```go
import "github.com/mrlm-net/cure/pkg/env"

e := env.Detect()

fmt.Println(e.OS)         // "darwin", "linux", "windows"
fmt.Println(e.Arch)       // "amd64", "arm64"
fmt.Println(e.Shell)      // "/bin/zsh" (value of $SHELL)
fmt.Println(e.GoVersion)  // "go1.25.0" — empty if Go is not on PATH
fmt.Println(e.GitVersion) // "git version 2.39.0" — empty if git is not on PATH
fmt.Println(e.WorkDir)    // current working directory
```

`Detect` uses `sync.Once` internally — the first call runs the detection (including two exec calls for Go and git versions), and every subsequent call returns the cached struct copy instantly (~7 ns). The cache is valid for the lifetime of the process.

## HasTool

```go
if !env.HasTool("docker") {
    fmt.Println("docker is not installed — skipping container build")
}
```

`HasTool` wraps `exec.LookPath`. It is intentionally not cached because tool availability can change between invocations (e.g., a `PATH` mutation), and `LookPath` is fast enough to call on demand.

## IsGitRepo

```go
if env.IsGitRepo() {
    // safe to run git commands
}
```

`IsGitRepo` walks up the directory tree from the current working directory looking for a `.git` entry. It does not invoke the `git` binary — the check is a pure filesystem stat walk. Returns `false` if `os.Getwd` fails or if no `.git` is found before reaching the filesystem root.

## Missing tools

Fields for tools that are not installed (Go, git) are set to empty strings. `HasTool` returns `false`. No errors are returned — absence of a tool is a normal runtime condition, not a failure.
