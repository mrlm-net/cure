# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
