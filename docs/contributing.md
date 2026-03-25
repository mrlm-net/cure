---
title: "Contributing"
description: "Development setup, conventions, and workflow for contributing to cure"
order: 1
section: "development"
---

# Contributing

Contributions are welcome. Cure uses a structured workflow based on GitHub Issues and Pull Requests.

## Prerequisites

- Go 1.25 or later
- `make`

## Getting started

Clone the repository, run tests, and build the binary:

```sh
git clone https://github.com/mrlm-net/cure.git
cd cure
make test
make build
./bin/cure help
```

## Build commands

| Command | Purpose |
|---------|---------|
| `make build` | Build the `cure` binary to `bin/` |
| `make test` | Run all tests with race detector |
| `make test-coverage` | Run tests with coverage report |
| `make lint` | Run `go vet` for static analysis |
| `make clean` | Remove build artifacts |

## Workflow

All work starts with a GitHub Issue describing the feature or bug.

1. Create or find an issue describing the change
2. Create a feature branch: `feat/<issue>-<description>` or `fix/<issue>-<description>`
3. Implement changes with tests
4. Submit a PR targeting `main`
5. PRs require passing `make test` and `make lint` before merge
6. Use Conventional Commits for commit messages: `feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`
7. PRs are squash-merged to maintain a clean commit history

## Code organization

- **`pkg/`** — New reusable functionality belongs here. Packages in `pkg/` must not import from `internal/` or `cmd/`.
- **`internal/`** — Application-specific wiring that glues `pkg/` packages together.
- **`cmd/cure/`** — Thin entry point. All logic lives in `internal/` or `pkg/`.

Follow the single responsibility principle — each package has one clear purpose.

## Conventions

The codebase follows standard Go conventions (`gofmt`, `go vet`, effective Go):

- Exported types and functions have doc comments
- Prefer returning `error` over panicking
- Use `io.Reader`/`io.Writer` interfaces for I/O — never hardcode `os.Stdout`
- Table-driven tests with `t.Run` subtests
- Benchmarks for performance-sensitive code paths
- Package names are short, lowercase, single-word

## Testing standards

Tests use the standard `testing` package with table-driven patterns:

```sh
make test           # Run all tests with race detector
make test-coverage  # Run tests with coverage report
```

Add test coverage for new code paths. Write benchmarks for performance-critical paths.

## Zero dependencies policy

Cure uses only Go's standard library for core functionality. New external dependencies require explicit discussion and approval. The tradeoff is implementing more functionality from scratch, but the benefits (supply chain safety, build speed, auditability) outweigh the cost for a foundational tool.

External dependencies are currently confined to:
- `github.com/anthropics/anthropic-sdk-go` — in `internal/agent/claude` only

For detailed guidance, see `CLAUDE.md` in the repository root.
