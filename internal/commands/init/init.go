// Package initcmd implements the "cure init" project bootstrap wizard.
// It orchestrates all generate subcommand entry points in a single interactive
// or non-interactive pass. Package name is initcmd to avoid shadowing the Go
// built-in init function.
package initcmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mrlm-net/cure/internal/commands/generate"
	"github.com/mrlm-net/cure/pkg/prompt"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// allAIToolIDs lists every AI-file generator value in canonical order.
const allAIToolIDs = "claude-md,agents-md,copilot-instructions,cursor-rules,windsurf-rules,gemini-md"

// languageOptions are the choices presented in the interactive language menu.
var languageOptions = []prompt.Option{
	{Label: "Go", Value: "go"},
	{Label: "Node.js", Value: "node"},
	{Label: "Python", Value: "python"},
	{Label: "Rust", Value: "rust"},
	{Label: "Other", Value: "other"},
}

// allAIToolOptions mirrors the scaffold options for the AI file multi-select.
var allAIToolOptions = []prompt.Option{
	{Label: "CLAUDE.md", Value: "claude-md", Description: "Project context for Claude"},
	{Label: "AGENTS.md", Value: "agents-md", Description: "Project context for Codex/OpenAI agents"},
	{Label: ".github/copilot-instructions.md", Value: "copilot-instructions", Description: "GitHub Copilot context"},
	{Label: ".cursor/rules/project.mdc", Value: "cursor-rules", Description: "Cursor IDE rules"},
	{Label: ".windsurfrules", Value: "windsurf-rules", Description: "Windsurf IDE rules"},
	{Label: "GEMINI.md", Value: "gemini-md", Description: "Project context for Gemini"},
}

// validAIToolIDs is a set of known AI tool IDs for fast validation.
var validAIToolIDs = func() map[string]bool {
	m := make(map[string]bool, len(allAIToolOptions))
	for _, opt := range allAIToolOptions {
		m[opt.Value] = true
	}
	return m
}()

// generatorResult pairs a component name with the error it produced (nil = success).
type generatorResult struct {
	name string
	err  error
}

// InitCommand implements "cure init".
type InitCommand struct {
	nonInteractive bool
	dryRun         bool
	force          bool

	// Project metadata
	name     string
	language string

	// AI tool selection (comma-separated IDs, default: all)
	aiTools string

	// Infrastructure component flags
	devcontainer bool
	ci           bool
	editorconfig bool
	gitignore    bool
}

// NewInitCommand constructs a new InitCommand and returns it as a terminal.Command.
func NewInitCommand() terminal.Command { return &InitCommand{} }

// Name returns the command name.
func (c *InitCommand) Name() string { return "init" }

// Description returns a short one-line description for help output.
func (c *InitCommand) Description() string {
	return "Initialize project tooling configuration"
}

// Usage returns detailed usage information including flags and examples.
func (c *InitCommand) Usage() string {
	return `Usage: cure init [flags]

Interactive wizard that generates all standard project configuration files in one pass.

In interactive mode, prompts for project details and lets you select which files to generate.
In --non-interactive mode, accepts all options via flags (defaults: all components enabled).

Flags:
  --non-interactive  Skip prompts; use flag values
  --dry-run          Preview output without writing files
  --force            Overwrite existing files without prompting
  --name             Project name
  --language         Primary language (go|node|python|rust|other)
  --ai-tools         Comma-separated AI tool IDs to generate (default: all)
                     Options: claude-md,agents-md,copilot-instructions,cursor-rules,windsurf-rules,gemini-md
  --devcontainer     Generate devcontainer configuration (default: true)
  --ci               Generate CI workflow (default: true)
  --editorconfig     Generate .editorconfig (default: true)
  --gitignore        Generate .gitignore (default: true)

Examples:
  cure init
  cure init --non-interactive --name myapp --language go
  cure init --non-interactive --name myapp --language go --ai-tools claude-md,cursor-rules
  cure init --non-interactive --name myapp --language go --dry-run
`
}

// Flags returns the flag set for this command.
func (c *InitCommand) Flags() *flag.FlagSet {
	fset := flag.NewFlagSet("init", flag.ContinueOnError)
	fset.BoolVar(&c.nonInteractive, "non-interactive", false, "Disable prompts; require all values via flags")
	fset.BoolVar(&c.dryRun, "dry-run", false, "Preview output without writing files")
	fset.BoolVar(&c.force, "force", false, "Overwrite existing files")
	fset.StringVar(&c.name, "name", "", "Project name")
	fset.StringVar(&c.language, "language", "", "Primary language (go|node|python|rust|other)")
	fset.StringVar(&c.aiTools, "ai-tools", "", "Comma-separated AI tool IDs (default: all)")
	fset.BoolVar(&c.devcontainer, "devcontainer", true, "Generate devcontainer configuration")
	fset.BoolVar(&c.ci, "ci", true, "Generate CI workflow")
	fset.BoolVar(&c.editorconfig, "editorconfig", true, "Generate .editorconfig")
	fset.BoolVar(&c.gitignore, "gitignore", true, "Generate .gitignore")
	return fset
}

// Run executes the init wizard, collecting input interactively or from flags,
// then orchestrating all selected generators.
func (c *InitCommand) Run(ctx context.Context, tc *terminal.Context) error {
	if !c.nonInteractive && prompt.IsInteractive(os.Stdin) {
		if err := c.collectInteractive(tc); err != nil {
			return err
		}
	} else {
		if err := c.validateNonInteractive(); err != nil {
			return err
		}
		// Default to all AI tools when --ai-tools was not provided.
		if c.aiTools == "" {
			c.aiTools = allAIToolIDs
		}
	}

	return c.runGenerators(ctx, tc)
}

// collectInteractive runs the interactive wizard to gather all inputs.
func (c *InitCommand) collectInteractive(tc *terminal.Context) error {
	p := prompt.NewPrompter(tc.Stdout, os.Stdin)

	var err error

	// Project name
	c.name, err = p.Required("What is the project name?", c.name)
	if err != nil {
		return fmt.Errorf("init: failed to read project name: %w", err)
	}

	// Language selection
	langOpt, err := p.SingleSelect("Select the primary language", languageOptions)
	if err != nil {
		return fmt.Errorf("init: failed to read language: %w", err)
	}
	c.language = langOpt.Value

	// AI tools multi-select
	chosen, err := p.MultiSelect("Select AI assistant files to generate", allAIToolOptions)
	if err != nil {
		return fmt.Errorf("init: failed to read AI tool selection: %w", err)
	}
	toolIDs := make([]string, len(chosen))
	for i, opt := range chosen {
		toolIDs[i] = opt.Value
	}
	c.aiTools = strings.Join(toolIDs, ",")

	// Infrastructure component confirmations
	c.devcontainer, err = p.Confirm("Generate devcontainer configuration?")
	if err != nil {
		return fmt.Errorf("init: failed to read devcontainer selection: %w", err)
	}

	c.ci, err = p.Confirm("Generate CI workflow (.github/workflows/ci.yml)?")
	if err != nil {
		return fmt.Errorf("init: failed to read CI selection: %w", err)
	}

	c.editorconfig, err = p.Confirm("Generate .editorconfig?")
	if err != nil {
		return fmt.Errorf("init: failed to read editorconfig selection: %w", err)
	}

	c.gitignore, err = p.Confirm("Generate .gitignore?")
	if err != nil {
		return fmt.Errorf("init: failed to read gitignore selection: %w", err)
	}

	return nil
}

// validateNonInteractive validates required flags in non-interactive mode.
func (c *InitCommand) validateNonInteractive() error {
	if c.name == "" {
		return fmt.Errorf("--name is required in non-interactive mode")
	}
	if c.language == "" {
		return fmt.Errorf("--language is required in non-interactive mode")
	}
	// Validate any explicitly provided AI tool IDs before running generators.
	if c.aiTools != "" {
		for _, id := range parseCSV(c.aiTools) {
			if !validAIToolIDs[id] {
				return fmt.Errorf("unknown AI tool %q; valid values: %s", id, allAIToolIDs)
			}
		}
	}
	return nil
}

// runGenerators builds base opts and calls every selected generator, collecting
// results. It prints a summary and returns an error if any generator failed.
func (c *InitCommand) runGenerators(ctx context.Context, tc *terminal.Context) error {
	baseOpts := generate.AIFileOpts{
		Name:           c.name,
		Language:       c.language,
		DryRun:         c.dryRun,
		Force:          c.force,
		NonInteractive: true,
	}

	var results []generatorResult

	// AI tools — generate each selected tool.
	for _, toolID := range parseCSV(c.aiTools) {
		results = append(results, c.runAITool(ctx, tc, toolID, baseOpts))
	}

	// Infrastructure components — each is conditional on its flag.
	if c.devcontainer {
		opts := generate.DevcontainerOpts{
			Name:           c.name,
			DryRun:         c.dryRun,
			Force:          c.force,
			NonInteractive: true,
		}
		err := generate.GenerateDevcontainer(ctx, tc.Stdout, opts)
		results = append(results, generatorResult{"devcontainer", err})
	}

	if c.ci {
		opts := generate.GithubWorkflowOpts{
			DryRun:         c.dryRun,
			Force:          c.force,
			NonInteractive: true,
		}
		err := generate.GenerateGithubWorkflow(ctx, tc.Stdout, opts)
		results = append(results, generatorResult{"ci", err})
	}

	if c.editorconfig {
		// Derive language key: map the init language value to an editorconfig
		// language key. "node" maps to "javascript"; others are identical when
		// they exist in the editorconfig language set; unknown keys are skipped.
		langs := editorconfigLanguages(c.language)
		opts := generate.EditorconfigOpts{
			Languages:      langs,
			DryRun:         c.dryRun,
			Force:          c.force,
			NonInteractive: true,
		}
		err := generate.GenerateEditorconfig(ctx, tc.Stdout, opts)
		results = append(results, generatorResult{"editorconfig", err})
	}

	if c.gitignore {
		profiles := gitignoreProfiles(c.language)
		opts := generate.GitignoreOpts{
			Profiles:       profiles,
			DryRun:         c.dryRun,
			Force:          c.force,
			NonInteractive: true,
		}
		err := generate.GenerateGitignore(ctx, tc.Stdout, opts)
		results = append(results, generatorResult{"gitignore", err})
	}

	return c.printSummary(tc, results)
}

// runAITool dispatches a single AI tool generator by its ID.
func (c *InitCommand) runAITool(ctx context.Context, tc *terminal.Context, toolID string, baseOpts generate.AIFileOpts) generatorResult {
	var err error
	switch toolID {
	case "claude-md":
		err = generate.GenerateClaudeMD(ctx, tc.Stdout, generate.ClaudeMDOpts{AIFileOpts: baseOpts})
	case "agents-md":
		err = generate.GenerateAgentsMD(ctx, tc.Stdout, generate.AgentsMDOpts{AIFileOpts: baseOpts})
	case "copilot-instructions":
		err = generate.GenerateCopilotInstructions(ctx, tc.Stdout, generate.CopilotInstructionsOpts{AIFileOpts: baseOpts})
	case "cursor-rules":
		err = generate.GenerateCursorRules(ctx, tc.Stdout, generate.CursorRulesOpts{AIFileOpts: baseOpts})
	case "windsurf-rules":
		err = generate.GenerateWindsurfRules(ctx, tc.Stdout, generate.WindsurfRulesOpts{AIFileOpts: baseOpts})
	case "gemini-md":
		err = generate.GenerateGeminiMD(ctx, tc.Stdout, generate.GeminiMDOpts{AIFileOpts: baseOpts})
	default:
		err = fmt.Errorf("unknown AI tool %q", toolID)
	}
	return generatorResult{toolID, err}
}

// printSummary writes the completion summary to tc.Stdout and returns a
// non-nil error if any generator failed.
func (c *InitCommand) printSummary(tc *terminal.Context, results []generatorResult) error {
	fmt.Fprintln(tc.Stdout)
	fmt.Fprintln(tc.Stdout, "cure init summary:")
	var failed int
	for _, r := range results {
		if r.err != nil {
			fmt.Fprintf(tc.Stdout, "  x %s: %v\n", r.name, r.err)
			failed++
		} else {
			fmt.Fprintf(tc.Stdout, "  ok %s\n", r.name)
		}
	}
	if failed > 0 {
		return fmt.Errorf("init: %d component(s) failed", failed)
	}
	return nil
}

// editorconfigLanguages maps the init language value to editorconfig language keys.
// "node" maps to "javascript" (which covers JS/TS in editorconfig). Unknown
// values that don't map to a valid editorconfig key are dropped silently so the
// .editorconfig is still generated with the universal [*] section.
func editorconfigLanguages(lang string) []string {
	switch strings.ToLower(lang) {
	case "go":
		return []string{"go"}
	case "node", "javascript", "typescript":
		return []string{"javascript"}
	case "python":
		return []string{"python"}
	case "rust":
		return []string{"rust"}
	default:
		return nil // generates [*] section only
	}
}

// gitignoreProfiles maps the init language value to .gitignore profile keys.
// "other" generates only the universal section (empty profiles slice).
func gitignoreProfiles(lang string) []string {
	switch strings.ToLower(lang) {
	case "go":
		return []string{"go"}
	case "node", "javascript", "typescript":
		return []string{"node"}
	case "python":
		return []string{"python"}
	case "rust":
		return []string{"rust"}
	default:
		return nil // universal section only
	}
}

// parseCSV splits a comma-separated string into trimmed, non-empty tokens.
// This mirrors the helper in the generate package but avoids cross-package
// access of an unexported function.
func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
