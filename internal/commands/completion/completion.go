package completion

import (
	"github.com/mrlm-net/cure/pkg/terminal"
)

// NewCompletionCommand creates the completion command group with bash/zsh subcommands.
// The registry parameter is the root Router, used to introspect registered commands
// and their flags for generating completion scripts.
func NewCompletionCommand(registry terminal.CommandRegistry) terminal.Command {
	router := terminal.New(
		terminal.WithName("completion"),
		terminal.WithDescription("Generate shell completion scripts"),
	)
	router.Register(&BashCommand{registry: registry})
	router.Register(&ZshCommand{registry: registry})
	return router
}
