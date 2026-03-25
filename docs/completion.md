---
title: "cure completion"
description: "Shell completion scripts for bash and zsh"
order: 4
section: "commands"
---

# cure completion

Generate shell completion scripts for bash and zsh. Completion scripts enable tab-completion for cure commands, subcommands, and flags in your shell.

## Subcommands

### cure completion bash

Generate a bash completion script and print it to stdout.

```sh
cure completion bash
```

To activate completion for the current session:

```sh
source <(cure completion bash)
```

To make it persistent, add it to your `~/.bashrc`:

```sh
cure completion bash >> ~/.bashrc
source ~/.bashrc
```

### cure completion zsh

Generate a zsh completion script and print it to stdout.

```sh
cure completion zsh
```

To activate completion for the current session:

```sh
source <(cure completion zsh)
```

To make it persistent, add it to your `~/.zshrc`:

```sh
cure completion zsh >> ~/.zshrc
source ~/.zshrc
```

## Dynamic introspection

Completion scripts are generated dynamically at runtime by inspecting the command registry via the `CommandRegistry` interface. This means completion always reflects the actual commands registered in the binary — there is no separate completion definition file to maintain.

When new commands are added to cure, completion support is automatic.
