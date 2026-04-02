# Cure

**A Go CLI tool for automating development tasks — generating templates for AI assistants, devcontainer configurations, and structured file formats.**

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/dl/) [![Release](https://img.shields.io/github/v/release/mrlm-net/cure)](https://github.com/mrlm-net/cure/releases) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/mrlm-net/cure)](https://goreportcard.com/report/github.com/mrlm-net/cure)

## Overview

Cure automates repetitive development tasks through AI context management, code generation, and network diagnostics. Manage multi-turn AI conversations from the terminal (`cure context`), generate templates for AI assistants (`CLAUDE.md`), trace HTTP/TCP/UDP connections with detailed timing and metadata, and output results in developer-friendly formats (NDJSON, HTML). Built with minimal external dependencies and only Go's standard library for core functionality, cure is designed as a foundation for developers who need reliable, auditable tooling without dependency bloat.

The project is under active development — currently at v0.10.0 with a stable API planned for v1.0.0. Cure's modular architecture separates reusable packages (`pkg/`) from application-specific logic (`internal/`), making it straightforward to extend with custom commands or embed cure's packages into other tools.

## Key Features

- **Project bootstrapping** — `cure init` bootstraps a complete project scaffold in one command: AI assistant files, devcontainer, CI workflow, editorconfig, and gitignore; interactive wizard or fully flag-driven for CI use
- **AI context management** — Start, resume, list, fork, delete, search, and export multi-turn AI conversations from the terminal; sessions are persisted to `~/.local/share/cure/sessions/` and work with any registered provider
- **Tool use** — All three providers (Claude, OpenAI, Gemini) execute multi-turn tool loops (up to 32 turns) within a single `context new` or `context resume` session; register tools via the `pkg/agent` API or activate named presets with `--skill <name>`
- **Skill registry** — `agent.RegisterSkill` bundles a system prompt with a set of tools under a named preset; `--skill <name>` on `context new` or `context resume` activates the preset for that session
- **Template generation** — Create `CLAUDE.md` project context files for AI assistants with interactive or flag-driven configuration; `--dry-run` prints output to stdout without writing files
- **Network tracing** — Trace HTTP requests (DNS resolution, TLS handshake, response timing), TCP connections, and UDP packet exchanges with detailed event streams
- **Flexible output** — Export data as NDJSON for log aggregation or HTML for visual inspection with syntax-highlighted JSON payloads
- **Hierarchical configuration** — Merge settings from defaults, global (`~/.cure.json`), local (`.cure.json`), environment variables (`CURE_` prefix), and CLI flags with clear precedence
- **Shell completion** — Generate bash and zsh completion scripts with dynamic command introspection
- **Project health checks** — `cure doctor` runs 7 checks (README, tests, CI, `.gitignore`, `CLAUDE.md`, build tool, dependency manifest) and exits 1 on failure

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

Bootstrap a new project with all standard configuration files in one pass:

```sh
cure init
```

Or fully non-interactive for CI and scripted environments:

```sh
cure init --non-interactive --name myapp --language go
```

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
- `cure init [flags]` — Bootstrap a complete project scaffold in one command (see [Project Bootstrapping](#project-bootstrapping))

### Project Bootstrapping

`cure init` generates all standard configuration files for a new project in a single interactive wizard or fully non-interactive pass. All generators run regardless of individual failures; a summary is printed at the end.

**Interactive mode** (default when stdin is a terminal):

```sh
cure init
```

Prompts for project name, primary language, which AI assistant files to generate, and whether to include devcontainer, CI workflow, editorconfig, and gitignore.

**Non-interactive mode** (for CI and scripts):

```sh
# Bootstrap a Go project with all defaults
cure init --non-interactive --name myapp --language go

# Select a subset of AI assistant files
cure init --non-interactive --name myapp --language go \
  --ai-tools claude-md,cursor-rules

# Preview without writing any files
cure init --non-interactive --name myapp --language go --dry-run
```

**Flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--non-interactive` | `false` | Skip prompts; use flag values |
| `--dry-run` | `false` | Preview output without writing files |
| `--force` | `false` | Overwrite existing files |
| `--name` | *(required in non-interactive)* | Project name |
| `--language` | *(required in non-interactive)* | Primary language: `go`, `node`, `python`, `rust`, `other` |
| `--ai-tools` | all | Comma-separated AI tool IDs to generate |
| `--devcontainer` | `true` | Generate `.devcontainer/devcontainer.json` |
| `--ci` | `true` | Generate `.github/workflows/ci.yml` |
| `--editorconfig` | `true` | Generate `.editorconfig` |
| `--gitignore` | `true` | Generate `.gitignore` |

**AI tool IDs** accepted by `--ai-tools`: `claude-md`, `agents-md`, `copilot-instructions`, `cursor-rules`, `windsurf-rules`, `gemini-md`.

**Summary output** — after all generators have run, `cure init` prints a per-component result:

```
cure init summary:
  ok claude-md
  ok cursor-rules
  ok devcontainer
  x ci: open .github/workflows/ci.yml: permission denied
```

### AI Context Management

- `cure context new --provider <name> --message <text>` — Start a new conversation session; streams the response to stdout and persists the session
- `cure context new --skill <name>` — Start a session with a named skill preset (system prompt + tool set)
- `cure context resume <id> --message <text>` — Continue an existing session with a new user message
- `cure context resume <id> --skill <name>` — Resume a session and activate a skill preset
- `cure context list [--format text|ndjson]` — List saved sessions sorted newest-first
- `cure context fork <id>` — Deep-copy a session with a new ID; prints the forked ID to stdout
- `cure context delete [--yes] <id>` — Delete a session (prompts for confirmation unless `--yes` is supplied)
- `cure context search <query> [--format table|ndjson]` — Search all session history for messages containing the query (case-insensitive); reports ID, provider, creation time, match count, and an excerpt
- `cure context export <session-id> [--format markdown|ndjson] [--output <file>]` — Export a session as Markdown (default) or NDJSON; read-only, never mutates the session
- `cure context` *(no args)* — Enter REPL mode for interactive multi-turn conversation; tool calls and results are annotated to stderr

Sessions are stored in `~/.local/share/cure/sessions/` (XDG-compliant). Set `ANTHROPIC_API_KEY` before using the `claude` provider.

#### Using tools and skills

All three providers (Claude, OpenAI, and Gemini) support multi-turn tool use via `sess.Tools`. Skills are named presets that combine a system prompt with a set of tools and are registered at program startup via `agent.RegisterSkill`. Skills do not imply a provider — `--provider` is still required on `context new`.

#### Provider capabilities

| Capability | Claude | OpenAI | Gemini |
|------------|--------|--------|--------|
| Streaming | Yes | Yes (SSE) | Yes (SSE) |
| Tool use | Yes | Yes | Yes |
| Token counting | Yes | Estimate | Estimate |

```sh
# Start a session with the "code-review" skill (system prompt + tools pre-loaded)
# Note: --provider is required; skills are provider-agnostic presets
cure context new --provider claude --skill code-review --message "Review this diff"

# Resume a session and apply a skill
cure context resume <id> --skill code-review --message "What about the tests?"
```

The `code-review` skill above is illustrative — skills must be registered at program startup via `agent.RegisterSkill` in your own program or in a future `skills.json` configuration.

During a tool-augmented session, the REPL annotates tool calls and results to stderr so stdout stays clean for piping. Each provider adapter automatically executes up to 32 sequential tool calls per session turn.

#### Searching session history

```sh
# Find sessions discussing authentication — table output
cure context search "authentication"

# Machine-readable output for scripting
cure context search "bug fix" --format ndjson
```

The table output shows `ID`, `PROVIDER`, `CREATED`, `MATCHES`, and `EXCERPT` columns. The excerpt is centred around the first match and is UTF-8 safe (rune-based windowing).

#### Exporting sessions

```sh
# Export to Markdown on stdout
cure context export abc123

# Export as NDJSON (one JSON object)
cure context export abc123 --format ndjson

# Write Markdown to a file (note: flags before the session ID)
cure context export --output session.md abc123
```

The Markdown export produces an H1 heading with the session ID, a metadata table (Provider, Model, Created, Updated, Fork of), and an H2 section per message with the role as heading.

The `cure context` commands are backed by [`pkg/agent`](#pkgagent) and [`pkg/agent/store`](#pkgagentstore--json-file-store) — see those sections if you want to embed session management into your own Go programs.

### Tracing

- `cure trace dns <hostname>` — Trace DNS resolution with IP addresses, CNAME chain, resolution timing, and RFC 1918 private IP classification
- `cure trace http <url>` — Trace HTTP request with DNS resolution, TLS handshake, request/response headers, and timing
- `cure trace tcp <address>` — Trace TCP connection with handshake timing and connection metadata
- `cure trace udp <address>` — Trace UDP packet exchange with send/receive timing

**Common flags**: `--format` (json|html), `--output <file>`, `--dry-run`

### Generation

| Command | Output | Notes |
|---------|--------|-------|
| `cure generate scaffold` | All AI context files in one pass | Interactive `MultiSelect` wizard; `--select` for a subset, `--non-interactive` to skip prompts |
| `cure generate claude-md` | `CLAUDE.md` | AI assistant context for Claude Code |
| `cure generate agents-md` | `AGENTS.md` | Cross-tool AI context (Copilot, Cursor, Devin, Gemini CLI, OpenAI Codex) |
| `cure generate copilot-instructions` | `.github/copilot-instructions.md` | GitHub Copilot instructions with YAML frontmatter |
| `cure generate cursor-rules` | `.cursor/rules/project.mdc` | Cursor rules with YAML frontmatter |
| `cure generate windsurf-rules` | `.windsurfrules` | Windsurf-style numbered rules |
| `cure generate gemini-md` | `GEMINI.md` | Google Gemini CLI auto-discovery format |
| `cure generate devcontainer` | `.devcontainer/devcontainer.json` | VS Code Dev Containers / GitHub Codespaces; optional `Dockerfile` stub via `--dockerfile` |
| `cure generate editorconfig` | `.editorconfig` | Per-language indent rules; supported: `go`, `javascript`, `python`, `rust`, `java`, `shell`, `markdown`, `yaml`, `generic` |
| `cure generate gitignore` | `.gitignore` | Built from 11 embedded profiles: `go`, `node`, `python`, `rust`, `java`, `macos`, `windows`, `linux`, `jetbrains`, `vscode`, `vim` |
| `cure generate github-workflow` | `.github/workflows/ci.yml` | GitHub Actions CI for Go; optional `--lint` and `--coverage` steps |

All `cure generate` subcommands support `--dry-run` (print to stdout without writing), `--force` (overwrite existing files), and `--non-interactive` (use defaults without prompting).

### Health checks

- `cure doctor` — Run project health checks against the current working directory and print a per-check summary

The doctor command runs 7 checks: README presence, test files, CI configuration, `.gitignore` (warning if absent), `CLAUDE.md`, a build tool (`Makefile` or similar), and a dependency manifest (`go.mod`, `package.json`, etc.). The command exits 0 when all checks pass or produce only warnings, and exits 1 when any check fails.

### Completion

- `cure completion bash` — Generate bash completion script
- `cure completion zsh` — Generate zsh completion script

Run `cure help <command>` for detailed usage and flag descriptions.

## Design Principles

Cure is built on three core principles: **zero external dependencies**, **reusable package architecture**, and **minimal abstraction**.

**Zero dependencies** — cure uses only Go's standard library. This eliminates supply chain risk, reduces build times, simplifies audits, and ensures cure remains buildable and maintainable for years without dependency updates. The tradeoff is implementing more functionality from scratch, but the benefits outweigh the cost for a foundational tool.

**Reusable packages** — the `pkg/` directory contains independently useful libraries that any Go project can import: `pkg/terminal` for CLI routing and flag parsing, `pkg/config` for hierarchical configuration merging, `pkg/tracer` for network event tracing, `pkg/template` for embedded template rendering, `pkg/mcp` for building stdlib-only MCP (Model Context Protocol) servers with stdio and HTTP Streamable transports, `pkg/agent` for provider-agnostic AI agent context management, `pkg/prompt` for interactive terminal prompts, `pkg/fs` for atomic filesystem operations, `pkg/style` for ANSI terminal styling with NO_COLOR support, `pkg/env` for cached runtime environment detection, and `pkg/doctor` for composable project health checks. Each package follows a single responsibility and can be used without importing cure's application logic.

**Minimal abstraction** — cure favors composition over complex abstractions. Commands implement a simple interface (`Name()`, `Description()`, `Usage()`, `Flags()`, `Run()`), configuration is plain `map[string]interface{}` with dot-notation access, and the router dispatches commands without heavy middleware stacks. This keeps the codebase readable and debuggable.

## pkg/terminal

`pkg/terminal` is cure's CLI framework. It provides the `Command` interface, a radix-tree `Router` for command dispatch, and three built-in execution strategies (runners). Any Go program can embed it to build structured CLI tooling without external dependencies.

Import path: `github.com/mrlm-net/cure/pkg/terminal`

### Command interface

Implement five methods to define a command:

```go
type VersionCommand struct{}

func (c *VersionCommand) Name()        string        { return "version" }
func (c *VersionCommand) Description() string        { return "Print version information" }
func (c *VersionCommand) Usage()       string        { return "Usage: myapp version" }
func (c *VersionCommand) Flags()       *flag.FlagSet { return nil }

func (c *VersionCommand) Run(ctx context.Context, tc *terminal.Context) error {
    fmt.Fprintln(tc.Stdout, "myapp version 1.0.0")
    return nil
}
```

Commands receive a `*terminal.Context` containing parsed positional arguments (`tc.Args`), a parsed `*flag.FlagSet` (`tc.Flags`), I/O streams (`tc.Stdout`, `tc.Stderr`, `tc.Stdin`), a structured logger (`tc.Logger`), and the merged config (`tc.Config`). Commands must write all output to these streams — never to `os.Stdout` directly.

### Router

`terminal.New` creates a router with functional options. Commands are registered and dispatched by name via a radix tree:

```go
router := terminal.New(
    terminal.WithStdout(os.Stdout),
    terminal.WithStderr(os.Stderr),
    terminal.WithRunner(&terminal.SerialRunner{}), // default
)
router.Register(&VersionCommand{})
router.Register(terminal.NewHelpCommand(router))

if err := router.RunContext(ctx, os.Args[1:]); err != nil {
    fmt.Fprintf(os.Stderr, "error: %v\n", err)
    os.Exit(1)
}
```

**Functional options:**

| Option | Default | Description |
|--------|---------|-------------|
| `WithStdout(w)` | `os.Stdout` | Standard output stream |
| `WithStderr(w)` | `os.Stderr` | Standard error stream |
| `WithRunner(r)` | `&SerialRunner{}` | Execution strategy |
| `WithConfig(cfg)` | `nil` | Merged config passed to commands |
| `WithSignalHandler()` | off | Cancel context on SIGINT/SIGTERM; second signal calls `os.Exit(1)` |
| `WithTimeout(d)` | none | Per-command execution deadline |
| `WithGracePeriod(d)` | `5s` | Time for cleanup after cancellation |

**Nested routers** — a `Router` implements `Command`, so sub-routers can be registered as subcommand groups:

```go
parent := terminal.New()
child  := terminal.New(terminal.WithName("context"), terminal.WithDescription("Manage sessions"))
child.Register(&NewCommand{})
child.Register(&ListCommand{})
parent.Register(child) // cure context new / cure context list
```

**Aliases** — register alternative names for a command:

```go
router.RegisterWithAliases(&VersionCommand{}, "v", "ver")
// now "myapp v", "myapp ver", and "myapp version" all work
```

### Execution strategies

The runner controls how matched commands are executed. Pass one via `WithRunner`.

#### SerialRunner *(default)*

Executes commands sequentially. Stops at the first error.

```go
router := terminal.New() // SerialRunner is the default
```

All commands share the same `*Context`. Suitable for single-command dispatch (the standard case) and ordered batch execution.

#### ConcurrentRunner

Executes commands concurrently using goroutines, up to a configurable worker limit. Each command receives its own `*Context` copy to prevent data races. All errors are collected and returned together via `errors.Join`.

```go
router := terminal.New(
    terminal.WithRunner(terminal.WithMaxWorkers(4)),
)
```

`WithMaxWorkers(0)` or negative values default to `runtime.NumCPU()`. Context cancellation prevents new commands from starting; in-flight commands receive the cancelled context.

#### PipelineRunner

Connects commands in sequence: the stdout of command N is piped directly to the stdin of command N+1 using `io.Pipe`. The first command reads from the router's stdin; the last writes to the router's stdout.

```go
router := terminal.New(
    terminal.WithRunner(&terminal.PipelineRunner{}),
)
router.Register(&FetchCommand{})
router.Register(&TransformCommand{})
router.Register(&FormatCommand{})
// FetchCommand → TransformCommand → FormatCommand, all running concurrently
```

All stages launch concurrently and are throttled naturally by data flow — a stage blocks until the previous stage produces output. A failed stage closes its write pipe with the error, causing downstream stages to receive `io.ErrClosedPipe` on their next read. The first non-nil error across all stages is returned.

### Signal handling and timeouts

```go
router := terminal.New(
    terminal.WithSignalHandler(),           // SIGINT/SIGTERM → cancel context
    terminal.WithTimeout(30*time.Second),   // hard deadline per command
    terminal.WithGracePeriod(5*time.Second),// cleanup window after cancellation
)
```

On the first SIGINT or SIGTERM the command context is cancelled — commands should return promptly. A second signal calls `os.Exit(1)` immediately.

## pkg/agent

> The [`cure context`](#ai-context-management) command group is the ready-to-use CLI front-end for this package. Use `pkg/agent` directly when you want to embed session management into your own Go programs.

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

### Tool use

Attach tools to a session via `Session.Tools` before calling `Agent.Run`. All three provider adapters (Claude, OpenAI, and Gemini) automatically execute tool calls and re-invoke the model until the model returns without requesting tools (up to 32 turns). Tool call and tool result events are emitted during the loop.

```go
// Define a tool with FuncTool — no struct needed.
getTime := agent.FuncTool(
    "get_time",
    "Return the current wall-clock time as HH:MM.",
    map[string]any{"type": "object", "properties": map[string]any{}},
    func(ctx context.Context, _ map[string]any) (string, error) {
        return time.Now().Format("15:04"), nil
    },
)

session := agent.NewSession("claude", "claude-opus-4-6")
session.AppendUserMessage("What time is it?")
session.Tools = []agent.Tool{getTime}

for ev, err := range a.Run(ctx, session) {
    if err != nil {
        log.Fatal(err)
    }
    switch ev.Kind {
    case agent.EventKindToolCall:
        fmt.Fprintf(os.Stderr, "[tool] %s(%s)\n", ev.ToolCall.ToolName, ev.ToolCall.InputJSON)
    case agent.EventKindToolResult:
        fmt.Fprintf(os.Stderr, "[tool result] %s → %s\n", ev.ToolResult.ToolName, ev.ToolResult.Result)
    case agent.EventKindToken:
        fmt.Print(ev.Text)
    }
}
```

`Session.Tools` is transient — it is excluded from JSON serialization (`json:"-"`) so tool registrations are never written to the session file. Reattach tools when loading a persisted session.

### Skill registry

A `Skill` is a named preset that bundles a system prompt with a set of tools. Register skills at program startup and activate them by name via `--skill <name>` on `context new` or `context resume`, or programmatically:

```go
// Register a skill at init time (typically in an init() function).
agent.RegisterSkill(agent.Skill{
    Name:         "time-assistant",
    Description:  "Answers questions about the current time.",
    SystemPrompt: "You are a helpful assistant that can tell the current time.",
    Tools:        []agent.Tool{getTime},
})

// Look up a skill and apply it to a session.
if skill, ok := agent.LookupSkill("time-assistant"); ok {
    session.SystemPrompt = skill.SystemPrompt
    session.Tools = skill.Tools
    session.SkillName = skill.Name
}

// List all registered skills.
for _, s := range agent.Skills() {
    fmt.Printf("  %s — %s\n", s.Name, s.Description)
}
```

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

## pkg/prompt

`pkg/prompt` provides interactive terminal prompts with validation. The `Prompter` struct wraps an `io.Writer` and `io.Reader` pair so all input and output can be redirected — enabling fully testable prompts without a real terminal.

Import path: `github.com/mrlm-net/cure/pkg/prompt`

```go
import (
    "os"
    "github.com/mrlm-net/cure/pkg/prompt"
)

p := prompt.NewPrompter(os.Stdout, os.Stdin)

// Required — repeats until the user provides a non-empty value.
// Pressing Enter with a default returns the default.
name, err := p.Required("Project name", "my-project")

// Confirm — accepts y/yes/n/no (case-insensitive).
ok, err := p.Confirm("Overwrite existing file?")

// MultiSelect — accepts comma-separated numbers, "all", or "none".
opts := []prompt.Option{
    {Label: "Logging", Value: "logging"},
    {Label: "Metrics", Value: "metrics"},
    {Label: "Tracing", Value: "tracing"},
}
selected, err := p.MultiSelect("Select features", opts)
```

`IsInteractive(stdin)` reports whether the reader is a real terminal. Use it to bypass prompts in CI or piped contexts:

```go
if !prompt.IsInteractive(os.Stdin) {
    // non-interactive: use defaults or flags
}
```

## pkg/fs

`pkg/fs` provides safe filesystem operations for CLI tools. The key primitive is `AtomicWrite`, which uses a write-then-rename sequence (create temp file → fsync → rename) so concurrent readers always observe either the old content or the new content, never a partial write.

Import path: `github.com/mrlm-net/cure/pkg/fs`

```go
import "github.com/mrlm-net/cure/pkg/fs"

// Write a config file atomically (create temp, fsync, rename).
err := fs.AtomicWrite("/etc/app/config.json", data, 0o600)

// Create a directory hierarchy; no-op if it already exists.
err = fs.EnsureDir("~/.local/share/myapp/cache", 0o700)

// Check existence without treating absence as an error.
exists, err := fs.Exists("/var/run/app.pid")

// Create a temporary directory; caller removes it when done.
dir, err := fs.TempDir("myapp-build-")
defer os.RemoveAll(dir)
```

`AtomicWrite` inherits permissions from the existing target file so a rewrite does not silently change access rights. The temp file is always created in the same directory as the target to guarantee same-filesystem rename semantics on POSIX.

## pkg/style

`pkg/style` provides minimal ANSI terminal styling with NO_COLOR support. All functions are standalone — there is no struct to initialise. Styling is enabled by default and automatically disabled when the `NO_COLOR` environment variable is set at program startup (see [no-color.org](https://no-color.org)).

Import path: `github.com/mrlm-net/cure/pkg/style`

```go
import "github.com/mrlm-net/cure/pkg/style"

// 8 foreground colors
fmt.Println(style.Red("error"))
fmt.Println(style.Green("ok"))
fmt.Println(style.Yellow("warning"))
fmt.Println(style.Blue("info"))

// 3 text styles
fmt.Println(style.Bold("heading"))
fmt.Println(style.Dim("secondary"))
fmt.Println(style.Underline("link"))

// Compose by nesting; the extra reset code is harmless
label := style.Bold(style.Red("FAIL"))

// Strip ANSI codes — useful for log files or display-width calculations
plain := style.Reset(label) // "FAIL"

// Runtime control
style.Disable() // turn off for this process
style.Enable()  // turn back on
```

## pkg/env

`pkg/env` detects the current runtime environment — OS, architecture, shell, Go/Git versions, and working directory. `Detect()` computes these values once using `sync.Once` and caches them; all subsequent calls return the cached struct without re-running any subprocesses.

Import path: `github.com/mrlm-net/cure/pkg/env`

```go
import "github.com/mrlm-net/cure/pkg/env"

e := env.Detect()
fmt.Println(e.OS)         // e.g. "darwin"
fmt.Println(e.Arch)       // e.g. "arm64"
fmt.Println(e.GoVersion)  // e.g. "go1.25.0"
fmt.Println(e.GitVersion) // e.g. "git version 2.39.0"
fmt.Println(e.WorkDir)    // e.g. "/Users/user/project"

// Check whether an external tool is available on PATH.
if env.HasTool("docker") {
    // docker is available
}

// Check whether the current directory is inside a git repository.
if env.IsGitRepo() {
    // inside a git repository
}
```

## pkg/doctor

`pkg/doctor` provides a public framework for running project health checks. The `cure doctor` command is built on top of this package, but any Go program can use it directly — either running the built-in checks or composing a custom suite.

Import path: `github.com/mrlm-net/cure/pkg/doctor`

### Core types

```go
// CheckFunc is the check unit — a plain function with no parameters.
type CheckFunc func() CheckResult

// CheckResult holds the name, status, and a human-readable message for one check.
type CheckResult struct {
    Name    string
    Status  CheckStatus
    Message string
}

// CheckStatus values: CheckPass, CheckWarn, CheckFail.
type CheckStatus int
```

### Running checks

`Run` executes a slice of `CheckFunc` values, writes a formatted and ANSI-styled line for each result, and returns tallies by status. Panicking checks are recovered and recorded as `CheckFail`; the remaining checks in the slice continue to run.

```go
import (
    "os"
    "github.com/mrlm-net/cure/pkg/doctor"
)

checks := doctor.BuiltinChecks() // 7 default checks
passed, warned, failed := doctor.Run(checks, os.Stdout)
if failed > 0 {
    os.Exit(1)
}
```

### Extending with custom checks

Append any `CheckFunc` to the slice before calling `Run`:

```go
checks := doctor.BuiltinChecks()

// Add a project-specific check.
checks = append(checks, func() doctor.CheckResult {
    ok, _ := fs.Exists("api/openapi.yaml")
    if ok {
        return doctor.CheckResult{Name: "OpenAPI", Status: doctor.CheckPass, Message: "api/openapi.yaml found"}
    }
    return doctor.CheckResult{Name: "OpenAPI", Status: doctor.CheckFail, Message: "api/openapi.yaml missing"}
})

doctor.Run(checks, os.Stdout)
```

### Built-in checks

`BuiltinChecks()` returns the 7 default checks in canonical order:

| Check | Pass condition | Fail/Warn |
|-------|---------------|-----------|
| README | `README.md` or `README` present | Fail |
| Tests | `*_test.go` files or `tests/` directory | Fail |
| CI Config | `.github/workflows/`, `.gitlab-ci.yml`, or `.circleci/` | Fail |
| .gitignore | `.gitignore` present | Warn |
| CLAUDE.md | `CLAUDE.md` present | Fail |
| Build Tool | `Makefile`, `package.json`, `Cargo.toml`, or `build.gradle` | Fail |
| Dependency Manifest | `go.mod`, `package.json`, `requirements.txt`, or `Cargo.toml` | Fail |

A new slice is returned on each call to `BuiltinChecks()` so callers can safely mutate it without affecting other callers.

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

Cure is currently at v0.10.0. The v0.10.0 release completed tool use support with multi-turn tool loops for the Claude provider, skill presets (`--skill`), and the MCP tool bridge. The v0.9.0 release added OpenAI and Gemini provider adapters, session tags, and `cure mcp serve`. The upcoming v0.11.0 release extends tool loop support to all three providers (Claude, OpenAI, Gemini).

Earlier milestones:

- **v0.10.0** — Tool use, skills, MCP integration: Claude tool loop (up to 32 turns), `--skill` flag, `ToolsFromMCPServer` bridge, REPL tool annotations
- **v0.9.0** — Multi-provider AI: OpenAI and Gemini adapters, session tags, `cure mcp serve`, API stability docs
- **v0.8.0** — Project Bootstrap: `cure init`, `cure context search`, `cure context export`, `pkg/doctor` extraction
- **v0.7.0** — Generation & Scaffolding: `cure generate scaffold`, `cure generate devcontainer`, `cure generate editorconfig`, `cure generate gitignore`, `cure generate github-workflow`; programmatic `Generate*` API for all AI-file subcommands
- **v0.6.x** — Developer experience foundation: `pkg/prompt`, `pkg/fs`, `pkg/style`, `pkg/env`; `cure doctor`; AI assistant template subcommands (`agents-md`, `copilot-instructions`, `cursor-rules`, `windsurf-rules`, `gemini-md`)
- **v0.5.0** — `pkg/agent`: provider-agnostic AI agent context management, `cure context` command group, Anthropic Claude adapter
- **v0.4.x** — `pkg/mcp` for stdlib-only MCP server implementation, shell auto-completion
- **v0.1.0–v0.3.x** — CLI framework (`pkg/terminal`), configuration (`pkg/config`), network tracing (`pkg/tracer`), template generation (`pkg/template`)

Upcoming milestones:

- **v1.0.0** — API stability for all `pkg/` packages; freeze of breaking changes

The v1.0.0 milestone marks API stability for `pkg/` packages and freeze of breaking changes. Track progress on the [GitHub Projects board](https://github.com/orgs/mrlm-net/projects/9).

## License

Licensed under the [Apache License 2.0](LICENSE).
