# Cure

**A Go CLI tool for automating development tasks — generating templates for AI assistants, devcontainer configurations, and structured file formats.**

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/dl/) [![Release](https://img.shields.io/github/v/release/mrlm-net/cure)](https://github.com/mrlm-net/cure/releases) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/mrlm-net/cure)](https://goreportcard.com/report/github.com/mrlm-net/cure)

## Overview

Cure automates repetitive development tasks through AI context management, code generation, and network diagnostics. Manage multi-turn AI conversations from the terminal (`cure context`), generate templates for AI assistants (`CLAUDE.md`), trace HTTP/TCP/UDP connections with detailed timing and metadata, and output results in developer-friendly formats (NDJSON, HTML). Built with minimal external dependencies and only Go's standard library for core functionality, cure is designed as a foundation for developers who need reliable, auditable tooling without dependency bloat.

The project is under active development — currently at v0.5.0 with a stable API planned for v1.0.0. Cure's modular architecture separates reusable packages (`pkg/`) from application-specific logic (`internal/`), making it straightforward to extend with custom commands or embed cure's packages into other tools.

## Key Features

- **AI context management** — Start, resume, list, fork, and delete multi-turn AI conversations from the terminal; sessions are persisted to `~/.local/share/cure/sessions/` and work with any registered provider
- **Template generation** — Create `CLAUDE.md` project context files for AI assistants with interactive or flag-driven configuration
- **Network tracing** — Trace HTTP requests (DNS resolution, TLS handshake, response timing), TCP connections, and UDP packet exchanges with detailed event streams
- **Flexible output** — Export data as NDJSON for log aggregation or HTML for visual inspection with syntax-highlighted JSON payloads
- **Hierarchical configuration** — Merge settings from defaults, global (`~/.cure.json`), local (`.cure.json`), environment variables (`CURE_` prefix), and CLI flags with clear precedence
- **Shell completion** — Generate bash and zsh completion scripts with dynamic command introspection

## Installation

Install the latest stable release using `go install`:

```sh
go install github.com/mrlm-net/cure/cmd/cure@latest
```

Verify installation:

```sh
cure version
```

To build from source, clone the repository and use the provided Makefile:

```sh
git clone https://github.com/mrlm-net/cure.git
cd cure
make build
./bin/cure version
```

## Quick Start

Start an AI conversation session (requires `ANTHROPIC_API_KEY`):

```sh
cure context new --provider claude --message "Summarise the Go 1.25 release notes."
```

Resume an existing session and continue the conversation:

```sh
cure context resume <session-id> --message "Which change is most impactful for CLI tools?"
```

Generate a `CLAUDE.md` template for configuring AI assistants:

```sh
cure generate claude-md
```

Trace an HTTP request with timing and TLS details in HTML format:

```sh
cure trace http https://api.github.com --format html --output trace.html
```

Enable shell completion for bash:

```sh
source <(cure completion bash)
```

## Commands

### Core

- `cure version` — Display version and build information
- `cure help [command]` — Show help for cure or a specific command

### AI Context Management

- `cure context new --provider <name> --message <text>` — Start a new conversation session; streams the response to stdout and persists the session
- `cure context resume <id> --message <text>` — Continue an existing session with a new user message
- `cure context list [--format text|ndjson]` — List saved sessions sorted newest-first
- `cure context fork <id>` — Deep-copy a session with a new ID; prints the forked ID to stdout
- `cure context delete [--yes] <id>` — Delete a session (prompts for confirmation unless `--yes` is supplied)
- `cure context` *(no args)* — Enter REPL mode for interactive multi-turn conversation

Sessions are stored in `~/.local/share/cure/sessions/` (XDG-compliant). Set `ANTHROPIC_API_KEY` before using the `claude` provider.

### Tracing

- `cure trace dns <hostname>` — Trace DNS resolution with IP addresses, CNAME chain, resolution timing, and RFC 1918 private IP classification
- `cure trace http <url>` — Trace HTTP request with DNS resolution, TLS handshake, request/response headers, and timing
- `cure trace tcp <address>` — Trace TCP connection with handshake timing and connection metadata
- `cure trace udp <address>` — Trace UDP packet exchange with send/receive timing

**Common flags**: `--format` (json|html), `--output <file>`, `--dry-run`

### Generation

- `cure generate claude-md` — Generate `CLAUDE.md` project context file with conventions and AI assistant configuration

### Completion

- `cure completion bash` — Generate bash completion script
- `cure completion zsh` — Generate zsh completion script

Run `cure help <command>` for detailed usage and flag descriptions.

## Design Principles

Cure is built on three core principles: **zero external dependencies**, **reusable package architecture**, and **minimal abstraction**.

**Zero dependencies** — cure uses only Go's standard library. This eliminates supply chain risk, reduces build times, simplifies audits, and ensures cure remains buildable and maintainable for years without dependency updates. The tradeoff is implementing more functionality from scratch, but the benefits outweigh the cost for a foundational tool.

**Reusable packages** — the `pkg/` directory contains independently useful libraries that any Go project can import: `pkg/terminal` for CLI routing and flag parsing, `pkg/config` for hierarchical configuration merging, `pkg/tracer` for network event tracing, `pkg/template` for embedded template rendering, `pkg/mcp` for building stdlib-only MCP (Model Context Protocol) servers with stdio and HTTP Streamable transports, and `pkg/agent` for provider-agnostic AI agent context management. Each package follows a single responsibility and can be used without importing cure's application logic.

**Minimal abstraction** — cure favors composition over complex abstractions. Commands implement a simple interface (`Name()`, `Description()`, `Usage()`, `Flags()`, `Run()`), configuration is plain `map[string]interface{}` with dot-notation access, and the router dispatches commands without heavy middleware stacks. This keeps the codebase readable and debuggable.

## pkg/agent

`pkg/agent` provides a provider-agnostic abstraction for AI agent context management. It defines the core interfaces, session lifecycle, a global provider registry, and a persistence interface — without coupling to any specific AI provider.

Import path: `github.com/mrlm-net/cure/pkg/agent`

### Registering a provider

Provider adapters live in `internal/agent/<provider>/` and self-register via `init()` using the blank-import driver pattern — the same convention as `database/sql` drivers.

```go
import (
    "github.com/mrlm-net/cure/pkg/agent"

    // Register the "claude" provider by importing its adapter package.
    // The adapter calls agent.Register("claude", factory) in its init() function.
    _ "github.com/mrlm-net/cure/internal/agent/claude"
)
```

After the blank import, `agent.New("claude", cfg)` is available. `agent.Registered()` returns a sorted list of all registered provider names.

### internal/agent/claude — Anthropic Claude adapter

The Claude adapter lives in `internal/agent/claude` and wires the Anthropic Go SDK into `pkg/agent`. It uses the blank-import driver pattern so your application code stays decoupled from the adapter package.

**Prerequisite** — set the API key in the environment before creating the agent:

```sh
export ANTHROPIC_API_KEY=sk-ant-...
```

**Full example** — blank-import the adapter, create a session, stream a response, and persist it with `JSONStore`:

```go
import (
    "context"
    "fmt"
    "log"
    "os"
    "strings"

    "github.com/mrlm-net/cure/pkg/agent"
    "github.com/mrlm-net/cure/pkg/agent/store"

    // Registers the "claude" provider via init().
    _ "github.com/mrlm-net/cure/internal/agent/claude"
)

func main() {
    ctx := context.Background()

    // Create a file-backed session store.
    s, err := store.NewJSONStore("~/.local/share/cure/sessions")
    if err != nil {
        log.Fatal(err)
    }

    // Instantiate the Claude agent.
    // "api_key_env" defaults to "ANTHROPIC_API_KEY".
    // "model" defaults to "claude-opus-4-6".
    // "max_tokens" defaults to 8192.
    a, err := agent.New("claude", map[string]any{
        "model":      "claude-opus-4-6",
        "max_tokens": 8192,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Build a session.
    session := agent.NewSession("claude", "claude-opus-4-6")
    session.SystemPrompt = "You are a concise technical assistant."
    session.AppendUserMessage("What is the difference between os.Rename and os.Link in Go?")

    // Stream the response using Go 1.23 range-over-function syntax.
    var response strings.Builder
    for ev, err := range a.Run(ctx, session) {
        if err != nil {
            log.Fatal(err)
        }
        switch ev.Kind {
        case agent.EventKindToken:
            response.WriteString(ev.Text)
            fmt.Print(ev.Text) // stream to terminal
        case agent.EventKindDone:
            fmt.Printf("\n\ntokens: in=%d out=%d stop=%s\n",
                ev.InputTokens, ev.OutputTokens, ev.StopReason)
        case agent.EventKindError:
            log.Fatalf("provider error: %s", ev.Err)
        }
    }

    // Append the assistant reply and persist the session.
    session.AppendAssistantMessage(response.String())
    if err := s.Save(ctx, session); err != nil {
        log.Fatal(err)
    }
    fmt.Printf("session saved: %s\n", session.ID)
}
```

**Configuration keys** accepted by `NewClaudeAgent`:

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `api_key_env` | `string` | `"ANTHROPIC_API_KEY"` | Name of the environment variable holding the API key |
| `model` | `string` | `"claude-opus-4-6"` | Anthropic model ID |
| `max_tokens` | `int` / `int64` / `float64` | `8192` | Maximum tokens in the completion |

**Token counting** — call `Agent.CountTokens` before streaming to check context budget:

```go
n, err := a.CountTokens(ctx, session)
if errors.Is(err, agent.ErrCountNotSupported) {
    n = agent.EstimateTokens(session.SystemPrompt) // fallback heuristic
}
fmt.Printf("estimated context tokens: %d\n", n)
```

**Error safety** — `sanitiseError` replaces the API key value with `[REDACTED]` in all error strings, so errors are safe to log or surface to users.

**Note** — `internal/agent/claude` is an internal package. It cannot be imported by external modules. This is intentional: cure ships the adapter as part of the application layer, while `pkg/agent` remains dependency-free. If you need to use the Claude adapter outside of cure, copy the adapter source into your own `internal/` package.

### Creating a session

`Session` holds the full conversation state. `NewSession` generates a 128-bit cryptographically random ID.

```go
session := agent.NewSession("claude", "claude-opus-4-5")
session.SystemPrompt = "You are a helpful assistant."
session.AppendUserMessage("Summarise the Go 1.23 release notes.")
```

Fork a session to branch a conversation without mutating the original:

```go
branch := session.Fork()
// branch.ForkOf == session.ID
// branch.History is a deep copy — appending to branch does not affect session
```

### Streaming a response

`Agent.Run` returns `iter.Seq2[Event, error]`, which is iterated with Go 1.23's range-over-function syntax.

```go
a, err := agent.New("claude", map[string]any{
    "api_key": os.Getenv("ANTHROPIC_API_KEY"),
})
if err != nil {
    log.Fatal(err)
}

var response strings.Builder

for ev, err := range a.Run(ctx, session) {
    if err != nil {
        log.Fatal(err)
    }
    switch ev.Kind {
    case agent.EventKindToken:
        response.WriteString(ev.Text)
    case agent.EventKindDone:
        fmt.Printf("tokens: in=%d out=%d stop=%s\n",
            ev.InputTokens, ev.OutputTokens, ev.StopReason)
    case agent.EventKindError:
        log.Fatalf("provider error: %s", ev.Err)
    }
}

session.AppendAssistantMessage(response.String())
```

Cancelling the context terminates the stream early.

### Persisting sessions

`SessionStore` is the interface for concrete persistence implementations (JSON file, database, in-memory). Implementations must be safe for concurrent use.

```go
type SessionStore interface {
    Save(ctx context.Context, s *Session) error
    Load(ctx context.Context, id string) (*Session, error)
    List(ctx context.Context) ([]*Session, error)
    Delete(ctx context.Context, id string) error
    Fork(ctx context.Context, id string) (*Session, error)
}
```

`Load`, `Delete`, and `Fork` return `ErrSessionNotFound` (or a wrapped form) when the session ID does not exist. Check with `errors.Is`:

```go
_, err := store.Load(ctx, id)
if errors.Is(err, agent.ErrSessionNotFound) {
    // session does not exist
}
```

### pkg/agent/store — JSON file store

`pkg/agent/store` provides `JSONStore`, a file-backed `SessionStore` implementation that is ready to use without any external dependencies.

Import path: `github.com/mrlm-net/cure/pkg/agent/store`

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

// Save a session. Writes are atomic (os.CreateTemp + os.Rename).
// Session files receive mode 0600.
if err := s.Save(ctx, session); err != nil {
    log.Fatal(err)
}

// Load it back by ID.
loaded, err := s.Load(ctx, session.ID)
if errors.Is(err, agent.ErrSessionNotFound) {
    fmt.Println("session not found")
} else if err != nil {
    log.Fatal(err)
}

// List all sessions sorted by UpdatedAt descending.
// Corrupt or unreadable files are silently skipped.
sessions, err := s.List(ctx)
if err != nil {
    log.Fatal(err)
}
for _, sess := range sessions {
    fmt.Printf("%s  %s  %s\n", sess.ID, sess.Provider, sess.UpdatedAt.Format(time.RFC3339))
}

// Fork creates an independent copy with a new ID.
branch, err := s.Fork(ctx, session.ID)
```

`JSONStore.Save` is protected by a `sync.Mutex` and is safe for concurrent use from multiple goroutines. `Load`, `List`, `Delete`, and `Fork` use direct filesystem reads and are inherently safe for concurrent access.

**ID validation** — session IDs must match `^[0-9a-f]{1,64}$` (1–64 lowercase hex characters). Any other value is rejected before touching the filesystem, eliminating path-traversal surface entirely.

### Testing a custom store

Use the shared test suite to verify any `SessionStore` implementation:

```go
func TestMyStore(t *testing.T) {
    store := NewMyStore(t.TempDir())
    agent.RunSessionStoreTests(t, store)
}
```

`RunSessionStoreTests` covers save/load round-trips, not-found errors, list, delete, and fork semantics.

### Token estimation

For quick context window budget calculations before calling a provider:

```go
n := agent.EstimateTokens(text) // len(text) / 4
```

For precise counts, use `Agent.CountTokens`. If the provider does not support it, `ErrCountNotSupported` is returned.

## Development

### Prerequisites

- Go 1.25 or later
- `make`

### Build Commands

| Command | Purpose |
|---------|---------|
| `make build` | Build the `cure` binary to `bin/` |
| `make test` | Run all tests with race detector |
| `make test-coverage` | Run tests with coverage report |
| `make lint` | Run `go vet` for static analysis |
| `make clean` | Remove build artifacts |

### Getting Started

Clone the repository, run tests, and build the binary:

```sh
git clone https://github.com/mrlm-net/cure.git
cd cure
make test
make build
./bin/cure help
```

The codebase follows standard Go conventions (`gofmt`, `go vet`, effective Go). Exported functions include doc comments, and tests use table-driven patterns with `t.Run` subtests.

## Contributing

Contributions are welcome. Cure uses a structured workflow based on GitHub Issues and Pull Requests. Before submitting a PR, review the project's conventions in `CLAUDE.md` to understand architectural patterns and coding standards.

**Workflow**: All work starts with a GitHub Issue describing the feature or bug. Create a feature branch (`feat/<issue>-<description>` or `fix/<issue>-<description>`), implement changes, and submit a PR targeting `main`. PRs require passing tests and `go vet` before merge. Use Conventional Commits for commit messages (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`). PRs are squash-merged to maintain a clean commit history.

**Code organization**: New reusable functionality belongs in `pkg/`, application-specific wiring in `internal/`, and command entry points in `cmd/cure/`. Packages in `pkg/` must not import from `internal/` or `cmd/`. Follow the single responsibility principle — each package has one clear purpose. Write tests for all exported functions and benchmarks for performance-critical paths.

**Testing standards**: Tests use the standard `testing` package with table-driven patterns. Run tests with race detection (`make test`) before submitting PRs. Add test coverage for new code paths and verify with `make test-coverage`.

For detailed guidance, see `CLAUDE.md` in the repository root.

## Roadmap

Cure is currently at v0.5.0. The v0.5.0 release delivered `pkg/agent` — provider-agnostic AI agent context management — along with the `cure context` command group and the Anthropic Claude adapter.

Upcoming milestones:

- **v0.6.0** — Foundation packages for developer experience: `pkg/prompt` for interactive prompts, `pkg/fs` for filesystem operations, `pkg/style` for terminal styling, and `pkg/env` for environment inspection; `cure doctor` command; `--dry-run` support across commands
- **v0.7.0+** — Expanded template generation (template directories, multi-file generation), additional AI provider adapters, and plugin architecture exploration

The v1.0.0 milestone marks API stability for `pkg/` packages and freeze of breaking changes. Track progress on the [GitHub Projects board](https://github.com/orgs/mrlm-net/projects/9).

## License

Licensed under the [Apache License 2.0](LICENSE).
