---
title: "cure context"
description: "Manage multi-turn AI conversation sessions from the terminal"
order: 1
section: "commands"
---

# cure context

Manage multi-turn AI conversation sessions from the terminal. Sessions are persisted to `~/.local/share/cure/sessions/` (XDG-compliant) and work with any registered provider.

Set `ANTHROPIC_API_KEY` before using the `claude` provider:

```sh
export ANTHROPIC_API_KEY=sk-ant-...
```

## Subcommands

### cure context new

Start a new conversation session. Streams the response to stdout and persists the session.

```sh
cure context new --provider claude --message "Summarise the Go 1.25 release notes."
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--provider <name>` | AI provider to use (e.g. `claude`) |
| `--message <text>` | Initial user message |

### cure context resume

Continue an existing session with a new user message.

```sh
cure context resume <session-id> --message "Which change is most impactful for CLI tools?"
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--message <text>` | User message to append |

### cure context list

List saved sessions sorted newest-first.

```sh
cure context list
cure context list --format ndjson
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--format text\|ndjson` | Output format (default: `text`) |

### cure context fork

Deep-copy a session with a new ID. Prints the forked session ID to stdout.

```sh
cure context fork <session-id>
```

### cure context delete

Delete a session. Prompts for confirmation unless `--yes` is supplied.

```sh
cure context delete <session-id>
cure context delete --yes <session-id>
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--yes` | Skip confirmation prompt |

### cure context (REPL mode)

When invoked without a subcommand, enters an interactive read-evaluate-print loop for multi-turn conversations:

```sh
cure context
```

## Session storage

Sessions are stored as JSON files in `~/.local/share/cure/sessions/`. Each file is named after the session ID (128-bit hex). Writes are atomic — a temporary file is renamed into place, so partial writes cannot corrupt the store.

## Backing library

The `cure context` commands are backed by [`pkg/agent`](/docs/pkg-agent) and [`pkg/agent/store`](/docs/pkg-agent-store). Use those packages directly if you want to embed session management into your own Go programs.
