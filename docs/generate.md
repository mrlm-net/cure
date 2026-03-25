---
title: "cure generate"
description: "Generate CLAUDE.md and other structured file templates"
order: 3
section: "commands"
---

# cure generate

Generate structured file templates for AI assistants and development tooling. Cure ships with embedded templates that produce well-structured output files.

## Subcommands

### cure generate claude-md

Generate a `CLAUDE.md` project context file. `CLAUDE.md` is a project-level configuration file read by AI coding assistants like Claude Code to understand your project's conventions, architecture, and workflow.

```sh
cure generate claude-md
```

The command writes `CLAUDE.md` to the current directory. If a `CLAUDE.md` already exists, cure prompts before overwriting.

## Design

Cure's template engine (`pkg/template`) uses Go's `text/template` package with templates embedded at compile time via `//go:embed`. This means the binary is fully self-contained — no template files need to be present at runtime.

Post-processing is applied after rendering to ensure consistent formatting:

- `CLAUDE.md` output is formatted as valid Markdown with normalized heading levels and whitespace.

## Adding templates

New templates are added by creating template files in the `pkg/template/` package and registering them with the global template registry. See the [Contributing](/docs/contributing) guide for the full workflow.
