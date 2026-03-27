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
	router.Register(&DevcontainerCommand{})
	router.Register(&EditorconfigCommand{})
	router.Register(&GitignoreCommand{})
	router.Register(&GithubWorkflowCommand{})
	// scaffold must be registered last so it can reference all other generators
	// via the scaffoldGenerators map (which captures the Generate* functions).
	router.Register(&ScaffoldCommand{})
	return router
}
