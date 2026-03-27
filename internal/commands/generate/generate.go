package generate

import "github.com/mrlm-net/cure/pkg/terminal"

// NewGenerateCommand returns a Router that groups all generate subcommands.
func NewGenerateCommand() terminal.Command {
	router := terminal.New(
		terminal.WithName("generate"),
		terminal.WithDescription("Generate project files (CLAUDE.md, configs, etc.)"),
	)
	router.Register(&ClaudeMDCommand{})
	router.Register(&K8sJobCommand{})
	router.Register(&AgentsMDCommand{})
	router.Register(&CopilotInstructionsCommand{})
	router.Register(&CursorRulesCommand{})
	router.Register(&WindsurfRulesCommand{})
	router.Register(&GeminiMDCommand{})
	router.Register(&GitignoreCommand{})
	return router
}
