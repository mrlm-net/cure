# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.6.0] - 2026-03-24

### Added

- `pkg/agent/store`: `JSONStore` — file-backed `SessionStore` that persists each session as a JSON file; tilde expansion in `dir`, lazy directory creation on first `Save`
- `pkg/agent/store`: `NewJSONStore(dir string) (*JSONStore, error)` — resolves `~` to the current user's home directory and returns an absolute path
- `pkg/agent/store`: atomic writes via `os.CreateTemp` + `os.Rename`; session files receive mode `0600`, directory receives mode `0700`
- `pkg/agent/store`: `Save` is protected by `sync.Mutex` — safe for concurrent use
- `pkg/agent/store`: `List` silently skips corrupt or unreadable JSON files and returns a non-nil empty slice when the store directory does not exist
- `pkg/agent/store`: ID validation rejects empty strings, `/`, `\`, and null bytes to prevent path traversal
- `pkg/agent/store`: compile-time interface assertion `var _ agent.SessionStore = (*JSONStore)(nil)`
- `internal/agent/claude` — Anthropic Claude provider adapter; registers as `"claude"` via `init()` using the blank-import driver pattern
- `internal/agent/claude`: `NewClaudeAgent(cfg map[string]any) (agent.Agent, error)` — reads `api_key_env` (default `ANTHROPIC_API_KEY`), `model` (default `claude-opus-4-6`), `max_tokens` (default `8192`)
- `internal/agent/claude`: `Run(ctx, session)` — `iter.Seq2[Event, error]` streaming response via goroutine+channel bridge over the Anthropic SDK
- `internal/agent/claude`: `CountTokens(ctx, session)` — calls Anthropic `/v1/messages/count_tokens`; HTTP 404 responses map to `agent.ErrCountNotSupported`
- `internal/agent/claude`: `sanitiseError` — redacts the API key value from all error strings before surfacing them
- New external dependency: `github.com/anthropics/anthropic-sdk-go v1.27.1`
- `pkg/agent` — provider-agnostic AI agent abstraction with core interfaces, session management, and a global registry
- `pkg/agent`: `Agent` interface with `Run(ctx, session) iter.Seq2[Event, error]`, `CountTokens(ctx, session) (int, error)`, and `Provider() string`
- `pkg/agent`: `AgentFactory` constructor type — `func(cfg map[string]any) (Agent, error)`
- `pkg/agent`: `EventKind` constants (`token`, `start`, `done`, `error`) and `Event` struct with JSON serialisation
- `pkg/agent`: `Role` constants (`user`, `assistant`, `system`) and `Message` struct
- `pkg/agent`: global registry — `Register(name, factory)` panics on empty/duplicate names (matches `http.Handle` semantics); `New(name, cfg)` wraps `ErrProviderNotFound`; `Registered()` returns sorted provider names
- `pkg/agent`: provider adapters live in `internal/agent/<provider>/` and self-register via `init()` using the blank-import driver pattern (same as `database/sql`)
- `pkg/agent`: `Session` struct with `ID`, `Provider`, `Model`, `SystemPrompt`, `History`, `CreatedAt`, `UpdatedAt`, `ForkOf`, `Tags`
- `pkg/agent`: `NewSession(provider, model)` — 128-bit `crypto/rand` session ID; `Fork()` — deep copy with new ID and `ForkOf` tracking; `AppendUserMessage` / `AppendAssistantMessage`
- `pkg/agent`: `SessionStore` interface — `Save`, `Load`, `List`, `Delete`, `Fork` — for concrete persistence implementations
- `pkg/agent`: `RunSessionStoreTests(t, store)` — shared test suite callable from any concrete store's test file
- `pkg/agent`: sentinel errors `ErrProviderNotFound`, `ErrSessionNotFound`, `ErrCountNotSupported`
- `pkg/agent`: `EstimateTokens(text)` — len/4 token heuristic for context window budget calculations
- `cure context new --provider <name> --message <text>` — start a new AI conversation session; streams the response and persists the session to `~/.local/share/cure/sessions/`
- `cure context resume <id> --message <text>` — continue an existing session by appending a new user message and streaming the response
- `cure context list [--format text|ndjson]` — list all saved sessions sorted newest-first; provider name truncated to 10 chars in text output
- `cure context fork <id>` — deep-copy a session with a new ID; prints the forked ID to stdout
- `cure context delete [--yes] <id>` — delete a session; prompts for confirmation unless `--yes` is supplied
- `cure context` REPL mode — when invoked without a subcommand, enters an interactive read-evaluate-print loop for multi-turn conversations
- `cmd/cure`: claude provider defaults (`agent.claude.model: claude-opus-4-6`, `agent.claude.max_tokens: 8192`) added to `loadConfig()`

### Security

- `pkg/agent/store`: `validateID` replaced deny-list with allow-list regex `^[0-9a-f]{1,64}$` — eliminates path-traversal surface for all characters outside lowercase hex
- `internal/commands/context`: `DefaultStoreDir()` now returns `(string, error)` — fails loudly if the home directory cannot be determined instead of silently falling back to a relative path

## [0.4.1] - 2026-03-21

### Added

- `pkg/mcp` — stdlib-only MCP (Model Context Protocol) server abstraction targeting protocol version `2025-03-26`
- `pkg/mcp`: `Tool`, `Resource`, and `Prompt` interfaces mirroring the `pkg/terminal.Command` pattern
- `pkg/mcp`: `Server` with functional options (`WithName`, `WithVersion`, `WithAddr`, `WithAllowedOrigins`); `RegisterTool`, `RegisterResource`, `RegisterPrompt` preserve registration order
- `pkg/mcp`: `FuncTool()` adapter for registering anonymous functions as MCP tools
- `pkg/mcp`: `Schema()` fluent builder — `.String()`, `.Number()`, `.Integer()`, `.Bool()`, `Required()`, `WithEnum()`, `WithDefault()`
- `pkg/mcp`: `Text()` and `Textf()` content constructors for tool call responses
- `pkg/mcp`: `ServeStdio(ctx)` — stdio transport for Claude Code and local MCP client integration
- `pkg/mcp`: `ServeHTTP(ctx, addr)` — HTTP Streamable transport with SSE, per-session management, and Origin validation; default bind `127.0.0.1:8080` (loopback only)
- `pkg/mcp`: `Serve(ctx)` — auto-detects transport based on stdin pipe state; `IsStdinPipe()` is exported for testing
- `pkg/mcp`: HTTP timeouts (ReadHeader 10s, Read 30s, Idle 120s); session IDs generated with 128-bit `crypto/rand`; `"null"` Origin explicitly rejected when an allowlist is set
- `pkg/tracer/dns` — DNS resolution tracer library with `TraceDNS(ctx, hostname, ...Option)` API; functional options `WithEmitter`, `WithDryRun`, `WithTimeout`, `WithServer`, `WithCount`, `WithInterval`
- `pkg/tracer/dns`: emits `dns_query_start` and `dns_query_done` events with resolved IP addresses, CNAME chain, resolution duration, optional error, and RFC 1918 private IP classification per address
- `cmd/cure`: `trace dns <hostname>` subcommand with `--format` (json|html), `--out-file`, `--dry-run`, `--timeout`, `--server` (IP or IP:port), `--count`, and `--interval` flags
- `cure trace dns --server`: accepts IP addresses only — hostnames are rejected to avoid DNS bootstrapping circularity
- `cure trace dns --count` + `--interval`: repeat query N times with configurable delay for detecting intermittent DNS flapping

## [0.3.0] - 2026-02-14

### Added

- `pkg/config` — hierarchical configuration management with `DeepMerge`, dot-notation `Get`/`Set`, `Environment` loader, `JSONFile` loader with tilde expansion
- `pkg/terminal`: config integration — `Config` field on `Router` and `Context`, `WithConfig` option, config precedence chain (defaults < global < local < env < flags)
- `pkg/tracer` — network tracing for HTTP (via `net/http/httptrace`), TCP, and UDP protocols with `Event`/`Emitter` architecture
- `pkg/tracer`: NDJSON and HTML output formatters
- `pkg/tracer`: header redaction for Authorization, Cookie, Set-Cookie
- `pkg/tracer`: dry-run mode for synthetic events without network I/O
- `cmd/cure`: `trace http|tcp|udp` subcommand with full flag support

## [0.2.0] - 2026-02-14

### Added

- `pkg/terminal`: command aliases via `AliasProvider` interface and `RegisterWithAliases`
- `pkg/terminal`: subcommand support — Router implements Command interface for nested routing
- `pkg/terminal`: advanced error handling — `CommandError`, `CommandNotFoundError`, `NoCommandError`, `FlagParseError` with "did you mean?" suggestions
- `pkg/terminal`: structured logging via `WithLogger` option using `log/slog`
- `pkg/terminal`: signal handling (SIGINT/SIGTERM) with grace period via `WithSignalHandler`, `WithTimeout`, `WithGracePeriod`
- `pkg/terminal`: `ConcurrentRunner` for parallel command execution
- `pkg/terminal`: `PipelineRunner` for sequential pipeline execution

### Fixed

- `pkg/terminal`: code review findings from v0.2.0 review

## [0.1.0] - 2026-02-14

### Added

- `pkg/terminal` — reusable CLI framework with command routing, flag handling, and help generation
  - `Command` interface for declarative command definitions
  - `Context` struct providing parsed args, flags, and output streams
  - `Router` with radix tree-based command dispatch and functional options (`WithStdout`, `WithStderr`, `WithRunner`)
  - `SerialRunner` for sequential command execution with context cancellation
  - `HelpCommand` with `CommandRegistry` interface for dynamic help generation
  - `ConcurrentRunner` and `PipelineRunner` stubs for future implementation
- `internal/commands` — cure-specific command implementations
  - `VersionCommand` printing version information
- `cmd/cure/main.go` — thin entry point wiring the terminal router
- Project scaffolding: Makefile, Go module, CI-ready test and lint targets

[Unreleased]: https://github.com/mrlm-net/cure/compare/v0.4.1...HEAD
[0.3.0]: https://github.com/mrlm-net/cure/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/mrlm-net/cure/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/mrlm-net/cure/releases/tag/v0.1.0
