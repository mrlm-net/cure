package generate

import (
	"github.com/mrlm-net/cure/pkg/terminal"
)

// NewGenerateCommand creates the generate command group with claude-md subcommand.
func NewGenerateCommand() terminal.Command {
	router := terminal.New(
		terminal.WithName("generate"),
		terminal.WithDescription("Generate project files (CLAUDE.md, configs, etc.)"),
	)
	router.Register(&ClaudeMDCommand{})
	return router
}
