// Package completion provides shell auto-completion script generation for cure commands.
//
// The completion command group generates bash and zsh completion scripts by
// introspecting the command registry at runtime. Generated scripts include:
//   - Command name completion for top-level commands
//   - Subcommand completion for nested routers
//   - Flag name completion for all registered flags
//   - Flag value completion for known flags (e.g., --format json|html)
//
// Usage:
//
//	// Bash completion
//	cure completion bash > /etc/bash_completion.d/cure
//	# or: source <(cure completion bash)
//
//	// Zsh completion
//	cure completion zsh > "${fpath[1]}/_cure"
//
// The completion command requires a CommandRegistry (typically the root Router)
// to introspect registered commands and their flags. Commands are never hardcoded;
// the scripts regenerate dynamically based on the current command tree.
package completion
