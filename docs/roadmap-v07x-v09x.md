# Cure v0.7.x–v0.9.x Milestone Roadmap

**Date:** 2026-03-26
**Planning Horizon:** v0.7.x through v0.9.x (Q2–Q3 2026)
**Decision context:** Architecture design, backlog creation, and sprint sequencing for the next three minor release families.

---

## Table of Contents

- [Current State Baseline (v0.6.3)](#current-state-baseline-v063)
- [Milestone Overview](#milestone-overview)
- [v0.7.x — Generation and Scaffolding](#v07x--generation-and-scaffolding)
- [v0.8.x — Project Bootstrap and Enhanced Doctor](#v08x--project-bootstrap-and-enhanced-doctor)
- [v0.9.x — Multi-Provider AI and MCP Serve](#v09x--multi-provider-ai-and-mcp-serve)
- [Cross-Milestone Dependency Map](#cross-milestone-dependency-map)
- [Plugin Capability Gap Analysis](#plugin-capability-gap-analysis)
- [Open Questions and Assumptions](#open-questions-and-assumptions)
- [Risk Assessment](#risk-assessment)

---

## Current State Baseline (v0.6.3)

### Commands

| Command | Subcommands | Notes |
|---------|-------------|-------|
| `cure generate` | `claude-md`, `agents-md`, `copilot-instructions`, `cursor-rules`, `windsurf-rules`, `gemini-md`, `k8s-job` | `--dry-run`, `--force`, `--output`, `--non-interactive` flags; embedded templates; custom dirs via `.cure/templates/` |
| `cure context` | `new`, `resume`, `list`, `fork`, `delete`, REPL | Claude-only; sessions in `~/.local/share/cure/sessions/`; `--format text\|ndjson` |
| `cure trace` | `http`, `tcp`, `udp`, `dns` | Stdlib-only HTTP/TCP/UDP/DNS tracing via `pkg/tracer` |
| `cure doctor` | (none) | 7 built-in checks; `CheckFunc` type; exit 1 on failure |
| `cure completion` | (introspective) | Shell auto-completion |

### Public Packages (`pkg/`)

| Package | Responsibility | Key Surface |
|---------|---------------|-------------|
| `pkg/terminal` | CLI framework | `Command`, `Router`, `Context`, `SerialRunner`, functional options |
| `pkg/agent` | Provider-agnostic AI abstractions | `Agent`, `Session`, `SessionStore`, `Event`, registry, `RunSessionStoreTests` |
| `pkg/agent/store` | JSON session persistence | `JSONStore`, atomic writes, allow-list ID validation |
| `pkg/mcp` | MCP server (stdio + HTTP Streamable) | `Server`, `Tool`, `Resource`, `Prompt`, `FuncTool`, `Schema` |
| `pkg/config` | Layered configuration | `ConfigObject`, `DeepMerge`, dot-notation `Get`/`Set` |
| `pkg/tracer` | HTTP/TCP/UDP tracing | Event/Emitter pattern, NDJSON + HTML formatters |
| `pkg/template` | Embedded template engine | `Render`, `List`, `RenderTo`, 4-tier overlay, `SetConfig` |
| `pkg/prompt` | Interactive terminal input | `Prompter`, `Required`, `Optional`, `Confirm`, `SingleSelect`, `MultiSelect` |
| `pkg/fs` | Crash-safe filesystem ops | `AtomicWrite`, `EnsureDir`, `Exists`, `TempDir` |
| `pkg/style` | ANSI terminal styling | 8 colour funcs, 3 style funcs, `NO_COLOR` support |
| `pkg/env` | Runtime environment detection | `Detect()` (cached), `HasTool`, `IsGitRepo` |

### External Dependencies

- `github.com/anthropics/anthropic-sdk-go v1.27.1` (sole non-stdlib dependency)
- Indirect: `tidwall/{gjson,match,pretty,sjson}`, `golang.org/x/sync`

---

## Milestone Overview

| Milestone | Theme | Entry Gate | Exit Gate |
|-----------|-------|-----------|-----------|
| **v0.7.x** | Generation and Scaffolding | v0.6.3 released | `cure generate scaffold` wizard + 4 new generate subcommands |
| **v0.8.x** | Project Bootstrap and Enhanced Doctor | v0.7.x released | `cure init` wizard + `pkg/doctor` public package + context search/export |
| **v0.9.x** | Multi-Provider AI and MCP Serve | v0.8.x released | OpenAI + Gemini adapters + session tags + `cure mcp serve` |

All milestones target stdlib-only additions. Any new external dependency requires explicit approval before implementation begins.

---

## v0.7.x — Generation and Scaffolding

### Problem Statement

`cure generate` can produce individual AI context files, but users who want to bootstrap a new project must run multiple generate commands manually, remember each subcommand name, and manage file conflicts themselves. There is no way to generate non-AI project scaffolding (devcontainer, editorconfig, gitignore, CI workflow) from a single tool. The result is friction at project creation time, which is the moment users most benefit from tooling consistency.

### Business Objective

Reduce the time from `git init` to a fully-configured project skeleton to a single interactive wizard invocation.

### Scope: In

- `cure generate scaffold` — MultiSelect wizard to generate any combination of AI context files in one pass
- `cure generate devcontainer` — `.devcontainer/devcontainer.json` and optional `Dockerfile`
- `cure generate editorconfig` — `.editorconfig` with language-aware rule sets
- `cure generate gitignore` — per-language `.gitignore` via embedded patterns
- `cure generate github-workflow` — GitHub Actions CI YAML (test + lint)

### Scope: Out

- `cure init` (project bootstrap wizard) — deferred to v0.8.x; `scaffold` does not wire new project metadata
- Plugin-provided generate templates — deferred to v1.0.0 (#19)
- Non-GitHub CI providers (GitLab, CircleCI) — deferred; may be added in patch releases within v0.7.x
- Language-specific Dockerfile templates beyond a single-stage base — deferred to v0.8.x

---

### Feature: `cure generate scaffold`

**User story:**

As a developer starting or maintaining a project, I want to run one command that lets me select which AI assistant context files to generate and then produces all of them in a single pass, so that I do not need to remember and invoke each subcommand individually.

**Acceptance Criteria:**

- [ ] `cure generate scaffold` presents a MultiSelect menu listing all AI file subcommands (`claude-md`, `agents-md`, `copilot-instructions`, `cursor-rules`, `windsurf-rules`, `gemini-md`).
- [ ] The user can select any combination including "all" or "none".
- [ ] Each selected file is generated sequentially using the same logic as its standalone subcommand, respecting `--force` and `--dry-run` flags.
- [ ] File conflicts (existing files) prompt individually with overwrite/skip per the existing `--force` flag semantics when run interactively; `--force` skips all prompts.
- [ ] `--non-interactive` mode plus `--select <comma-list>` generates the specified files without any prompts.
- [ ] Exit code is 0 when all selected files are written successfully; non-zero if any file write fails.
- [ ] `--dry-run` prints all selected outputs to stdout without writing any files.
- [ ] `cure generate scaffold --help` lists all selectable file types.

**Edge Cases and Error Handling:**

- Selecting zero items (`none`) exits with code 0 and prints "No files selected."
- A write failure for one selected file does not abort remaining files; all errors are collected and reported at the end.
- Running in a non-TTY without `--non-interactive` and without `--select` defaults to generating all files (same as `all` selection).

**Technical Details:**

- `internal/commands/generate/scaffold.go` — new command implementation.
- Reuses existing `claudemd.go`, `agentsmd.go`, etc. as function-level callers; scaffold calls each subcommand's `run` function directly rather than re-invoking the CLI.
- `pkg/prompt.MultiSelect` is already implemented and suitable for the selection step.
- No new external dependencies.

**Dependencies:**

- Requires v0.6.3 merged; all existing generate subcommands must accept a programmatic `run(ctx, tc)` entry point (refactor gate for scaffold).

---

### Feature: `cure generate devcontainer`

**User story:**

As a developer who uses VS Code Dev Containers or GitHub Codespaces, I want `cure generate devcontainer` to create a `.devcontainer/devcontainer.json` (and optionally a `Dockerfile`) so that I have a reproducible development environment without manually writing JSON.

**Acceptance Criteria:**

- [ ] The command creates `.devcontainer/devcontainer.json` with fields: `name`, `image` (or `build.dockerfile`), `features`, `postCreateCommand`, `customizations.vscode.extensions`.
- [ ] The interactive wizard prompts: project name, base image or Dockerfile, VS Code extension IDs (optional), post-create command (optional).
- [ ] `--non-interactive` generates a minimal devcontainer using sensible defaults (Ubuntu base image, no extensions, no post-create command).
- [ ] When the user selects "Dockerfile", a stub `Dockerfile` is also created in `.devcontainer/`.
- [ ] The generated JSON is valid against the devcontainer spec schema (verified by a round-trip `encoding/json.Unmarshal` in tests).
- [ ] `--dry-run` prints the generated files to stdout without writing.
- [ ] An existing `.devcontainer/devcontainer.json` is not overwritten unless `--force` is supplied.
- [ ] `pkg/fs.EnsureDir` is used to create `.devcontainer/` atomically.

**Edge Cases and Error Handling:**

- If `.devcontainer/` exists but is a file (not a directory), the command returns a clear error and exits non-zero.
- Extension IDs provided by the user that contain spaces are trimmed before insertion.

**Technical Details:**

- New embedded template `pkg/template/templates/devcontainer.tmpl` (JSON format).
- `internal/commands/generate/devcontainer.go`.
- Dockerfile template lives at `pkg/template/templates/devcontainer-dockerfile.tmpl`.
- No new external dependencies; `encoding/json` for struct marshalling.

---

### Feature: `cure generate editorconfig`

**User story:**

As a developer working across editors and languages, I want `cure generate editorconfig` to produce a `.editorconfig` file with per-language indent rules so that my project has consistent formatting settings without manual research.

**Acceptance Criteria:**

- [ ] The command presents a MultiSelect menu of supported languages: Go, JavaScript/TypeScript, Python, Rust, Java, Shell, Markdown, YAML, Generic.
- [ ] Each selected language produces a corresponding `[*.{ext}]` section with `indent_style`, `indent_size`, `end_of_line`, `charset`, and `trim_trailing_whitespace`.
- [ ] A `[*]` root section is always emitted with `root = true` and universal defaults.
- [ ] `--non-interactive` generates a `[*]` section with sensible universal defaults only (no language sections).
- [ ] An existing `.editorconfig` is not overwritten unless `--force`.
- [ ] `--dry-run` prints output to stdout without writing.

**Edge Cases and Error Handling:**

- User selects zero languages: emits `[*]` section only; exits 0.
- Unsupported language string passed via `--non-interactive` flag (future extension point): command ignores unknown language identifiers and logs a warning to stderr.

**Technical Details:**

- Embedded template `pkg/template/templates/editorconfig.tmpl`.
- Language rule sets defined as a static map in `internal/commands/generate/editorconfig.go` — no external data files.
- No new external dependencies.

---

### Feature: `cure generate gitignore`

**User story:**

As a developer initializing a repository, I want `cure generate gitignore` to create a `.gitignore` with patterns relevant to my project's languages and tooling, so that I do not need to look up or copy patterns manually.

**Acceptance Criteria:**

- [ ] The command presents a MultiSelect menu of language/tool profiles: Go, Node.js, Python, Rust, Java, macOS, Windows, Linux, JetBrains IDEs, VS Code, Vim/Emacs.
- [ ] Each profile contributes a labeled section to the output `.gitignore`.
- [ ] Patterns within each section are the canonical patterns for that profile (embedded at compile time, not fetched at runtime).
- [ ] Profiles can be combined; duplicate patterns across profiles are deduplicated.
- [ ] `--non-interactive` defaults to generating a single universal section (OS temp files, editor swap files).
- [ ] An existing `.gitignore` is not overwritten unless `--force`.
- [ ] `--dry-run` prints output to stdout without writing.

**Edge Cases and Error Handling:**

- User selects zero profiles: emits a minimal comment-only file; exits 0.
- Fetching patterns from `gitignore.io` or GitHub is explicitly out of scope — all patterns must be embedded.

**Technical Details:**

- Pattern data stored as Go string constants (or a small embedded data file) in `internal/commands/generate/gitignore_patterns.go`.
- Template `pkg/template/templates/gitignore.tmpl`.
- No new external dependencies.

---

### Feature: `cure generate github-workflow`

**User story:**

As a developer publishing a repository to GitHub, I want `cure generate github-workflow` to create a basic CI workflow YAML so that my project runs tests and linting automatically on every pull request without manually writing YAML.

**Acceptance Criteria:**

- [ ] The command creates `.github/workflows/ci.yml`.
- [ ] The generated workflow runs on `push` to `main` and on `pull_request` targeting `main`.
- [ ] The wizard prompts: Go version (default: `1.25`), whether to include lint step, whether to include test coverage upload.
- [ ] The generated YAML is syntactically valid (verified by a round-trip `encoding/json.Unmarshal` is not applicable; instead, the template renders clean YAML with no trailing spaces, verified via string assertions in tests).
- [ ] `--non-interactive` generates a minimal Go workflow (checkout, setup-go, test, vet) without coverage upload.
- [ ] An existing `.github/workflows/ci.yml` is not overwritten unless `--force`.
- [ ] `pkg/fs.EnsureDir` creates `.github/workflows/` if absent.
- [ ] `--dry-run` prints output without writing.

**Edge Cases and Error Handling:**

- `.github/` exists as a file: command returns a clear error.
- Go version string not matching `\d+\.\d+` pattern (interactive validation): prompt repeats with error message.

**Technical Details:**

- Template `pkg/template/templates/github-workflow-go.tmpl`.
- `internal/commands/generate/githubworkflow.go`.
- No new external dependencies.

---

## v0.8.x — Project Bootstrap and Enhanced Doctor

### Problem Statement

Two distinct gaps drive this milestone.

First, `cure generate` produces individual files but no single command initialises an entire project skeleton. Users must decide themselves which files to generate, in what order, and how to wire them together. `cure init` closes this gap as a high-level wizard that orchestrates generate commands.

Second, `cure doctor`'s `CheckFunc` framework lives in `internal/commands/doctor`, making it inaccessible to external projects that want to run health checks. Extracting it to `pkg/doctor` enables reuse and enables `.cure.json` to declare custom checks.

Third, `cure context` has no search or export capability. Users with many sessions cannot locate a session by content, and there is no way to share a session outside of cure itself.

### Business Objective

Make `cure` the single command a developer needs to run after cloning or initialising a new repository, and make the doctor framework reusable beyond the CLI itself.

### Scope: In

- `cure init` — interactive project scaffold wizard; calls existing generate commands
- `pkg/doctor` — extract `CheckFunc`, `CheckResult`, `CheckStatus` into the public package; `internal/commands/doctor` imports from `pkg/doctor`
- Custom doctor checks via `.cure.json` (`doctor.checks` array of `{ name, command, pass_on }`)
- `cure context search <query>` — full-text search across saved sessions
- `cure context export <id>` — export one session to Markdown or NDJSON

### Scope: Out

- `cure init` creating a `go.mod` or language-specific project files — cure is not a language scaffold tool; it configures tooling around an existing or empty project
- Remote check endpoints (HTTP health check targets) in `pkg/doctor` — deferred to v1.0.x
- Session import (`cure context import`) — deferred; dependent on agreed file format stability
- Bulk export (`cure context export --all`) — deferred to v0.9.x or later

---

### Feature: `cure init`

**User story:**

As a developer starting a new project or onboarding a repository that has no tooling configuration, I want to run `cure init` and answer a short wizard so that cure generates all the relevant files for my project in a single pass.

**Acceptance Criteria:**

- [ ] The wizard collects: project name, primary language (Go / Node.js / Python / Rust / Other), which AI tool context files to generate (MultiSelect), whether to generate a devcontainer, whether to generate a CI workflow, whether to generate `.editorconfig`, whether to generate `.gitignore`.
- [ ] For each selected item, `cure init` delegates to the corresponding `cure generate` subcommand's programmatic entry point.
- [ ] `cure init` does not duplicate generate logic; it is a pure orchestration command.
- [ ] `--non-interactive` accepts all prompts via flags: `--name`, `--language`, `--ai-tools <comma-list>`, `--devcontainer`, `--ci`, `--editorconfig`, `--gitignore`.
- [ ] If no flags are provided in `--non-interactive` mode, all components default to enabled.
- [ ] A summary of files written (or skipped) is printed on completion.
- [ ] Exit code is 0 if all writes succeed; non-zero if any file write fails.
- [ ] `--dry-run` propagates to all delegated generate calls.

**Edge Cases and Error Handling:**

- Running `cure init` in a directory that already has most files: non-conflicting files are written, conflicts prompt (or skip with `--force`).
- User cancels mid-wizard (Ctrl-C): no partial files are written (each generate call is atomic via `pkg/fs.AtomicWrite`).

**Technical Details:**

- `internal/commands/init/init.go` — new command package.
- Must be registered last in `cmd/cure/main.go` after all generate subcommands (so completion introspection sees the full command list).
- No new external dependencies.

**Dependencies:**

- `cure generate scaffold` (v0.7.x) must expose a programmatic entry point before `cure init` can delegate to it.
- `cure generate devcontainer`, `cure generate editorconfig`, `cure generate gitignore`, `cure generate github-workflow` (all v0.7.x) must be complete.

---

### Feature: `pkg/doctor` — Public Package Extraction

**User story:**

As a developer using cure's `pkg/` libraries in my own Go project, I want to import `pkg/doctor` and register custom `CheckFunc` implementations so that I can run project health checks in my own tooling without duplicating the framework.

**Acceptance Criteria:**

- [ ] `pkg/doctor` exports `CheckFunc`, `CheckResult`, and `CheckStatus` with identical semantics to the current `internal/commands/doctor` types.
- [ ] `pkg/doctor` exports `Run(checks []CheckFunc, w io.Writer) (passed, warned, failed int)` — runs checks and writes formatted results to `w`.
- [ ] `pkg/doctor` has zero imports from `internal/` or `cmd/`.
- [ ] `internal/commands/doctor` is refactored to import from `pkg/doctor`; the 7 built-in `CheckFunc` implementations move to `pkg/doctor`.
- [ ] Compile-time: `var _ = pkg/doctor.CheckFunc(nil)` assertion confirms type compatibility.
- [ ] All existing `doctor_test.go` tests pass unchanged after the refactor (behaviour is identical).
- [ ] `pkg/doctor` has table-driven tests for `Run` covering all three status outcomes.
- [ ] `pkg/doctor` has a benchmark for `Run` with 10 no-op checks.

**Edge Cases and Error Handling:**

- A `CheckFunc` that panics: `Run` must recover and record that check as `CheckFail` with the panic message as the result message. This prevents one bad check from aborting all others.

**Technical Details:**

- New package directory `pkg/doctor/`.
- `pkg/doctor/doctor.go` exports types and `Run`.
- `pkg/doctor/checks.go` contains the 7 built-in checks moved from `internal/commands/doctor/`.
- `internal/commands/doctor/doctor.go` shrinks to wiring: imports `pkg/doctor`, calls `pkg/doctor.Run`, formats the summary line.

**Dependencies:**

- No upstream dependencies on other v0.8.x features.

---

### Feature: Custom Doctor Checks via `.cure.json`

**User story:**

As a developer with project-specific health requirements, I want to declare custom `cure doctor` checks in `.cure.json` so that my project-specific checks run alongside the built-in ones without modifying cure's source code.

**Acceptance Criteria:**

- [ ] `.cure.json` accepts a `doctor.checks` array where each entry has: `name` (string), `command` (string — shell command), `pass_on` (`"exit_0"` | `"stdout_contains:<pattern>"`).
- [ ] `cure doctor` loads `.cure.json` at runtime and appends custom checks after the 7 built-in checks.
- [ ] Each custom check runs the declared command via `os/exec` with a 10-second timeout; the exit status or stdout is evaluated per `pass_on`.
- [ ] A command that times out is recorded as `CheckWarn` with message "check timed out".
- [ ] A command not found on `$PATH` is recorded as `CheckFail` with a "command not found" message.
- [ ] Custom checks that fail do not prevent built-in checks from running.
- [ ] `cure doctor --no-custom` skips all custom checks.

**Edge Cases and Error Handling:**

- `.cure.json` present but `doctor.checks` is not an array: warn to stderr and continue with built-in checks only.
- `pass_on` value not in the allowed set: the check is recorded as `CheckFail` with "invalid pass_on value".
- Command contains shell metacharacters: `os/exec` is invoked directly (not through a shell) using `strings.Fields` splitting — no shell injection surface.

**Technical Details:**

- `internal/commands/doctor/custom.go` — `loadCustomChecks(cfgPath string) ([]pkg/doctor.CheckFunc, error)`.
- Shell metacharacter concern: use `exec.Command(parts[0], parts[1:]...)` — no `sh -c` wrapper.
- Timeout: `exec.CommandContext` with a 10-second `context.WithTimeout`.

**Dependencies:**

- `pkg/doctor` public package must be complete before custom check loading is added.

---

### Feature: `cure context search <query>`

**User story:**

As a developer with many saved sessions, I want to run `cure context search <query>` to find sessions whose content contains my search terms, so that I can locate a past conversation without remembering its ID.

**Acceptance Criteria:**

- [ ] `cure context search <query>` performs a case-insensitive substring search across `Session.History[*].Content` for all sessions in the store.
- [ ] Results are printed as a table: session ID, provider, creation date, matched message count, first matched excerpt (max 80 characters).
- [ ] `--format ndjson` emits one JSON object per matching session with fields: `id`, `provider`, `created_at`, `match_count`, `excerpt`.
- [ ] If no sessions match, the command prints "No sessions matched." and exits 0.
- [ ] Search term is required; omitting it prints usage and exits non-zero.
- [ ] Search operates entirely in-memory on loaded sessions; no index is built or persisted.

**Edge Cases and Error Handling:**

- Store directory does not exist or is empty: command prints "No sessions found." and exits 0.
- A session file is corrupt: it is skipped (consistent with `JSONStore.List` behaviour).
- Query contains regex metacharacters: query is treated as a literal substring, not a regex.

**Technical Details:**

- `internal/commands/context/search.go` — new subcommand.
- Loads all sessions via `JSONStore.List`; loads individual session content via `JSONStore.Load`.
- Pure stdlib: `strings.Contains` for matching.
- No new external dependencies.

**Dependencies:**

- `pkg/agent/store.JSONStore` must be complete (already shipped in v0.5.0).

---

### Feature: `cure context export <id>`

**User story:**

As a developer who wants to share or archive a conversation, I want `cure context export <id>` to write the session to a Markdown or NDJSON file, so that I can read it outside of cure or share it with a colleague.

**Acceptance Criteria:**

- [ ] `cure context export <id>` defaults to Markdown output printed to stdout.
- [ ] `--format ndjson` emits the raw session JSON (same format as stored on disk).
- [ ] `--output <path>` writes to the specified file using `pkg/fs.AtomicWrite`; stdout is used when `--output` is absent.
- [ ] Markdown format: H1 = session ID, metadata block (provider, model, created, updated), followed by alternating `## User` / `## Assistant` sections with message content.
- [ ] An unknown `<id>` returns a clear "session not found: <id>" error and exits non-zero.
- [ ] Export does not modify the session (read-only operation).

**Edge Cases and Error Handling:**

- Session with empty history: Markdown is emitted with metadata block and a note "No messages in this session."
- `--output` path parent directory does not exist: `pkg/fs.EnsureDir` is called; if that fails, error is returned and nothing is written.

**Technical Details:**

- `internal/commands/context/export.go`.
- Markdown rendering uses `text/template` (stdlib); no external markdown library.
- Template for Markdown export lives as a Go string constant in the same file (not in `pkg/template` registry — it is not user-overridable).

**Dependencies:**

- `pkg/agent/store.JSONStore` (v0.5.0).

---

## v0.9.x — Multi-Provider AI and MCP Serve

### Problem Statement

`cure context` is locked to the Claude provider. Developers who use OpenAI or Google Gemini models cannot use cure's session management without switching providers. Additionally, cure's generate and doctor capabilities are not accessible to AI assistants that use the Model Context Protocol (MCP): there is no way to invoke `cure generate claude-md` or `cure doctor` from within an MCP-aware AI tool. Finally, sessions have no tagging system, making it impossible to group or filter them by project or topic.

### Business Objective

Make cure provider-agnostic for AI sessions, expose cure's capabilities as MCP tools consumable by AI assistants, and provide session metadata management ahead of the v1.0.0 API freeze.

### Scope: In

- `internal/agent/openai` — OpenAI adapter (GPT-4o, GPT-4o-mini); registered as `"openai"`
- `internal/agent/gemini` — Google Gemini adapter (Gemini 2.5 Pro); registered as `"gemini"`
- Session tags: `cure context new --tag <tag>`, `cure context list --tag <tag>` (filter)
- `cure mcp serve` — exposes cure's generate and doctor capabilities as MCP tools
- `pkg/` API freeze notes — document public API stability commitments ahead of v1.0.0

### Scope: Out

- OpenAI assistant threads API (stateful server-side sessions) — deferred; cure's session model maps to the stateless messages API
- Gemini multimodal input (images, video) — deferred; cure sessions are text-only
- MCP Resources and Prompts registration in `cure mcp serve` — tools only in v0.9.x; Resources/Prompts deferred
- Bulk tag operations (`cure context retag`) — deferred
- `pkg/` API freeze enforcement (semver locking) — informational docs only in v0.9.x; enforcement via CI deferred to v1.0.0

---

### Feature: `internal/agent/openai` — OpenAI Adapter

**User story:**

As a developer who uses OpenAI models, I want to run `cure context new --provider openai --model gpt-4o` so that I can use cure's session management with OpenAI, without being restricted to the Claude provider.

**Acceptance Criteria:**

- [ ] `internal/agent/openai` registers as provider `"openai"` via `init()` using the blank-import driver pattern; `cmd/cure/main.go` adds the blank import.
- [ ] `NewOpenAIAgent(cfg map[string]any) (agent.Agent, error)` reads `api_key_env` (default `OPENAI_API_KEY`), `model` (default `gpt-4o`), `max_tokens` (default `4096`).
- [ ] `Agent.Run(ctx, session)` streams tokens via the OpenAI Chat Completions API, mapping `agent.Message.Role` to OpenAI roles (`user`, `assistant`, `system`).
- [ ] `Agent.CountTokens(ctx, session)` returns `agent.ErrCountNotSupported` (OpenAI does not expose a count-tokens endpoint equivalent to Anthropic's).
- [ ] `sanitiseError` redacts the API key value from all error strings before surfacing them.
- [ ] The adapter works without any new external dependency: uses `net/http` + `encoding/json` directly (no OpenAI SDK).
- [ ] Unit tests cover: factory config reading, role mapping, error sanitisation, `CountTokens` returning `ErrCountNotSupported`.
- [ ] E2E test uses `OPENAI_BASE_URL` env var to inject a mock HTTP server (same pattern as Claude adapter's `ANTHROPIC_BASE_URL`).

**Edge Cases and Error Handling:**

- `OPENAI_API_KEY` env var not set: `NewOpenAIAgent` returns a clear error at construction time, not at first `Run` call.
- HTTP 429 (rate limit) from OpenAI: surfaced as an `EventKindError` event with the status code in the message.
- HTTP 401 (bad key): surfaced as `EventKindError`; key value redacted from the error string.

**Technical Details:**

- `internal/agent/openai/openai.go`.
- Streaming: OpenAI Chat Completions supports `stream: true` with SSE (`text/event-stream`); parse `data: {...}` lines using `bufio.Scanner` with the `ScanLines` split function.
- No new external dependency — direct HTTP client only.

**Assumption:** The OpenAI Chat Completions SSE streaming format is stable. If OpenAI deprecates SSE in favour of a new transport before v0.9.0, this work item must be re-scoped.

---

### Feature: `internal/agent/gemini` — Google Gemini Adapter

**User story:**

As a developer who uses Google Gemini models, I want to run `cure context new --provider gemini --model gemini-2.5-pro` so that I can use cure's session management with Gemini.

**Acceptance Criteria:**

- [ ] `internal/agent/gemini` registers as provider `"gemini"` via `init()`; `cmd/cure/main.go` adds the blank import.
- [ ] `NewGeminiAgent(cfg map[string]any) (agent.Agent, error)` reads `api_key_env` (default `GEMINI_API_KEY`), `model` (default `gemini-2.5-pro`), `max_tokens` (default `8192`).
- [ ] `Agent.Run(ctx, session)` uses the Gemini `generateContent` REST API with `stream=true`, mapping `agent.Message.Role` to Gemini roles (`user`, `model`).
- [ ] Gemini uses `"model"` for the assistant role; the adapter maps `agent.RoleAssistant` → `"model"` and `agent.RoleUser` → `"user"` bidirectionally.
- [ ] `Agent.CountTokens(ctx, session)` calls the Gemini `countTokens` endpoint and returns the token count.
- [ ] `sanitiseError` redacts the API key from error strings.
- [ ] No new external dependency.
- [ ] Unit and E2E tests follow the same patterns as the OpenAI adapter.

**Edge Cases and Error Handling:**

- Gemini `"model"` role mapping must not pass `"model"` through to the `agent.Message.Role` field in stored history — the adapter maps outbound (to Gemini) and translates inbound (from Gemini response) back to `agent.RoleAssistant`.

**Technical Details:**

- `internal/agent/gemini/gemini.go`.
- Streaming: Gemini uses newline-delimited JSON (NDJSON) response bodies when `?alt=sse` is set; parse with `bufio.Scanner`.
- `countTokens` endpoint: `POST https://generativelanguage.googleapis.com/v1beta/models/{model}:countTokens`.

---

### Feature: Session Tags

**User story:**

As a developer managing many sessions across projects, I want to tag sessions when I create them and filter by tag when listing, so that I can group and locate sessions without searching through content.

**Acceptance Criteria:**

- [ ] `cure context new --tag <tag>` stores the tag in `Session.Tags` before the first save.
- [ ] `--tag` may be specified multiple times to apply multiple tags.
- [ ] `cure context list --tag <tag>` filters the output to sessions that include the specified tag (exact match, case-sensitive).
- [ ] `cure context list` without `--tag` lists all sessions (unchanged from current behaviour).
- [ ] Tags are persisted in the JSON store's `tags` field (already present in the `Session` struct since v0.5.0).
- [ ] `cure context list` text output adds a `Tags` column when at least one listed session has tags; the column is omitted when no sessions have tags.
- [ ] `--format ndjson` output already includes the `tags` field; no change needed for NDJSON format.
- [ ] Tag values are arbitrary strings with no character restrictions beyond JSON string validity.

**Edge Cases and Error Handling:**

- `--tag` with an empty string value: the command returns an error "tag value cannot be empty".
- `cure context list --tag <tag>` matching zero sessions: prints "No sessions matched." and exits 0.
- `cure context resume` does not accept `--tag`; tags on existing sessions are preserved as stored.

**Technical Details:**

- Flag changes in `internal/commands/context/new.go` and `list.go`.
- `Session.Tags` field already exists in `pkg/agent/session.go`; no model changes required.
- Text output column logic in `list.go`; width is truncated at 30 characters with a trailing ellipsis for display.

**Dependencies:**

- None within v0.9.x; depends on `Session.Tags` field (shipped in v0.5.0).

---

### Feature: `cure mcp serve`

**User story:**

As a developer using an MCP-aware AI assistant (Claude Desktop, Cursor, Windsurf), I want to run `cure mcp serve` to expose cure's generate and doctor capabilities as MCP tools, so that the AI assistant can invoke them on my behalf during a conversation.

**Acceptance Criteria:**

- [ ] `cure mcp serve` starts an MCP server using `pkg/mcp.Server`.
- [ ] The server auto-detects the appropriate transport via `pkg/mcp.IsStdinPipe()`: stdio when stdin is a pipe, HTTP Streamable on `127.0.0.1:8080` otherwise.
- [ ] `--addr <host:port>` overrides the HTTP listen address.
- [ ] The following tools are registered:
  - `generate_claude_md` — generates CLAUDE.md for the current directory; parameter: `force` (bool, default false).
  - `generate_agents_md` — generates AGENTS.md; parameter: `force`.
  - `generate_scaffold` — generates selected AI files; parameter: `tools` (array of strings, e.g. `["claude-md", "agents-md"]`), `force`.
  - `doctor` — runs all doctor checks; returns JSON array of `{ name, status, message }`.
- [ ] Each tool's handler invokes the existing command logic via the programmatic entry point (same refactor gate as `cure generate scaffold`).
- [ ] Tool output is returned as `[]mcp.Content` with a single `mcp.Text(result)` item.
- [ ] `doctor` tool returns `isError: true` when any check fails, with the failure summary as the content text.
- [ ] `cure mcp serve --help` lists all registered tools and their parameters.

**Edge Cases and Error Handling:**

- A generate tool that encounters an existing file without `force: true` returns an MCP error result (`isError: true`) with a human-readable message — it does not panic or crash the server.
- HTTP transport: CORS allowed origins default to all (same as `pkg/mcp.Server` default); `--allowed-origins <comma-list>` restricts origins.
- The server runs until interrupted (SIGINT / SIGTERM); `context.WithCancel` is used to propagate shutdown to `pkg/mcp.Server.Serve`.

**Technical Details:**

- `internal/commands/mcp/serve.go` — new command; registers with the top-level `mcp` subcommand router.
- `cmd/cure/main.go` wires `mcp serve` command.
- Programmatic entry point refactor: each generate command must expose `RunWith(ctx context.Context, tc *terminal.Context, opts GenerateOpts) error` (or equivalent) so the MCP tool handler can call it without constructing a CLI invocation.
- No new external dependencies.

**Dependencies:**

- `pkg/mcp` (v0.4.1) — already shipped.
- `cure generate scaffold` programmatic entry point (v0.7.x).
- `pkg/doctor.Run` (v0.8.x) for the `doctor` tool handler.

---

### Feature: `pkg/` API Freeze Notes

**User story:**

As a Go developer importing cure's `pkg/` packages, I want to know which packages have stabilised APIs before I take a dependency, so that I can plan for upgrade compatibility.

**Acceptance Criteria:**

- [ ] A `docs/api-stability.md` document lists all `pkg/` packages with stability classification: `stable`, `candidate`, or `experimental`.
- [ ] The document defines what "stable" means for this project: no breaking changes without a major version bump (`v1.0.0+`).
- [ ] Each package entry includes: current stability tier, known planned breaking changes (if any), and the target version when it will stabilise.
- [ ] `pkg/terminal`, `pkg/agent`, `pkg/agent/store`, `pkg/mcp`, `pkg/config`, `pkg/template`, `pkg/prompt`, `pkg/fs`, `pkg/style`, `pkg/env` are all assessed.
- [ ] `pkg/doctor` (new in v0.8.x) is assessed as `candidate` in v0.9.x.
- [ ] The document is reviewed for accuracy before v0.9.0 release.

**Technical Details:**

- New file `docs/api-stability.md`.
- No code changes required; documentation only.

---

## Cross-Milestone Dependency Map

```
v0.7.x
├── generate scaffold          ──────────────────────────────────┐
│   └── requires: programmatic entry point for each generate cmd │
├── generate devcontainer                                        │
├── generate editorconfig                                        │
├── generate gitignore                                           │
└── generate github-workflow                                     │
                                                                 │
v0.8.x                                                           │
├── cure init  ────────────────────────── depends on: scaffold ──┘
├── pkg/doctor (extraction)
│   └── required by: custom checks, cure mcp serve (v0.9.x)
├── custom doctor checks  ──── depends on: pkg/doctor
├── context search  ──────────────────── no upstream deps
└── context export  ──────────────────── no upstream deps

v0.9.x
├── openai adapter  ──────────────────── no upstream deps
├── gemini adapter  ──────────────────── no upstream deps
├── session tags  ────────────────────── no upstream deps (Session.Tags exists)
├── cure mcp serve  ─── depends on: scaffold entry point (v0.7.x), pkg/doctor (v0.8.x)
└── api stability docs  ─────────────── no code deps; depends on all prior work
```

**Critical path:** `cure generate scaffold` programmatic entry point (v0.7.x) is required by both `cure init` (v0.8.x) and `cure mcp serve` (v0.9.x). This entry point refactor must be designed for reuse from the start, not bolted on later.

---

## Plugin Capability Gap Analysis

This section identifies capabilities required for the v0.7.x–v0.9.x roadmap that are not yet present in the codebase.

### Gap 1: No Programmatic Entry Point for Generate Subcommands

**Status:** Missing
**Blocking:** `cure generate scaffold` (v0.7.x), `cure init` (v0.8.x), `cure mcp serve` (v0.9.x)

Each generate subcommand currently exposes only a `terminal.Command` interface (`Run(ctx, tc)`). The MCP serve handler and the scaffold/init orchestrators need to call generate logic without constructing a fake CLI context. A `RunWith(ctx, tc, opts)` or equivalent functional signature is required.

**Recommended approach:** Each generate command exposes a package-level `Generate(ctx context.Context, w io.Writer, opts GenerateOpts) error` function. The `terminal.Command.Run` implementation wraps it. The orchestrators call the function directly.

---

### Gap 2: No Shell Command Execution in Doctor

**Status:** Missing
**Blocking:** Custom doctor checks (v0.8.x)

The current `CheckFunc` type is `func() CheckResult`. Custom shell-based checks need `os/exec` invocation with timeout. The framework needs a safe `ExecCheck(name, command string, passOn PassOnRule) CheckFunc` constructor in `pkg/doctor`.

---

### Gap 3: No OpenAI or Gemini HTTP Streaming Client

**Status:** Missing
**Blocking:** OpenAI adapter (v0.9.x), Gemini adapter (v0.9.x)

The Claude adapter delegates to the Anthropic SDK. For OpenAI and Gemini, the constraint is stdlib-only (no SDK). Both APIs use SSE or NDJSON streaming over HTTP. A shared internal utility for reading SSE lines from an `http.Response.Body` would prevent duplication between the two adapters.

**Recommended approach:** `internal/agent/sseutil/reader.go` — a minimal SSE line reader using `bufio.Scanner`. Both adapters import this internal utility.

---

### Gap 4: No Session Full-Text Index

**Status:** Accepted gap — deliberate design choice
**Impact:** `cure context search` (v0.8.x) will perform an O(n) in-memory scan across all sessions.

For v0.8.x this is acceptable given expected session counts (<1000). If session counts grow significantly, an index (e.g., an inverted map persisted to disk) becomes necessary. This is a known deferred concern, not a current blocker.

---

### Gap 5: No Tag-Based Filtering in `JSONStore`

**Status:** Missing — must be added
**Blocking:** Session tags filtering in `cure context list --tag` (v0.9.x)

`SessionStore.List` returns all sessions. Tag filtering either (a) happens in the command layer after `List` returns (simple, no interface change), or (b) requires a `ListByTag(ctx, tag string)` method on the store (requires interface change, breaks existing implementations). Option (a) is recommended for v0.9.x to avoid a breaking interface change before API freeze.

---

### Gap 6: No MCP Tool-to-Command Binding Convention

**Status:** Missing
**Blocking:** `cure mcp serve` (v0.9.x)

There is no established pattern for binding a `pkg/mcp.FuncTool` to a cure command's programmatic entry point. This is a design convention gap, not a code gap. The convention must be agreed before `cure mcp serve` implementation begins to prevent ad-hoc wiring in `serve.go`.

**Recommended approach:** Document the convention in `internal/commands/mcp/README.md` before implementation: each tool handler signature is `func(ctx context.Context, args map[string]any) ([]mcp.Content, error)`, and tool argument schemas mirror the generate command's flag types.

---

### Gap 7: `pkg/doctor` Does Not Exist as a Public Package

**Status:** Missing
**Blocking:** External reuse, `cure mcp serve` `doctor` tool (v0.9.x)

`CheckFunc`, `CheckResult`, and `CheckStatus` are currently in `internal/commands/doctor`. Extraction to `pkg/doctor` is a prerequisite for the `doctor` MCP tool and for external projects importing the framework.

---

## Open Questions and Assumptions

### Open Questions

1. **Devcontainer Dockerfile scope (v0.7.x):** Should the generated `Dockerfile` be a single-stage build using the project's primary language image, or a multi-stage build? A multi-stage build is more correct for production but adds template complexity. Decision needed before implementation.

2. **Gitignore pattern source (v0.7.x):** Are embedded patterns sufficient, or should the tool support fetching from `api.github.com/gitignore/templates` with a local cache? Fetching adds an HTTP dependency and complicates offline use; embedded patterns require manual updates when language tooling changes. This analysis recommends embedded-only for v0.7.x.

3. **`cure init` vs. `cure generate scaffold` overlap (v0.7.x / v0.8.x):** `scaffold` selects AI files only; `init` orchestrates all generate types. Is there user confusion between two wizard-style commands? Consider whether `scaffold` should be folded into `init` or remain a standalone subcommand of `cure generate`. Recommendation: keep them separate — `scaffold` is a generate subcommand for AI files only; `init` is a top-level project wizard. Document the distinction clearly in help text.

4. **API key management for multi-provider (v0.9.x):** Should cure emit a warning when `OPENAI_API_KEY` or `GEMINI_API_KEY` is not set and the user selects that provider, or should it fail silently at construction time? Recommendation: fail loudly at `cure context new` with a "set OPENAI_API_KEY environment variable" error, not a silent no-op.

5. **`cure mcp serve` tool registration policy (v0.9.x):** Should all generate subcommands be exposed as individual MCP tools (`generate_claude_md`, `generate_agents_md`, etc.) or only via the `generate_scaffold` batch tool? Individual tools give AI assistants finer-grained control; the batch tool is simpler. This analysis recommends both: scaffold for batch, individual tools for targeted generation.

6. **v1.0.0 timeline:** Is v0.9.x explicitly a pre-freeze release, or might additional minor versions (v0.10.x, etc.) precede v1.0.0? The answer affects how aggressively breaking changes are permitted in v0.9.x. This analysis assumes v0.9.x is the last minor release before the v1.0.0 API stability commitment.

### Assumptions

| Assumption | Confidence | Risk if Wrong |
|------------|------------|---------------|
| OpenAI Chat Completions SSE streaming format remains stable | High | OpenAI adapter must be rewritten |
| Gemini `generateContent` REST API remains accessible without a Go SDK | High | Gemini adapter may require SDK, adding an external dependency |
| Session counts remain below 1,000 for typical users in v0.8.x timeframe | Medium | Context search performance degrades; index needed sooner |
| `pkg/mcp` Server interface is stable enough for `cure mcp serve` in v0.9.x | High | MCP serve must be re-scoped if pkg/mcp breaks |
| The OpenAI and Gemini key env var conventions (`OPENAI_API_KEY`, `GEMINI_API_KEY`) are stable | High | Config key names must be updated |
| `.cure.json` schema for `doctor.checks` does not conflict with existing config keys | High | Config namespace conflict requires migration |

---

## Risk Assessment

| # | Risk | Probability | Impact | Score | Mitigation |
|---|------|------------|--------|-------|-----------|
| 1 | Programmatic entry point refactor scope creeps and delays v0.7.x | M | H | 6 | Scope the refactor strictly to `run()` extraction; do not redesign command architecture |
| 2 | OpenAI SSE streaming format changes before v0.9.0 | L | H | 3 | Pin to a documented API version; add an integration test against the live API in CI |
| 3 | `pkg/doctor` extraction introduces a breaking change in `internal/commands/doctor` tests | M | M | 4 | Run `make test` after every extraction step; keep the internal package as a thin shim |
| 4 | Session full-text search (v0.8.x) is too slow for users with large session stores | L | M | 2 | Document the O(n) characteristic in help text; add a performance benchmark in tests |
| 5 | `cure init` wizard UX is confusing alongside `cure generate scaffold` | M | M | 4 | Write clear help text distinguishing the two; user-test with 3 developers before v0.8.0 release |
| 6 | MCP tool argument schema conflicts with future MCP protocol version | L | M | 2 | Peg `pkg/mcp` protocol version to `2025-03-26`; upgrade when MCP spec stabilises |
| 7 | External dependency approval delayed for new AI provider SDKs | M | H | 6 | Design both adapters as stdlib-only from the start; no SDK dependency planned |

**Scoring:** H=3, M=2, L=1. Score = Probability × Impact. Critical >= 6, High >= 4, Medium >= 2, Low = 1.

**Top risks:**
- Risk 1 (programmatic entry point scope): Mitigate by defining the `Generate()` function signature in a design comment on the GitHub issue before implementation begins.
- Risk 7 (SDK dependency): Mitigate by committing to stdlib-only adapters in the issue acceptance criteria; any SDK proposal requires explicit approval.
