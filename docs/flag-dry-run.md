---
title: "--dry-run"
description: "Preview generated output without writing to disk"
order: 4
section: "commands"
---

# --dry-run

The `--dry-run` flag prints the generated output to stdout instead of writing it to a file. No files are created or modified. The command exits `0`.

## Supported commands

| Command | Flag |
|---------|------|
| `cure generate claude-md` | `--dry-run` |

## Usage

```sh
cure generate claude-md --dry-run
```

The output begins with a comment line showing the path that *would* have been written:

```
# Dry run mode: would write to CLAUDE.md

# My Project
...
```

## Use cases

- **Review before committing** — inspect the generated file before it touches the filesystem
- **CI validation** — pipe the output to a diff tool to detect drift between the generated and committed file
- **Scripting** — capture the output with `$(cure generate claude-md --dry-run)` or redirect with `>`
