# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/mrlm-net/cure/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/mrlm-net/cure/releases/tag/v0.1.0
