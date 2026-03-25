---
title: "Quick Start"
description: "First steps with cure — context management, template generation, and tracing"
order: 2
section: "getting-started"
---

# Quick Start

Cure automates repetitive development tasks through AI context management, code generation, and network diagnostics. This guide covers the most common workflows.

## AI context management

Start an AI conversation session (requires `ANTHROPIC_API_KEY`):

```sh
export ANTHROPIC_API_KEY=sk-ant-...
cure context new --provider claude --message "Summarise the Go 1.25 release notes."
```

Resume an existing session and continue the conversation:

```sh
cure context resume <session-id> --message "Which change is most impactful for CLI tools?"
```

List all saved sessions:

```sh
cure context list
```

## Template generation

Generate a `CLAUDE.md` template for configuring AI assistants in your project:

```sh
cure generate claude-md
```

## Network tracing

Trace an HTTP request with timing and TLS details in HTML format:

```sh
cure trace http https://api.github.com --format html --output trace.html
```

Trace DNS resolution for a hostname:

```sh
cure trace dns api.github.com
```

## Shell completion

Enable shell completion for bash:

```sh
source <(cure completion bash)
```

Enable shell completion for zsh:

```sh
source <(cure completion zsh)
```

To make completion persistent, add the source command to your shell profile (`.bashrc`, `.zshrc`).

## Getting help

```sh
cure help
cure help context
cure help trace
```

Run `cure help <command>` for detailed usage and flag descriptions.
