# Cure v1.0.0 — Full Platform Vision Requirements

**Date:** 2026-04-09
**Author:** Business Analyst
**Decision context:** Define the complete scope, user stories, and boundaries for cure v1.0.0, transforming it from a CLI tool into an AI-first development platform.
**Status:** Draft for stakeholder review

---

## Table of Contents

- [Problem Statement](#problem-statement)
- [Current State Baseline (v0.11.3)](#current-state-baseline-v0113)
- [Scope Boundaries](#scope-boundaries)
- [Domain A: Multi-Instance Orchestration](#domain-a-multi-instance-orchestration)
- [Domain B: Agent-Human Communication (Notifications)](#domain-b-agent-human-communication-notifications)
- [Domain C: Project Management](#domain-c-project-management)
- [Domain D: GUI Evolution](#domain-d-gui-evolution)
- [Domain E: Multi-Level Config Sync](#domain-e-multi-level-config-sync)
- [Domain F: Smart Doctor](#domain-f-smart-doctor)
- [Domain G: AI Config Distribution](#domain-g-ai-config-distribution)
- [Cross-Domain Dependencies](#cross-domain-dependencies)
- [Risk Assessment](#risk-assessment)
- [Open Questions and Assumptions](#open-questions-and-assumptions)
- [Plugin Capability Gap Analysis](#plugin-capability-gap-analysis)

---

## Problem Statement

### The Gap

Cure v0.11.3 is a capable single-agent CLI tool: it can generate templates, run AI sessions against four providers, serve MCP tools, and present a basic GUI. However, it operates as a **single-agent, single-repo, single-config** tool. Modern AI-assisted development demands something fundamentally different:

1. **Single-agent bottleneck.** Developers can only run one agent session at a time. Real-world AI workflows benefit from 2-4 agents working in parallel on different aspects of a problem (build, test, review, document), each in an isolated environment. Cure cannot orchestrate multiple concurrent agents.

2. **No communication channel.** Agents work silently. When an agent completes a task, encounters a blocker, or needs a decision, the developer must actively check. There is no notification system, no ability for the agent to reach the developer outside the terminal, and no way for the developer to respond without context-switching to the CLI.

3. **No project abstraction.** Cure operates on a single repository. Multi-repo projects, monorepo-within-org structures, and cross-repo agent coordination are not supported. Configuration lives per-repo with no higher-level orchestration.

4. **Thin GUI.** The current GUI shows sessions, chat, doctor results, and config. It cannot edit files, run terminals, or serve as a primary development interface. Developers must keep their IDE open alongside cure.

5. **No config distribution.** AI tooling configuration (CLAUDE.md, .cursor/rules, MCP configs, system prompts) is manually maintained per-repo. There is no registry, no sync, no drift detection, and no runtime assembly of agent context from project-level definitions.

6. **Single-stack doctor.** The doctor checks Go toolchain requirements only. Projects using Node, Python, Rust, or Java get no benefit from doctor's diagnostic capabilities.

7. **No backlog integration.** Developers context-switch to GitHub, Azure DevOps, or Jira to manage issues. Agents cannot read or write work items natively, breaking the "agent as team member" model.

### The Opportunity

v1.0.0 transforms cure from a CLI tool into an **AI development platform** where:

- Multiple agents work autonomously in isolated containers, coordinated by a single host cure instance
- Agents communicate with the developer through Microsoft Teams and OS notifications, like team members in a chat channel
- A project abstraction above repositories enables multi-repo coordination with synced configuration
- The GUI evolves toward a self-contained development environment with Monaco editor, terminal, and diff viewer
- AI tooling configuration is managed centrally and distributed to repos/agents automatically, with drift detection
- Doctor understands multiple tech stacks and runs project-wide diagnostics
- Backlog management (GitHub Issues, Azure DevOps) is native, enabling agents to create, update, and close work items

---

## Current State Baseline (v0.11.3)

### Commands

| Command | Subcommands | Notes |
|---------|-------------|-------|
| `cure context` | `new`, `resume`, `list`, `fork`, `delete`, REPL | 4 providers (Claude, Claude Code, OpenAI, Gemini), tool loops, skills |
| `cure generate` | `claude-md`, `agents-md`, `copilot-instructions`, `cursor-rules`, `windsurf-rules`, `gemini-md`, `k8s-job`, `scaffold`, `devcontainer`, `editorconfig`, `gitignore`, `github-workflow` | 4-tier template overlay, `--dry-run`, `--force` |
| `cure doctor` | (none) | 7 built-in Go checks + custom checks from config |
| `cure init` | (none) | Interactive project bootstrap wizard |
| `cure gui` | (none) | SvelteKit 5 SPA, Go HTTP server, SSE streaming |
| `cure mcp serve` | (none) | Stdio + HTTP Streamable transports |
| `cure trace` | `http`, `tcp`, `udp`, `dns` | Network diagnostics |
| `cure completion` | (introspective) | Shell auto-completion |

### Public Packages (`pkg/`)

| Package | Responsibility |
|---------|---------------|
| `pkg/terminal` | CLI framework (Command, Router, Context, runners) |
| `pkg/agent` | Provider-agnostic AI abstractions (Agent, Session, Tool, Skill, Event) |
| `pkg/agent/store` | JSON session persistence |
| `pkg/mcp` | MCP server (stdio + HTTP Streamable), FuncTool with schema validation |
| `pkg/config` | 5-layer hierarchical config merging |
| `pkg/template` | Embedded template engine with 4-tier overlay |
| `pkg/doctor` | Health check framework |
| `pkg/prompt` | Interactive terminal prompts |
| `pkg/fs` | Atomic filesystem operations |
| `pkg/style` | ANSI terminal styling |
| `pkg/env` | Cached runtime environment detection |
| `pkg/tracer` | HTTP/TCP/UDP tracing |

### Internal Packages

| Package | Responsibility |
|---------|---------------|
| `internal/agent/claude` | Anthropic API adapter with tool loop |
| `internal/agent/claudecode` | Claude Code CLI adapter |
| `internal/agent/claudestream` | Claude CLI text streaming adapter |
| `internal/agent/openai` | OpenAI API adapter with tool loop |
| `internal/agent/gemini` | Gemini API adapter with tool loop |
| `internal/agent/tools` | MCP-to-agent tool bridge |
| `internal/gui` | Go HTTP server + SPA embedding + API routes |
| `internal/commands/*` | CLI command implementations |

### External Dependencies

- `github.com/anthropics/anthropic-sdk-go` (sole non-stdlib dep in current codebase)
- Indirect: `tidwall/{gjson,match,pretty,sjson}`, `golang.org/x/sync`

### Open Work

| Issue | Status | Description |
|-------|--------|-------------|
| #175 | In progress | GUI PTY streaming + markdown rendering |
| #104 | Open (epic) | GUI stabilization |
| #19 | Open (parked) | Plugin spike (deferred to v1.0.0) |

---

## Scope Boundaries

### In Scope for v1.0.0

| Domain | Capability | Decision Reference |
|--------|-----------|-------------------|
| A | Multi-instance orchestration: 2-4 concurrent agents in devcontainers | Locked |
| A | Host-container MCP protocol | Locked |
| A | Docker Compose generation AND lifecycle management | Locked |
| B | Microsoft Teams bot with thread-per-session semantics | Locked |
| B | OS local notifications (macOS/Linux) | Locked |
| B | Full bidirectional messaging (agent notifies, user responds) | Locked |
| C | VCS operations (git) from within cure | Locked |
| C | Backlog management: GitHub Issues + Azure DevOps (full CRUD) | Locked |
| C | Project skeleton creation from cure | Locked |
| D | Monaco editor with LSP integration | Locked |
| D | File viewer with diff support | Locked |
| D | Integrated terminal emulator | Locked |
| D | Config editor for cure's own configuration | Locked |
| E | Project entity abstraction above repositories | Locked |
| E | Project config at `~/.cure/projects/<name>/project.json` | Locked |
| E | Config sync: local user <-> project <-> remote tracker | Locked |
| E | `cure project init` interactive command | Locked |
| F | Multi-stack doctor (Node, Python, Rust, Java, Go) | Locked |
| F | Project-scoped doctor (across all repos in a project) | Locked |
| G | AI config control plane with source registry (bundled + external git repos) | Locked |
| G | Managed configs: CLAUDE.md, settings.json, .mcp.json, .cursor/rules, copilot-instructions, agents.md, LSP, skills, agent defs | Locked |
| G | Injection triggers: explicit sync, implicit on init/doctor/session, file watcher | Locked |
| G | Drift detection via managed-file markers + git-based detection | Locked |
| G | Runtime assembly of system prompt, tools, and context from registry + project config | Locked |

### Out of Scope (Deferred to Post-v1.0.0)

| Item | Rationale |
|------|-----------|
| Slack integration | Locked: deferred to next version after v1.0.0 |
| Jira integration | Locked: deferred to next version after v1.0.0 (same timeline as Slack) |
| Lockfile-based drift detection | Deferred: managed-file markers + git-based detection are sufficient for v1.0.0 |
| More than 4 concurrent agents | 2-4 is the design target; scaling beyond requires operational validation |
| Mobile notifications (iOS/Android push) | Channels are Teams + OS notifications for v1.0.0 |
| Remote agent execution (cloud-hosted containers) | v1.0.0 is local Docker only |
| Plugin system (Go plugins or RPC) | #19 spike is parked; evaluation may inform v1.0.0 architecture but a full plugin API is not a deliverable |

---

## Domain A: Multi-Instance Orchestration

### Context

Today cure runs a single agent session. The user described a model where the host cure instance coordinates 2-4 agent instances, each running in its own devcontainer. The host uses the GUI server as the coordination point (no separate daemon). Communication between host and container cure instances uses MCP protocol.

### User Stories

#### A-1: Launch Multiple Agent Instances

**Type:** User Story
**Priority:** Critical
**Component:** internal/orchestrator, internal/gui

**Description**

As a developer,
I want to launch 2-4 agent sessions that run simultaneously in isolated devcontainer instances,
so that I can parallelize work across build, test, review, and documentation tasks.

**Acceptance Criteria**

- [ ] **Given** a project with a devcontainer definition, **when** the user runs `cure orchestrate start --agents 3`, **then** 3 Docker containers are created, each with its own cure instance running inside
- [ ] **Given** 4 running agent containers, **when** each agent starts a session, **then** all 4 sessions stream events concurrently to the host GUI without interference
- [ ] **Given** an agent container, **when** it is created, **then** it has the project repository mounted (read-write on its own branch or worktree) and all required tools installed per the devcontainer spec
- [ ] **Given** a running orchestration, **when** the user views the GUI dashboard, **then** all active agent sessions are visible with their current status (idle, running, blocked, completed)
- [ ] The host cure instance does not require a separate daemon process; the GUI server acts as the coordinator
- [ ] Maximum concurrent agents is configurable (default: 4, minimum: 1)
- [ ] Each container is isolated: filesystem, network namespace, and environment variables do not leak between agents

**Edge Cases and Error Handling**

- If Docker is not available, `cure orchestrate start` exits with a clear error message naming the missing dependency
- If a container fails to start, the remaining containers still launch; the failed container is reported with its error
- If the host cure process is terminated (SIGINT/SIGTERM), all managed containers are stopped gracefully (SIGTERM, then SIGKILL after timeout)

**Out of Scope**

- Remote container execution (cloud VMs, Kubernetes pods)
- Agent-to-agent direct communication (agents communicate through the host only)

---

#### A-2: Host-Container MCP Protocol

**Type:** User Story
**Priority:** Critical
**Component:** pkg/mcp, internal/orchestrator

**Description**

As the host cure instance,
I want to communicate with each container's cure instance via MCP protocol,
so that I can dispatch tasks, collect results, and relay events using a standardized interface.

**Acceptance Criteria**

- [ ] **Given** a running container with a cure instance inside, **when** the host sends an MCP `tools/call` request, **then** the container executes the tool and returns the result over the same transport
- [ ] **Given** the host and container are on the same Docker network, **when** the MCP connection is established, **then** the transport used is the most reliable available (evaluate: HTTP Streamable over Docker network, stdio over `docker exec`, Unix socket via volume mount)
- [ ] **Given** a container cure instance, **when** it starts, **then** it automatically starts its MCP server on the agreed transport and the host can discover and connect to it
- [ ] **Given** a network interruption between host and container, **when** the connection drops, **then** the host detects the failure within 10 seconds and reports the container as disconnected
- [ ] The protocol is symmetric: the host can call tools on the container, and the container can call tools on the host (for session store access, notification relay, etc.)
- [ ] All MCP messages between host and container are authenticated (at minimum, a shared secret per session)

**Edge Cases and Error Handling**

- If the container MCP server is not ready when the host attempts connection, the host retries with exponential backoff (max 30 seconds)
- If a tool call exceeds a configurable timeout (default: 5 minutes), the host receives a timeout error and can decide whether to retry or fail

**Out of Scope**

- Cross-machine MCP (host and container on different hosts)

**Dependencies**

- Requires A-1 (container lifecycle must exist before protocol can be tested)

---

#### A-3: Docker Compose Generation and Lifecycle Management

**Type:** User Story
**Priority:** High
**Component:** internal/orchestrator, internal/commands/orchestrate

**Description**

As a developer,
I want cure to generate a Docker Compose stack definition from my devcontainer config and manage its lifecycle (up/down/restart/logs),
so that I do not need to manually write or manage Docker Compose files for agent orchestration.

**Acceptance Criteria**

- [ ] **Given** a project with a `.devcontainer/devcontainer.json`, **when** the user runs `cure orchestrate init`, **then** cure generates a `docker-compose.cure.yml` with one service per agent slot (configurable count)
- [ ] **Given** a generated compose file, **when** the user runs `cure orchestrate up`, **then** `docker compose up -d` is invoked and all services start
- [ ] **Given** running services, **when** the user runs `cure orchestrate down`, **then** `docker compose down` is invoked and all containers stop
- [ ] **Given** running services, **when** the user runs `cure orchestrate logs [agent-name]`, **then** the logs for that agent's container are streamed to stdout
- [ ] **Given** a running agent container that crashes, **when** cure detects the exit, **then** it reports the failure and optionally restarts the container (configurable restart policy)
- [ ] The compose file references the same Dockerfile and features as the project's devcontainer.json
- [ ] The compose file is deterministic: running `cure orchestrate init` twice with the same inputs produces the same output

**Edge Cases and Error Handling**

- If no `devcontainer.json` exists, `cure orchestrate init` prompts the user to create one first (or run `cure generate devcontainer`)
- If Docker Compose v2 is not installed, exit with error naming the dependency

**Out of Scope**

- Kubernetes deployment (containers run via Docker Compose only)
- Custom compose overrides beyond what devcontainer.json specifies

---

## Domain B: Agent-Human Communication (Notifications)

### Context

Agents should be treated as team members who can notify the developer and receive responses. The primary channel for v1.0.0 is Microsoft Teams, with OS local notifications as a secondary channel. Each agent session maps to a Teams thread.

### User Stories

#### B-1: Microsoft Teams Bot Integration

**Type:** User Story
**Priority:** High
**Component:** internal/notifications/teams

**Description**

As a developer,
I want my agent sessions to send messages to a Microsoft Teams channel as a bot,
so that I can receive notifications about agent activity (completions, blockers, decisions needed) in my normal communication tool.

**Acceptance Criteria**

- [ ] **Given** a configured Teams bot connection (webhook URL or Bot Framework registration), **when** an agent session completes a task, **then** a message is posted to the configured Teams channel
- [ ] **Given** an agent session, **when** it starts, **then** a new thread is created in the Teams channel with the session name and context (project, branch, task description)
- [ ] **Given** an active agent thread in Teams, **when** the agent encounters a blocker or needs a decision, **then** a message is posted to that thread with clear options or a request for input
- [ ] **Given** a Teams thread for an agent session, **when** the user replies in that thread, **then** the response is relayed back to the agent session as a user message
- [ ] **Given** multiple concurrent agent sessions, **when** each posts messages, **then** each session uses its own thread (no cross-contamination)
- [ ] Bot configuration is stored in project config (`~/.cure/projects/<name>/project.json`) under a `notifications.teams` key
- [ ] The Teams integration can be disabled per-project or globally

**Edge Cases and Error Handling**

- If the Teams webhook/bot is unreachable, the notification is queued locally and retried (max 3 attempts, exponential backoff); the agent does not block on notification delivery
- If the user's Teams reply cannot be parsed as a valid response, the agent re-prompts with clarification
- If the bot token expires, cure logs a clear error and continues without Teams notifications (falls back to OS notifications only)

**Out of Scope**

- Teams admin setup (tenant registration, bot permissions) — cure documents requirements but does not automate Azure AD setup
- Rich adaptive cards beyond text + action buttons (post-v1.0.0)

---

#### B-2: OS Local Notifications

**Type:** User Story
**Priority:** Medium
**Component:** internal/notifications/local

**Description**

As a developer,
I want to receive OS-level notifications (macOS Notification Center, Linux desktop notifications) when an agent needs attention,
so that I am alerted even when Teams is not open.

**Acceptance Criteria**

- [ ] **Given** macOS, **when** an agent session completes or encounters a blocker, **then** a native notification appears in Notification Center with the session name and a summary
- [ ] **Given** Linux with a notification daemon (libnotify/notify-send), **when** an agent event occurs, **then** a desktop notification is displayed
- [ ] **Given** an OS notification, **when** the user clicks it, **then** the cure GUI opens (or focuses) on the relevant session
- [ ] OS notifications can be configured: enabled/disabled, event types to notify on (completion, blocker, decision-needed, error)
- [ ] Notifications include: session name, event type, one-line summary

**Edge Cases and Error Handling**

- If no notification daemon is available (headless Linux, SSH session), notifications are silently skipped and a warning is logged once
- On macOS, if the user has denied notification permissions, cure logs a warning and continues

**Out of Scope**

- Windows notifications (evaluate for post-v1.0.0)
- Sound customization

---

#### B-3: Bidirectional Messaging Architecture

**Type:** User Story
**Priority:** High
**Component:** internal/notifications

**Description**

As a developer,
I want to send responses back to agents through the same channel they notified me on (Teams reply or GUI),
so that I can make decisions and unblock agents without switching to the CLI.

**Acceptance Criteria**

- [ ] **Given** an agent waiting for user input, **when** the user responds via Teams thread reply, **then** the response is injected into the agent's session as a user message and processing resumes
- [ ] **Given** an agent waiting for user input, **when** the user responds via the GUI chat interface, **then** the response is processed identically to a Teams response
- [ ] **Given** a response received from any channel, **when** it is processed, **then** the other channels are updated to reflect that the decision was made (e.g., Teams thread shows "Resolved via GUI")
- [ ] The messaging architecture supports adding new channels (Slack, email) without modifying the core notification dispatcher
- [ ] Response routing uses session ID to match incoming messages to the correct agent session

**Edge Cases and Error Handling**

- If the same user responds on two channels simultaneously, the first response received wins; the second is acknowledged with "Already responded"
- If a response arrives for a session that has already timed out or been cancelled, it is logged and the user is notified

**Out of Scope**

- Multi-user collaboration (single developer per project for v1.0.0)
- Message encryption at rest

---

## Domain C: Project Management

### Context

Cure should minimize context-switching by bringing VCS operations and backlog management into the platform. Agents are first-class participants: they do most implementation, developers review and enhance.

### User Stories

#### C-1: Git Operations from Cure

**Type:** User Story
**Priority:** High
**Component:** internal/commands/vcs, pkg/vcs (new)

**Description**

As a developer,
I want to perform common git operations (branch, commit, push, pull, merge, status, diff, log) from within cure,
so that I do not need to switch to a separate terminal for version control.

**Acceptance Criteria**

- [ ] **Given** a git repository, **when** the user runs `cure vcs status`, **then** the current branch, staged changes, unstaged changes, and untracked files are displayed
- [ ] **Given** a git repository, **when** the user runs `cure vcs branch <name>`, **then** a new branch is created and checked out
- [ ] **Given** staged changes, **when** the user runs `cure vcs commit -m "<message>"`, **then** a commit is created with the given message (Conventional Commits format validated)
- [ ] **Given** a branch with commits, **when** the user runs `cure vcs push`, **then** the branch is pushed to the configured remote
- [ ] **Given** a remote with new commits, **when** the user runs `cure vcs pull`, **then** the local branch is updated
- [ ] **Given** a merge conflict, **when** the user runs `cure vcs merge <branch>`, **then** conflicts are reported with file paths and the user is prompted to resolve
- [ ] All VCS operations are also available through the GUI (Domain D)
- [ ] All VCS operations work within orchestrated agent containers (Domain A)

**Edge Cases and Error Handling**

- If the current directory is not a git repository, exit with a clear error
- If `git` binary is not available, exit with error naming the dependency
- Protected branch push (main/master) shows a warning and requires `--force` confirmation

**Out of Scope**

- Git hosting provider operations (PR creation, review) — these are covered by backlog management (C-2)
- Rebase workflows (post-v1.0.0)

---

#### C-2: Backlog Management — GitHub Issues

**Type:** User Story
**Priority:** High
**Component:** internal/commands/backlog, internal/backlog/github

**Description**

As a developer or agent,
I want to create, read, update, and close GitHub Issues from within cure,
so that work tracking is part of the development workflow without requiring a browser.

**Acceptance Criteria**

- [ ] **Given** a project with a GitHub remote, **when** the user runs `cure backlog list`, **then** open issues are displayed with number, title, labels, assignee, and status
- [ ] **Given** a project, **when** the user runs `cure backlog create --title "..." --body "..." --label "..."`, **then** a new issue is created on GitHub and added to the project board
- [ ] **Given** an issue number, **when** the user runs `cure backlog view <number>`, **then** the full issue body, comments, labels, and linked PRs are displayed
- [ ] **Given** an issue number, **when** the user runs `cure backlog update <number> --state closed --comment "Resolved in PR #X"`, **then** the issue is closed with the comment
- [ ] **Given** an agent session with backlog tools registered, **when** the agent decides to create an issue (e.g., bug found during implementation), **then** it can invoke the backlog tool to create it without human intervention
- [ ] **Given** the project board configuration in project.json, **when** an issue is created or updated, **then** the Projects v2 board status is updated accordingly
- [ ] Backlog operations are available via CLI, GUI, and as MCP tools for agents

**Edge Cases and Error Handling**

- If the `gh` CLI is not installed, exit with error naming the dependency
- If authentication fails, display a clear message about `gh auth login`
- Rate limiting from GitHub API is handled with backoff and retry

**Out of Scope**

- Pull request creation and review (use `cure vcs` + `gh` directly)
- GitHub Actions management

**Dependencies**

- Requires E (Project entity) for project board configuration

---

#### C-3: Backlog Management — Azure DevOps Work Items

**Type:** User Story
**Priority:** Medium
**Component:** internal/commands/backlog, internal/backlog/azdo

**Description**

As a developer or agent,
I want to create, read, update, and close Azure DevOps work items from within cure,
so that teams using Azure DevOps have the same backlog integration as GitHub users.

**Acceptance Criteria**

- [ ] **Given** a project configured with Azure DevOps, **when** the user runs `cure backlog list`, **then** open work items are displayed with ID, title, type, state, and assigned-to
- [ ] **Given** a project, **when** the user runs `cure backlog create --type "User Story" --title "..."`, **then** a new work item is created in Azure DevOps
- [ ] **Given** a work item ID, **when** the user runs `cure backlog view <id>`, **then** the full details, discussion, and relations are displayed
- [ ] **Given** a work item ID, **when** the user runs `cure backlog update <id> --state "Resolved"`, **then** the state is updated with an audit comment
- [ ] The backlog command auto-detects the tracker type from project.json configuration
- [ ] Azure DevOps operations require `az boards` CLI and valid authentication

**Edge Cases and Error Handling**

- If `az` CLI is not installed or `azure-devops` extension is missing, exit with error
- WIQL query failures are reported with the query text for debugging

**Out of Scope**

- Azure DevOps Pipelines management
- Board layout customization (columns, swimlanes)

**Dependencies**

- Requires E (Project entity) for tracker configuration

---

#### C-4: Project Skeleton Creation

**Type:** User Story
**Priority:** Medium
**Component:** internal/commands/project

**Description**

As a developer,
I want to create a new project skeleton with all required configuration from a single cure command,
so that new projects start with consistent structure, AI configs, CI/CD, and devcontainer setup.

**Acceptance Criteria**

- [ ] **Given** the user runs `cure project create <name>`, **when** prompted, **then** they can select: language/stack, git hosting (GitHub/Azure DevOps), AI providers, notification channels, and devcontainer features
- [ ] **Given** selections made, **when** the wizard completes, **then** a new directory is created with: git init, CLAUDE.md, devcontainer.json, editorconfig, gitignore, CI workflow, and project.json
- [ ] **Given** a created project, **when** `cure doctor` is run, **then** all checks pass
- [ ] The skeleton uses the same template system as `cure generate` (embedded + overlay)
- [ ] The project entity is registered at `~/.cure/projects/<name>/project.json`

**Edge Cases and Error Handling**

- If the target directory already exists, prompt for confirmation before overwriting
- If required tools are missing (git, docker), warn but allow creation of partial skeleton

**Out of Scope**

- Remote repository creation (use `gh repo create` or `az repos create` separately)

---

## Domain D: GUI Evolution

### Context

The GUI is moving from a dashboard/chat tool toward a primary development interface. This is incremental — v1.0.0 establishes the boundaries with Monaco editor, terminal, file viewer, and config editor.

### User Stories

#### D-1: Monaco Editor with LSP Integration

**Type:** User Story
**Priority:** High
**Component:** frontend (SvelteKit), internal/gui/api

**Description**

As a developer,
I want to edit project files in the cure GUI using Monaco editor (the VS Code engine) with language server protocol support,
so that I can review and modify code without leaving the cure interface.

**Acceptance Criteria**

- [ ] **Given** the GUI file browser, **when** the user opens a file, **then** it loads in a Monaco editor instance with syntax highlighting for the detected language
- [ ] **Given** a file open in Monaco, **when** the user makes edits and saves (Ctrl+S / Cmd+S), **then** the file is written to disk via the API
- [ ] **Given** a supported language (Go, TypeScript, Python, Rust, Java), **when** a file is opened, **then** LSP features are available: autocomplete, go-to-definition, hover documentation, diagnostics
- [ ] **Given** the Monaco editor, **when** multiple files are open, **then** a tab bar allows switching between them
- [ ] **Given** an agent editing a file in a container, **when** the change is committed, **then** the GUI file viewer reflects the updated content on refresh
- [ ] The editor supports light and dark themes, matching the GUI's theme preference
- [ ] File operations (create, rename, delete) are available from the file browser

**Edge Cases and Error Handling**

- If a file is binary (images, compiled artifacts), display a placeholder instead of loading in Monaco
- If LSP is not available for a language, syntax highlighting still works (Monaco built-in grammars); LSP features degrade gracefully
- If a file is modified externally while open in Monaco, the user is notified and can reload or keep their version
- Maximum file size for editor loading is configurable (default: 5 MB); larger files show a warning

**Out of Scope**

- Multi-cursor collaborative editing
- Extension marketplace (VS Code extensions are not compatible with Monaco standalone)

---

#### D-2: File Viewer with Diff Support

**Type:** User Story
**Priority:** Medium
**Component:** frontend (SvelteKit), internal/gui/api

**Description**

As a developer,
I want to view file contents and git diffs in the GUI,
so that I can review agent-generated changes before committing.

**Acceptance Criteria**

- [ ] **Given** a file path, **when** the user opens it in the viewer, **then** the file content is displayed with syntax highlighting
- [ ] **Given** a file with uncommitted changes, **when** the user selects "Show diff", **then** a side-by-side or unified diff view is displayed using Monaco's diff editor
- [ ] **Given** two commits or branches, **when** the user selects "Compare", **then** all changed files are listed and each can be viewed in diff mode
- [ ] **Given** the diff view, **when** viewing agent-generated changes, **then** the user can approve (stage) or discard individual file changes

**Edge Cases and Error Handling**

- If the file does not exist on one side of the diff (new file or deleted file), show it as added/removed
- Large diffs (>10,000 lines) show a warning and offer to load incrementally

**Out of Scope**

- Three-way merge editor
- Inline commenting on diffs

---

#### D-3: Integrated Terminal Emulator

**Type:** User Story
**Priority:** High
**Component:** frontend (SvelteKit), internal/gui/api

**Description**

As a developer,
I want to open a terminal emulator within the cure GUI,
so that I can run commands without leaving the interface.

**Acceptance Criteria**

- [ ] **Given** the GUI, **when** the user opens a terminal pane, **then** a PTY-backed terminal emulator appears with the user's default shell
- [ ] **Given** a terminal session, **when** the user types commands, **then** they execute in the project's working directory with the user's environment
- [ ] **Given** the terminal, **when** output includes ANSI escape codes (colors, cursor movement), **then** they are rendered correctly
- [ ] **Given** multiple terminal sessions, **when** the user opens additional terminals, **then** a tab bar allows switching between them
- [ ] **Given** an orchestrated agent container, **when** the user opens a terminal for that container, **then** the terminal session is inside the container (via `docker exec`)
- [ ] The terminal supports copy-paste, scrollback buffer (configurable size), and search

**Edge Cases and Error Handling**

- If the shell process exits, the terminal tab shows "Session ended" and offers to restart
- If the WebSocket connection to the backend drops, the terminal reconnects automatically

**Out of Scope**

- Terminal multiplexer features (split panes like tmux)
- Serial port / SSH connections through the terminal

---

#### D-4: Config Editor

**Type:** User Story
**Priority:** Medium
**Component:** frontend (SvelteKit), internal/gui/api

**Description**

As a developer,
I want to edit cure's configuration files (global, project, local) through the GUI with validation and preview,
so that I can adjust settings without manually editing JSON files.

**Acceptance Criteria**

- [ ] **Given** the GUI settings page, **when** the user navigates to config, **then** all configuration layers are displayed (global, project, local) with their current values
- [ ] **Given** a config key, **when** the user edits its value, **then** the change is validated against the expected type/schema and saved to the appropriate config file
- [ ] **Given** the config editor, **when** the user modifies a setting, **then** a preview shows the effective (merged) configuration before saving
- [ ] **Given** project.json, **when** the user edits notification channels, AI providers, or repo lists, **then** the changes take effect on the next relevant operation (no restart required)
- [ ] The config editor highlights which layer a value originates from (inherited vs overridden)

**Edge Cases and Error Handling**

- If a config file has syntax errors, the editor highlights them and prevents saving until fixed
- If a config value is invalid (wrong type, out of range), validation errors are shown inline

**Out of Scope**

- Schema-driven form generation (config editor uses JSON editing with validation, not custom forms)

---

## Domain E: Multi-Level Config Sync

### Context

The project entity is an abstraction ABOVE repositories. A project can be a monorepo or span multiple repos. Project config lives at `~/.cure/projects/<name>/project.json` (user-level, outside repos). Config syncs between local user config, project repo config, and remote trackers (GitHub/Azure DevOps).

### User Stories

#### E-1: Project Entity and Config Structure

**Type:** User Story
**Priority:** Critical
**Component:** pkg/project (new), internal/commands/project

**Description**

As a developer,
I want to define a project that groups one or more repositories with shared configuration,
so that cure can operate across repos with consistent settings.

**Acceptance Criteria**

- [ ] **Given** the user runs `cure project init`, **when** prompted interactively, **then** they provide: project name, description, list of repos (local paths + remote URLs), default AI provider, default tracker (GitHub/Azure DevOps), devcontainer stack definition, and notification channels
- [ ] **Given** a completed wizard, **when** the project is created, **then** a `~/.cure/projects/<name>/project.json` file is written with all configured fields
- [ ] **Given** a project.json, **when** the user runs any cure command from within a project repo, **then** cure auto-detects the project by matching the current directory against registered repo paths
- [ ] **Given** multiple projects, **when** the user runs `cure project list`, **then** all registered projects are displayed with their name, repo count, and status
- [ ] **Given** a project name, **when** the user runs `cure project show <name>`, **then** the full configuration is displayed

**Minimum project.json fields:**

```json
{
  "name": "my-project",
  "description": "A multi-repo Go project",
  "repos": [
    { "path": "/home/user/src/api", "remote": "git@github.com:org/api.git" },
    { "path": "/home/user/src/web", "remote": "git@github.com:org/web.git" }
  ],
  "defaults": {
    "provider": "claude",
    "tracker": {
      "type": "github",
      "owner": "org",
      "project_number": 9
    }
  },
  "devcontainer": {
    "image": "mcr.microsoft.com/devcontainers/go:1.25",
    "features": ["ghcr.io/devcontainers/features/node:1"]
  },
  "notifications": {
    "teams": { "webhook_url": "https://..." },
    "local": { "enabled": true }
  }
}
```

**Edge Cases and Error Handling**

- If a repo path does not exist, warn during init but allow creation (repo may be cloned later)
- If two projects register the same repo path, warn about the conflict
- Project names must be unique, lowercase, alphanumeric with hyphens

**Out of Scope**

- Project templates (pre-defined project.json for common setups)
- Cloud-synced project config

---

#### E-2: Config Sync Between Layers

**Type:** User Story
**Priority:** High
**Component:** pkg/config (extended), internal/config/sync

**Description**

As a developer,
I want cure to synchronize configuration between my local settings, project-level config, and remote tracker settings,
so that all team members and agents operate with consistent configuration.

**Acceptance Criteria**

- [ ] **Given** a project.json and a repo's `.cure.json`, **when** cure loads config, **then** the merge order is: pkg defaults < global `~/.cure.json` < project `project.json` < repo `.cure.json` < env vars < CLI flags
- [ ] **Given** a change to project.json, **when** `cure config sync` is run, **then** each repo's `.cure.json` is updated with the project-level defaults (without overriding repo-specific overrides)
- [ ] **Given** a remote tracker (GitHub/Azure DevOps), **when** project metadata is read, **then** cure can pull project board settings, labels, and field IDs into project.json
- [ ] **Given** a config key modified in `.cure.json`, **when** it conflicts with project.json, **then** the repo-level value wins (most specific wins)
- [ ] Config sync is non-destructive: it never deletes values, only adds or updates

**Edge Cases and Error Handling**

- If a repo `.cure.json` does not exist, sync creates it with project defaults
- If sync encounters a conflict (same key, different values across repos), it reports the conflict and keeps the existing repo value

**Out of Scope**

- Real-time sync (config sync is explicit via `cure config sync` or triggered by specific operations)
- Conflict resolution UI

---

## Domain F: Smart Doctor

### Context

Doctor currently runs 7 Go-specific checks. v1.0.0 extends it to detect and check multiple tech stacks, and to run across all repos in a project.

### User Stories

#### F-1: Multi-Stack Detection and Checks

**Type:** User Story
**Priority:** High
**Component:** pkg/doctor (extended), internal/commands/doctor

**Description**

As a developer working on a multi-language project,
I want cure doctor to detect the tech stacks in use (Go, Node, Python, Rust, Java) and run appropriate checks for each,
so that I get health diagnostics regardless of the primary language.

**Acceptance Criteria**

- [ ] **Given** a repository with `package.json`, **when** `cure doctor` is run, **then** Node.js checks are executed: node version, npm/yarn/pnpm presence, `node_modules` status, lockfile consistency
- [ ] **Given** a repository with `requirements.txt` or `pyproject.toml`, **when** `cure doctor` is run, **then** Python checks are executed: python version, venv presence, dependency installation status
- [ ] **Given** a repository with `Cargo.toml`, **when** `cure doctor` is run, **then** Rust checks are executed: rustc version, cargo presence, clippy availability
- [ ] **Given** a repository with `pom.xml` or `build.gradle`, **when** `cure doctor` is run, **then** Java checks are executed: JDK version, Maven/Gradle presence
- [ ] **Given** a repository with multiple stacks (e.g., Go backend + Node frontend), **when** `cure doctor` is run, **then** checks for all detected stacks are executed
- [ ] **Given** a `doctor` section in `.cure.json`, **when** custom checks override default checks, **then** the custom configuration takes precedence
- [ ] Stack detection is based on file presence in the repository root and common subdirectory patterns (e.g., `frontend/package.json`)
- [ ] Each stack's checks are documented in `cure doctor --list`

**Edge Cases and Error Handling**

- If a tool binary is not found, the check reports "SKIP" with a message, not "FAIL"
- If stack detection finds ambiguous signals (e.g., both `package.json` and `Cargo.toml` in root), both stacks are checked independently

**Out of Scope**

- Auto-installation of missing tools
- Stack-specific build or test execution

---

#### F-2: Project-Scoped Doctor

**Type:** User Story
**Priority:** Medium
**Component:** internal/commands/doctor, pkg/project

**Description**

As a developer managing a multi-repo project,
I want to run doctor across all repositories in a project,
so that I get a unified health report for the entire project.

**Acceptance Criteria**

- [ ] **Given** a project with 3 repos, **when** the user runs `cure doctor --project <name>`, **then** doctor runs in each repo and produces a combined report
- [ ] **Given** a project-scoped doctor run, **when** results are displayed, **then** each repo's results are grouped under the repo name with pass/fail/skip counts
- [ ] **Given** a project-scoped doctor run, **when** any repo has a failing check, **then** the overall exit code is 1
- [ ] **Given** a project-scoped doctor run, **when** run from the GUI, **then** results are displayed per-repo in an expandable tree view

**Edge Cases and Error Handling**

- If a repo path in project.json does not exist, skip it with a warning
- If a repo is not a git repository, run doctor for file-based checks only

**Out of Scope**

- Cross-repo dependency validation (e.g., "repo A depends on repo B's package")

**Dependencies**

- Requires E-1 (project entity must exist)

---

## Domain G: AI Config Distribution

### Context

Cure becomes the control plane for AI tooling configuration. It manages a registry of config sources (bundled defaults + external git repos as overlays, like Homebrew taps), detects drift, and dynamically assembles agent context at runtime.

### User Stories

#### G-1: Source Registry for AI Configs

**Type:** User Story
**Priority:** Critical
**Component:** pkg/registry (new), internal/commands/registry

**Description**

As a developer,
I want to register external git repositories as AI config sources (overlays) alongside cure's bundled defaults,
so that I can share and reuse AI tooling configuration across projects and teams.

**Acceptance Criteria**

- [ ] **Given** the user runs `cure registry add <name> <git-url>`, **when** the repository is cloned, **then** its templates, skills, agent definitions, and config fragments are available as an overlay
- [ ] **Given** multiple registered sources, **when** cure resolves a config template (e.g., CLAUDE.md template), **then** the resolution order is: bundled defaults < registered overlays (in registration order) < project-level overrides
- [ ] **Given** a registered source, **when** the user runs `cure registry update <name>`, **then** the local clone is pulled to the latest version
- [ ] **Given** a registered source, **when** the user runs `cure registry remove <name>`, **then** the overlay is removed and configs revert to the next-lower layer
- [ ] **Given** the user runs `cure registry list`, **then** all registered sources are displayed with name, URL, last-updated date, and item counts (templates, skills, configs)
- [ ] Registered sources are stored at `~/.cure/registry/<name>/` (git clones)
- [ ] The registry supports the same directory structure as cure's embedded templates (so existing template overlays work as registry sources)

**Edge Cases and Error Handling**

- If a git clone fails (auth, network), the error is reported and the source is not registered
- If two sources provide the same template name, the last-registered source wins (with a warning)
- If a registered source has a malformed structure, warn on registration but do not block other sources

**Out of Scope**

- Publishing to a central registry (sources are git repos, not a package index)
- Source signing or verification (post-v1.0.0)

---

#### G-2: Managed Config Files

**Type:** User Story
**Priority:** Critical
**Component:** internal/config/managed, pkg/registry

**Description**

As a developer,
I want cure to manage AI tooling configuration files in my repositories,
so that CLAUDE.md, .cursor/rules, .mcp.json, and other AI configs are maintained consistently.

**Acceptance Criteria**

- [ ] **Given** a project, **when** the user runs `cure sync`, **then** the following files are generated or updated from the registry + project config:
  - `CLAUDE.md`
  - `.claude/settings.json`
  - `.mcp.json`
  - `.cursor/rules/*.mdc`
  - `.github/copilot-instructions.md`
  - `agents.md`
  - LSP configuration files
  - Skill definitions
  - Agent definitions
- [ ] **Given** a managed file, **when** cure generates it, **then** a marker comment or metadata field is inserted (e.g., `<!-- managed by cure -->`) to identify it as managed
- [ ] **Given** a managed file with local modifications, **when** `cure sync` is run, **then** the user is warned about the drift and prompted to accept the update or keep local changes
- [ ] **Given** a managed file, **when** `cure sync --force` is run, **then** the file is overwritten regardless of local modifications
- [ ] Config file generation uses the template engine with project config values as template context (project name, repos, providers, etc.)

**Edge Cases and Error Handling**

- If a config file type is not relevant to the project (e.g., no Cursor usage), it is skipped
- If the template for a managed file does not exist in any registry source, the file is skipped with a warning

**Out of Scope**

- Auto-committing generated config changes (the developer decides when to commit)
- Partial file management (cure manages the entire file, not sections)

---

#### G-3: Drift Detection

**Type:** User Story
**Priority:** High
**Component:** internal/config/drift

**Description**

As a developer,
I want cure to detect when managed config files have been modified outside of cure,
so that I can identify and resolve configuration drift before it causes inconsistencies.

**Acceptance Criteria**

- [ ] **Given** a managed file with a cure marker, **when** `cure doctor` or `cure sync --check` is run, **then** cure compares the current file content against the expected generated content
- [ ] **Given** drift is detected, **when** reported, **then** the output shows which files have drifted, the nature of the change (added, modified, deleted), and a diff preview
- [ ] **Given** drift in a managed file, **when** the user runs `cure sync`, **then** they are prompted per-file: apply cure's version, keep local version, or show diff
- [ ] Drift detection uses git-based comparison: cure compares the current working tree against the last `cure sync` commit for managed files
- [ ] Managed-file markers survive common editing operations (not easily deleted by formatters or linters)

**Edge Cases and Error Handling**

- If the marker is removed from a managed file, cure treats it as unmanaged and does not attempt to update it
- If the git history for a managed file is not available (new repo, shallow clone), cure falls back to marker-only detection

**Out of Scope**

- Lockfile-based drift detection (deferred)
- Automatic drift resolution (always requires user decision)

---

#### G-4: Runtime Assembly of Agent Context

**Type:** User Story
**Priority:** Critical
**Component:** internal/agent (extended), pkg/registry

**Description**

As a developer starting an agent session,
I want cure to dynamically assemble the right system prompt, tools, MCP connections, and context from the registry + project config,
so that every agent session is configured correctly for the current project without manual setup.

**Acceptance Criteria**

- [ ] **Given** a project with registered AI config sources, **when** an agent session starts (`cure context new`), **then** the system prompt is assembled from: base system prompt (from registry) + project-specific instructions (from project.json) + repo-specific context (from .cure.json)
- [ ] **Given** a project with MCP server definitions in the registry, **when** an agent session starts, **then** MCP connections are established automatically to the configured servers
- [ ] **Given** a project with skill definitions in the registry, **when** an agent session starts, **then** registered skills are available to the agent
- [ ] **Given** a project with tool definitions in the registry, **when** an agent session starts, **then** tools are registered and available for the agent's tool loop
- [ ] **Given** multiple agents in an orchestration (Domain A), **when** each starts, **then** each receives the same base context but can have agent-specific overrides (e.g., build agent gets build-specific tools, review agent gets review-specific tools)
- [ ] Runtime assembly is logged (which sources contributed which context elements) for debugging

**Edge Cases and Error Handling**

- If a registry source is unavailable (deleted, corrupted), the session starts with available sources and warns about the missing source
- If system prompt assembly exceeds a token limit, cure truncates the least-specific context layers (project defaults first)
- If an MCP server fails to connect, the session starts without that server's tools and warns the user

**Out of Scope**

- Dynamic context update during a running session (context is assembled at session start)
- Token-aware context pruning with LLM assistance

---

## Cross-Domain Dependencies

The following dependency map shows which domains and stories must be completed before others can start.

```
E-1 (Project Entity)
 ├── A-1 (Multi-Instance) — needs project context for devcontainer config
 ├── A-3 (Docker Compose) — needs devcontainer definition from project
 ├── C-2 (GitHub Backlog) — needs tracker config from project.json
 ├── C-3 (AzDO Backlog) — needs tracker config from project.json
 ├── C-4 (Skeleton Creation) — creates project entity
 ├── F-2 (Project-Scoped Doctor) — needs repo list from project
 └── G-2 (Managed Configs) — needs project config for template context

G-1 (Source Registry)
 ├── G-2 (Managed Configs) — registry provides templates for managed files
 ├── G-3 (Drift Detection) — must know which files are managed
 └── G-4 (Runtime Assembly) — registry provides context components

A-1 (Multi-Instance)
 ├── A-2 (Host-Container MCP) — needs running containers
 └── A-3 (Docker Compose) — composition depends on orchestrator design

B-3 (Bidirectional Messaging)
 ├── B-1 (Teams Bot) — Teams is a channel in the messaging architecture
 └── B-2 (OS Notifications) — OS notifications are a channel

D-1 (Monaco Editor) — independent, but enhanced by:
 └── D-2 (Diff Viewer) — uses Monaco diff editor component

F-1 (Multi-Stack Doctor) — independent
```

### Suggested Implementation Order

Based on dependencies:

1. **Foundation (parallel start):**
   - E-1 (Project Entity) — everything depends on this
   - G-1 (Source Registry) — G-2 through G-4 depend on this
   - F-1 (Multi-Stack Doctor) — independent, can start immediately
   - D-1 (Monaco Editor) — independent, can start immediately

2. **Second wave (after foundation):**
   - G-2 (Managed Configs) — after G-1 + E-1
   - G-4 (Runtime Assembly) — after G-1
   - C-2 (GitHub Backlog) — after E-1
   - C-3 (AzDO Backlog) — after E-1
   - C-1 (Git Operations) — independent, can parallel with wave 1
   - D-3 (Integrated Terminal) — independent
   - D-4 (Config Editor) — after E-1
   - B-1 (Teams Bot) — after E-1 (needs notification config from project.json)
   - B-2 (OS Notifications) — independent

3. **Third wave (after second):**
   - G-3 (Drift Detection) — after G-2
   - B-3 (Bidirectional Messaging) — after B-1 + B-2
   - A-1 (Multi-Instance) — after E-1
   - D-2 (Diff Viewer) — after D-1 + C-1
   - F-2 (Project-Scoped Doctor) — after E-1 + F-1
   - C-4 (Skeleton Creation) — after E-1

4. **Final wave (after third):**
   - A-2 (Host-Container MCP) — after A-1
   - A-3 (Docker Compose) — after A-1

---

## Risk Assessment

| # | Risk | Probability | Impact | Score | Mitigation | Owner |
|---|------|-------------|--------|-------|------------|-------|
| 1 | **Monaco + LSP integration complexity.** Monaco standalone lacks VS Code's built-in LSP client. Integrating LSP in browser requires a language server running on the Go backend and a WebSocket bridge. | H | H | 9 | Spike (D-1) to validate Monaco-LSP architecture before committing. Consider monaco-languageclient library. If LSP proves too complex, ship Monaco with syntax highlighting only and defer LSP to post-v1.0.0. | Architect |
| 2 | **Teams Bot Framework auth complexity.** Microsoft Bot Framework requires Azure AD app registration, OAuth flows, and webhook infrastructure. The bot-to-user reply path (bidirectional) is significantly more complex than outbound webhooks. | H | M | 6 | Start with outbound-only Teams notifications via Incoming Webhook (much simpler). Add bidirectional messaging as a follow-on within v1.0.0 if webhook path validates. Budget time for Azure AD setup docs. | Platform Engineer |
| 3 | **Docker-in-Docker reliability.** Devcontainer orchestration on macOS uses Docker Desktop, which has known performance and stability issues with mounted volumes and nested containers. | M | H | 6 | Test early on macOS Docker Desktop. Use Docker socket mounting rather than Docker-in-Docker where possible. Document supported Docker versions. | Platform Engineer |
| 4 | **MCP transport selection for host-container communication.** Three transport options exist (HTTP over Docker network, stdio via docker exec, Unix socket via volume mount). None has been validated for reliability under concurrent agent load. | M | H | 6 | Spike (A-2) to benchmark all three transports under 4-agent concurrent load. Select based on: reliability > latency > complexity. | Architect |
| 5 | **Scope size leading to quality/timeline risk.** 19 user stories across 7 domains is a large scope. Risk of partial delivery or quality degradation. | H | M | 6 | Phase delivery: foundation stories first (E-1, G-1), GUI evolution (D-*) and orchestration (A-*) can be staged across multiple release candidates. Define clear quality gates per wave. | Delivery Manager |
| 6 | **External SDK dependency management.** v1.0.0 introduces multiple new dependencies (Docker SDK, Microsoft Bot Framework, Azure DevOps SDK, go-github). Dependency conflicts, API changes, and security vulnerabilities become a concern. | M | M | 4 | Pin dependency versions. Set up Dependabot. Prefer shelling to CLI tools (gh, az, docker) where SDK adds marginal value over CLI invocation. | Build Engineer |
| 7 | **Terminal emulator in browser.** WebSocket-backed PTY requires careful handling of encoding, resize events, and connection drops. Libraries like xterm.js are mature but add significant frontend bundle size. | M | M | 4 | Use xterm.js (proven, widely used). Test with large output buffers and rapid output (e.g., `find /`). Set scrollback limits. | Frontend Engineer |
| 8 | **Config sync conflict resolution.** Multi-level config merge (6 layers) with sync operations introduces potential for data loss if conflicts are not handled carefully. | L | H | 3 | Config sync is non-destructive (never deletes). Report conflicts, never auto-resolve. Require explicit user confirmation for overwrites. | Architect |

**Scoring:** H=3, M=2, L=1. Score = Probability x Impact. Critical >=6, High >=4, Medium >=2, Low =1.

### Top Risks Summary

1. **Monaco + LSP (Score 9):** Highest risk item. Recommend a time-boxed spike (2-3 days) before committing to LSP in v1.0.0. Fallback: ship Monaco with syntax highlighting only.
2. **Teams bidirectional messaging (Score 6):** Start with outbound webhooks, prove the architecture, then add reply-path.
3. **Docker reliability on macOS (Score 6):** Test early, document constraints, consider Colima as alternative.
4. **MCP transport selection (Score 6):** Spike required to select transport before building orchestrator.
5. **Scope size (Score 6):** Phased delivery with quality gates per wave is essential.

---

## Open Questions and Assumptions

### Open Questions

| # | Question | Impact | Proposed Resolution |
|---|----------|--------|-------------------|
| Q1 | **What Teams license tier is required for bot integration?** Microsoft Teams bot registration requires Azure AD and may require specific Teams licensing. | B-1 scope and documentation | Research minimum Teams license. Document in setup guide. If enterprise-only, consider Incoming Webhook as lower-barrier alternative. |
| Q2 | **Should agents share a single Docker network or have isolated networks?** Network isolation affects security but complicates host-container communication. | A-1, A-2 architecture | Default to shared Docker network with MCP auth. Offer isolated networks as a config option for security-sensitive environments. |
| Q3 | **What LSP servers should cure ship/manage for v1.0.0?** Each language needs its own LSP server binary. Shipping them increases binary size and maintenance burden. | D-1 scope | Do not ship LSP servers. Document how to configure them. Cure launches the user's installed LSP server via config. |
| Q4 | **Should the GUI terminal share the Go backend's PTY library or use a separate process?** Architecture decision that affects latency and resource usage. | D-3 architecture | Use a separate PTY process per terminal session. Communicate via WebSocket. This isolates terminal crashes from the Go server. |
| Q5 | **How should registry sources be authenticated for private git repos?** SSH keys, personal access tokens, or git credential helper? | G-1 scope | Delegate to the system's git credential configuration. Cure runs `git clone` and `git pull` — if the user's git is configured for private repos, it works. No cure-specific auth. |
| Q6 | **What is the maximum practical number of managed config files per repo?** More file types mean more maintenance and more potential for drift. | G-2 scope | Start with the 9 file types listed. Each is opt-in via project config. Add more in post-v1.0.0 based on user demand. |
| Q7 | **Should `cure vcs` shell out to `git` or use a Go git library (go-git)?** Shelling is simpler but slower. go-git avoids the git binary dependency but is a large external dependency. | C-1 architecture | Shell out to `git`. It is a universal dependency for developers. Avoids adding go-git (~40K LOC) to the dependency tree. |
| Q8 | **How should multi-repo workspaces handle different branches per repo?** When orchestrating agents across repos, each may need a different branch. | A-1, E-1 interaction | Each agent container works on its own branch (agent-created or user-specified). The project entity does not enforce branch policy. |

### Assumptions

| # | Assumption | Risk if Wrong |
|---|-----------|---------------|
| A1 | Docker (Docker Desktop or Docker Engine) is available on the developer's machine. | Domain A is entirely blocked. Mitigation: doctor check for Docker, clear error messaging. |
| A2 | The developer has a Microsoft 365 account with Teams access for bot integration. | B-1 is unavailable. Mitigation: B-2 (OS notifications) works without Teams. |
| A3 | The `gh` CLI is installed and authenticated for GitHub backlog operations. | C-2 degrades to manual issue management. Mitigation: doctor check for `gh`. |
| A4 | The `az` CLI with `azure-devops` extension is installed for AzDO operations. | C-3 degrades. Mitigation: AzDO is optional; GitHub is the primary tracker. |
| A5 | External git repos used as registry sources are accessible (public or user has credentials). | G-1 registry `add` fails for inaccessible repos. Mitigation: clear error on clone failure. |
| A6 | Official SDKs from integration targets (Docker, Teams, GitHub, AzDO) are acceptable as dependencies in `internal/` packages. | If not, CLI shelling is the fallback for all integrations (slower, harder to test). |
| A7 | macOS and Linux are the supported platforms for v1.0.0. Windows is not a primary target. | Windows users cannot use orchestration (Docker path differences) or OS notifications. |
| A8 | The SvelteKit 5 frontend framework is retained for GUI evolution. No framework migration. | If migrated, all D-* stories are rescoped. This is unlikely given recent investment. |

---

## Plugin Capability Gap Analysis

This section assesses what skills and agents in the mrlm devstack plugin need to be created or modified to support building v1.0.0.

### Existing Skills — Assessment

| Skill | Status for v1.0.0 | Gaps |
|-------|-------------------|------|
| `mrlm:planning` | **Sufficient.** Used for this document. | None. |
| `mrlm:backlog-writing` | **Sufficient.** Templates cover User Story, Bug, Task, Spike, Epic. | None — all 19 stories fit existing templates. |
| `mrlm:github-issues` | **Sufficient.** Full CRUD, Projects v2 integration, state transitions, audit trail. | None — GitHub backlog features (C-2) rely on this skill directly. |
| `mrlm:azure-devops-work-items` | **Sufficient.** Full CRUD, WIQL queries, state transitions. | None — AzDO backlog features (C-3) rely on this skill. |
| `mrlm:golang` | **Needs extension.** Current skill covers Go best practices but not Docker SDK, Bot Framework SDK, or multi-process orchestration patterns. | Add guidance for: Docker SDK usage patterns, WebSocket server patterns in Go stdlib, PTY management in Go, process supervision patterns. |
| `mrlm:svelte` | **Sufficient for core work.** Covers SvelteKit 5 patterns. | Needs guidance for: Monaco editor integration in SvelteKit, xterm.js integration, WebSocket client patterns for terminal/real-time features. |
| `mrlm:typescript` | **Sufficient.** Covers TypeScript patterns used in the frontend. | None. |
| `mrlm:technical-writing` | **Sufficient.** Covers documentation standards. | None. |
| `mrlm:version-control` | **Sufficient.** Covers git workflow, Conventional Commits, PR standards. | None. |
| `mrlm:estimation` | **Sufficient.** Will be needed for sprint planning. | None. |
| `mrlm:stakeholder-reporting` | **Sufficient.** Will be needed for milestone reports. | None. |
| `mrlm:retrospective` | **Sufficient.** Will be needed after each phase. | None. |
| `mrlm:competitive-analysis` | **Not needed** for v1.0.0 build. | N/A. |
| `mrlm:okr-definition` | **Not needed** for v1.0.0 build. | N/A. |
| `mrlm:terraform` | **Not needed.** No infrastructure provisioning in v1.0.0. | N/A. |
| `mrlm:wails` | **Not needed.** Cure uses embedded SvelteKit, not Wails. | N/A. |

### New Skills Needed

| Skill | Purpose | Domain Coverage | Priority |
|-------|---------|----------------|----------|
| `mrlm:docker` | Docker SDK patterns, Dockerfile/Compose generation, container lifecycle management, volume mounting, network configuration. Needed for Domain A orchestration. | A-1, A-2, A-3 | High |
| `mrlm:microsoft-teams` | Microsoft Bot Framework SDK patterns, Azure AD app registration, Incoming Webhook setup, Adaptive Cards, thread management. Needed for Domain B. | B-1, B-3 | High |
| `mrlm:monaco-editor` | Monaco editor integration in web apps (non-VS Code), LSP client setup (monaco-languageclient), theme configuration, diff editor API. Needed for Domain D. | D-1, D-2 | High |
| `mrlm:terminal-emulation` | xterm.js integration, PTY management in Go, WebSocket transport for terminal I/O, ANSI rendering. Needed for Domain D. | D-3 | Medium |

### Existing Skills — Modifications Needed

| Skill | Modification | Reason |
|-------|-------------|--------|
| `mrlm:golang` | Add section on Docker SDK (`github.com/docker/docker/client`), process supervision (launching and monitoring child processes), WebSocket server patterns using `golang.org/x/net/websocket` or `nhooyr.io/websocket`, and PTY allocation (`os/exec` + `github.com/creack/pty`). | Domain A (orchestration) and Domain D (terminal) require Go patterns not covered today. |
| `mrlm:svelte` | Add section on integrating third-party JS libraries in SvelteKit 5 components (Monaco, xterm.js) with proper SSR avoidance (`onMount`, dynamic imports), and WebSocket client state management. | Domain D GUI stories require embedding complex JS libraries that need SSR-safe loading. |
| `mrlm:github-issues` | No modifications needed — but the C-2 story builds a cure-native abstraction over `gh` CLI operations. The skill itself is used by agents during development, not embedded in cure's runtime. | Clarity note only. |

### Agent Gaps

The current agent set (business-analyst, software-engineer, code-reviewer, qa-engineer, platform-engineer, delivery-manager, security-specialist) is sufficient for v1.0.0 development. However:

| Agent | Gap | Recommendation |
|-------|-----|----------------|
| software-engineer | No experience with Docker SDK, Bot Framework, or Monaco editor integration. | Equip with new skills (`mrlm:docker`, `mrlm:microsoft-teams`, `mrlm:monaco-editor`). |
| platform-engineer | Manages deployment but not Docker Compose orchestration from application code. | Equip with `mrlm:docker` skill. May need to handle Docker Compose testing. |
| qa-engineer | No test patterns for WebSocket-based features (terminal, real-time editor). | Add WebSocket testing guidance to existing test skill or create test-specific addendum. |

### Spikes Required Before Implementation

Based on the risk assessment and gaps identified, the following spikes should be completed before committing to implementation:

| Spike | Time Box | Question | Deliverable | Blocks |
|-------|----------|----------|-------------|--------|
| S-1: MCP Transport for Host-Container | 3 days | Which MCP transport (HTTP Streamable, stdio via docker exec, Unix socket) is most reliable under 4-agent concurrent load? | Benchmark results + recommendation | A-2 |
| S-2: Monaco + LSP in SvelteKit | 3 days | Can monaco-languageclient provide LSP features (autocomplete, diagnostics) in a SvelteKit 5 SPA backed by a Go HTTP server? | Working PoC or "defer LSP" decision | D-1 |
| S-3: Teams Bot Bidirectional Messaging | 2 days | Can a Teams bot receive and relay user replies to a non-Azure-hosted Go backend? What Azure AD configuration is required? | Architecture document + Azure AD setup guide | B-1, B-3 |
| S-4: xterm.js + Go PTY over WebSocket | 2 days | Can xterm.js connect to a Go backend PTY via WebSocket with acceptable latency and correct encoding? | Working PoC | D-3 |

---

## Next Steps

1. **Review this document** with the project owner (Martin Hrasek) and confirm scope, priorities, and risk mitigations.
2. **Execute spikes** S-1 through S-4 to derisk critical technical unknowns.
3. **Create epic issues** on GitHub for each domain (A through G) with child issues for each user story.
4. **Estimate effort** using T-shirt sizing for each story to inform phasing and release planning.
5. **Define milestone plan** mapping stories to release candidates (v1.0.0-rc.1 through rc.N).
6. **Create or update mrlm devstack skills** identified in the gap analysis before implementation begins.
