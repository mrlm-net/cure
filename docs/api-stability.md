# API Stability Classification

This document classifies the stability of each `pkg/` package in cure. Use it to plan upgrade compatibility when taking a dependency on a package.

## Stability Tiers

| Tier | Meaning |
|------|---------|
| **stable** | No breaking changes without a major version bump (`v1.0.0+`). Safe to depend on. |
| **candidate** | API is mostly settled; minor breaking changes may still occur in `v0.x.y` releases. Announced in advance. |
| **experimental** | Under active design; breaking changes may occur without notice in `v0.x.y` releases. |

Breaking changes in `pkg/` packages are noted in the [CHANGELOG](changelog.md).

---

## Package Classifications

### `pkg/terminal` — **candidate**

Command router, flag handling, help generation, and execution contexts.

- **Planned changes**: `RunnerFunc` and execution mode types may be refined before v1.0.0.
- **Stabilises at**: v1.0.0

---

### `pkg/agent` — **experimental**

Provider-agnostic AI agent interface, session management, event streaming, tool dispatch, skill registry, and `MessageContent` codec.

- **Planned changes**: The `MessageContent` / `ContentBlock` type codec landed in v0.10.0 and is backward-compatible but the broader tool-use API (`Tool`, `Skill`, `Session.Tools`) is new and may be adjusted as provider coverage grows.
- **Known planned breaking change**: `v0.11.x` will add tool-use support to the Gemini and OpenAI adapters, which may require `Tool.Schema()` to accommodate provider-specific schema shapes.
- **Stabilises at**: v1.0.0

---

### `pkg/agent/store` — **candidate**

JSON-backed session persistence with `Store` interface.

- **Planned changes**: None. The `Store` interface (`Save`, `Load`, `List`, `Delete`) is stable.
- **Stabilises at**: v1.0.0

---

### `pkg/mcp` — **candidate**

Stdlib-only MCP (Model Context Protocol) server with stdio and HTTP Streamable transports.

- **Planned changes**: `Server.Tools()` accessor added in v0.10.0. The MCP spec is still evolving; transport-level changes may follow spec updates.
- **Stabilises at**: v1.0.0

---

### `pkg/config` — **stable**

Hierarchical configuration merging with dot-notation access.

- No breaking changes planned. `ConfigObject`, `DeepMerge`, `Get`/`Set`, and layered merge (`File`, `Environment`, `NewConfig`) are stable.
- **Stabilises at**: v0.x.y (already stable)

---

### `pkg/template` — **candidate**

Template generation engine with embedded templates and custom directory support.

- **Planned changes**: Custom template directories (added v0.6.1) may gain watch/reload support; the `Render` / `List` API itself is stable.
- **Stabilises at**: v1.0.0

---

### `pkg/prompt` — **candidate**

Interactive terminal prompts with validation and menu support.

- **Planned changes**: Potential addition of multi-select and date-picker widgets; existing `Prompter` API is stable.
- **Stabilises at**: v1.0.0

---

### `pkg/fs` — **stable**

Atomic filesystem operations (`WriteFile`, `EnsureDir`, `Exists`, `TempDir`).

- No breaking changes planned. Thin wrappers around `os` stdlib.
- **Stabilises at**: v0.x.y (already stable)

---

### `pkg/style` — **stable**

ANSI terminal styling (8 colors, 3 text styles, `NO_COLOR` support).

- No breaking changes planned. `Enable`, `Disable`, `Reset`, `Dim`, `Red`, etc. are stable.
- **Stabilises at**: v0.x.y (already stable)

---

### `pkg/env` — **stable**

Cached runtime environment detection (OS, arch, tool availability, git context).

- No breaking changes planned. `Env()` singleton and all `Detect*` helpers are stable.
- **Stabilises at**: v0.x.y (already stable)

---

### `pkg/doctor` — **candidate**

Pluggable health check framework (`Check`, `CheckFunc`, `Runner`, `Result`).

- New in v0.8.x. The core interfaces are settled but the built-in check library may grow.
- **Planned changes**: Additional built-in checks may be added; existing types will not break.
- **Stabilises at**: v1.0.0

---

## Summary Table

| Package | Tier | Stabilises At |
|---------|------|---------------|
| `pkg/config` | stable | already stable |
| `pkg/fs` | stable | already stable |
| `pkg/style` | stable | already stable |
| `pkg/env` | stable | already stable |
| `pkg/terminal` | candidate | v1.0.0 |
| `pkg/agent/store` | candidate | v1.0.0 |
| `pkg/mcp` | candidate | v1.0.0 |
| `pkg/template` | candidate | v1.0.0 |
| `pkg/prompt` | candidate | v1.0.0 |
| `pkg/doctor` | candidate | v1.0.0 |
| `pkg/agent` | experimental | v1.0.0 |

---

*Last updated for cure v0.10.0.*
