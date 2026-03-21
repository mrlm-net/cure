# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/mrlm-net/cure/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/mrlm-net/cure/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/mrlm-net/cure/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/mrlm-net/cure/releases/tag/v0.1.0
