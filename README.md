# Cure

**A Go CLI tool for automating development tasks — generating templates for AI assistants, devcontainer configurations, and structured file formats.**

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev/dl/) [![Build](https://github.com/mrlm-net/cure/actions/workflows/test.yml/badge.svg)](https://github.com/mrlm-net/cure/actions) [![Release](https://img.shields.io/github/v/release/mrlm-net/cure)](https://github.com/mrlm-net/cure/releases) [![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/mrlm-net/cure)](https://goreportcard.com/report/github.com/mrlm-net/cure)

## Overview

Cure automates repetitive development tasks through code generation and network diagnostics. Generate templates for AI assistants (`CLAUDE.md`), trace HTTP/TCP/UDP connections with detailed timing and metadata, and output results in developer-friendly formats (NDJSON, HTML). Built with zero external dependencies using only Go's standard library, cure is designed as a foundation for developers who need reliable, auditable tooling without dependency bloat.

The project is under active development — currently at v0.4.0 with a stable API planned for v1.0.0. Cure's modular architecture separates reusable packages (`pkg/`) from application-specific logic (`internal/`), making it straightforward to extend with custom commands or embed cure's packages into other tools.

## Key Features

- **Template generation** — Create `CLAUDE.md` project context files for AI assistants with interactive or flag-driven configuration
- **Network tracing** — Trace HTTP requests (DNS resolution, TLS handshake, response timing), TCP connections, and UDP packet exchanges with detailed event streams
- **Flexible output** — Export trace data as NDJSON for log aggregation or HTML for visual inspection with syntax-highlighted JSON payloads
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

### Tracing

- `cure trace http <url>` — Trace HTTP request with DNS resolution, TLS handshake, request/response headers, and timing
- `cure trace tcp <address>` — Trace TCP connection with handshake timing and connection metadata
- `cure trace udp <address>` — Trace UDP packet exchange with send/receive timing

**Common flags**: `--format` (ndjson|html), `--output <file>`, `--dry-run`

### Generation

- `cure generate claude-md` — Generate `CLAUDE.md` project context file with conventions and AI assistant configuration

### Completion

- `cure completion bash` — Generate bash completion script
- `cure completion zsh` — Generate zsh completion script

Run `cure help <command>` for detailed usage and flag descriptions.

## Design Principles

Cure is built on three core principles: **zero external dependencies**, **reusable package architecture**, and **minimal abstraction**.

**Zero dependencies** — cure uses only Go's standard library. This eliminates supply chain risk, reduces build times, simplifies audits, and ensures cure remains buildable and maintainable for years without dependency updates. The tradeoff is implementing more functionality from scratch, but the benefits outweigh the cost for a foundational tool.

**Reusable packages** — the `pkg/` directory contains independently useful libraries that any Go project can import: `pkg/terminal` for CLI routing and flag parsing, `pkg/config` for hierarchical configuration merging, `pkg/tracer` for network event tracing, and `pkg/template` for embedded template rendering. Each package follows a single responsibility and can be used without importing cure's application logic.

**Minimal abstraction** — cure favors composition over complex abstractions. Commands implement a simple interface (`Name()`, `Description()`, `Usage()`, `Flags()`, `Run()`), configuration is plain `map[string]interface{}` with dot-notation access, and the router dispatches commands without heavy middleware stacks. This keeps the codebase readable and debuggable.

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

Cure is currently at v0.4.0. The next milestone is v0.5.0, which focuses on foundation packages for developer experience: `pkg/prompt` for interactive prompts, `pkg/fs` for filesystem operations, `pkg/style` for terminal styling, and `pkg/env` for environment inspection. This release will also introduce the `cure doctor` command for validating the development environment and `--dry-run` support across commands.

The v1.0.0 milestone marks API stability for `pkg/` packages and freeze of breaking changes. Track progress on the [GitHub Projects board](https://github.com/orgs/mrlm-net/projects/9).

Future releases (v0.6.0+) will expand template generation capabilities, introduce template directories for multi-file generation, and explore plugin architecture for extensibility.

## License

Licensed under the [Apache License 2.0](LICENSE).
