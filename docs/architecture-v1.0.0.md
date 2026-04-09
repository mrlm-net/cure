# Cure v1.0.0 Architecture Design

**Date:** 2026-04-09
**Author:** Architect
**Status:** Draft for review
**Preceding document:** [requirements-v1.0.0.md](requirements-v1.0.0.md)

---

## Table of Contents

- [Executive Summary](#executive-summary)
- [Architecture Principles](#architecture-principles)
- [System Context](#system-context)
- [Package Map — New and Extended](#package-map--new-and-extended)
- [Domain Architectures](#domain-architectures)
  - [E: Project Entity and Config Sync](#e-project-entity-and-config-sync)
  - [G: AI Config Distribution](#g-ai-config-distribution)
  - [F: Smart Doctor](#f-smart-doctor)
  - [C: Project Management](#c-project-management)
  - [B: Notifications](#b-notifications)
  - [D: GUI Evolution](#d-gui-evolution)
  - [A: Multi-Instance Orchestration](#a-multi-instance-orchestration)
- [Data Model Changes](#data-model-changes)
- [Communication Architecture](#communication-architecture)
- [Interface Definitions](#interface-definitions)
- [Dependency Map](#dependency-map)
- [ADR Log](#adr-log)
- [Incremental Delivery Strategy](#incremental-delivery-strategy)
- [Risk Mitigation Through Architecture](#risk-mitigation-through-architecture)
- [Appendix: CC CLI Integration Model](#appendix-cc-cli-integration-model)

---

## Executive Summary

Cure v1.0.0 transforms a single-agent CLI tool into an AI development platform. The architecture introduces six new public packages (`pkg/project`, `pkg/registry`, `pkg/vcs`, `pkg/notify`, `pkg/orchestrator`, `pkg/doctor/stack`), extends three existing ones (`pkg/agent`, `pkg/config`, `pkg/doctor`), and adds significant internal wiring. The design preserves the stdlib-only constraint for `pkg/` packages (with the exception of approved SDKs in `internal/`), maintains the `pkg/` versus `internal/` boundary, and supports incremental delivery through 5-6 minor releases before the v1.0.0 tag.

The foundational packages — `pkg/project` (Project entity) and `pkg/registry` (AI config source registry) — unlock all other domains and must be built first. The architecture is designed so that each domain can be delivered independently once its foundation dependency is satisfied.

---

## Architecture Principles

These principles extend the existing codebase conventions for v1.0.0 scope.

1. **Shell out, do not embed.** For external tools (git, docker, gh, az), use subprocess execution with structured output parsing. This avoids massive dependency trees (go-git is ~40K LOC, Docker SDK is even larger) and leverages tools developers already have installed. The `pkg/` packages provide typed wrappers over CLI tools.

2. **Project is the new root entity.** Every cross-repo, cross-agent, and cross-notification operation flows through a Project. The Project entity replaces ad-hoc per-repo configuration as the primary configuration scope.

3. **Channels, not destinations.** The notification system uses a channel abstraction. Teams, OS notifications, GUI, and future channels (Slack, email) all implement the same interface. The dispatcher routes events to all enabled channels.

4. **GUI server is the coordinator.** No separate daemon. The `cure gui` HTTP server manages active sessions, orchestrated containers, WebSocket terminals, and notification dispatch. CLI commands are thin wrappers that either operate standalone or communicate with the running GUI server.

5. **MCP is the inter-process protocol.** Host-to-container communication uses MCP over HTTP Streamable transport on a Docker bridge network. This reuses the existing `pkg/mcp` server and adds a client.

6. **Registry is append-only overlays.** The AI config registry follows the same overlay model as `pkg/template` (embedded < external sources < project < repo). No new concept — just a wider scope of managed artifacts.

7. **CC CLI as the primary agent runtime.** Cure assembles Claude Code CLI invocations from project config + registry rather than reimplementing agent orchestration. The `--settings`, `--mcp-config`, `--system-prompt`, `--tools`, and `--agents` flags give cure full programmatic control over CC's behavior.

---

## System Context

```
+------------------------------------------------------------------+
|                     Developer Workstation                         |
|                                                                   |
|  +------------------------------+    +------------------------+  |
|  |         cure CLI             |    |      cure GUI          |  |
|  | cure project init            |    | HTTP :port/            |  |
|  | cure context new             |    | SSE /api/sessions/     |  |
|  | cure orchestrate up          |    | WS  /api/terminal/     |  |
|  | cure backlog list            |    | WS  /api/editor/lsp    |  |
|  | cure sync                    |    | Monaco + xterm.js      |  |
|  | cure doctor                  |    +----------+-------------+  |
|  | cure registry add            |               |                |
|  +------------------------------+               |                |
|         |            |                          |                |
|  +------+------+     |    +--------------------+                 |
|  | pkg/project  |     |    | internal/gui/api                   |
|  | pkg/registry |     |    | internal/gui (HTTP server)          |
|  | pkg/vcs      |     |    +----+-------+--------+---------+    |
|  | pkg/notify   |     |         |       |        |         |    |
|  | pkg/doctor   |     |    +----+--+ +--+----+ +-+------+  |   |
|  | pkg/agent    |     |    |Session| |Backlog| |Terminal |  |   |
|  | pkg/config   |     |    |  API  | |  API  | |  API    |  |   |
|  | pkg/mcp      |     |    +---+---+ +---+---+ +---+-----+  |   |
|  +--------------+     |        |         |         |         |   |
|                       |        v         v         v         |   |
|  +--------------------+-----------------------------------+  |   |
|  |              internal/ wiring layer                     |  |   |
|  | internal/orchestrator  (Docker Compose lifecycle)       |  |   |
|  | internal/agent/*       (provider adapters)              |  |   |
|  | internal/backlog/*     (GitHub, AzDO adapters)          |  |   |
|  | internal/notifications (Teams, local dispatch)          |  |   |
|  | internal/config/sync   (config sync logic)              |  |   |
|  | internal/config/drift  (managed file drift detection)   |  |   |
|  +---------------------------------------------------------+  |   |
|                       |                                       |   |
|  +--------------------v----+   +---+   +---+   +---+         |   |
|  | Docker bridge network   |   |   |   |   |   |   |         |   |
|  | +---------+ +---------+ |   |   |   |   |   |   |         |   |
|  | |agent-1  | |agent-2  | |   |   |   |   |   |   |         |   |
|  | |cure MCP | |cure MCP | |   |   |   |   |   |   |         |   |
|  | |server   | |server   | |   |   |   |   |   |   |         |   |
|  | +---------+ +---------+ |   +---+   +---+   +---+         |   |
|  +--------------------------+  Repos (git)                    |   |
|                                                               |   |
+---------------------------------------------------------------+   |
         |                                                          |
         v                                                          |
+--------+----------+    +------------------+                       |
| Microsoft Teams   |    | OS Notification   |                      |
| (Bot / Webhook)   |    | (macOS / Linux)   |                      |
+-------------------+    +------------------+                       |
```

---

## Package Map -- New and Extended

### New Public Packages

| Package | Responsibility | Dependencies (pkg/ only) |
|---------|---------------|------------------------|
| `pkg/project` | Project entity: load, save, list, auto-detect from cwd, repo matching | `pkg/config` |
| `pkg/registry` | Source registry: add/remove/update/list overlays, resolve templates and configs by overlay order | `pkg/project` (optional), `pkg/fs` |
| `pkg/vcs` | Git operations: typed wrappers over `git` CLI (status, branch, commit, push, pull, merge, diff, log) | none (shells out) |
| `pkg/notify` | Notification channel interface + dispatcher, event types, routing by session ID | none |
| `pkg/orchestrator` | Container lifecycle: Compose generation, up/down/restart, health monitoring, MCP client | `pkg/mcp`, `pkg/project` |
| `pkg/doctor/stack` | Stack detection + per-stack check suites (Node, Python, Rust, Java); extends `pkg/doctor` | `pkg/doctor` |

### Extended Public Packages

| Package | Changes |
|---------|---------|
| `pkg/agent` | Session model extensions (name, project, branch, linked work items, provider display) |
| `pkg/agent` | `SessionStore` interface: add `Search(ctx, filter)` method |
| `pkg/config` | Add project layer to merge chain (6th layer between global and repo) |
| `pkg/doctor` | Add `CheckSkip` status, structured JSON output option, `--list` support |
| `pkg/mcp` | Add MCP client (currently server-only); needed for host-to-container calls |
| `pkg/template` | Registry-aware overlay resolution (registry sources between embedded and user directories) |

### New Internal Packages

| Package | Responsibility |
|---------|---------------|
| `internal/orchestrator` | Docker Compose generation from devcontainer.json, container lifecycle (up/down/logs), health polling |
| `internal/backlog/github` | GitHub Issues CRUD via `gh` CLI |
| `internal/backlog/azdo` | Azure DevOps work items CRUD via `az boards` CLI |
| `internal/notifications/teams` | Teams Bot Framework or Incoming Webhook adapter |
| `internal/notifications/local` | OS notification adapter (macOS `osascript`, Linux `notify-send`) |
| `internal/config/sync` | Config sync between project, repos, and remote trackers |
| `internal/config/drift` | Managed file drift detection (marker comparison, git-based) |
| `internal/config/managed` | Managed config file generation from registry + project context |
| `internal/commands/project` | `cure project init/list/show` CLI wiring |
| `internal/commands/orchestrate` | `cure orchestrate init/up/down/logs` CLI wiring |
| `internal/commands/backlog` | `cure backlog list/create/view/update` CLI wiring |
| `internal/commands/vcs` | `cure vcs status/branch/commit/push/pull/merge` CLI wiring |
| `internal/commands/registry` | `cure registry add/remove/update/list` CLI wiring |
| `internal/commands/sync` | `cure sync` CLI wiring |
| `internal/gui/ws` | WebSocket hub for terminal sessions and LSP proxy |

### Extended Internal Packages

| Package | Changes |
|---------|---------|
| `internal/gui` | WebSocket upgrade handler, file browser API, terminal session manager |
| `internal/gui/api` | New routes: `/api/files/*`, `/api/terminal/*`, `/api/backlog/*`, `/api/project/*`, `/api/orchestrate/*`, `/api/vcs/*` |
| `internal/commands/gui` | Extended to start WebSocket handlers, register orchestrator, inject project context |
| `internal/commands/doctor` | Extended for `--project`, `--list`, JSON output |
| `internal/agent/claudecode` | Extended to assemble CC CLI invocations from project config + registry (the CC CLI control model) |

---

## Domain Architectures

### E: Project Entity and Config Sync

This is the foundational domain. Everything else depends on having a project abstraction.

#### pkg/project

The Project package defines the entity, persistence, and lookup operations.

```
~/.cure/
  projects/
    my-project/
      project.json       <- Project entity
    another-project/
      project.json
  registry/
    company-defaults/    <- Git clone (see Domain G)
    team-overrides/
  cure.json              <- Global config (existing)
```

**Project entity schema** (see Data Model section for full schema):

```go
// pkg/project/project.go

// Project is the top-level entity grouping repositories, configuration,
// and operational settings for a multi-repo development effort.
type Project struct {
    Name          string           `json:"name"`
    Description   string           `json:"description,omitempty"`
    Repos         []Repo           `json:"repos"`
    Defaults      Defaults         `json:"defaults"`
    Devcontainer  *Devcontainer    `json:"devcontainer,omitempty"`
    Notifications NotificationsCfg `json:"notifications,omitempty"`
    Workflow      *WorkflowCfg     `json:"workflow,omitempty"`
    CreatedAt     time.Time        `json:"created_at"`
    UpdatedAt     time.Time        `json:"updated_at"`
}
```

**Key interfaces:**

```go
// ProjectStore persists and retrieves Project entities.
type ProjectStore interface {
    Save(p *Project) error
    Load(name string) (*Project, error)
    List() ([]*Project, error)
    Delete(name string) error
}

// Detector finds the project associated with a working directory
// by matching the cwd against registered repo paths.
type Detector interface {
    Detect(cwd string) (*Project, error)
}
```

**Workflow enforcement** is a property of the Project, not a separate system:

```go
// WorkflowCfg defines development workflow rules enforced by cure.
type WorkflowCfg struct {
    BranchPattern   string   `json:"branch_pattern,omitempty"`   // regex, e.g. "^(feat|fix|docs)/\\d+-.*$"
    CommitPattern   string   `json:"commit_pattern,omitempty"`   // regex, Conventional Commits
    RequireReview   bool     `json:"require_review,omitempty"`
    ProtectedBranch []string `json:"protected_branches,omitempty"` // e.g. ["main","release/*"]
}
```

Workflow rules are checked by `cure vcs commit`, `cure vcs push`, and the GUI before allowing operations on protected resources.

#### Config Layer Extension

The merge chain extends from 5 to 6 layers:

```
pkg defaults < global ~/.cure.json < project project.json < repo .cure.json < env vars < CLI flags
```

The project layer slots between global and repo. `pkg/config` already supports arbitrary `ConfigObject` merging via `NewConfig(objs...)` — the project layer is simply a new object in the merge list. No interface changes needed in `pkg/config`; the wiring layer (`internal/commands/*`) adds the project object to the merge call.

#### Config Sync Flow

```
                          cure config sync
                                |
                    +-----------+-----------+
                    |                       |
             Read project.json        Read each repo .cure.json
                    |                       |
                    +--------> Merge <------+
                               |
                    Detect conflicts (same key, different values)
                               |
                    +----------+----------+
                    |                     |
              No conflict           Conflict detected
                    |                     |
              Write repo .cure.json  Report + keep repo value
              with project defaults
```

Config sync is non-destructive: it adds project defaults to repos that do not override them. Conflicts are reported, never auto-resolved.

---

### G: AI Config Distribution

The registry and managed config system builds on the existing `pkg/template` overlay model.

#### pkg/registry

```go
// Source is a registered config source (git clone or local directory).
type Source struct {
    Name      string    `json:"name"`
    URL       string    `json:"url"`        // git remote URL (empty for local)
    Path      string    `json:"path"`       // local filesystem path (~/.cure/registry/<name>/)
    UpdatedAt time.Time `json:"updated_at"`
}

// Registry manages AI config sources and resolves artifacts
// from the overlay stack.
type Registry struct { /* ... */ }

// RegistryStore persists the list of registered sources.
type RegistryStore interface {
    Save(s *Source) error
    Load(name string) (*Source, error)
    List() ([]*Source, error)
    Delete(name string) error
}
```

**Resolution order** for any artifact (template, skill definition, agent definition, config fragment):

```
embedded (pkg/template/templates/) < registry sources (in registration order) < project overrides < repo overrides
```

This is the same layering as `pkg/template` today, extended to cover more artifact types.

**Source directory structure** (each registry source follows this layout):

```
<source-root>/
  templates/           <- Template files (same format as cure's embedded templates)
    claude-md.tmpl
    custom-template.tmpl
  skills/              <- Skill definitions (JSON)
    code-review.json
  agents/              <- Agent definitions (JSON, maps to CC --agents flag)
    build-agent.json
    review-agent.json
  configs/             <- Config fragments (merged into .cure.json defaults)
    defaults.json
  mcp/                 <- MCP server configurations
    servers.json
  prompts/             <- System prompt fragments
    base.txt
    security-addendum.txt
```

#### Managed Config Files

`internal/config/managed` generates config files by:

1. Resolving the template from the registry overlay stack
2. Rendering the template with project + repo context as template variables
3. Inserting a managed-file marker (e.g., `<!-- managed by cure: sha256:<hash> -->`)
4. Writing the file via `pkg/fs` atomic write

**Marker format:**

```
<!-- managed by cure: sha256:<content-hash-without-marker> -->
```

The marker is placed at the top of the file (or in a language-appropriate comment format). The SHA-256 hash is computed from the generated content excluding the marker line itself. This enables drift detection without requiring git history.

#### Runtime Assembly

When an agent session starts, `internal/agent` assembles context from:

1. **System prompt:** base prompt (registry) + project instructions (project.json `defaults.system_prompt`) + repo context (.cure.json)
2. **Tools:** registered tools (registry `mcp/servers.json`) + project tools + session-specific tools
3. **Skills:** registered skills (registry `skills/`) + project skills
4. **MCP config:** assembled from registry + project config, written to temp file, passed to CC via `--mcp-config`
5. **Settings:** assembled from registry + project config, written to temp file, passed to CC via `--settings`
6. **Agent definitions:** assembled from registry `agents/`, passed to CC via `--agents`

The assembly is logged at debug level so operators can trace which source contributed which element.

---

### F: Smart Doctor

#### pkg/doctor/stack

Stack detection runs before checks. Each detected stack registers its checks.

```go
// Stack represents a detected technology stack with its health checks.
type Stack struct {
    Name   string             // e.g., "go", "node", "python", "rust", "java"
    Detect func(dir string) bool  // returns true if stack is present in dir
    Checks func() []doctor.CheckFunc // returns checks for this stack
}

// DetectStacks scans a directory and returns all detected stacks.
func DetectStacks(dir string) []Stack
```

Detection heuristics (file presence in root or immediate subdirectories):

| Stack | Trigger files |
|-------|--------------|
| Go | `go.mod`, `go.sum` |
| Node | `package.json` |
| Python | `requirements.txt`, `pyproject.toml`, `setup.py`, `Pipfile` |
| Rust | `Cargo.toml` |
| Java | `pom.xml`, `build.gradle`, `build.gradle.kts` |

Each stack module registers its checks. Go checks are the existing 7 built-in checks. New stacks add their own suites.

#### Project-Scoped Doctor

`cure doctor --project <name>` iterates the project's repo list, runs doctor in each repo, and aggregates results:

```go
// ProjectDoctorResult holds aggregated results across repos.
type ProjectDoctorResult struct {
    Project string
    Repos   []RepoDoctorResult
}

type RepoDoctorResult struct {
    RepoPath string
    Stacks   []string
    Passed   int
    Warned   int
    Failed   int
    Skipped  int
    Results  []doctor.CheckResult
}
```

The `CheckSkip` status is added to `pkg/doctor` for cases where a tool binary is missing (not a failure, but the check cannot run).

---

### C: Project Management

#### pkg/vcs

Typed wrappers over `git` CLI. Every function shells out to `git` and parses output.

```go
// Status returns the working tree status for the given directory.
func Status(dir string) (*StatusResult, error)

// Branch creates and checks out a new branch.
func Branch(dir, name string) error

// Commit creates a commit with the given message. Validates against
// the optional pattern (Conventional Commits).
func Commit(dir, message string, opts ...CommitOption) error

// Push pushes the current branch to the remote.
func Push(dir string, opts ...PushOption) error

// Pull pulls the current branch from the remote.
func Pull(dir string) error

// Diff returns the diff for the given pathspec.
func Diff(dir string, opts ...DiffOption) (*DiffResult, error)

// Log returns commit history.
func Log(dir string, opts ...LogOption) ([]LogEntry, error)
```

The `CommitOption` functional options pattern allows passing `WithValidatePattern(regex)` for workflow enforcement.

#### Backlog Abstraction

```go
// internal/backlog/backlog.go

// Tracker is the abstraction over work item backends.
type Tracker interface {
    List(ctx context.Context, filter Filter) ([]WorkItem, error)
    Get(ctx context.Context, id string) (*WorkItem, error)
    Create(ctx context.Context, item *WorkItem) (*WorkItem, error)
    Update(ctx context.Context, id string, changes *WorkItemUpdate) error
    Close(ctx context.Context, id string, comment string) error
}

// WorkItem is the provider-agnostic work item model.
type WorkItem struct {
    ID          string
    Title       string
    Body        string
    State       string
    Labels      []string
    Assignee    string
    TrackerType string // "github" or "azdo"
    URL         string
}
```

`internal/backlog/github` implements `Tracker` by shelling out to `gh issue list --json ...`, `gh issue create`, etc.

`internal/backlog/azdo` implements `Tracker` by shelling out to `az boards work-item show`, `az boards work-item create`, etc.

The tracker type is determined from `project.json` `defaults.tracker.type`. The `cure backlog` command auto-selects the implementation.

Backlog operations are also exposed as MCP tools for agents:

```go
// internal/agent/tools/backlog.go

// BacklogTools returns agent.Tool implementations backed by a Tracker.
func BacklogTools(t backlog.Tracker) []agent.Tool
```

---

### B: Notifications

#### pkg/notify

```go
// Channel sends notifications and optionally receives responses.
type Channel interface {
    // Name returns the channel identifier (e.g., "teams", "local", "gui").
    Name() string

    // Send delivers a notification. Returns a receipt ID for tracking responses.
    Send(ctx context.Context, n Notification) (string, error)

    // Responses returns a channel of incoming responses for this notification channel.
    // Returns nil if the channel does not support bidirectional messaging.
    Responses() <-chan Response
}

// Notification is a message sent from an agent session to the developer.
type Notification struct {
    SessionID   string
    SessionName string
    ProjectName string
    EventType   EventType  // Completion, Blocker, DecisionNeeded, Error
    Summary     string
    Details     string
}

// Response is a message from the developer back to an agent session.
type Response struct {
    SessionID string
    ChannelID string // which channel the response came from
    Text      string
}

// EventType classifies notification events.
type EventType string

const (
    EventCompletion     EventType = "completion"
    EventBlocker        EventType = "blocker"
    EventDecisionNeeded EventType = "decision_needed"
    EventError          EventType = "error"
)

// Dispatcher routes notifications to all enabled channels and
// multiplexes responses back to sessions.
type Dispatcher struct { /* ... */ }

// NewDispatcher creates a dispatcher with the given channels.
func NewDispatcher(channels ...Channel) *Dispatcher

// Notify sends a notification to all enabled channels.
func (d *Dispatcher) Notify(ctx context.Context, n Notification) error

// WaitResponse blocks until a response arrives for the given session,
// or the context is cancelled. The first response from any channel wins.
func (d *Dispatcher) WaitResponse(ctx context.Context, sessionID string) (Response, error)
```

#### Teams Channel

`internal/notifications/teams` implements `notify.Channel`:

- **Phase 1 (outbound-only):** Uses Incoming Webhook to post messages. Each session maps to a thread via the `replyToId` field. Configuration is a webhook URL in project.json.
- **Phase 2 (bidirectional):** Upgrades to Bot Framework. The bot receives replies via Azure Bot Service webhook, matches them to sessions by thread ID, and routes them through `Responses()`.

The phased approach derisks the Teams integration. Phase 1 is achievable with zero Azure AD configuration.

#### OS Local Channel

`internal/notifications/local` implements `notify.Channel`:

- macOS: `osascript -e 'display notification ...'`
- Linux: `notify-send --app-name=cure ...`
- Clicking a notification opens the cure GUI URL (passed as the notification URL).
- `Responses()` returns nil (OS notifications are unidirectional).

#### GUI Channel

The GUI itself is a notification channel. `internal/gui` implements `notify.Channel`:

- `Send` pushes the notification to SSE-connected browsers.
- `Responses()` receives replies from the chat interface.
- This ensures the GUI is treated identically to Teams in the dispatcher.

---

### D: GUI Evolution

#### Monaco Editor

**Technology:** `monaco-editor` npm package (the VS Code editor extracted for browser use).

**Architecture:**

```
Browser                          Go Backend
+------------------+             +------------------+
| Monaco Editor    |  REST API   | File API         |
| (JS component)   |<---------->| GET  /api/files/* |
|                  |             | PUT  /api/files/* |
|                  |             | POST /api/files   |
|                  |             | DEL  /api/files/* |
+--------+---------+             +------------------+
         |
         | WebSocket /api/editor/lsp
         v
+--------+---------+             +------------------+
| monaco-          |  WebSocket  | LSP Proxy         |
| languageclient   |<---------->| Spawns LSP server |
| (JS library)     |             | per language,     |
+------------------+             | bridges stdio     |
                                 +------------------+
```

The Go backend serves files via REST. For LSP, the Go backend spawns the language server (user-installed, path configured in project.json) and bridges its stdio to the WebSocket. `monaco-languageclient` on the frontend speaks LSP over WebSocket.

If the spike (S-2) determines LSP is too complex for v1.0.0, Monaco ships with syntax highlighting only. The LSP proxy architecture is prepared but not wired.

**File API:**

```go
// internal/gui/api routes (added to existing api router)

// GET  /api/files?path=<dir>              -> directory listing
// GET  /api/files/<path>                  -> file content
// PUT  /api/files/<path>                  -> write file content
// POST /api/files?path=<dir>&name=<name>  -> create file
// DEL  /api/files/<path>                  -> delete file
```

All file operations are scoped to project repos. Path traversal is prevented by validating that resolved paths are within project boundaries.

#### Integrated Terminal

**Technology:** `xterm.js` npm package + `xterm-addon-fit` + `xterm-addon-webgl`.

**Architecture:**

```
Browser                          Go Backend
+------------------+             +------------------+
| xterm.js         |  WebSocket  | Terminal Manager  |
| (JS component)   |<---------->| PTY allocation    |
|                  |  binary     | per session       |
|                  |  frames     | (os/exec + pty)   |
+------------------+             +------------------+
```

The Go backend allocates a PTY per terminal session using `os/exec` with `os.StartProcess` and PTY allocation (via the `github.com/creack/pty` package, approved as an `internal/` dependency). Each terminal session gets a unique WebSocket connection. The terminal manager tracks active sessions and handles resize events.

For container terminals, the PTY is allocated via `docker exec -it <container> /bin/sh` instead of a local shell.

**WebSocket protocol:**

```
Client -> Server:
  { "type": "input",  "data": "<base64-encoded-keystrokes>" }
  { "type": "resize", "cols": 120, "rows": 40 }

Server -> Client:
  { "type": "output", "data": "<base64-encoded-terminal-output>" }
  { "type": "exit",   "code": 0 }
```

#### Diff Viewer

Uses Monaco's built-in diff editor (`monaco.editor.createDiffEditor`). The Go backend provides the diff data via:

```
GET /api/vcs/diff?path=<file>&base=<ref>  -> { original: "...", modified: "..." }
```

#### Config Editor

The config editor is a specialized Monaco instance loading JSON with schema validation. The Go backend provides:

```
GET  /api/config/layers          -> list of config layers with their sources
GET  /api/config/effective       -> merged effective config
GET  /api/config/layer/<name>    -> raw config for a specific layer
PUT  /api/config/layer/<name>    -> update a specific layer
POST /api/config/validate        -> validate a config object against schema
```

---

### A: Multi-Instance Orchestration

This is the most complex domain. It depends on E (project), G (runtime assembly), and existing MCP infrastructure.

#### Architecture Overview

```
cure GUI (host)
    |
    +-> Orchestrator
    |     |
    |     +-> Docker Compose (generated)
    |     |     |
    |     |     +-> agent-1 container (cure MCP server inside)
    |     |     +-> agent-2 container (cure MCP server inside)
    |     |     +-> agent-3 container (cure MCP server inside)
    |     |     +-> agent-4 container (cure MCP server inside)
    |     |
    |     +-> MCP Client (per container)
    |           |
    |           +-> HTTP Streamable over Docker bridge network
    |
    +-> Session Manager
          |
          +-> Session per agent (host session store)
          +-> Events aggregated to GUI via SSE
```

#### Docker Compose Generation

`internal/orchestrator` reads the project's devcontainer.json and generates a `docker-compose.cure.yml`:

```yaml
# Generated by cure — do not edit manually
version: "3.8"
services:
  agent-1:
    build:
      context: .
      dockerfile: .devcontainer/Dockerfile
    volumes:
      - .:/workspace:cached
    working_dir: /workspace
    environment:
      - CURE_AGENT_NAME=agent-1
      - CURE_MCP_PORT=9100
      - CURE_HOST_URL=http://host.docker.internal:${CURE_GUI_PORT}
    ports:
      - "9101:9100"  # MCP server port
    command: ["cure", "mcp", "serve", "--http", ":9100"]
    networks:
      - cure-net

  agent-2:
    # ... same pattern, port 9102:9100

networks:
  cure-net:
    driver: bridge
```

Each container runs `cure mcp serve` on startup, exposing tools for the host to call. The host discovers containers by their mapped ports and establishes MCP client connections.

#### MCP Client

`pkg/mcp` is extended with a client:

```go
// pkg/mcp/client.go

// Client connects to a remote MCP server and invokes tools.
type Client struct { /* ... */ }

// NewClient creates an MCP client for the given HTTP Streamable endpoint.
func NewClient(endpoint string, opts ...ClientOption) *Client

// CallTool invokes a tool on the remote MCP server.
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (string, error)

// ListTools returns the tools available on the remote server.
func (c *Client) ListTools(ctx context.Context) ([]ToolInfo, error)
```

#### Container Lifecycle

```go
// internal/orchestrator/orchestrator.go

// Orchestrator manages the lifecycle of agent containers.
type Orchestrator struct { /* ... */ }

// Init generates docker-compose.cure.yml from project config.
func (o *Orchestrator) Init(project *project.Project) error

// Up starts all agent containers.
func (o *Orchestrator) Up(ctx context.Context) error

// Down stops all agent containers.
func (o *Orchestrator) Down(ctx context.Context) error

// Status returns the health status of all containers.
func (o *Orchestrator) Status(ctx context.Context) ([]ContainerStatus, error)

// Logs streams logs from a specific container.
func (o *Orchestrator) Logs(ctx context.Context, name string, w io.Writer) error

// MCPClient returns the MCP client for a specific container.
func (o *Orchestrator) MCPClient(name string) (*mcp.Client, error)
```

The orchestrator shells out to `docker compose` for lifecycle operations. It monitors container health via `docker compose ps --format json` on a polling interval (default: 5 seconds).

#### Host-Container MCP Authentication

Each orchestration session generates a random shared secret (32 bytes, hex-encoded). The secret is passed to containers via the `CURE_MCP_SECRET` environment variable. The MCP HTTP Streamable transport includes the secret as a `Bearer` token in the `Authorization` header. The host MCP client includes the same token. This prevents unauthorized access to container MCP servers on the Docker bridge network.

---

## Data Model Changes

### Session Model Extension

The `pkg/agent.Session` struct is extended with new fields:

```go
type Session struct {
    // Existing fields (unchanged)
    ID           string           `json:"id"`
    Provider     string           `json:"provider"`
    Model        string           `json:"model"`
    SystemPrompt string           `json:"system_prompt,omitempty"`
    History      []Message        `json:"history"`
    CreatedAt    time.Time        `json:"created_at"`
    UpdatedAt    time.Time        `json:"updated_at"`
    ForkOf       string           `json:"fork_of,omitempty"`
    Tags         []string         `json:"tags,omitempty"`
    SkillName    string           `json:"skill_name,omitempty"`
    Tools        []Tool           `json:"-"`

    // New fields for v1.0.0
    Name         string           `json:"name,omitempty"`          // human-readable name
    ProjectName  string           `json:"project_name,omitempty"`  // associated project
    BranchName   string           `json:"branch_name,omitempty"`   // git branch at session start
    RepoName     string           `json:"repo_name,omitempty"`     // repository name/path
    GitDirty     bool             `json:"git_dirty,omitempty"`     // working tree status at start
    WorkItems    []string         `json:"work_items,omitempty"`    // linked issue/ticket IDs
    AgentRole    string           `json:"agent_role,omitempty"`    // e.g., "build", "review", "test"
    ContainerID  string           `json:"container_id,omitempty"`  // Docker container ID if orchestrated
}
```

All new fields are optional with `omitempty`. Existing session files remain compatible (backward-compatible JSON deserialization). The `Name` field defaults to a generated name (e.g., `<provider>-<first-4-chars-of-id>`) if not set explicitly.

### SessionStore Extension

```go
// SessionFilter defines criteria for searching sessions.
type SessionFilter struct {
    ProjectName string
    Provider    string
    BranchName  string
    HasWorkItem string   // filter by linked work item ID
    SkillName   string
    Limit       int
}

// SessionStore is extended with Search.
type SessionStore interface {
    Save(ctx context.Context, s *Session) error
    Load(ctx context.Context, id string) (*Session, error)
    List(ctx context.Context) ([]*Session, error)
    Delete(ctx context.Context, id string) error
    Fork(ctx context.Context, id string) (*Session, error)

    // Search returns sessions matching the filter criteria.
    Search(ctx context.Context, filter SessionFilter) ([]*Session, error)
}
```

### Project Entity Schema (Full)

```json
{
  "name": "my-project",
  "description": "A multi-repo Go project",
  "repos": [
    {
      "path": "/home/user/src/api",
      "remote": "git@github.com:org/api.git",
      "default_branch": "main"
    }
  ],
  "defaults": {
    "provider": "claude",
    "model": "claude-sonnet-4-6",
    "system_prompt": "You are a senior Go developer...",
    "tracker": {
      "type": "github",
      "owner": "org",
      "repo": "api",
      "project_number": 9,
      "project_id": "PVT_kwDOBxaH0c4BPROP"
    },
    "max_agents": 4,
    "max_turns": 32,
    "max_budget_usd": 5.0
  },
  "devcontainer": {
    "image": "mcr.microsoft.com/devcontainers/go:1.25",
    "features": ["ghcr.io/devcontainers/features/node:1"],
    "dockerfile": ".devcontainer/Dockerfile"
  },
  "notifications": {
    "teams": {
      "webhook_url": "https://...",
      "bot_app_id": "",
      "bot_app_secret": "",
      "bidirectional": false
    },
    "local": {
      "enabled": true,
      "events": ["completion", "blocker", "decision_needed", "error"]
    }
  },
  "workflow": {
    "branch_pattern": "^(feat|fix|docs|refactor|test|chore)/\\d+-.*$",
    "commit_pattern": "^(feat|fix|docs|test|refactor|chore)(\\(.+\\))?!?: .+",
    "require_review": true,
    "protected_branches": ["main", "release/*"]
  },
  "registry": {
    "sources": ["company-defaults", "team-overrides"]
  },
  "ai_config": {
    "managed_files": [
      "claude-md",
      "claude-settings",
      "mcp-json",
      "cursor-rules",
      "copilot-instructions",
      "agents-md"
    ],
    "sync_triggers": ["init", "doctor", "session_start"],
    "watch": false
  },
  "created_at": "2026-04-09T12:00:00Z",
  "updated_at": "2026-04-09T12:00:00Z"
}
```

### Registry Source Schema

```json
{
  "sources": [
    {
      "name": "company-defaults",
      "url": "git@github.com:org/cure-config-defaults.git",
      "path": "/home/user/.cure/registry/company-defaults",
      "updated_at": "2026-04-09T12:00:00Z"
    }
  ]
}
```

Stored at `~/.cure/registry/registry.json`.

---

## Communication Architecture

### Protocol Summary

| Communication Path | Protocol | Transport | Use Case |
|-------------------|----------|-----------|----------|
| Browser <-> GUI Backend (pages) | HTTP | TCP | SPA serving, REST APIs |
| Browser <-> GUI Backend (streaming) | SSE | HTTP | Agent event streaming (existing) |
| Browser <-> GUI Backend (terminal) | WebSocket | TCP | PTY I/O (binary frames) |
| Browser <-> GUI Backend (LSP) | WebSocket | TCP | Language server protocol proxy |
| Host <-> Container (tools) | MCP/HTTP Streamable | TCP (Docker bridge) | Tool invocation, result collection |
| Host <-> Teams (outbound) | HTTPS | TCP | Webhook POST to Teams channel |
| Teams <-> Host (inbound) | HTTPS | TCP | Bot Framework callback to host |
| Host <-> OS (notification) | subprocess | stdio | osascript / notify-send |
| Host <-> git | subprocess | stdio | All VCS operations |
| Host <-> gh/az CLIs | subprocess | stdio | Backlog CRUD |
| Host <-> docker compose | subprocess | stdio | Container lifecycle |

### WebSocket Hub

The GUI backend maintains a WebSocket hub for multiplexing terminal and LSP connections:

```go
// internal/gui/ws/hub.go

// Hub manages active WebSocket connections grouped by session type.
type Hub struct {
    terminals map[string]*TerminalSession // keyed by session ID
    lsp       map[string]*LSPSession      // keyed by language + workspace
}

// TerminalSession bridges a WebSocket to a PTY.
type TerminalSession struct {
    ID        string
    PTY       *os.File  // PTY master side
    Cmd       *exec.Cmd
    Container string    // empty for local, container name for remote
}
```

### SSE Extension

The existing SSE streaming for agent events is extended with notification events:

```
event: notification
data: {"session_id":"abc","event_type":"completion","summary":"Build succeeded"}

event: orchestration
data: {"container":"agent-1","status":"running","health":"healthy"}
```

---

## Interface Definitions

### Core Interfaces Summary

| Interface | Package | Methods | Used By |
|-----------|---------|---------|---------|
| `ProjectStore` | `pkg/project` | `Save, Load, List, Delete` | CLI, GUI, orchestrator |
| `Detector` | `pkg/project` | `Detect(cwd)` | CLI (auto-detect project from cwd) |
| `RegistryStore` | `pkg/registry` | `Save, Load, List, Delete` | CLI, sync, runtime assembly |
| `Channel` | `pkg/notify` | `Name, Send, Responses` | Teams, local, GUI channels |
| `Dispatcher` | `pkg/notify` | `Notify, WaitResponse` | Orchestrator, agent loop |
| `Tracker` | `internal/backlog` | `List, Get, Create, Update, Close` | CLI, GUI, agent tools |
| `SessionStore` (extended) | `pkg/agent` | `Save, Load, List, Delete, Fork, Search` | CLI, GUI |
| `Client` | `pkg/mcp` | `CallTool, ListTools` | Host-to-container communication |
| `Orchestrator` | `internal/orchestrator` | `Init, Up, Down, Status, Logs, MCPClient` | CLI, GUI |

### API Routes Summary (New)

| Method | Route | Domain | Description |
|--------|-------|--------|-------------|
| GET | `/api/project` | E | List projects |
| GET | `/api/project/:name` | E | Get project details |
| POST | `/api/project` | E | Create project |
| PUT | `/api/project/:name` | E | Update project |
| GET | `/api/files` | D | Directory listing |
| GET | `/api/files/*path` | D | Read file |
| PUT | `/api/files/*path` | D | Write file |
| POST | `/api/files` | D | Create file |
| DELETE | `/api/files/*path` | D | Delete file |
| GET | `/api/backlog` | C | List work items |
| GET | `/api/backlog/:id` | C | Get work item |
| POST | `/api/backlog` | C | Create work item |
| PUT | `/api/backlog/:id` | C | Update work item |
| GET | `/api/vcs/status` | C | Git status |
| GET | `/api/vcs/diff` | C | Git diff |
| POST | `/api/vcs/commit` | C | Git commit |
| POST | `/api/vcs/push` | C | Git push |
| POST | `/api/vcs/pull` | C | Git pull |
| POST | `/api/vcs/branch` | C | Create branch |
| WS | `/api/terminal/:id` | D | Terminal WebSocket |
| WS | `/api/editor/lsp` | D | LSP WebSocket |
| GET | `/api/orchestrate/status` | A | Container status |
| POST | `/api/orchestrate/up` | A | Start containers |
| POST | `/api/orchestrate/down` | A | Stop containers |
| GET | `/api/orchestrate/logs/:name` | A | Stream container logs |
| GET | `/api/config/layers` | D | Config layers |
| GET | `/api/config/effective` | D | Effective merged config |
| PUT | `/api/config/layer/:name` | D | Update config layer |
| GET | `/api/registry` | G | List registry sources |
| POST | `/api/registry` | G | Add registry source |
| DELETE | `/api/registry/:name` | G | Remove registry source |
| POST | `/api/sync` | G | Trigger config sync |
| GET | `/api/sync/status` | G | Drift detection results |

---

## Dependency Map

### Domain Dependencies

```
                    E-1 (Project Entity) [FOUNDATION]
                    /    |    |    |    \
                   /     |    |    |     \
                  v      v    v    v      v
               G-1    C-2  C-3  F-2    A-1
            (Registry)(GH)(AzDO)(Proj  (Orch)
               /|\              Doctor)  |
              / | \                     / \
             v  v  v                   v   v
           G-2 G-3 G-4              A-2   A-3
          (Mgd)(Drift)(Asm)      (MCP)  (Compose)

    B-3 (Bidirectional)          D-2 (Diff)
     /        \                    |
    v          v                   v
  B-1        B-2               D-1 (Monaco)
 (Teams)   (Local)

    Independent (can start immediately):
    - F-1 (Multi-Stack Doctor)
    - C-1 (Git Operations / pkg/vcs)
    - D-1 (Monaco Editor)
    - D-3 (Terminal Emulator)
    - B-2 (OS Notifications)
```

### Package Import Graph (new packages only)

```
pkg/project   -> pkg/config, pkg/fs
pkg/registry  -> pkg/fs
pkg/vcs       -> (none, shells out to git)
pkg/notify    -> (none)
pkg/orchestrator -> pkg/mcp, pkg/project
pkg/doctor/stack -> pkg/doctor

internal/orchestrator     -> pkg/orchestrator, pkg/project, pkg/mcp
internal/backlog/github   -> (shells out to gh)
internal/backlog/azdo     -> (shells out to az)
internal/notifications/*  -> pkg/notify
internal/config/sync      -> pkg/project, pkg/config, pkg/fs
internal/config/drift     -> pkg/project, pkg/registry, pkg/fs
internal/config/managed   -> pkg/registry, pkg/template, pkg/project, pkg/fs
internal/gui/ws           -> (os/exec, pty)
internal/commands/*       -> (wiring, imports from pkg/ and internal/)
```

---

## ADR Log

### ADR-010: Shell out to CLI tools instead of embedding SDKs

**Status:** Accepted

**Context:** v1.0.0 integrates with git, Docker, GitHub (gh), Azure DevOps (az), and OS notification tools. Two approaches exist: (a) use Go SDKs (go-git, Docker SDK, go-github, azure-devops-go-api), or (b) shell out to CLI tools with structured output parsing.

**Decision:** Shell out to CLI tools for all external integrations in `pkg/` packages. Use Go SDKs only in `internal/` packages where the CLI approach is insufficient (specifically: `github.com/creack/pty` for PTY allocation in terminal emulator).

**Rationale:**
- go-git alone is ~40K LOC and incomplete (no rebase, limited worktree support)
- Docker SDK pulls in massive transitive dependencies
- gh and az CLIs handle authentication, pagination, and rate limiting already
- CLI tools are developer prerequisites anyway (cure doctor validates their presence)
- Structured output (JSON flags: `gh issue list --json`, `docker compose ps --format json`) provides typed data without an SDK
- Keeps `pkg/` stdlib-only (except for `pkg/mcp` which already depends on the Anthropic SDK indirectly via `internal/`)

**Consequences:**
- Requires CLI tools installed on PATH (doctor checks validate this)
- Subprocess spawning adds ~10-50ms latency per operation (acceptable for CLI/GUI workflows)
- Parsing structured output requires maintenance when CLI output formats change (mitigated by pinning to stable output flags like `--json`)
- `github.com/creack/pty` is the sole new dependency approved for `internal/gui/ws`

---

### ADR-011: Project entity at ~/.cure/projects/ with filesystem-based store

**Status:** Accepted

**Context:** The Project entity needs persistence. Options: (a) SQLite database, (b) filesystem JSON files, (c) ~/.config/cure/ XDG-compliant directory.

**Decision:** Filesystem JSON files at `~/.cure/projects/<name>/project.json`. One directory per project. The `~/.cure/` directory is the root for all cure user data (already used for global config and registry).

**Rationale:**
- Consistent with existing `pkg/agent/store` pattern (JSON files in a directory)
- Human-readable and manually editable
- No database dependency
- `~/.cure/` is already the cure data root; XDG compliance would split data across `~/.config/cure/`, `~/.local/share/cure/`, etc., adding complexity for no user benefit
- One directory per project allows future per-project state files (logs, cache) alongside project.json

**Consequences:**
- Project names must be filesystem-safe (lowercase alphanumeric + hyphens)
- No built-in querying beyond file listing (acceptable given expected project count: 1-20)
- Backup is simple: copy `~/.cure/`

---

### ADR-012: MCP HTTP Streamable as host-container transport

**Status:** Proposed (pending spike S-1 validation)

**Context:** Three transport options exist for host-container MCP communication: (a) HTTP Streamable over Docker bridge network, (b) stdio via `docker exec`, (c) Unix socket via volume mount.

**Decision (proposed):** HTTP Streamable over Docker bridge network, with shared-secret Bearer token authentication.

**Rationale:**
- HTTP Streamable is already implemented in `pkg/mcp` server-side
- Docker bridge networking provides reliable TCP connectivity between containers
- Each container exposes port 9100 (mapped to unique host ports for debugging)
- Authentication via Bearer token prevents unauthorized access on the shared network
- stdio via `docker exec` creates a new process per request (expensive, no connection reuse)
- Unix socket requires volume mount coordination and has macOS Docker Desktop reliability issues

**Consequences:**
- Requires MCP client implementation in `pkg/mcp` (currently server-only)
- Docker bridge network means all containers can potentially reach each other (mitigated by auth)
- Must handle container startup race (MCP server not ready when host connects) — solved by retry with backoff

**Risk:** If spike S-1 reveals reliability issues under concurrent load, fall back to stdio via docker exec as a proven-reliable alternative.

---

### ADR-013: Monaco + xterm.js for GUI editor and terminal

**Status:** Accepted

**Context:** The GUI needs a code editor and terminal emulator. Options evaluated: (a) Monaco + xterm.js, (b) CodeMirror 6 + custom terminal, (c) Ace editor + xterm.js.

**Decision:** Monaco editor for code editing and file viewing (including diff). xterm.js for terminal emulation. Both are loaded via npm and bundled with the SvelteKit frontend.

**Rationale:**
- Monaco is the VS Code editor engine — developers are already familiar with it
- Monaco includes a built-in diff editor (`createDiffEditor`)
- `monaco-languageclient` provides LSP client integration for Monaco
- xterm.js is the de-facto browser terminal (used by VS Code, Theia, Hyper, JupyterLab)
- Both libraries are MIT-licensed and actively maintained
- CodeMirror 6 is excellent for lightweight editing but lacks Monaco's LSP ecosystem
- Ace editor is stable but has less active development than Monaco

**Consequences:**
- Monaco adds ~2-4 MB to the frontend bundle (mitigated by dynamic import, loaded only when editor is opened)
- xterm.js adds ~200 KB (with WebGL addon for performance)
- SSR must be avoided for both libraries (SvelteKit `onMount` + dynamic import pattern)
- Monaco requires a Web Worker for syntax highlighting (worker bundling configuration in Vite)

---

### ADR-014: Teams Incoming Webhook first, Bot Framework later

**Status:** Accepted

**Context:** Teams notification can use (a) Incoming Webhook (simple, outbound-only), or (b) Bot Framework (complex, bidirectional). The requirements specify bidirectional messaging (B-3).

**Decision:** Ship outbound-only via Incoming Webhook first (Phase 1), then upgrade to Bot Framework for bidirectional messaging (Phase 2) within the v1.0.0 release cycle.

**Rationale:**
- Incoming Webhook requires zero Azure AD configuration (just a URL)
- Bot Framework requires Azure AD app registration, bot channel configuration, and a publicly accessible callback URL
- Phase 1 delivers immediate value (agent-to-developer notifications) with minimal setup
- Phase 2 adds reply capability after the architecture is proven
- The `notify.Channel` interface supports both — `Responses()` returns nil for Phase 1

**Consequences:**
- Phase 1: developers receive notifications but cannot reply via Teams (must use GUI)
- Phase 2: requires Azure AD documentation and potentially a tunneling solution (ngrok or Azure Bot Service) for the callback URL
- The channel abstraction allows this phased approach without changing the dispatcher

---

### ADR-015: Notification dispatcher with channel abstraction

**Status:** Accepted

**Context:** Notifications must be sent to multiple channels (Teams, OS, GUI) and responses must be routed back to the correct session. Options: (a) direct integration per channel, (b) pub/sub message bus, (c) dispatcher with channel interface.

**Decision:** Dispatcher pattern with a `Channel` interface. The dispatcher sends notifications to all enabled channels and multiplexes responses.

**Rationale:**
- Simple pattern that handles the v1.0.0 use case (3 channels, single developer)
- Adding a new channel requires only implementing the `Channel` interface
- No external message bus dependency
- First-response-wins semantics for bidirectional messaging (prevents duplicate responses)
- The dispatcher is a regular Go struct, not a service — it lives in the GUI server process

**Consequences:**
- Single-process only (no distributed pub/sub)
- Adding channels post-v1.0.0 (Slack, email) requires only interface implementation
- Response deduplication is simple (first response wins, reject subsequent)

---

### ADR-016: CC CLI as primary agent runtime in orchestrated mode

**Status:** Accepted

**Context:** Orchestrated agents in containers need a runtime. Options: (a) cure's built-in agent loop (internal/agent/claude), (b) Claude Code CLI invoked by cure with assembled flags, (c) custom agent framework.

**Decision:** Use Claude Code CLI as the primary agent runtime. Cure assembles CC CLI invocations from project config + registry, passing all context via CC's command-line flags.

**Rationale:**
- CC CLI already handles: tool execution, permission management, context management, subagent spawning, worktree isolation, session persistence
- Cure's role is configuration assembly and orchestration, not agent runtime
- CC CLI flags provide complete programmatic control: `--settings`, `--mcp-config`, `--system-prompt`, `--tools`, `--agents`, `--permission-mode`, `--max-turns`, `--max-budget-usd`
- The `--output-format stream-json --input-format stream-json` flags enable full programmatic I/O
- Cure's existing `internal/agent/claudecode` adapter already handles CC CLI invocation
- For non-Claude providers (OpenAI, Gemini), cure's built-in agent loop remains available

**Consequences:**
- Claude Code CLI must be installed in orchestrated containers
- CC CLI version changes may require cure adapter updates
- The adapter must assemble all flags correctly from project config
- Cure does not need to implement tool execution, permission management, or context management for CC-backed sessions
- For orchestrated sessions, cure provides tools via MCP server (container runs `cure mcp serve`), and CC CLI connects to them via `--mcp-config`

---

## Incremental Delivery Strategy

The delivery is structured as minor releases, each delivering a cohesive vertical slice. Each release is independently valuable and shippable.

### Phase 0: Spikes (before any implementation)

| Spike | Time Box | Question | Blocks |
|-------|----------|----------|--------|
| S-1: MCP Transport | 3 days | HTTP Streamable reliability under 4-agent concurrent load on Docker bridge | A-2 |
| S-2: Monaco + LSP in SvelteKit | 3 days | Can monaco-languageclient work in SvelteKit 5 with Go backend LSP proxy? | D-1 (LSP part) |
| S-3: Teams Bot bidirectional | 2 days | Can a Teams bot relay replies to a non-Azure-hosted Go backend? | B-1 Phase 2 |
| S-4: xterm.js + Go PTY over WebSocket | 2 days | Latency and encoding correctness for PTY-over-WebSocket | D-3 |

Spikes produce: working PoC or "defer to post-v1.0.0" decision + architecture notes.

### v0.12.0 -- Foundation

**Theme:** Project entity, config layer extension, session model enrichment.

| Story | Domain | Size |
|-------|--------|------|
| E-1: Project entity (pkg/project, cure project init/list/show) | E | L |
| Session model extensions (name, project, branch, work items, provider) | Agent | M |
| SessionStore.Search method | Agent | S |
| Config merge chain extension (project layer) | E | S |
| Workflow enforcement in project config | E | M |

**Value delivered:** Developers can define projects, sessions show rich context in GUI, project auto-detection works.

### v0.13.0 -- AI Config Distribution

**Theme:** Registry, managed configs, runtime assembly.

| Story | Domain | Size |
|-------|--------|------|
| G-1: Source registry (pkg/registry, cure registry add/remove/update/list) | G | L |
| G-2: Managed config files (cure sync, managed-file markers) | G | L |
| G-4: Runtime assembly of agent context from registry + project | G | L |
| G-3: Drift detection (cure sync --check, doctor integration) | G | M |

**Value delivered:** AI configs are centrally managed, synced to repos, drift is detected. Agent sessions get automatically assembled context.

### v0.14.0 -- Project Management and Smart Doctor

**Theme:** VCS, backlog, multi-stack doctor.

| Story | Domain | Size |
|-------|--------|------|
| C-1: Git operations (pkg/vcs, cure vcs) | C | M |
| C-2: GitHub backlog (cure backlog, MCP tools) | C | L |
| C-3: Azure DevOps backlog (cure backlog) | C | L |
| F-1: Multi-stack doctor | F | M |
| F-2: Project-scoped doctor | F | S |
| C-4: Project skeleton creation | C | M |

**Value delivered:** VCS and backlog management from within cure. Doctor understands all major stacks.

### v0.15.0 -- GUI Evolution

**Theme:** Monaco editor, terminal emulator, diff viewer, config editor.

| Story | Domain | Size |
|-------|--------|------|
| D-1: Monaco editor (syntax highlighting + file browser) | D | L |
| D-1 (continued): LSP integration (conditional on spike S-2) | D | XL |
| D-3: Integrated terminal (xterm.js + WebSocket + PTY) | D | L |
| D-2: Diff viewer (Monaco diff editor + VCS API) | D | M |
| D-4: Config editor | D | M |

**Value delivered:** GUI becomes a usable development environment with editor, terminal, and diff review.

### v0.16.0 -- Notifications

**Theme:** Teams integration, OS notifications, bidirectional messaging.

| Story | Domain | Size |
|-------|--------|------|
| B-2: OS local notifications | B | S |
| B-1 Phase 1: Teams outbound webhook | B | M |
| B-1 Phase 2: Teams Bot Framework (conditional on spike S-3) | B | L |
| B-3: Bidirectional messaging architecture | B | L |

**Value delivered:** Agents notify developers via Teams and OS notifications. Developers can respond through Teams or GUI.

### v0.17.0 -- Multi-Instance Orchestration

**Theme:** Container orchestration, host-container MCP, Docker Compose.

| Story | Domain | Size |
|-------|--------|------|
| A-1: Multi-instance orchestration (internal/orchestrator, Docker Compose generation) | A | XL |
| A-2: Host-container MCP protocol (pkg/mcp client, auth) | A | L |
| A-3: Docker Compose lifecycle management (up/down/restart/logs) | A | L |

**Value delivered:** 2-4 agents run simultaneously in containers, coordinated by the host.

### v1.0.0 -- Stabilization

**Theme:** Integration testing, documentation, performance, polish.

- Cross-domain integration tests
- Performance testing under multi-agent load
- Documentation: architecture guide, user guide, API reference
- GUI polish: responsive design, error states, loading states
- Security audit (MCP auth, file API path traversal, WebSocket security)
- Release candidate cycle (rc.1, rc.2, ...)

### Delivery Sequence Rationale

1. **Foundation first (v0.12.0):** Project entity is the root dependency for 10+ stories. Shipping it first unblocks all other domains.
2. **AI config second (v0.13.0):** The registry and managed configs are the second most-depended-upon capability. Runtime assembly is critical for orchestration.
3. **VCS + backlog + doctor third (v0.14.0):** These are high-value, moderate-risk features that can be developed in parallel after foundation.
4. **GUI fourth (v0.15.0):** GUI evolution has the highest frontend complexity but is independent of backend domains once file and VCS APIs exist.
5. **Notifications fifth (v0.16.0):** Notifications require project config (for channel configuration) and benefit from the GUI channel being mature.
6. **Orchestration last (v0.17.0):** This is the highest-complexity domain, depending on project, registry, MCP client, and container infrastructure. By building it last, all dependencies are stable.

---

## Risk Mitigation Through Architecture

| Risk | Score | Architectural Mitigation |
|------|-------|-------------------------|
| Monaco + LSP complexity (9) | LSP is isolated behind the WebSocket proxy. If spike S-2 fails, Monaco ships with syntax highlighting only. The `WS /api/editor/lsp` route is simply not wired. Zero impact on other features. |
| Teams bidirectional (6) | Phase 1 (webhook) and Phase 2 (bot) are separate channel implementations behind the same `notify.Channel` interface. Phase 1 ships regardless. Phase 2 is additive. |
| Docker reliability (6) | Orchestration is the last domain (v0.17.0). All other features work without Docker. Doctor checks for Docker availability. The orchestrator gracefully degrades. |
| MCP transport (6) | Spike S-1 validates before implementation. The MCP client interface is transport-agnostic — switching from HTTP Streamable to stdio-via-docker-exec requires only changing the connection factory, not the caller code. |
| Scope size (6) | 6 minor releases with clear scope per release. Each release is independently shippable. Quality gates per release (tests pass, review complete, E2E coverage). |
| SDK dependencies (4) | Shelling to CLI tools eliminates SDK dependencies for git, docker, gh, az. Only `github.com/creack/pty` is added to `internal/`. |

---

## Appendix: CC CLI Integration Model

Cure assembles Claude Code CLI invocations from project config + registry. This section documents how project.json fields map to CC CLI flags.

### Flag Assembly

| project.json field | CC CLI flag | Notes |
|-------------------|-------------|-------|
| `defaults.model` | `--model` | Model selection |
| `defaults.max_turns` | `--max-turns` | Turn limit |
| `defaults.max_budget_usd` | `--max-budget-usd` | Budget cap |
| `defaults.system_prompt` | `--system-prompt-file` (temp file) | Assembled from registry + project + repo |
| Registry `mcp/servers.json` | `--mcp-config` (temp file) | Assembled from all sources |
| Registry `agents/` | `--agents` (JSON) | Agent definitions from registry |
| `workflow.*` | `--append-system-prompt` | Workflow rules injected as system prompt addendum |
| Session tools | `--tools` | Tool list from session config |
| Permission mode | `--permission-mode` | From project config or default |

### Invocation Pattern

```go
// internal/agent/claudecode — extended for v1.0.0

func (a *claudeCodeAdapter) buildArgs(session *agent.Session, project *project.Project) []string {
    args := []string{
        "-p",
        "--output-format", "stream-json",
        "--input-format", "stream-json",
        "--model", a.model,
    }

    // System prompt: assembled from registry + project + repo
    if promptFile := a.assembleSystemPrompt(session, project); promptFile != "" {
        args = append(args, "--system-prompt-file", promptFile)
    }

    // MCP config: assembled from registry + project
    if mcpFile := a.assembleMCPConfig(project); mcpFile != "" {
        args = append(args, "--mcp-config", mcpFile, "--strict-mcp-config")
    }

    // Agent definitions: from registry
    if agentDefs := a.assembleAgentDefs(project); agentDefs != "" {
        args = append(args, "--agents", agentDefs)
    }

    // Resource limits
    if project != nil && project.Defaults.MaxTurns > 0 {
        args = append(args, "--max-turns", strconv.Itoa(project.Defaults.MaxTurns))
    }
    if project != nil && project.Defaults.MaxBudgetUSD > 0 {
        args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", project.Defaults.MaxBudgetUSD))
    }

    // Session management
    if session.ID != "" {
        args = append(args, "--session-id", session.ID)
    }

    return args
}
```

### Container Agent Invocation

In orchestrated mode, each container runs cure with CC CLI inside. The host:

1. Generates project-specific config files (system prompt, MCP config, settings) via runtime assembly
2. Mounts them into the container via Docker volume
3. The container's entrypoint runs `cure mcp serve` to expose host-callable tools
4. The container's agent run invokes CC CLI with the mounted config files
5. Agent events flow back to the host via the MCP connection

This model means cure does not need to implement agent orchestration logic — CC CLI handles tool loops, context management, and autonomous operation. Cure handles configuration, lifecycle, and coordination.

---

*End of architecture document.*
