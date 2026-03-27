package generate

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mrlm-net/cure/pkg/fs"
	"github.com/mrlm-net/cure/pkg/prompt"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// ClaudeMDCommand generates a CLAUDE.md file via interactive prompts or flags.
type ClaudeMDCommand struct {
	// Flags
	nonInteractive bool
	force          bool
	dryRun         bool
	outputPath     string

	// Field values (from flags or prompts)
	name          string
	description   string
	language      string
	buildTool     string
	testFramework string
	conventions   string // comma-separated
}

func (c *ClaudeMDCommand) Name() string        { return "claude-md" }
func (c *ClaudeMDCommand) Description() string { return "Generate CLAUDE.md project context file" }
func (c *ClaudeMDCommand) Usage() string {
	return `Usage: cure generate claude-md [flags]

Generate a CLAUDE.md file with project context for AI assistants.

Interactive mode (default):
  cure generate claude-md

Non-interactive mode (for CI/CD):
  cure generate claude-md --non-interactive \
    --name myapp \
    --description "A CLI tool for X" \
    --language go \
    --build-tool make \
    --test-framework testing \
    --conventions "gofmt,go vet"

Flags:
  --non-interactive   Disable prompts, require all values via flags
  --dry-run           Preview generated output without writing to disk
  --name              Project name (required in non-interactive)
  --description       Short description (required in non-interactive)
  --language          Primary language (required in non-interactive)
  --build-tool        Build tool (default: make)
  --test-framework    Test framework (default: language-specific)
  --conventions       Comma-separated conventions (optional)
  --output            Output file path (default: ./CLAUDE.md)
  --force             Overwrite existing file without prompting

Examples:
  # Interactive mode with defaults from config
  cure generate claude-md

  # Non-interactive with all values
  cure generate claude-md --non-interactive \
    --name cure \
    --description "Go CLI for dev automation" \
    --language go

  # Preview output without writing to disk
  cure generate claude-md --non-interactive \
    --name cure \
    --description "Go CLI for dev automation" \
    --language go \
    --dry-run

  # Custom output path
  cure generate claude-md --output docs/CLAUDE.md
`
}

func (c *ClaudeMDCommand) Flags() *flag.FlagSet {
	fset := flag.NewFlagSet("claude-md", flag.ContinueOnError)
	fset.BoolVar(&c.nonInteractive, "non-interactive", false, "Disable prompts, require all values via flags")
	fset.BoolVar(&c.force, "force", false, "Overwrite existing file without prompting")
	fset.BoolVar(&c.dryRun, "dry-run", false, "Preview output without writing file")
	fset.StringVar(&c.outputPath, "output", "./CLAUDE.md", "Output file path")
	fset.StringVar(&c.name, "name", "", "Project name")
	fset.StringVar(&c.description, "description", "", "Project description")
	fset.StringVar(&c.language, "language", "", "Primary programming language")
	fset.StringVar(&c.buildTool, "build-tool", "", "Build tool (e.g., make, npm, cargo)")
	fset.StringVar(&c.testFramework, "test-framework", "", "Test framework")
	fset.StringVar(&c.conventions, "conventions", "", "Comma-separated key conventions")
	return fset
}

func (c *ClaudeMDCommand) Run(ctx context.Context, tc *terminal.Context) error {
	c.loadDefaults(tc)

	if err := c.gatherInput(tc); err != nil {
		return err
	}

	// In interactive mode, prompt the user when the target file already exists.
	// This check runs before Generate*, which will honour opts.Force.
	if !c.nonInteractive && !c.dryRun {
		if err := c.checkOverwrite(tc); err != nil {
			return err
		}
	}

	if err := GenerateClaudeMD(ctx, tc.Stdout, ClaudeMDOpts{c.toOpts()}); err != nil {
		return err
	}

	if !c.dryRun {
		c.printSuccess(tc)
	}
	return nil
}

// checkOverwrite prompts for confirmation when the output file already exists
// and --force has not been set (interactive mode only).
func (c *ClaudeMDCommand) checkOverwrite(tc *terminal.Context) error {
	exists, err := fs.Exists(c.outputPath)
	if err != nil {
		return fmt.Errorf("failed to check if %s exists: %w", c.outputPath, err)
	}
	if !exists || c.force {
		return nil
	}
	prompter := prompt.NewPrompter(tc.Stdout, os.Stdin)
	confirm, err := prompter.Confirm(fmt.Sprintf("%s already exists. Overwrite?", c.outputPath))
	if err != nil {
		return err
	}
	if !confirm {
		return fmt.Errorf("aborted: file exists and overwrite declined")
	}
	c.force = true // signal to GenerateClaudeMD that overwrite is permitted
	return nil
}

// toOpts converts the command's internal state into an AIFileOpts value.
func (c *ClaudeMDCommand) toOpts() AIFileOpts {
	return AIFileOpts{
		Name:           c.name,
		Description:    c.description,
		Language:       c.language,
		BuildTool:      c.buildTool,
		TestFramework:  c.testFramework,
		Conventions:    c.conventions,
		OutputPath:     c.outputPath,
		Force:          c.force,
		DryRun:         c.dryRun,
		NonInteractive: c.nonInteractive,
	}
}

// loadDefaults reads default values from tc.Config if available.
func (c *ClaudeMDCommand) loadDefaults(tc *terminal.Context) {
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

// gatherInput collects values via prompts (interactive) or validates flags (non-interactive).
func (c *ClaudeMDCommand) gatherInput(tc *terminal.Context) error {
	if c.nonInteractive {
		return c.validateFlags()
	}
	return c.promptUser(tc)
}

// validateFlags ensures required flags are present in non-interactive mode.
func (c *ClaudeMDCommand) validateFlags() error {
	if c.name == "" {
		return fmt.Errorf("--name is required in non-interactive mode")
	}
	if c.description == "" {
		return fmt.Errorf("--description is required in non-interactive mode")
	}
	if c.language == "" {
		return fmt.Errorf("--language is required in non-interactive mode")
	}
	// Optional fields get defaults if not set — GenerateClaudeMD will also apply
	// these, but setting them here keeps validateFlags self-contained.
	if c.buildTool == "" {
		c.buildTool = "make"
	}
	if c.testFramework == "" {
		c.testFramework = defaultTestFramework(c.language)
	}
	return nil
}

// promptUser runs interactive prompts to gather input.
func (c *ClaudeMDCommand) promptUser(tc *terminal.Context) error {
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
	c.buildTool, err = prompter.Optional(fmt.Sprintf("Build tool (e.g., make, npm, cargo) [%s]:", c.buildTool), c.buildTool)
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

// printSuccess writes success message and next steps to stdout.
func (c *ClaudeMDCommand) printSuccess(tc *terminal.Context) {
	relPath, _ := filepath.Rel(".", c.outputPath)
	if relPath == "" {
		relPath = c.outputPath
	}

	fmt.Fprintf(tc.Stdout, "Generated %s successfully.\n\n", relPath)
	fmt.Fprintln(tc.Stdout, "Next steps:")
	fmt.Fprintln(tc.Stdout, "1. Review CLAUDE.md and customize sections as needed")
	fmt.Fprintln(tc.Stdout, "2. Commit to version control (git add CLAUDE.md && git commit -m \"Add CLAUDE.md\")")
	fmt.Fprintln(tc.Stdout, "3. Share with your team and AI tools")
	fmt.Fprintln(tc.Stdout, "")
	fmt.Fprintln(tc.Stdout, "Example usage:")
	fmt.Fprintln(tc.Stdout, "  - GitHub Copilot: Place CLAUDE.md in repo root")
	fmt.Fprintln(tc.Stdout, "  - Anthropic Claude: Reference in project context")
}
