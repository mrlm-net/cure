package generate

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mrlm-net/cure/pkg/prompt"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// scaffoldOptions are the selectable AI assistant file generators shown in
// the MultiSelect menu when running scaffold interactively.
var scaffoldOptions = []prompt.Option{
	{Label: "CLAUDE.md", Value: "claude-md", Description: "Project context for Claude"},
	{Label: "AGENTS.md", Value: "agents-md", Description: "Project context for Codex/OpenAI agents"},
	{Label: ".github/copilot-instructions.md", Value: "copilot-instructions", Description: "GitHub Copilot context"},
	{Label: ".cursor/rules/project.mdc", Value: "cursor-rules", Description: "Cursor IDE rules"},
	{Label: ".windsurfrules", Value: "windsurf-rules", Description: "Windsurf IDE rules"},
	{Label: "GEMINI.md", Value: "gemini-md", Description: "Project context for Gemini"},
}

// scaffoldEntry groups a generator function with its default output path.
type scaffoldEntry struct {
	fn          func(ctx context.Context, w io.Writer, opts AIFileOpts) error
	defaultPath string
}

// scaffoldGenerators maps each selectable value to its generator entry.
var scaffoldGenerators = map[string]scaffoldEntry{
	"claude-md": {
		fn:          func(ctx context.Context, w io.Writer, opts AIFileOpts) error { return GenerateClaudeMD(ctx, w, ClaudeMDOpts{opts}) },
		defaultPath: "./CLAUDE.md",
	},
	"agents-md": {
		fn:          func(ctx context.Context, w io.Writer, opts AIFileOpts) error { return GenerateAgentsMD(ctx, w, AgentsMDOpts{opts}) },
		defaultPath: "./AGENTS.md",
	},
	"copilot-instructions": {
		fn:          func(ctx context.Context, w io.Writer, opts AIFileOpts) error { return GenerateCopilotInstructions(ctx, w, CopilotInstructionsOpts{opts}) },
		defaultPath: "./.github/copilot-instructions.md",
	},
	"cursor-rules": {
		fn:          func(ctx context.Context, w io.Writer, opts AIFileOpts) error { return GenerateCursorRules(ctx, w, CursorRulesOpts{opts}) },
		defaultPath: "./.cursor/rules/project.mdc",
	},
	"windsurf-rules": {
		fn:          func(ctx context.Context, w io.Writer, opts AIFileOpts) error { return GenerateWindsurfRules(ctx, w, WindsurfRulesOpts{opts}) },
		defaultPath: "./.windsurfrules",
	},
	"gemini-md": {
		fn:          func(ctx context.Context, w io.Writer, opts AIFileOpts) error { return GenerateGeminiMD(ctx, w, GeminiMDOpts{opts}) },
		defaultPath: "./GEMINI.md",
	},
}

// scaffoldError pairs a generator name with the error it produced.
type scaffoldError struct {
	name string
	err  error
}

// ScaffoldCommand generates multiple AI assistant context files in one pass.
type ScaffoldCommand struct {
	nonInteractive bool
	force          bool
	dryRun         bool
	selectFlag     string // --select: comma-separated subcommand names

	// Shared AI-file inputs
	name          string
	description   string
	language      string
	buildTool     string
	testFramework string
	conventions   string
}

func (c *ScaffoldCommand) Name() string        { return "scaffold" }
func (c *ScaffoldCommand) Description() string { return "Generate multiple AI context files in one pass" }
func (c *ScaffoldCommand) Usage() string {
	return `Usage: cure generate scaffold [flags]

Generate multiple AI assistant context files in one pass. In interactive mode a
multi-select menu is shown; in non-interactive mode use --select to specify which
files to generate (defaults to all when --select is omitted).

Interactive mode (default):
  cure generate scaffold

Non-interactive mode (generate all files):
  cure generate scaffold --non-interactive \
    --name myapp \
    --description "A CLI tool for X" \
    --language go

Non-interactive mode (select specific files):
  cure generate scaffold --non-interactive \
    --select claude-md,agents-md \
    --name myapp \
    --description "A CLI tool for X" \
    --language go

Available file generators (use with --select):
  claude-md             CLAUDE.md (Anthropic Claude)
  agents-md             AGENTS.md (cross-tool standard)
  copilot-instructions  .github/copilot-instructions.md (GitHub Copilot)
  cursor-rules          .cursor/rules/project.mdc (Cursor IDE)
  windsurf-rules        .windsurfrules (Windsurf IDE)
  gemini-md             GEMINI.md (Google Gemini CLI)

Flags:
  --non-interactive   Disable prompts; require all values via flags
  --select            Comma-separated list of generators to run (default: all)
  --dry-run           Preview generated output without writing to disk
  --force             Overwrite existing files without prompting
  --name              Project name (required in non-interactive)
  --description       Short description (required in non-interactive)
  --language          Primary language (required in non-interactive)
  --build-tool        Build tool (default: make)
  --test-framework    Test framework (default: language-specific)
  --conventions       Comma-separated conventions (optional)
`
}

func (c *ScaffoldCommand) Flags() *flag.FlagSet {
	fset := flag.NewFlagSet("scaffold", flag.ContinueOnError)
	fset.BoolVar(&c.nonInteractive, "non-interactive", false, "Disable prompts, require all values via flags")
	fset.BoolVar(&c.force, "force", false, "Overwrite existing files without prompting")
	fset.BoolVar(&c.dryRun, "dry-run", false, "Preview output without writing files")
	fset.StringVar(&c.selectFlag, "select", "", "Comma-separated list of generators to run")
	fset.StringVar(&c.name, "name", "", "Project name")
	fset.StringVar(&c.description, "description", "", "Project description")
	fset.StringVar(&c.language, "language", "", "Primary programming language")
	fset.StringVar(&c.buildTool, "build-tool", "", "Build tool (e.g., make, npm, cargo)")
	fset.StringVar(&c.testFramework, "test-framework", "", "Test framework")
	fset.StringVar(&c.conventions, "conventions", "", "Comma-separated key conventions")
	return fset
}

func (c *ScaffoldCommand) Run(ctx context.Context, tc *terminal.Context) error {
	c.loadDefaults(tc)

	// Step 1: Determine which generators to run.
	selected, err := c.resolveSelection(tc)
	if err != nil {
		return err
	}

	// Step 2: Gather shared input (prompts or flag validation).
	if err := c.gatherSharedInput(tc); err != nil {
		return err
	}

	// Step 3: Early exit when nothing was selected.
	if len(selected) == 0 {
		fmt.Fprintln(tc.Stdout, "No files selected.")
		return nil
	}

	// Step 4: Run each selected generator; collect errors, continue on failure.
	baseOpts := AIFileOpts{
		Name:           c.name,
		Description:    c.description,
		Language:       c.language,
		BuildTool:      c.buildTool,
		TestFramework:  c.testFramework,
		Conventions:    c.conventions,
		Force:          c.force,
		DryRun:         c.dryRun,
		NonInteractive: c.nonInteractive,
	}

	var errs []scaffoldError
	for _, name := range selected {
		entry := scaffoldGenerators[name]
		opts := baseOpts
		opts.OutputPath = entry.defaultPath

		if err := entry.fn(ctx, tc.Stdout, opts); err != nil {
			errs = append(errs, scaffoldError{name: name, err: err})
		}
	}

	// Step 5: Report failures.
	if len(errs) > 0 {
		fmt.Fprintln(tc.Stderr, "The following generators failed:")
		for i, e := range errs {
			fmt.Fprintf(tc.Stderr, "  %d. %s: %v\n", i+1, e.name, e.err)
		}
		return fmt.Errorf("scaffold: %w", errs[0].err)
	}

	// Step 6: Print summary.
	if !c.dryRun {
		fmt.Fprintf(tc.Stdout, "\nGenerated %d file(s) successfully.\n", len(selected))
	}
	return nil
}

// resolveSelection determines which generators to run based on --select flag,
// --non-interactive flag, and TTY detection.
func (c *ScaffoldCommand) resolveSelection(tc *terminal.Context) ([]string, error) {
	// --select provided: parse and validate.
	if c.selectFlag != "" {
		return c.parseSelectFlag()
	}

	// No --select: default to all (both interactive and non-interactive).
	if c.nonInteractive || !prompt.IsInteractive(os.Stdin) {
		return allScaffoldNames(), nil
	}

	// Interactive TTY: show MultiSelect menu.
	prompter := prompt.NewPrompter(tc.Stdout, os.Stdin)
	chosen, err := prompter.MultiSelect("Select files to generate", scaffoldOptions)
	if err != nil {
		return nil, fmt.Errorf("scaffold: failed to read selection: %w", err)
	}

	names := make([]string, len(chosen))
	for i, opt := range chosen {
		names[i] = opt.Value
	}
	return names, nil
}

// parseSelectFlag parses the --select flag value into validated generator names.
// Returns an error if any name is not in the known set.
func (c *ScaffoldCommand) parseSelectFlag() ([]string, error) {
	if c.selectFlag == "" {
		return allScaffoldNames(), nil
	}

	parts := strings.Split(c.selectFlag, ",")
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if _, ok := scaffoldGenerators[name]; !ok {
			return nil, fmt.Errorf("unknown generator %q; valid values: %s",
				name, strings.Join(allScaffoldNames(), ", "))
		}
		names = append(names, name)
	}
	return names, nil
}

// gatherSharedInput collects name/description/language via prompts or validates
// them as flags in non-interactive mode.
func (c *ScaffoldCommand) gatherSharedInput(tc *terminal.Context) error {
	if c.nonInteractive {
		return c.validateFlags()
	}
	return c.promptUser(tc)
}

// validateFlags checks required flags in non-interactive mode and applies defaults.
func (c *ScaffoldCommand) validateFlags() error {
	if c.name == "" {
		return fmt.Errorf("--name is required in non-interactive mode")
	}
	if c.description == "" {
		return fmt.Errorf("--description is required in non-interactive mode")
	}
	if c.language == "" {
		return fmt.Errorf("--language is required in non-interactive mode")
	}
	if c.buildTool == "" {
		c.buildTool = "make"
	}
	if c.testFramework == "" {
		c.testFramework = defaultTestFramework(c.language)
	}
	return nil
}

// promptUser runs interactive prompts to gather shared AI file inputs.
func (c *ScaffoldCommand) promptUser(tc *terminal.Context) error {
	prompter := prompt.NewPrompter(tc.Stdout, os.Stdin)

	var err error
	c.name, err = prompter.Required("What is the project name?", c.name)
	if err != nil {
		return err
	}

	c.description, err = prompter.Required("Short description (1-2 sentences):", c.description)
	if err != nil {
		return err
	}

	c.language, err = prompter.Required("Primary language (e.g., Go, Python, TypeScript):", c.language)
	if err != nil {
		return err
	}

	if c.buildTool == "" {
		c.buildTool = "make"
	}
	c.buildTool, err = prompter.Optional(fmt.Sprintf("Build tool [%s]:", c.buildTool), c.buildTool)
	if err != nil {
		return err
	}

	defaultTest := c.testFramework
	if defaultTest == "" {
		defaultTest = defaultTestFramework(c.language)
	}
	c.testFramework, err = prompter.Optional(fmt.Sprintf("Test framework [%s]:", defaultTest), defaultTest)
	if err != nil {
		return err
	}

	c.conventions, err = prompter.Optional("Key conventions (comma-separated):", c.conventions)
	if err != nil {
		return err
	}

	return nil
}

// loadDefaults reads default values from tc.Config if available.
func (c *ScaffoldCommand) loadDefaults(tc *terminal.Context) {
	if tc.Config == nil {
		return
	}
	if c.language == "" {
		if val := tc.Config.Get("generate.language", ""); val != nil {
			if str, ok := val.(string); ok {
				c.language = str
			}
		}
	}
	if c.buildTool == "" {
		if val := tc.Config.Get("generate.build-tool", ""); val != nil {
			if str, ok := val.(string); ok {
				c.buildTool = str
			}
		}
	}
	if c.testFramework == "" {
		if val := tc.Config.Get("generate.test-framework", ""); val != nil {
			if str, ok := val.(string); ok {
				c.testFramework = str
			}
		}
	}
	if c.conventions == "" {
		if val := tc.Config.Get("generate.conventions", ""); val != nil {
			if str, ok := val.(string); ok {
				c.conventions = str
			}
		}
	}
}

// allScaffoldNames returns all known generator names in definition order.
func allScaffoldNames() []string {
	names := make([]string, len(scaffoldOptions))
	for i, opt := range scaffoldOptions {
		names[i] = opt.Value
	}
	return names
}
