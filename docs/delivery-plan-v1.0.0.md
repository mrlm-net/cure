# Cure v1.0.0 Delivery Plan

**Date:** 2026-04-09
**Author:** Delivery Manager
**Status:** Draft for stakeholder approval
**Preceding documents:** [requirements-v1.0.0.md](requirements-v1.0.0.md), [architecture-v1.0.0.md](architecture-v1.0.0.md)
**Workload management:** GitHub Issues + GitHub Projects v2 on mrlm-net/cure, project #9

---

## Table of Contents

- [Overview](#overview)
- [Prerequisites: Complete GUI Epic #104](#prerequisites-complete-gui-epic-104)
- [Epic Summary Table](#epic-summary-table)
- [Dependency Graph](#dependency-graph)
- [RACI Matrix](#raci-matrix)
- [Epic Details](#epic-details)
  - [Epic: Spikes (Phase 0)](#epic-spikes-phase-0)
  - [Epic: v0.12.0 — Foundation](#epic-v0120--foundation)
  - [Epic: v0.13.0 — AI Config Distribution](#epic-v0130--ai-config-distribution)
  - [Epic: v0.14.0 — Project Management and Smart Doctor](#epic-v0140--project-management-and-smart-doctor)
  - [Epic: v0.15.0 — GUI Evolution](#epic-v0150--gui-evolution)
  - [Epic: v0.16.0 — Notifications](#epic-v0160--notifications)
  - [Epic: v0.17.0 — Multi-Instance Orchestration](#epic-v0170--multi-instance-orchestration)
  - [Epic: v1.0.0 — Stabilization and Release](#epic-v100--stabilization-and-release)
- [Risk Register](#risk-register)
- [Workload Management Configuration](#workload-management-configuration)

---

## Overview

This plan transforms cure from a single-agent CLI tool (v0.11.3) into an AI-first development platform (v1.0.0) through seven incremental minor releases plus a stabilization release. Each release delivers a cohesive vertical slice and is independently shippable.

**Delivery approach:**

1. **Complete existing GUI epic (#104)** — finish #109 (CLI wiring) and #114 (E2E tests) before new work begins
2. **Execute spikes** — derisk 4 critical technical unknowns (MCP transport, Monaco+LSP, Teams bidirectional, xterm.js+PTY)
3. **Build foundation first (v0.12.0)** — Project entity and session enrichment unblock all other domains
4. **Layer incrementally** — each release builds on the previous, adding one domain at a time
5. **Stabilize last (v1.0.0)** — integration tests, security audit, documentation, and polish

**Playbook selection:** This is a large multi-domain feature development initiative. The plan uses Playbook 1 (Feature Development) as the primary pattern, with Playbook 5 (Architecture/Refactoring) for foundation work, and Playbook 6 (Plugin/Configuration) for registry and config distribution tasks.

**Workload tracking:** All issues are tracked on the mrlm-net/cure GitHub Projects v2 board (#9). Agents post decisions, blockers, and outcomes to issues — not progress updates.

---

## Prerequisites: Complete GUI Epic #104

Before any v1.0.0 work begins, the existing GUI epic must be closed.

| Issue | Title | Status | Action |
|-------|-------|--------|--------|
| #109 | CLI wiring | In progress | Complete and merge |
| #114 | E2E tests | Pending | Implement and merge |
| #175 | PTY streaming + markdown | In progress | Complete and merge |

**Gate:** All three issues merged and #104 closed before Phase 0 spikes begin.

---

## Epic Summary Table

| Epic | Release | Theme | Task Count | Total Size Points | Domains |
|------|---------|-------|------------|-------------------|---------|
| Spikes | Phase 0 | Technical validation | 4 | 4 x S = ~4 S | A, B, D |
| Foundation | v0.12.0 | Project entity, session enrichment, config layer | 9 | 1 XL + 3 L + 3 M + 2 S | E, Agent |
| AI Config Distribution | v0.13.0 | Registry, managed configs, runtime assembly, drift | 8 | 3 L + 3 M + 2 S | G |
| Project Mgmt + Smart Doctor | v0.14.0 | VCS, backlog, multi-stack doctor | 11 | 2 L + 5 M + 3 S + 1 XS | C, F |
| GUI Evolution | v0.15.0 | Monaco, terminal, diff, config editor, theme | 10 | 1 XL + 3 L + 4 M + 2 S | D |
| Notifications | v0.16.0 | Teams, OS notifications, bidirectional messaging | 7 | 2 L + 3 M + 2 S | B |
| Multi-Instance Orchestration | v0.17.0 | Containers, host-container MCP, Docker Compose | 8 | 1 XL + 3 L + 2 M + 2 S | A |
| Stabilization | v1.0.0 | Integration tests, security, docs, polish | 8 | 2 L + 4 M + 2 S | Cross-domain |
| **Total** | | | **65** | | |

---

## Dependency Graph

```
                          Prerequisites
                          #109, #114, #175
                                |
                                v
                    +--- Phase 0: Spikes ---+
                    |  S-1  S-2  S-3  S-4  |
                    +-----------+-----------+
                                |
            +-------------------+-------------------+
            |                                       |
            v                                       v
    v0.12.0 Foundation                     (S-1 result feeds A-2)
    [E-1 Project Entity]                   (S-2 result feeds D-1)
    [Session Enrichment]                   (S-3 result feeds B-1)
    [Config Layer]                         (S-4 result feeds D-3)
    [Workflow Enforcement]
            |
            +------+--------+--------+--------+
            |      |        |        |        |
            v      v        v        v        v
    v0.13.0    v0.14.0   v0.15.0  v0.16.0  v0.17.0
    AI Config  Proj Mgmt  GUI     Notify   Orchestrate
    Distrib.   +Doctor    Evolve
            |      |        |        |        |
            +------+--------+--------+--------+
                                |
                                v
                    v1.0.0 Stabilization
```

### Cross-Epic Task Dependencies

```
E-1 (v0.12.0) ──> G-1, G-2, G-4 (v0.13.0)
E-1 (v0.12.0) ──> C-2, C-3, C-4 (v0.14.0)
E-1 (v0.12.0) ──> F-2 (v0.14.0)
E-1 (v0.12.0) ──> D-4 (v0.15.0)
E-1 (v0.12.0) ──> B-1 (v0.16.0)
E-1 (v0.12.0) ──> A-1 (v0.17.0)

G-1 (v0.13.0) ──> G-2, G-3, G-4 (v0.13.0)
G-2 (v0.13.0) ──> G-3 (v0.13.0)

C-1 (v0.14.0) ──> D-2 (v0.15.0)

S-1 (Phase 0) ──> A-2 (v0.17.0)
S-2 (Phase 0) ──> D-1 LSP (v0.15.0)
S-3 (Phase 0) ──> B-1 Phase 2 (v0.16.0)
S-4 (Phase 0) ──> D-3 (v0.15.0)

B-1, B-2 (v0.16.0) ──> B-3 (v0.16.0)

A-1 (v0.17.0) ──> A-2, A-3 (v0.17.0)

All (v0.12.0–v0.17.0) ──> v1.0.0 Stabilization
```

---

## RACI Matrix

**Legend:** R = Responsible (does the work), A = Accountable (delivery-manager), C = Consulted (input before), I = Informed (notified after).

### Per-Epic RACI

| Activity | `mrlm:software-architect` | `mrlm:software-engineer` | `mrlm:ui-designer` | `mrlm:code-reviewer` | `mrlm:qa-engineer` | `mrlm:security-specialist` | `mrlm:personal-writer` | `mrlm:platform-engineer` |
|----------|--------------------------|-------------------------|--------------------|-----------------------|--------------------|-----------------------------|------------------------|--------------------------|
| **Spikes (Phase 0)** | R (design decisions) | R (PoC code) | C (D spikes) | I | I | C (S-1 auth) | — | C (S-1 Docker) |
| **v0.12.0 Foundation** | R (pkg/project design) | R (implementation) | C (GUI changes) | R (pre-merge) | R (E2E) | I | I | — |
| **v0.13.0 AI Config** | R (registry design) | R (implementation) | — | R (pre-merge) | R (E2E) | C (config security) | R (docs) | — |
| **v0.14.0 Proj Mgmt + Doctor** | C (backlog abstraction) | R (implementation) | — | R (pre-merge) | R (E2E) | I | R (docs) | — |
| **v0.15.0 GUI Evolution** | C (LSP architecture) | R (backend) | R (UI design) | R (pre-merge) | R (E2E + UX) | I | — | — |
| **v0.16.0 Notifications** | R (channel design) | R (implementation) | C (GUI channel) | R (pre-merge) | R (E2E) | C (Teams auth) | R (setup docs) | C (Teams infra) |
| **v0.17.0 Orchestration** | R (MCP client design) | R (implementation) | I | R (pre-merge) | R (E2E) | R (MCP auth audit) | R (docs) | R (Docker/Compose) |
| **v1.0.0 Stabilization** | I | R (fixes) | R (polish) | R (final review) | R (integration) | R (security audit) | R (user guide) | R (release) |

---

## Epic Details

### Epic: Spikes (Phase 0)

**Playbook:** Custom (research spikes before implementation)
**Objective:** Derisk 4 critical technical unknowns identified in the architecture. Each spike produces a working PoC or a "defer to post-v1.0.0" decision with architecture notes.
**Prerequisite:** GUI Epic #104 fully closed (issues #109, #114, #175 merged).
**Execution:** PARALLEL (all 4 spikes are independent)

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| S-1 | spike(mcp): benchmark host-container MCP transport options | Spike | P0 | S | `mrlm:software-architect` | — | `mrlm:platform-engineer`, `mrlm:security-specialist` | **Time box:** 3 days. **Question:** Which MCP transport (HTTP Streamable over Docker bridge, stdio via docker exec, Unix socket via volume mount) is most reliable under 4-agent concurrent load? **Acceptance criteria:** (1) Benchmark all three transports with simulated 4-agent concurrent tool calls. (2) Measure: requests/sec, p95 latency, error rate under sustained load. (3) Produce recommendation document with selected transport and rationale. (4) If HTTP Streamable is selected, validate shared-secret Bearer token auth pattern. **Deliverable:** Benchmark results + transport recommendation committed to `docs/spikes/`. |
| S-2 | spike(gui): validate Monaco + LSP in SvelteKit 5 | Spike | P0 | S | `mrlm:software-engineer` | — | `mrlm:software-architect` | **Time box:** 3 days. **Question:** Can `monaco-languageclient` provide LSP features (autocomplete, hover, diagnostics) in a SvelteKit 5 SPA backed by a Go HTTP server proxying an LSP server via WebSocket? **Acceptance criteria:** (1) Working PoC with Monaco editor rendering Go files with syntax highlighting in SvelteKit 5. (2) Validate SSR-safe dynamic import pattern (`onMount` + `import()`). (3) If LSP works: demonstrate autocomplete against `gopls` via WebSocket proxy. (4) If LSP fails: document blockers, recommend "syntax highlighting only" for v1.0.0. (5) Measure frontend bundle size impact. **Deliverable:** PoC branch or "defer LSP" decision document. |
| S-3 | spike(notify): validate Teams Bot bidirectional messaging | Spike | P1 | S | `mrlm:software-engineer` | — | `mrlm:platform-engineer` | **Time box:** 2 days. **Question:** Can a Microsoft Teams bot receive and relay user thread replies to a non-Azure-hosted Go HTTP backend? What Azure AD configuration is required? **Acceptance criteria:** (1) Document Azure AD app registration steps for a Teams bot. (2) Working PoC: bot posts a message to a Teams channel, user replies in thread, Go backend receives the reply via Bot Framework webhook. (3) Document minimum Teams license tier required. (4) If bidirectional proves too complex for v1.0.0: confirm Incoming Webhook outbound-only path as fallback. **Deliverable:** Architecture document + Azure AD setup guide. |
| S-4 | spike(gui): validate xterm.js + Go PTY over WebSocket | Spike | P1 | S | `mrlm:software-engineer` | — | `mrlm:software-architect` | **Time box:** 2 days. **Question:** Can xterm.js connect to a Go backend PTY via WebSocket with acceptable latency (<50ms) and correct ANSI encoding? **Acceptance criteria:** (1) Working PoC: xterm.js in browser connects to Go HTTP server via WebSocket, Go server allocates PTY using `github.com/creack/pty`, bidirectional I/O works. (2) Test with: interactive commands (vim, htop), rapid output (`find /`), resize events. (3) Measure input-to-output latency. (4) Document the WebSocket message protocol (binary vs JSON frames). **Deliverable:** PoC branch + latency measurements. |

---

### Epic: v0.12.0 — Foundation

**Playbook:** Playbook 5 (Architecture/Refactoring) then Playbook 1 (Feature Development)
**Objective:** Establish the Project entity as the root abstraction, enrich the Session model, extend the config merge chain, add workflow enforcement, and update the GUI to surface enriched session data.
**Domains:** E (Project Entity and Config Sync), Agent (Session model)
**Prerequisite:** Spikes complete (results inform design but do not block foundation work).

**Phase 1: Architecture and Design (SERIAL)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| F-1 | design(project): architecture and interfaces for pkg/project | Task | P0 | M | `mrlm:software-architect` | — | `mrlm:software-engineer`, `mrlm:security-specialist` | Define the `Project`, `Repo`, `Defaults`, `DevcontainerCfg`, `NotificationsCfg`, `WorkflowCfg` structs. Define `ProjectStore` and `Detector` interfaces. Define the filesystem layout at `~/.cure/projects/<name>/project.json`. Document the 6-layer config merge chain extension (pkg defaults < global < project < repo < env < flags). Publish design as a comment on the epic issue. **Acceptance criteria:** (1) Go interface definitions for `ProjectStore` and `Detector` approved. (2) `Project` struct schema matches the architecture document. (3) Config merge chain extension approach documented. (4) Backward compatibility with existing session files confirmed. |

**Phase 2: Implementation (PARALLEL where independent)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| F-2 | feat(project): implement pkg/project — Project entity and ProjectStore | Feature | P0 | XL | `mrlm:software-engineer` | F-1 | — | Implement `pkg/project/project.go` (Project struct, Repo, Defaults, WorkflowCfg, NotificationsCfg, DevcontainerCfg), `pkg/project/store.go` (filesystem-based ProjectStore at `~/.cure/projects/`), `pkg/project/detect.go` (Detector matching cwd against registered repo paths). **Acceptance criteria:** (1) `ProjectStore.Save/Load/List/Delete` operations work with JSON files at `~/.cure/projects/<name>/project.json`. (2) `Detector.Detect(cwd)` returns the correct project when cwd is within a registered repo path. (3) Project names validated: lowercase alphanumeric + hyphens, unique. (4) All exported functions have tests. (5) Zero external dependencies. |
| F-3 | feat(project): implement cure project init/list/show CLI commands | Feature | P0 | L | `mrlm:software-engineer` | F-2 | — | Create `internal/commands/project/` package (package name `projcmd`). Implement `cure project init` (interactive wizard using `pkg/prompt` — name, description, repos, provider, tracker, devcontainer, notifications), `cure project list` (tabular output), `cure project show <name>` (full JSON display). Wire into `cmd/cure/main.go` router. **Acceptance criteria:** (1) `cure project init` creates a valid `project.json` at `~/.cure/projects/<name>/project.json`. (2) `cure project list` displays all registered projects with name, repo count, last updated. (3) `cure project show <name>` outputs the full project config. (4) Auto-detect from cwd works when running any cure command inside a project repo. (5) Unit tests and CLI integration tests pass. |
| F-4 | feat(agent): extend Session model with enrichment fields | Feature | P0 | M | `mrlm:software-engineer` | F-1 | — | Add fields to `pkg/agent.Session`: `Name` (string), `ProjectName` (string), `BranchName` (string), `RepoName` (string), `GitDirty` (bool), `WorkItems` ([]string), `AgentRole` (string), `ContainerID` (string). All fields use `omitempty` JSON tags for backward compatibility. Add `SessionFilter` struct and `Search(ctx, filter)` method to `SessionStore` interface. Implement `Search` in `pkg/agent/store/` filesystem store. **Acceptance criteria:** (1) Existing session JSON files deserialize without error (backward compatible). (2) New fields persist correctly in JSON. (3) `Search` filters by project name, provider, branch, work item, skill, with limit. (4) Session name auto-generates from `<provider>-<first-4-chars-of-id>` if not set. (5) All new fields and methods have tests. |
| F-5 | feat(config): extend merge chain with project layer | Feature | P0 | S | `mrlm:software-engineer` | F-2 | — | Modify the config loading in `internal/commands/` to add the project layer between global and repo in the `pkg/config.NewConfig(objs...)` call. When a project is detected (via `Detector`), load its `project.json` defaults as a `ConfigObject` and inject it into the merge chain. **Acceptance criteria:** (1) Config precedence: pkg defaults < global ~/.cure.json < project project.json < repo .cure.json < env vars < CLI flags. (2) Project layer only active when a project is detected. (3) Existing behavior unchanged when no project is detected. (4) Tests verify merge order with all 6 layers. |
| F-6 | feat(project): implement WorkflowCfg enforcement | Feature | P1 | M | `mrlm:software-engineer` | F-2, F-3 | — | Implement workflow rule validation. `WorkflowCfg.BranchPattern` (regex) is checked when cure creates branches. `WorkflowCfg.CommitPattern` (regex) is checked when cure creates commits. `WorkflowCfg.ProtectedBranch` list prevents direct pushes. `WorkflowCfg.RequireReview` flag is advisory (logged, not enforced at CLI level). Validation functions live in `pkg/project/workflow.go`. **Acceptance criteria:** (1) `ValidateBranch(name, pattern)` returns error if branch name does not match pattern. (2) `ValidateCommit(message, pattern)` returns error if commit message does not match pattern. (3) `IsProtected(branch, protectedList)` returns true for protected branches. (4) Tests cover: valid/invalid patterns, empty patterns (no enforcement), protected branch matching with glob support. |

**Phase 3: GUI and Integration (PARALLEL)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| F-7 | feat(gui): display enriched session metadata in session list and chat view | Feature | P1 | M | `mrlm:software-engineer` | F-4 | `mrlm:ui-designer` | Update the GUI session list (`/context`) to show: session name, project name, branch, provider, linked work items. Update the chat view (`/context/[id]`) header to show: session name, project, branch, git status, active skill, agent role. Update the API responses in `internal/gui/api/` to include the new session fields. **Acceptance criteria:** (1) Session list shows name, project, branch, and provider columns. (2) Chat view header displays all enrichment fields. (3) Empty fields (no project, no branch) degrade gracefully — show dashes or omit. (4) Existing sessions without new fields render correctly. |
| F-8 | feat(gui): add project management views — list and detail | Feature | P1 | L | `mrlm:software-engineer` | F-3 | `mrlm:ui-designer` | Add `/project` route to the SvelteKit frontend displaying all registered projects in a card layout (name, description, repo count, last updated). Add `/project/[name]` route showing full project details: repos, defaults, devcontainer config, notifications config, workflow rules. Add API routes: `GET /api/project` (list), `GET /api/project/:name` (detail). **Acceptance criteria:** (1) `/project` lists all projects with name, description, repo count. (2) `/project/[name]` displays full config in a readable format. (3) Navigation between project list and project detail works. (4) No projects state shows an informative empty state. |

**Phase 4: Quality Gates (PARALLEL)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| F-9 | review(v0.12.0): code review for all v0.12.0 PRs | Task | P0 | S | `mrlm:code-reviewer` | F-2 through F-8 | — | Systematic code review of all PRs in v0.12.0. Focus on: correctness of Project entity and store, backward compatibility of Session model changes, config merge chain ordering, test coverage, adherence to Go conventions and project structure rules. **Acceptance criteria:** (1) All PRs reviewed and approved. (2) No critical or high-severity findings remain open. (3) Test coverage for new code exceeds 80%. |

---

### Epic: v0.13.0 — AI Config Distribution

**Playbook:** Playbook 1 (Feature Development)
**Objective:** Build the AI config registry, managed config file system, runtime assembly, and drift detection. This epic makes cure the control plane for AI tooling configuration.
**Domains:** G (AI Config Distribution)
**Prerequisite:** v0.12.0 released (Project entity available).

**Phase 1: Architecture (SERIAL)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| G-0 | design(registry): architecture and interfaces for pkg/registry and config management | Task | P0 | M | `mrlm:software-architect` | v0.12.0 | `mrlm:software-engineer`, `mrlm:security-specialist` | Define `Source`, `Registry`, `RegistryStore` interfaces. Define the source directory layout (templates/, skills/, agents/, configs/, mcp/, prompts/). Define the resolution order (embedded < registry sources < project < repo). Define managed-file marker format and drift detection algorithm. Document the runtime assembly pipeline. **Acceptance criteria:** (1) Interfaces approved. (2) Source directory structure documented. (3) Marker format specified. (4) Runtime assembly data flow documented. |

**Phase 2: Implementation (SERIAL then PARALLEL)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| G-1 | feat(registry): implement pkg/registry — Source registry and store | Feature | P0 | L | `mrlm:software-engineer` | G-0 | — | Implement `pkg/registry/source.go` (Source struct), `pkg/registry/registry.go` (Registry with overlay resolution), `pkg/registry/store.go` (filesystem-based RegistryStore at `~/.cure/registry/`). The registry resolves artifacts from the overlay stack: embedded < registered sources (in registration order) < project overrides < repo overrides. **Acceptance criteria:** (1) `RegistryStore.Save/Load/List/Delete` for source entries in `~/.cure/registry/registry.json`. (2) `Registry.Resolve(artifactType, name)` returns the highest-priority artifact from the overlay stack. (3) Sources are directories at `~/.cure/registry/<name>/` following the defined directory layout. (4) Last-registered source wins when multiple sources provide the same artifact (with warning). (5) All exported functions have tests. |
| G-2 | feat(registry): implement cure registry CLI commands | Feature | P0 | M | `mrlm:software-engineer` | G-1 | — | Create `internal/commands/registry/` package. Implement: `cure registry add <name> <git-url>` (clone repo to `~/.cure/registry/<name>/`, register source), `cure registry update <name>` (git pull), `cure registry remove <name>` (delete clone, deregister), `cure registry list` (display name, URL, last updated, item counts). Wire into router. **Acceptance criteria:** (1) `add` clones a git repo and registers it. (2) `update` pulls latest from remote. (3) `remove` deletes the clone and deregisters. (4) `list` shows all sources with metadata. (5) Clear error messages for: auth failure, network error, invalid repo structure. |
| G-3 | feat(sync): implement managed config file generation | Feature | P0 | L | `mrlm:software-engineer` | G-1, v0.12.0 (E-1) | — | Create `internal/config/managed/` package. Implement managed config file generation: resolve template from registry overlay, render with project+repo context, insert managed-file marker (`<!-- managed by cure: sha256:<hash> -->`), write via `pkg/fs` atomic write. Implement `cure sync` command (`internal/commands/sync/`). Supported managed files: CLAUDE.md, .claude/settings.json, .mcp.json, .cursor/rules/*.mdc, .github/copilot-instructions.md, agents.md, skill definitions, agent definitions. Each file type is opt-in via `project.json` `ai_config.managed_files` array. **Acceptance criteria:** (1) `cure sync` generates all configured managed files in the current repo. (2) Managed-file marker inserted at top of each generated file. (3) Template rendering uses project config values as context variables. (4) Files not in managed_files list are skipped. (5) `--force` flag overwrites without prompt. (6) Without `--force`, warns about existing files that would be overwritten. |
| G-4 | feat(agent): implement runtime assembly of agent context from registry + project | Feature | P0 | L | `mrlm:software-engineer` | G-1, v0.12.0 (F-4) | — | Extend `internal/agent/claudecode/` adapter to assemble CC CLI invocations from project config + registry. Implement assembly pipeline: (1) System prompt: base (registry) + project instructions + repo context. (2) MCP config: assembled from registry `mcp/servers.json` + project config, written to temp file. (3) Settings: assembled from registry + project, written to temp file. (4) Agent definitions: from registry `agents/`, passed via `--agents`. (5) Skills: from registry `skills/`. Map project.json fields to CC CLI flags: `--model`, `--max-turns`, `--max-budget-usd`, `--system-prompt-file`, `--mcp-config`, `--agents`, `--permission-mode`, `--settings`. **Acceptance criteria:** (1) When `cure context new` is run in a project repo, the system prompt is assembled from all layers. (2) CC CLI is invoked with all assembled flags. (3) Assembly is logged at debug level showing which source contributed which element. (4) Missing registry sources produce warnings, not errors. (5) Non-CC providers (OpenAI, Gemini) receive assembled system prompt and tools without CC-specific flags. |
| G-5 | feat(sync): implement drift detection | Feature | P1 | M | `mrlm:software-engineer` | G-3 | — | Create `internal/config/drift/` package. Implement drift detection: (1) Read managed-file marker from existing file. (2) Regenerate expected content from registry + project. (3) Compare SHA-256 hash in marker with hash of current content (excluding marker). (4) Report drifted files with nature of change. Integrate with `cure sync --check` (report-only mode) and `cure doctor` (drift check registered as a doctor check). **Acceptance criteria:** (1) `cure sync --check` reports which managed files have drifted. (2) `cure sync` prompts per-file: apply cure's version, keep local, or show diff. (3) If marker is removed from a file, file is treated as unmanaged and skipped. (4) Doctor integration: drift check appears in `cure doctor` output. |

**Phase 3: Quality Gates (PARALLEL)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| G-6 | review(v0.13.0): code review for all v0.13.0 PRs | Task | P0 | S | `mrlm:code-reviewer` | G-1 through G-5 | — | Review all PRs. Focus on: registry overlay resolution correctness, managed-file marker security (no injection), runtime assembly correctness, config sync safety (non-destructive). **Acceptance criteria:** All PRs approved, no critical findings. |
| G-7 | test(v0.13.0): E2E tests for registry, sync, and runtime assembly | Task | P1 | M | `mrlm:qa-engineer` | G-1 through G-5 | `mrlm:software-engineer` | E2E tests covering: (1) Registry add/update/remove lifecycle. (2) Managed config generation and drift detection cycle. (3) Runtime assembly with mock project and registry sources. (4) Backward compatibility with sessions created pre-v0.13.0. **Acceptance criteria:** All E2E tests pass. Edge cases covered: empty registry, missing sources, corrupted source directories. |
| G-8 | docs(v0.13.0): registry and sync documentation | Task | P2 | S | `mrlm:personal-writer` | G-1 through G-5 | `mrlm:software-engineer` | Write documentation for: `cure registry` command usage, `cure sync` command usage, registry source directory structure, creating custom registry sources, managed file list, drift detection behavior. **Acceptance criteria:** Docs committed to `docs/` directory. |

---

### Epic: v0.14.0 — Project Management and Smart Doctor

**Playbook:** Playbook 1 (Feature Development)
**Objective:** Bring VCS operations, backlog management (GitHub + Azure DevOps), multi-stack doctor, project-scoped doctor, and project skeleton creation into cure.
**Domains:** C (Project Management), F (Smart Doctor)
**Prerequisite:** v0.12.0 released (Project entity available). Independent of v0.13.0.

**Phase 1: Implementation (PARALLEL — independent packages)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| C-1 | feat(vcs): implement pkg/vcs — typed git CLI wrappers | Feature | P0 | M | `mrlm:software-engineer` | v0.12.0 | — | Create `pkg/vcs/` package with typed wrappers over `git` CLI: `Status(dir)`, `Branch(dir, name)`, `Commit(dir, message, opts...)`, `Push(dir, opts...)`, `Pull(dir)`, `Diff(dir, opts...)`, `Log(dir, opts...)`. Use `os/exec` to shell out, parse structured output. `CommitOption` functional options support `WithValidatePattern(regex)` for workflow enforcement integration. **Acceptance criteria:** (1) All 7 git operations work via CLI shelling. (2) `WithValidatePattern` validates commit message against regex before executing. (3) Error handling: not-a-git-repo, git-not-installed, merge conflicts reported cleanly. (4) All functions have tests (using `git init` in `t.TempDir()`). |
| C-2 | feat(vcs): implement cure vcs CLI commands | Feature | P1 | M | `mrlm:software-engineer` | C-1 | — | Create `internal/commands/vcs/` package. Implement: `cure vcs status`, `cure vcs branch <name>`, `cure vcs commit -m "<msg>"`, `cure vcs push`, `cure vcs pull`, `cure vcs diff`, `cure vcs log`. Integrate workflow enforcement from project config. Wire into router. **Acceptance criteria:** (1) All VCS subcommands work and delegate to `pkg/vcs`. (2) Commit validates against `WorkflowCfg.CommitPattern` if project is detected. (3) Branch validates against `WorkflowCfg.BranchPattern` if project is detected. (4) Push to protected branch shows warning, requires `--force` flag. |
| C-3 | feat(backlog): implement backlog abstraction and GitHub adapter | Feature | P0 | L | `mrlm:software-engineer` | v0.12.0 | `mrlm:software-architect` | Create `internal/backlog/backlog.go` with `Tracker` interface (List, Get, Create, Update, Close) and `WorkItem` model. Create `internal/backlog/github/` adapter using `gh` CLI (JSON output mode). Implement: `List` via `gh issue list --json`, `Get` via `gh issue view --json`, `Create` via `gh issue create`, `Update` via `gh issue edit`, `Close` via `gh issue close`. Handle Projects v2 board integration per project.json config. **Acceptance criteria:** (1) Tracker interface defined with 5 methods. (2) GitHub adapter implements all 5 via `gh` CLI. (3) Projects v2 board status updated on create/update/close. (4) Rate limiting handled with backoff. (5) Clear errors for: gh not installed, auth failure. |
| C-4 | feat(backlog): implement Azure DevOps adapter | Feature | P1 | L | `mrlm:software-engineer` | C-3 | — | Create `internal/backlog/azdo/` adapter using `az boards` CLI. Implement all 5 `Tracker` methods via: `az boards work-item create`, `az boards work-item show`, `az boards query`, `az boards work-item update`, `az boards work-item delete`. Auto-detect tracker type from project.json `defaults.tracker.type`. **Acceptance criteria:** (1) AzDO adapter implements all 5 Tracker methods. (2) WIQL queries used for list/search. (3) State transitions follow Agile template by default (configurable). (4) Clear errors for: az not installed, extension missing, auth failure. |
| C-5 | feat(backlog): implement cure backlog CLI commands and MCP tools | Feature | P1 | M | `mrlm:software-engineer` | C-3, C-4 | — | Create `internal/commands/backlog/` package. Implement: `cure backlog list`, `cure backlog create --title "..." --body "..."`, `cure backlog view <id>`, `cure backlog update <id> --state "..."`. Auto-select tracker implementation from project.json. Create `internal/agent/tools/backlog.go` — `BacklogTools(tracker)` returning `[]agent.Tool` for agent use via MCP. **Acceptance criteria:** (1) CLI commands work with both GitHub and AzDO backends. (2) MCP tools registered for agent sessions: `backlog_list`, `backlog_create`, `backlog_view`, `backlog_update`. (3) Tracker auto-selection from project config. |
| F-1 | feat(doctor): implement multi-stack detection and checks | Feature | P0 | M | `mrlm:software-engineer` | — | — | Create `pkg/doctor/stack/` package. Implement `Stack` struct (Name, Detect func, Checks func) and `DetectStacks(dir)`. Implement stack detection for: Go (`go.mod`), Node (`package.json`), Python (`requirements.txt`, `pyproject.toml`), Rust (`Cargo.toml`), Java (`pom.xml`, `build.gradle`). Each stack provides 3-5 checks (tool version, package manager, dependency status). Add `CheckSkip` status to `pkg/doctor` for missing tool binaries. Add `--list` flag to show available checks. **Acceptance criteria:** (1) `DetectStacks` identifies all 5 stacks by trigger files. (2) Multi-stack repos get checks for all detected stacks. (3) Missing tool binary = SKIP, not FAIL. (4) `cure doctor --list` shows all registered checks grouped by stack. (5) Existing Go checks still work. |
| F-2 | feat(doctor): implement project-scoped doctor | Feature | P1 | S | `mrlm:software-engineer` | F-1, v0.12.0 | — | Extend `internal/commands/doctor/` to support `--project <name>` flag. When set, iterate all repos in the project's repo list, run `cure doctor` in each, aggregate results. Add JSON output option (`--json`). **Acceptance criteria:** (1) `cure doctor --project <name>` runs doctor in each registered repo. (2) Results grouped by repo with pass/fail/skip counts. (3) Exit code 1 if any repo has a failing check. (4) `--json` flag outputs structured JSON for programmatic consumption. (5) Missing repo paths produce warnings, not errors. |
| C-6 | feat(project): implement project skeleton creation | Feature | P2 | M | `mrlm:software-engineer` | v0.12.0, C-1 | — | Implement `cure project create <name>` wizard in `internal/commands/project/`. Interactive prompts for: language/stack, git hosting, AI providers, notification channels, devcontainer features. Generate: git init, CLAUDE.md, devcontainer.json, editorconfig, gitignore, CI workflow, project.json. Reuse existing `pkg/template` and `cure generate` infrastructure. **Acceptance criteria:** (1) Wizard collects all inputs interactively. (2) Generated project passes `cure doctor`. (3) Project entity registered at `~/.cure/projects/<name>/project.json`. (4) Existing directory prompts for confirmation before overwriting. |

**Phase 2: Quality Gates (PARALLEL)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| C-7 | review(v0.14.0): code review for all v0.14.0 PRs | Task | P0 | S | `mrlm:code-reviewer` | C-1 through C-6, F-1, F-2 | — | Review all PRs. Focus on: subprocess command injection risks in VCS/backlog wrappers, Tracker abstraction correctness, stack detection reliability, test coverage. **Acceptance criteria:** All PRs approved. |
| C-8 | test(v0.14.0): E2E tests for VCS, backlog, and doctor | Task | P1 | M | `mrlm:qa-engineer` | C-1 through C-6, F-1, F-2 | `mrlm:software-engineer` | E2E tests: (1) VCS operations in a temp git repo. (2) Backlog operations with mock gh/az CLI. (3) Multi-stack doctor with test repos containing different stack files. (4) Project-scoped doctor across multiple repos. **Acceptance criteria:** All E2E tests pass. |
| C-9 | docs(v0.14.0): VCS, backlog, and doctor documentation | Task | P2 | XS | `mrlm:personal-writer` | C-1 through C-6, F-1, F-2 | `mrlm:software-engineer` | Document: `cure vcs` commands, `cure backlog` commands, multi-stack doctor behavior, project-scoped doctor usage. **Acceptance criteria:** Docs committed to `docs/`. |

---

### Epic: v0.15.0 — GUI Evolution

**Playbook:** Playbook 1 (Feature Development)
**Objective:** Evolve the GUI from a dashboard into a development environment with Monaco editor, integrated terminal, diff viewer, config editor, and light/dark theme system.
**Domains:** D (GUI Evolution)
**Prerequisite:** v0.14.0 released (VCS API needed for diff viewer). Spike S-2 and S-4 results inform scope.

**Phase 1: Design (SERIAL)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| D-0 | design(gui): component layout, design system, and theme specification | Task | P0 | M | `mrlm:ui-designer` | v0.14.0, S-2, S-4 | `mrlm:software-architect`, `mrlm:software-engineer` | Define: (1) Layout for editor, terminal, file browser, and session panels. (2) Design system: color tokens, typography, spacing, component library. (3) Light and dark theme color palettes with OS preference detection (`prefers-color-scheme`). (4) Responsive behavior for desktop viewport sizes (min 1024px). (5) Accessibility spec: WCAG 2.1 AA compliance targets. **Acceptance criteria:** Design specification document with color tokens, component hierarchy, and theme toggle behavior. |

**Phase 2: Implementation (PARALLEL where independent)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| D-1 | feat(gui): implement design system and light/dark theme | Feature | P0 | M | `mrlm:software-engineer` | D-0 | `mrlm:ui-designer` | Implement CSS custom properties (variables) for the design system. Implement theme toggle (light/dark) with: (1) OS preference detection via `prefers-color-scheme`. (2) User override stored in localStorage. (3) Theme toggle button in the top bar. Update all existing components to use theme tokens instead of hardcoded colors. **Acceptance criteria:** (1) Theme toggles between light and dark. (2) OS preference detected on first visit. (3) User preference persists across sessions. (4) All existing pages (home, context, doctor, config, generate) use theme tokens. (5) No hardcoded color values remain. |
| D-2 | feat(gui): implement Monaco editor with file browser | Feature | P0 | L | `mrlm:software-engineer` | D-0 | `mrlm:ui-designer` | Add `/editor` route. Implement file browser (tree view) on the left panel using `GET /api/files?path=<dir>` API. Implement Monaco editor in the main panel with: syntax highlighting, tab bar for multiple files, save (Ctrl+S/Cmd+S via `PUT /api/files/<path>`), file CRUD (create, rename, delete). Implement Go backend file API (`GET/PUT/POST/DELETE /api/files/*`). Path traversal prevention: validate all resolved paths are within project boundaries. **Acceptance criteria:** (1) File browser shows directory tree for project repos. (2) Opening a file loads it in Monaco with syntax highlighting. (3) Saving writes to disk. (4) Tab bar supports multiple open files. (5) Binary files show a placeholder. (6) Max file size configurable (default 5 MB). (7) SSR-safe: Monaco loaded via dynamic import in onMount. |
| D-3 | feat(gui): implement Monaco editor LSP integration (conditional) | Feature | P1 | XL | `mrlm:software-engineer` | D-2, S-2 | `mrlm:software-architect` | **Conditional on spike S-2 success.** Implement LSP proxy in Go backend: spawn language server (user-configured path in project.json), bridge stdio to WebSocket at `WS /api/editor/lsp`. Implement `monaco-languageclient` on the frontend: connect to WebSocket, provide autocomplete, hover, go-to-definition, diagnostics. If S-2 concluded "defer LSP": skip this task, document as post-v1.0.0. **Acceptance criteria:** (1) LSP features work for at least Go (gopls). (2) Autocomplete, hover, diagnostics visible in Monaco. (3) Graceful degradation if LSP server not configured: syntax highlighting still works. (4) LSP server path configurable in project.json. |
| D-4 | feat(gui): implement integrated terminal emulator | Feature | P0 | L | `mrlm:software-engineer` | D-0, S-4 | — | Add terminal pane (bottom panel or tab). Implement: xterm.js on frontend connecting via WebSocket to `WS /api/terminal/:id`. Go backend: allocate PTY per session using `github.com/creack/pty`, manage terminal sessions in `internal/gui/ws/`. Support: multiple terminal tabs, resize events, scrollback buffer (configurable), copy-paste, user's default shell. WebSocket protocol: binary frames for I/O, JSON for control messages (resize, exit). **Acceptance criteria:** (1) Terminal opens with user's default shell in project working directory. (2) ANSI colors and cursor movement render correctly. (3) Multiple terminals via tab bar. (4) Resize events propagate. (5) Shell exit shows "Session ended" with restart option. (6) WebSocket reconnection on disconnect. |
| D-5 | feat(gui): implement diff viewer with VCS integration | Feature | P1 | M | `mrlm:software-engineer` | D-2, C-1 (v0.14.0) | — | Implement diff view using Monaco's `createDiffEditor`. Add `GET /api/vcs/diff?path=<file>&base=<ref>` API endpoint returning original and modified file contents. UI: file list showing all changed files, clicking a file opens side-by-side diff. Support: uncommitted changes diff, branch-to-branch comparison. Integrate with VCS operations: approve (stage) or discard individual file changes. **Acceptance criteria:** (1) Diff view shows side-by-side comparison with syntax highlighting. (2) File list of all changed files. (3) Stage/discard individual files from diff view. (4) New/deleted files handled correctly. |
| D-6 | feat(gui): implement config editor | Feature | P1 | M | `mrlm:software-engineer` | D-2, v0.12.0 | `mrlm:ui-designer` | Add `/config/editor` route. Implement config editor using Monaco with JSON language mode. Backend APIs: `GET /api/config/layers` (list layers with sources), `GET /api/config/effective` (merged config), `GET /api/config/layer/<name>` (raw layer), `PUT /api/config/layer/<name>` (update), `POST /api/config/validate` (schema validation). UI: layer selector showing which layer each value comes from, effective config preview, validation feedback. **Acceptance criteria:** (1) All config layers displayed with their sources. (2) Editing a layer validates and saves. (3) Effective (merged) config preview updates on change. (4) Syntax errors highlighted, save blocked until fixed. (5) Layer origin indicator for each config value. |

**Phase 3: Quality Gates (PARALLEL)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| D-7 | review(v0.15.0): code review for all v0.15.0 PRs | Task | P0 | S | `mrlm:code-reviewer` | D-1 through D-6 | — | Review all PRs. Focus on: file API path traversal prevention, WebSocket security, PTY resource cleanup, Monaco bundle size, theme consistency. **Acceptance criteria:** All PRs approved. |
| D-8 | test(v0.15.0): E2E and UX testing for GUI features | Task | P1 | M | `mrlm:qa-engineer` | D-1 through D-6 | `mrlm:ui-designer` | E2E tests: (1) File browser navigation and file CRUD. (2) Monaco editor open/edit/save cycle. (3) Terminal open, command execution, output verification. (4) Diff viewer with test repo. (5) Config editor layer display and edit. (6) Theme toggle light/dark. UX testing: keyboard navigation, screen reader basics, color contrast in both themes. **Acceptance criteria:** All E2E tests pass. No critical UX issues. |
| D-9 | test(v0.15.0): visual regression testing for GUI themes | Task | P2 | S | `mrlm:qa-engineer` | D-1 | `mrlm:ui-designer` | Capture baseline screenshots for all views in both light and dark themes. Set up visual regression comparison for future changes. **Acceptance criteria:** Baseline screenshots captured. No visual anomalies in either theme. |

---

### Epic: v0.16.0 — Notifications

**Playbook:** Playbook 1 (Feature Development)
**Objective:** Enable agents to notify developers via Microsoft Teams and OS notifications, with bidirectional messaging support.
**Domains:** B (Agent-Human Communication)
**Prerequisite:** v0.12.0 released (Project entity for notification config).

**Phase 1: Architecture (SERIAL)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| B-0 | design(notify): notification channel architecture and dispatcher | Task | P0 | M | `mrlm:software-architect` | v0.12.0, S-3 | `mrlm:software-engineer`, `mrlm:security-specialist` | Define `Channel` interface (Name, Send, Responses), `Notification` struct, `Response` struct, `EventType` enum, `Dispatcher` struct (Notify, WaitResponse). Define first-response-wins semantics for bidirectional messaging. Define channel configuration schema in project.json. If S-3 concluded "defer bidirectional": adjust B-1 Phase 2 scope. **Acceptance criteria:** Interfaces approved. Dispatcher design documented. Channel configuration schema defined. |

**Phase 2: Implementation (SERIAL for core, PARALLEL for channels)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| B-1 | feat(notify): implement pkg/notify — channel interface and dispatcher | Feature | P0 | M | `mrlm:software-engineer` | B-0 | — | Implement `pkg/notify/` package: `Channel` interface, `Notification` struct, `Response` struct, `EventType` constants (`completion`, `blocker`, `decision_needed`, `error`), `Dispatcher` struct with `Notify()` (fan-out to all channels) and `WaitResponse()` (first-response-wins with channel multiplexing). **Acceptance criteria:** (1) Dispatcher sends to all enabled channels. (2) WaitResponse blocks until any channel returns a response. (3) First response wins; subsequent responses for same session are rejected. (4) Context cancellation propagates correctly. (5) All exported functions have tests. |
| B-2 | feat(notify): implement OS local notification channel | Feature | P0 | S | `mrlm:software-engineer` | B-1 | — | Create `internal/notifications/local/` implementing `notify.Channel`. macOS: `osascript -e 'display notification ...'`. Linux: `notify-send --app-name=cure ...`. `Responses()` returns nil (unidirectional). Configuration: enabled/disabled, event types filter. **Acceptance criteria:** (1) Notifications appear on macOS and Linux. (2) Event type filtering works. (3) Missing notification daemon: silently skip, log warning once. (4) `Responses()` returns nil. |
| B-3 | feat(notify): implement Teams outbound webhook channel (Phase 1) | Feature | P1 | M | `mrlm:software-engineer` | B-1 | `mrlm:platform-engineer` | Create `internal/notifications/teams/` implementing `notify.Channel`. Phase 1: Outbound-only via Incoming Webhook. POST JSON to webhook URL. Map session to thread via `replyToId`. Configuration from project.json `notifications.teams.webhook_url`. `Responses()` returns nil in Phase 1. **Acceptance criteria:** (1) Messages posted to Teams channel via webhook. (2) Each session creates its own thread. (3) Subsequent messages in same session reply to the thread. (4) Failed webhook: queue locally, retry 3x with exponential backoff. (5) Bot token expiry: log error, continue without Teams. |
| B-4 | feat(notify): implement Teams Bot Framework bidirectional channel (Phase 2) | Feature | P1 | L | `mrlm:software-engineer` | B-3, S-3 | `mrlm:platform-engineer`, `mrlm:security-specialist` | **Conditional on spike S-3 success.** Upgrade Teams channel to Bot Framework. Implement: Azure AD app registration flow documentation, bot webhook callback handler in GUI server, thread-to-session mapping, `Responses()` returning a channel of user replies. If S-3 concluded "defer bidirectional": skip this task, document as post-v1.0.0. **Acceptance criteria:** (1) User replies in Teams thread are received by Go backend. (2) Replies routed to correct session via thread ID. (3) Cross-channel response dedup works (responded-via-GUI shown in Teams). (4) Setup documentation for Azure AD registration. |
| B-5 | feat(notify): implement GUI notification channel | Feature | P1 | S | `mrlm:software-engineer` | B-1 | — | Implement GUI as a `notify.Channel` in `internal/gui/`. `Send` pushes notification events via existing SSE stream (new event type: `notification`). `Responses()` receives replies from the chat interface input. **Acceptance criteria:** (1) Notifications appear in GUI as toast/banner elements. (2) Chat interface can send responses to waiting agents. (3) GUI channel integrates with dispatcher like any other channel. |
| B-6 | feat(notify): implement bidirectional messaging dispatch | Feature | P1 | L | `mrlm:software-engineer` | B-1, B-2, B-3 or B-4, B-5 | — | Wire the dispatcher into the agent session lifecycle. When an agent session emits a notification event, the dispatcher fans it out to all enabled channels. When an agent is waiting for user input (`WaitResponse`), the dispatcher multiplexes across all channels. Implement cross-channel acknowledgment: when a response arrives on one channel, other channels are updated (e.g., Teams thread shows "Resolved via GUI"). **Acceptance criteria:** (1) Notification events from agent sessions reach all enabled channels. (2) User responses from any channel unblock the agent. (3) Cross-channel acknowledgment works. (4) Disabled channels are silently skipped. |

**Phase 3: Quality Gate**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| B-7 | review(v0.16.0): code review for all v0.16.0 PRs | Task | P0 | S | `mrlm:code-reviewer` | B-1 through B-6 | — | Review all PRs. Focus on: channel interface correctness, dispatcher concurrency safety, Teams auth credential handling, response routing correctness. **Acceptance criteria:** All PRs approved. |

---

### Epic: v0.17.0 — Multi-Instance Orchestration

**Playbook:** Playbook 1 (Feature Development)
**Objective:** Enable 2-4 concurrent agent sessions running in isolated devcontainers, coordinated by the host cure instance via MCP protocol.
**Domains:** A (Multi-Instance Orchestration)
**Prerequisite:** v0.12.0 (Project entity), v0.13.0 (registry + runtime assembly), S-1 (MCP transport decision).

**Phase 1: Architecture (SERIAL)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| A-0 | design(orchestrate): orchestration architecture, MCP client, and container lifecycle | Task | P0 | M | `mrlm:software-architect` | v0.13.0, S-1 | `mrlm:software-engineer`, `mrlm:platform-engineer`, `mrlm:security-specialist` | Define: (1) `pkg/mcp/Client` interface (CallTool, ListTools). (2) `internal/orchestrator/Orchestrator` struct (Init, Up, Down, Status, Logs, MCPClient). (3) Docker Compose generation from devcontainer.json. (4) Container-to-host MCP authentication (shared secret Bearer token). (5) Health monitoring via `docker compose ps --format json`. (6) GUI integration points for orchestration status and container terminals. **Acceptance criteria:** Architecture document approved. MCP Client interface defined. Compose generation schema documented. Auth model specified. |

**Phase 2: Implementation (SERIAL core, PARALLEL layers)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| A-1 | feat(mcp): implement MCP client in pkg/mcp | Feature | P0 | L | `mrlm:software-engineer` | A-0 | — | Add `Client` to `pkg/mcp`: `NewClient(endpoint, opts...)`, `CallTool(ctx, name, args)`, `ListTools(ctx)`. Support HTTP Streamable transport (per S-1 result). Implement Bearer token authentication via `ClientOption`. **Acceptance criteria:** (1) Client connects to a remote MCP server. (2) `CallTool` invokes a tool and returns the result. (3) `ListTools` returns available tools. (4) Bearer token sent in Authorization header. (5) Connection failure detected within 10 seconds. (6) Retry with exponential backoff (max 30 seconds) for server-not-ready. (7) All functions have tests against a test MCP server. |
| A-2 | feat(orchestrate): implement Docker Compose generation from devcontainer.json | Feature | P0 | L | `mrlm:software-engineer` | A-0, v0.12.0 | `mrlm:platform-engineer` | Create `internal/orchestrator/compose.go`. Read project's `.devcontainer/devcontainer.json`. Generate `docker-compose.cure.yml` with: one service per agent slot (configurable count, default 4), same Dockerfile/features as devcontainer, workspace volume mount, MCP port mapping (9100 + offset), `CURE_AGENT_NAME`, `CURE_MCP_PORT`, `CURE_HOST_URL`, `CURE_MCP_SECRET` environment variables, shared `cure-net` bridge network. `cure orchestrate init` CLI command. **Acceptance criteria:** (1) `cure orchestrate init` generates deterministic compose file. (2) Agent count configurable via `--agents` flag. (3) Compose file references devcontainer Dockerfile. (4) Each service has unique port mapping. (5) MCP secret auto-generated (32 bytes hex). (6) Running init twice produces identical output. |
| A-3 | feat(orchestrate): implement container lifecycle management | Feature | P0 | XL | `mrlm:software-engineer` | A-2, A-1 | `mrlm:platform-engineer` | Create `internal/orchestrator/orchestrator.go`. Implement: `Up(ctx)` (docker compose up -d), `Down(ctx)` (docker compose down), `Status(ctx)` (container health via docker compose ps --format json, polled every 5s), `Logs(ctx, name, w)` (stream container logs), `MCPClient(name)` (return MCP client for container). Wire CLI commands: `cure orchestrate up`, `cure orchestrate down`, `cure orchestrate status`, `cure orchestrate logs [name]`. Handle graceful shutdown: SIGINT/SIGTERM stops all containers. Failed container: report error, other containers continue. **Acceptance criteria:** (1) `cure orchestrate up` starts all containers. (2) `cure orchestrate down` stops all containers. (3) `cure orchestrate status` shows health per container. (4) Logs stream correctly. (5) MCP client connects to each container. (6) SIGINT triggers graceful container shutdown. (7) Individual container failure does not affect others. |
| A-4 | feat(orchestrate): implement multi-agent session management and GUI integration | Feature | P0 | L | `mrlm:software-engineer` | A-3 | `mrlm:ui-designer` | Integrate orchestration with the GUI server: (1) `/api/orchestrate/status` returns all container statuses. (2) `/api/orchestrate/up` and `/api/orchestrate/down` control lifecycle via GUI. (3) SSE events for orchestration status changes. (4) Container terminal sessions via `docker exec -it`. (5) All agent sessions visible in session list with `ContainerID` field populated. (6) Agent role display (build, review, test, etc.). **Acceptance criteria:** (1) GUI dashboard shows all orchestrated agents with status. (2) Starting/stopping containers from GUI works. (3) Container terminals accessible from GUI. (4) Agent sessions enriched with container info. |

**Phase 3: Quality Gates (PARALLEL)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| A-5 | review(v0.17.0): code review for all v0.17.0 PRs | Task | P0 | S | `mrlm:code-reviewer` | A-1 through A-4 | — | Review all PRs. Focus on: MCP client security (auth, connection handling), Docker Compose generation correctness, resource cleanup, SIGINT handling, path traversal in container interactions. **Acceptance criteria:** All PRs approved. |
| A-6 | test(v0.17.0): E2E tests for orchestration | Task | P0 | M | `mrlm:qa-engineer` | A-1 through A-4 | `mrlm:software-engineer` | E2E tests: (1) MCP client-server roundtrip with auth. (2) Docker Compose generation validation. (3) Container lifecycle (up/status/down). (4) Multi-agent concurrent tool calls (stress test). (5) Graceful shutdown behavior. **Note:** Requires Docker available in test environment. **Acceptance criteria:** All E2E tests pass. Concurrent tool calls succeed under 4-agent load. |
| A-7 | security(v0.17.0): security review of MCP auth and container isolation | Task | P0 | S | `mrlm:security-specialist` | A-1, A-3 | — | Review: (1) MCP shared-secret generation (entropy, randomness). (2) Bearer token transmission security. (3) Container network isolation. (4) File API path traversal prevention for container-mounted volumes. (5) Environment variable handling for secrets. **Acceptance criteria:** No critical or high-severity findings. Recommendations documented. |
| A-8 | docs(v0.17.0): orchestration documentation | Task | P1 | S | `mrlm:personal-writer` | A-1 through A-4 | `mrlm:software-engineer` | Document: `cure orchestrate` commands, devcontainer requirements, Docker prerequisites, multi-agent workflow guide, troubleshooting. **Acceptance criteria:** Docs committed to `docs/`. |

---

### Epic: v1.0.0 — Stabilization and Release

**Playbook:** Playbook 4 (Release Preparation)
**Objective:** Integration testing, security audit, documentation, performance testing, GUI polish, and release. This is the quality gate before the v1.0.0 tag.
**Prerequisite:** All minor releases (v0.12.0 through v0.17.0) complete.

**Execution: SERIAL (quality gates must pass sequentially)**

| # | Task | Type | Priority | Size | Agent | Depends On | Consult | Description |
|---|------|------|----------|------|-------|------------|---------|-------------|
| R-1 | test(v1.0.0): cross-domain integration tests | Task | P0 | L | `mrlm:qa-engineer` | v0.17.0 | `mrlm:software-engineer` | Integration tests covering cross-domain workflows: (1) Project init -> registry add -> sync -> context new -> agent gets assembled context. (2) Orchestrate up -> multi-agent session -> notifications dispatched -> container terminals. (3) Backlog create from agent via MCP tool -> issue appears in project board. (4) VCS operations from GUI (commit, diff, push). (5) Config editor changes reflected in next agent session. **Acceptance criteria:** All cross-domain integration tests pass. No P0/P1 bugs found or all fixed. |
| R-2 | test(v1.0.0): performance testing under multi-agent load | Task | P0 | M | `mrlm:qa-engineer` | v0.17.0 | `mrlm:platform-engineer` | Performance tests: (1) 4-agent concurrent MCP tool calls — sustained throughput for 30 minutes. (2) GUI with 4 active SSE streams + 2 terminal WebSockets — memory and CPU usage stable. (3) Session store Search with 1000+ sessions — response time <200ms. (4) File API with large directory trees — response time acceptable. **Acceptance criteria:** No memory leaks. p95 latencies within targets. Resource usage stable under sustained load. |
| R-3 | secure(v1.0.0): comprehensive security audit | Task | P0 | L | `mrlm:security-specialist` | v0.17.0 | `mrlm:software-engineer` | Full security review: (1) MCP auth (shared secret entropy, token handling). (2) File API path traversal (all `/api/files/*` routes). (3) WebSocket security (terminal, LSP). (4) Teams credential storage. (5) Docker Compose secret handling. (6) SBOM generation. (7) Dependency vulnerability scan. (8) Config file permission checks. **Acceptance criteria:** No critical findings. SBOM generated. All high-severity findings resolved or accepted with documented rationale. |
| R-4 | feat(gui): final GUI polish — responsive design, error states, loading states | Task | P1 | M | `mrlm:software-engineer` | R-1 | `mrlm:ui-designer` | Address: (1) Loading indicators for all async operations. (2) Error states with actionable messages for all API failures. (3) Empty states for all list views. (4) Responsive behavior at common viewport sizes (1024-2560px). (5) Keyboard navigation for critical workflows. (6) Consistent animation and transition timing. **Acceptance criteria:** No unhandled loading or error states in any view. WCAG 2.1 AA color contrast in both themes. |
| R-5 | review(v1.0.0): final code review sweep | Task | P0 | M | `mrlm:code-reviewer` | R-1, R-3, R-4 | — | Final review sweep across the entire codebase for: (1) TODO/FIXME items that should be resolved. (2) API consistency across all routes. (3) Error handling completeness. (4) Test coverage gaps. (5) Documentation accuracy. **Acceptance criteria:** No open critical or high-severity findings. |
| R-6 | docs(v1.0.0): user guide, architecture guide, and API reference | Task | P0 | M | `mrlm:personal-writer` | R-4 | `mrlm:software-engineer`, `mrlm:software-architect` | Write: (1) User guide covering all cure commands and workflows. (2) Architecture guide for contributors. (3) API reference for all GUI HTTP/WebSocket endpoints. (4) Getting started guide updated for v1.0.0 capabilities. (5) Upgrade guide from v0.11.x to v1.0.0. **Acceptance criteria:** All docs committed. No stale references to pre-v1.0.0 behavior. |
| R-7 | test(v1.0.0): release readiness assessment | Task | P0 | S | `mrlm:qa-engineer` | R-1 through R-6 | `mrlm:code-reviewer`, `mrlm:security-specialist` | Final release readiness check: (1) All tests pass (unit, E2E, integration, performance). (2) Security audit findings resolved. (3) Documentation complete. (4) No P0 or P1 bugs open. (5) Changelog complete. **Acceptance criteria:** Release readiness report signed off. |
| R-8 | chore(v1.0.0): tag, GitHub Release, and deployment | Task | P0 | S | `mrlm:platform-engineer` | R-7 | `mrlm:personal-writer` | Tag `v1.0.0`, create GitHub Release with release notes (title: "v1.0.0"), verify CI passes, verify Go module proxy picks up the tag. **Acceptance criteria:** (1) Tag v1.0.0 pushed. (2) GitHub Release published with title "v1.0.0". (3) `go install github.com/mrlm-net/cure@v1.0.0` works. |

---

## Risk Register

| # | Risk | Probability | Impact | Score | Mitigation | Owner |
|---|------|-------------|--------|-------|------------|-------|
| 1 | **Monaco + LSP integration too complex for v1.0.0.** monaco-languageclient may not work reliably in SvelteKit 5 with Go backend LSP proxy. | H | H | 9 | Spike S-2 validates before commitment. Fallback: Monaco ships with syntax highlighting only. LSP task (D-3) is explicitly conditional. | `mrlm:software-architect` |
| 2 | **Teams Bot Framework auth complexity exceeds time box.** Azure AD registration, OAuth flows, and webhook infrastructure may be too complex. | H | M | 6 | Start with outbound-only Incoming Webhook (B-3). Bidirectional (B-4) is conditional on S-3. Phase 1 delivers value independently. | `mrlm:platform-engineer` |
| 3 | **Docker reliability on macOS.** Docker Desktop has known performance and stability issues with mounted volumes and nested containers. | M | H | 6 | Orchestration is v0.17.0 (last domain). All other features work without Docker. Doctor checks validate Docker availability. Test early on macOS Docker Desktop. | `mrlm:platform-engineer` |
| 4 | **MCP transport selection wrong.** HTTP Streamable may be unreliable under 4-agent concurrent load on Docker bridge. | M | H | 6 | Spike S-1 benchmarks all three transports before implementation. MCP Client interface is transport-agnostic — switching requires only changing the connection factory. | `mrlm:software-architect` |
| 5 | **Scope size leads to quality degradation.** 65 tasks across 8 epics is substantial. Risk of rushing late-stage work. | H | M | 6 | Each epic is independently shippable. Quality gates per epic (review + E2E). Stabilization epic provides buffer. Conditional tasks (D-3, B-4) can be deferred without blocking v1.0.0. | delivery-manager |
| 6 | **Context window exhaustion on large tasks.** Build agents may run out of context on XL tasks. | M | M | 4 | XL tasks broken into focused sub-PRs. One agent owns main.go wiring to avoid conflicts. Test artifacts cleaned before committing. | delivery-manager |
| 7 | **Config sync data loss.** Multi-level config merge with 6 layers introduces potential for accidental overwrites. | L | H | 3 | Config sync is non-destructive (never deletes). Conflicts reported, never auto-resolved. User confirmation required for overwrites. Atomic writes via `pkg/fs`. | `mrlm:software-architect` |
| 8 | **Breaking changes in external CLI tools.** `gh`, `az`, `docker compose` output format changes could break parsers. | L | M | 2 | Pin to stable output flags (`--json`, `--format json`). Integration tests catch format changes early. | `mrlm:software-engineer` |

**Scoring:** H=3, M=2, L=1. Score = Probability x Impact. Critical >=6, High >=4, Medium >=2, Low =1.

---

## Workload Management Configuration

All issues created from this plan use the following GitHub Projects v2 configuration:

- **Repository:** mrlm-net/cure
- **Project:** #9 "CURE CLI" (org: mrlm-net)
- **Project ID:** `PVT_kwDOBxaH0c4BPROP`

### Field IDs

| Field | ID | Options |
|-------|----|---------|
| Status | `PVTSSF_lADOBxaH0c4BPROPzg9ufVk` | Backlog (`f75ad846`), Ready (`61e4505c`), In progress (`47fc9ee4`), In review (`df73e18b`), Done (`98236657`) |
| Priority | `PVTSSF_lADOBxaH0c4BPROPzg9ufYY` | P0 (`79628723`), P1 (`0a877460`), P2 (`da944a9c`) |
| Size | `PVTSSF_lADOBxaH0c4BPROPzg9ufYc` | XS (`6c6483d2`), S (`f784b110`), M (`7515a9f1`), L (`817d0097`), XL (`db339eb2`) |

### Agent Workflow for Issues

1. **Issue created:** Add to project board, set Status to "Backlog"
2. **Issue refined and ready:** Set Status to "Ready"
3. **Implementation started:** Set Status to "In progress"
4. **PR opened:** Set Status to "In review"
5. **PR merged and issue closed:** Set Status to "Done"

### Labels

Each issue should receive appropriate labels:
- **Type:** `type/epic`, `type/feature`, `type/task`, `type/spike`
- **Priority:** `priority/p0`, `priority/p1`, `priority/p2`
- **Status:** `status/backlog`, `status/ready`, `status/in-progress`, `status/review`
- **Milestone:** `milestone/v0.12.x`, `milestone/v0.13.x`, ..., `milestone/v1.0.x`
- **Domain:** Create as needed (e.g., `domain/project`, `domain/registry`, `domain/gui`, `domain/notify`, `domain/orchestrate`, `domain/vcs`, `domain/doctor`)

---

*End of delivery plan. This document is the input for Phase 4: Backlog Creation, where each task becomes a GitHub Issue on the project board.*
