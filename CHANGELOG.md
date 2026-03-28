# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.8.0] - 2026-03-27

### Added

- `pkg/doctor` — public reusable health-check framework extracted from `internal/commands/doctor`; zero imports from `internal/` or `cmd/`
- `pkg/doctor`: `CheckFunc` type — `func() CheckResult`; follows `http.HandlerFunc` pattern for extensible, composable checks
- `pkg/doctor`: `CheckResult` struct with `Name`, `Status`, and `Message` fields
- `pkg/doctor`: `CheckStatus` constants — `CheckPass`, `CheckWarn`, `CheckFail`
- `pkg/doctor`: `Run(checks []CheckFunc, w io.Writer) (passed, warned, failed int)` — executes checks in order, formats results with ANSI styling, recovers panics (panicking checks recorded as `CheckFail`; remaining checks continue to run)
- `pkg/doctor`: `BuiltinChecks() []CheckFunc` — returns the 7 default health checks in canonical order; new slice per call to prevent mutation by callers
- `internal/commands/doctor`: refactored to thin adapter over `pkg/doctor`; type aliases (`CheckFunc`, `CheckResult`, `CheckStatus`) and var aliases for built-in checks preserve backward compatibility for internal tests
- `cure doctor` custom checks — reads `doctor.checks` array from `.cure.json`; each entry: `name` (string), `command` (string), `pass_on` (`"exit_0"` | `"stdout_contains:<pattern>"`)
- `cure doctor` custom checks: each check runs via `os/exec` directly (no `sh -c`, no shell injection surface); `strings.Fields` splits the command into argv; quoted arguments are not supported
- `cure doctor` custom checks: 10-second per-check timeout → `CheckWarn`; command not found → `CheckFail`; unknown `pass_on` rule rejected at load time (not silent fallback)
- `cure doctor` custom checks: missing `.cure.json` silently produces no checks (opt-in, not required); parse errors print a warning and skip custom checks without aborting built-in checks
- `cure doctor --no-custom` — skips all custom checks from `.cure.json`
- `cure context search <query> [--format table|ndjson]` — case-insensitive substring search across all saved session message history; reports ID, provider, creation time, match count, and a short excerpt from the first match per session
- `cure context search`: `firstExcerpt` — rune-based UTF-8–safe excerpt windowing (80 runes, 20 rune leading context); adds leading/trailing ellipses when the window does not cover content boundaries
- `cure context search --format ndjson` — one JSON object per matching session with fields `id`, `provider`, `created_at`, `match_count`, `excerpt`
- `cure context export <session-id> [--format markdown|ndjson] [--output <file>]` — read-only export of a saved session; never mutates the session in the store
- `cure context export`: Markdown format — H1 heading with session ID, metadata table (Provider, Model, Created, Updated, Fork of), H2 section per message with role as heading
- `cure context export`: NDJSON format — single compact JSON object followed by a newline; compatible with `jq` in line-by-line mode
- `cure context export --output <file>` — writes output to a file via `pkg/fs.AtomicWrite`; creates parent directories with `pkg/fs.EnsureDir`; flags must precede the positional argument
- `cure init [flags]` — interactive wizard that bootstraps a complete project scaffold in a single command; continues on failure and prints a `ok <component>` / `x <component>: <error>` summary
- `cure init` interactive mode — prompts for project name (`Required`), language (`SingleSelect`: Go, Node.js, Python, Rust, Other), AI assistant files (`MultiSelect`), devcontainer, CI workflow, editorconfig, gitignore
- `cure init --non-interactive` — accepts all values from flags; `--name` and `--language` are required; defaults all infrastructure components to enabled; defaults `--ai-tools` to all six AI file generators
- `cure init --ai-tools <ids>` — comma-separated subset of `claude-md,agents-md,copilot-instructions,cursor-rules,windsurf-rules,gemini-md`; validated before any generator runs
- `cure init --devcontainer`, `--ci`, `--editorconfig`, `--gitignore` — boolean flags (default `true`) to opt out individual infrastructure components in non-interactive mode
- `cure init --dry-run` — previews all generator output without touching the filesystem
- `cure init --force` — passes `--force` to every generator; overwrites existing files without prompting

## [0.7.0] - 2026-03-27

### Added

- `cure generate scaffold` — interactive `MultiSelect` wizard that generates all AI context files in one pass; calls `GenerateClaudeMD`, `GenerateAgentsMD`, `GenerateCopilotInstructions`, `GenerateCursorRules`, `GenerateWindsurfRules`, `GenerateGeminiMD` with continue-on-error semantics; flags: `--select` (comma-separated subset), `--non-interactive` (generates all without prompts), `--dry-run`, `--force`
- `cure generate devcontainer` — generates `.devcontainer/devcontainer.json` (and an optional `Dockerfile` stub) for VS Code Dev Containers and GitHub Codespaces; uses `encoding/json.MarshalIndent` for JSON generation (no template injection risk); flags: `--name`, `--base-image`, `--dockerfile`, `--extensions`, `--post-create-command`, `--output-dir`, `--dry-run`, `--force`, `--non-interactive`
- `cure generate editorconfig` — generates `.editorconfig` with per-language indent rules; supported languages: `go`, `javascript`, `python`, `rust`, `java`, `shell`, `markdown`, `yaml`, `generic`; sections emitted in canonical order for deterministic output; flags: `--languages` (comma-separated), `--output`, `--dry-run`, `--force`, `--non-interactive`
- `cure generate gitignore` — generates `.gitignore` from 11 embedded profiles: `go`, `node`, `python`, `rust`, `java`, `macos`, `windows`, `linux`, `jetbrains`, `vscode`, `vim`; cross-profile deduplication removes duplicate patterns; flags: `--profiles` (comma-separated), `--output`, `--dry-run`, `--force`, `--non-interactive`
- `cure generate github-workflow` — generates `.github/workflows/ci.yml` for GitHub Actions CI targeting Go projects; configurable `--go-version` with format validation; optional `--lint` (adds `go vet` step) and `--coverage` (adds codecov upload); flags: `--go-version`, `--lint`, `--coverage`, `--output`, `--dry-run`, `--force`, `--non-interactive`
- Programmatic API for all AI-file subcommands — each existing subcommand (`claude-md`, `agents-md`, `copilot-instructions`, `cursor-rules`, `windsurf-rules`, `gemini-md`) now exports a `Generate*(ctx, w io.Writer, opts XxxOpts) error` function callable without the `terminal.Context` layer

### Security

- `cure generate devcontainer`: `--base-image` validated against `^[a-zA-Z0-9][a-zA-Z0-9._/:@-]*$` to prevent Dockerfile injection
- All output paths sanitised with `filepath.Clean`

## [0.6.3] - 2026-03-26

### Added

- `cure generate agents-md` — generates `AGENTS.md` (cross-tool standard; adopted by GitHub Copilot, Cursor, Devin, Gemini CLI, OpenAI Codex)
- `cure generate copilot-instructions` — generates `.github/copilot-instructions.md` with YAML frontmatter (`applyTo: "**"`); creates `.github/` directory automatically
- `cure generate cursor-rules` — generates `.cursor/rules/project.mdc` with YAML frontmatter (`alwaysApply: true`); creates `.cursor/rules/` directory automatically
- `cure generate windsurf-rules` — generates `.windsurfrules` (plain text, Windsurf-style numbered rules)
- `cure generate gemini-md` — generates `GEMINI.md` (Google Gemini CLI auto-discovery format)
- All new subcommands support `--dry-run`, `--force`, `--output`, `--non-interactive`, and the same prompt flags as `cure generate claude-md`

## [0.6.2] - 2026-03-25

### Changed

- `internal/commands/generate`: migrated from internal `Prompter` struct to `pkg/prompt.NewPrompter`; migrated from `os.WriteFile` to `pkg/fs.AtomicWrite` and `pkg/fs.Exists` — deleted `internal/commands/generate/prompt.go` and `prompt_test.go` (308 lines removed)

### Added

- `pkg/prompt`, `pkg/fs`, `pkg/style`, `pkg/env`: added `example_test.go` with runnable `Example*` functions for Go documentation and test coverage
- `CLAUDE.md`: expanded architecture diagram with all current `pkg/` packages
- `README.md`: added sections for `pkg/prompt`, `pkg/fs`, `pkg/style`, `pkg/env`; documented `cure doctor` and `--dry-run` flag; updated roadmap to reflect v0.6.x releases

## [0.6.1] - 2026-03-25

### Added

- `cure doctor` — project health check command; runs 7 checks (README, tests, CI, .gitignore, CLAUDE.md, build tool, dependency manifest) and prints pass/warn/fail per check with an exit code of 1 on any failure
- `cure doctor`: `CheckFunc` type — `func() CheckResult`; follows `http.HandlerFunc` pattern for extensible, composable checks
- `pkg/template`: `SetConfig(cfg)` — injects `*config.Config` into the template registry; triggers lazy rebuild on next use
- `pkg/template`: custom template directories — 4-level search order: embedded → config-defined dirs → `~/.cure/templates/` → `.cure/templates/`; silently skips missing directories
- `cure generate claude-md --dry-run` — prints the generated output to stdout instead of writing to disk; exits 0 without touching the filesystem

## [0.6.0] - 2026-03-25

### Added

- `pkg/prompt` — interactive terminal input with `Prompter` struct injecting `io.Writer`/`io.Reader` for full testability
- `pkg/prompt`: `Required(prompt, default)` — repeats until non-empty; returns default on Enter
- `pkg/prompt`: `Optional(prompt, default)` — returns default on Enter; never repeats
- `pkg/prompt`: `Confirm(prompt)` — accepts y/yes/n/no case-insensitively; repeats on invalid input
- `pkg/prompt`: `SingleSelect(prompt, options)` — numbered 1-based menu; re-prompts on invalid selection
- `pkg/prompt`: `MultiSelect(prompt, options)` — comma-separated numbers, "all", or "none"; preserves original option order; deduplicates
- `pkg/prompt`: `IsInteractive(stdin)` — detects terminal via `*os.File` + `os.ModeCharDevice`; no syscall imports
- `pkg/fs` — crash-safe filesystem operations with atomic write semantics
- `pkg/fs`: `AtomicWrite(path, content, perm)` — temp file in same directory → fsync → atomic rename; preserves existing file permissions; cleans up temp on error
- `pkg/fs`: `EnsureDir(path, perm)` — `os.MkdirAll` wrapper; returns error if path exists and is not a directory
- `pkg/fs`: `Exists(path)` — stat wrapper returning `(bool, error)`; error only for permission/I/O failures
- `pkg/fs`: `TempDir(prefix)` — `os.MkdirTemp` wrapper returning the created path
- `pkg/fs`: `SetPermissions(path, mode)` — `os.Chmod` wrapper
- `pkg/style` — minimal ANSI terminal styling (~92 lines); standalone functions per ADR-001
- `pkg/style`: 8 foreground color functions: `Red`, `Green`, `Yellow`, `Blue`, `Magenta`, `Cyan`, `White`, `Gray`
- `pkg/style`: 3 text style functions: `Bold`, `Dim`, `Underline`
- `pkg/style`: `Reset(text)` — strips all ANSI escape sequences via compiled regexp
- `pkg/style`: `Enabled()`, `Disable()`, `Enable()` — runtime toggle; `NO_COLOR` env var disables styling at startup
- `pkg/env` — runtime environment detection with cached singleton per ADR-004
- `pkg/env`: `Environment` struct with `OS`, `Arch`, `Shell`, `GoVersion`, `GitVersion`, `WorkDir` fields
- `pkg/env`: `Detect()` — `sync.Once`-cached detection; ~7 ns/op on subsequent calls
- `pkg/env`: `HasTool(name)` — `exec.LookPath` wrapper returning bool; intentionally uncached
- `pkg/env`: `IsGitRepo()` — walks directory tree looking for `.git`; no git binary invocation

### Fixed

- `pkg/tracer/http`, `pkg/tracer/tcp`, `pkg/tracer/udp`: extracted nil-safe `emit()` helper eliminating repeated `if cfg.emitter != nil` guard branches; reduces cyclomatic complexity of `TraceURL` (19→11) and `TraceAddr` tcp (22→11), udp (17→10)
- `internal/commands/trace/http`: removed dead `timeout` variable that was computed but never passed to `http.TraceURL` (ineffassign)
- `pkg/terminal/errors_test.go`: split intentional misspelling literals in `BenchmarkLevenshtein` with string concatenation to suppress misspell linter false positives

## [0.5.0] - 2026-03-24

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

[Unreleased]: https://github.com/mrlm-net/cure/compare/v0.7.0...HEAD
[0.7.0]: https://github.com/mrlm-net/cure/compare/v0.6.3...v0.7.0
[0.6.3]: https://github.com/mrlm-net/cure/compare/v0.6.2...v0.6.3
[0.6.2]: https://github.com/mrlm-net/cure/compare/v0.6.1...v0.6.2
[0.6.1]: https://github.com/mrlm-net/cure/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/mrlm-net/cure/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/mrlm-net/cure/compare/v0.4.1...v0.5.0
[0.3.0]: https://github.com/mrlm-net/cure/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/mrlm-net/cure/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/mrlm-net/cure/releases/tag/v0.1.0
