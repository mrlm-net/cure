# Cure

A Go CLI tool for automating development tasks — generating templates for AI assistants, devcontainer configurations, and other structured file formats (YAML, JSON, etc.). Designed as a public OSS project under Apache 2.0.

## Tech Stack

- **Language**: Go 1.25+
- **Dependencies**: Standard library only (no external packages unless explicitly requested)
- **Build tool**: `make` + `go build`
- **Test framework**: `testing` (stdlib)

## Architecture

```
cmd/cure/              Entry point — thin main, wires internal packages
internal/              Private application logic (not importable by other projects)
  internal/commands/   CLI command implementations (VersionCommand, etc.)
pkg/                   Public reusable libraries (importable by anyone)
  pkg/terminal/        Command routing, flag handling, help generation, execution modes
  pkg/trace/           HTTP tracing utilities
  pkg/template/        Template generation engine
  ...                  Each package follows single responsibility principle
```

**Design principles**:
- `pkg/` packages are the public API — they must be independently useful, well-tested, and stable
- `internal/` contains cure-specific wiring that glues `pkg/` packages together
- `cmd/cure/main.go` is a thin entry point — all logic lives in `internal/` or `pkg/`
- Zero external dependencies unless explicitly approved — stdlib is the foundation
- Every exported function has tests, every package has benchmarks for performance-critical paths

## Development

### Prerequisites

- Go 1.25+
- `make`

### Commands

| Command | Purpose |
|---------|---------|
| `make build` | Build the `cure` binary to `bin/` |
| `make test` | Run all tests with race detector |
| `make test-coverage` | Run tests with coverage report |
| `make lint` | Run `go vet` |
| `make clean` | Remove build artifacts |

### Getting Started

```sh
git clone git@github.com:mrlm-net/cure.git
cd cure
make test
make build
./bin/cure help
```

## Conventions

### Code

- Follow standard Go conventions (`gofmt`, `go vet`, effective Go)
- Package names are short, lowercase, single-word (or hyphenated for `pkg/` if needed)
- Exported types and functions have doc comments
- Prefer returning `error` over panicking
- Use `io.Reader`/`io.Writer` interfaces for I/O — never hardcode `os.Stdout`
- Table-driven tests with `t.Run` subtests
- Benchmarks for performance-sensitive code paths

### Project Structure

- `pkg/` packages MUST NOT import from `internal/` or `cmd/`
- `internal/` packages MAY import from `pkg/`
- `cmd/` packages MAY import from both `internal/` and `pkg/`
- Each `pkg/` package has a clear, single responsibility
- New reusable functionality goes in `pkg/`, cure-specific logic in `internal/`

### Git & Workflow

- All work happens via PRs based on assigned GitHub Issues
- Branch naming: `feat/<issue>-<short-description>`, `fix/<issue>-<short-description>`
- Commit messages: Conventional Commits (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`, `chore:`)
- PRs require passing CI (tests + lint) before merge
- PRs must request review from **Copilot** (`gh pr edit <number> --add-reviewer copilot`)
- Squash merge to `main`

### Versioning

- This is a Go module — **all version tags MUST use the `v` prefix** (e.g., `v0.1.0`, `v1.2.3`)
- Tags without the `v` prefix are invisible to Go tooling (`go get`, `go install`, module proxy)
- Follow [Semantic Versioning 2.0.0](https://semver.org/): `vMAJOR.MINOR.PATCH`
- Pre-release: `v0.x.y` until API stabilizes, then `v1.0.0`

## Workload Management

Agents track work decisions, blockers, and outcomes in GitHub Issues.

**System**: GitHub Issues
**Repository**: mrlm-net/cure
**Configuration**:
- Use the `github-issues` skill for issue management
- Agents post decisions (e.g., "Chose X over Y because Z"), blockers, quality gate failures, and phase outcomes
- Agents do NOT post progress notifications or status updates — keep it human-consumable

## MRLM Plugin Usage

This project uses the [mrlm devstack plugin](https://github.com/mrlm-net/devstack) for AI-assisted development. Available commands:

| Command | What it does |
|---------|-------------|
| `/spec` | Gather requirements, write user stories and acceptance criteria |
| `/design` | Design system architecture, define interfaces and technical patterns |
| `/build` | Implement code and unit tests (engineer only, no review) |
| `/review` | Systematic code review for correctness, style, and performance |
| `/test` | Run E2E, performance, UX, and accessibility testing |
| `/secure` | Vulnerability scan, SBOM generation, OWASP compliance check |
| `/deploy` | Infrastructure provisioning and deployment automation |
| `/make` | Full SDLC pipeline — from requirements through security scan |
| `/ask` | Ask any question using full agent toolkit (read-only) |
| `/write` | Generate articles, documentation, or marketing content |
| `/release` | Publish versioned release with changelog, git tag, and GitHub Release |
| `/init` | Initialize project structure and CLAUDE.md |

### Recommended Workflow

For new features, use the full pipeline: `/make [feature description]`

For focused work, chain individual commands:
1. `/spec` — define what to build
2. `/design` — plan how to build it
3. `/build` — implement it
4. `/review` — review the code
5. `/test` — verify it works
6. `/secure` — check for vulnerabilities
