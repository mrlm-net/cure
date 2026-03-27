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

// WindsurfRulesCommand generates a .windsurfrules file for Windsurf IDE.
type WindsurfRulesCommand struct {
	nonInteractive bool
	force          bool
	dryRun         bool
	outputPath     string

	name          string
	description   string
	language      string
	buildTool     string
	testFramework string
	conventions   string
}

func (c *WindsurfRulesCommand) Name() string { return "windsurf-rules" }
func (c *WindsurfRulesCommand) Description() string {
	return "Generate .windsurfrules for Windsurf IDE"
}
func (c *WindsurfRulesCommand) Usage() string {
	return `Usage: cure generate windsurf-rules [flags]

Generate a .windsurfrules file (plain text, Windsurf-style numbered rules) for Windsurf IDE.

Interactive mode (default):
  cure generate windsurf-rules

Non-interactive mode (for CI/CD):
  cure generate windsurf-rules --non-interactive \
    --name myapp \
    --description "A CLI tool for X" \
    --language go

Flags:
  --non-interactive   Disable prompts, require all values via flags
  --dry-run           Preview generated output without writing to disk
  --name              Project name (required in non-interactive)
  --description       Short description (required in non-interactive)
  --language          Primary language (required in non-interactive)
  --build-tool        Build tool (default: make)
  --test-framework    Test framework (default: language-specific)
  --conventions       Comma-separated conventions (optional)
  --output            Output file path (default: ./.windsurfrules)
  --force             Overwrite existing file without prompting
`
}

func (c *WindsurfRulesCommand) Flags() *flag.FlagSet {
	fset := flag.NewFlagSet("windsurf-rules", flag.ContinueOnError)
	fset.BoolVar(&c.nonInteractive, "non-interactive", false, "Disable prompts, require all values via flags")
	fset.BoolVar(&c.force, "force", false, "Overwrite existing file without prompting")
	fset.BoolVar(&c.dryRun, "dry-run", false, "Preview output without writing file")
	fset.StringVar(&c.outputPath, "output", "./.windsurfrules", "Output file path")
	fset.StringVar(&c.name, "name", "", "Project name")
	fset.StringVar(&c.description, "description", "", "Project description")
	fset.StringVar(&c.language, "language", "", "Primary programming language")
	fset.StringVar(&c.buildTool, "build-tool", "", "Build tool (e.g., make, npm, cargo)")
	fset.StringVar(&c.testFramework, "test-framework", "", "Test framework")
	fset.StringVar(&c.conventions, "conventions", "", "Comma-separated key conventions")
	return fset
}

func (c *WindsurfRulesCommand) Run(ctx context.Context, tc *terminal.Context) error {
	c.loadDefaults(tc)

	if err := c.gatherInput(tc); err != nil {
		return err
	}

	if !c.nonInteractive && !c.dryRun {
		if err := c.checkOverwrite(tc); err != nil {
			return err
		}
	}

	if err := GenerateWindsurfRules(ctx, tc.Stdout, WindsurfRulesOpts{c.toOpts()}); err != nil {
		return err
	}

	if !c.dryRun {
		c.printSuccess(tc)
	}
	return nil
}

func (c *WindsurfRulesCommand) checkOverwrite(tc *terminal.Context) error {
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
	c.force = true
	return nil
}

// toOpts converts the command's internal state into an AIFileOpts value.
func (c *WindsurfRulesCommand) toOpts() AIFileOpts {
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

func (c *WindsurfRulesCommand) loadDefaults(tc *terminal.Context) {
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

func (c *WindsurfRulesCommand) gatherInput(tc *terminal.Context) error {
	if c.nonInteractive {
		return c.validateFlags()
	}
	return c.promptUser(tc)
}

func (c *WindsurfRulesCommand) validateFlags() error {
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

func (c *WindsurfRulesCommand) promptUser(tc *terminal.Context) error {
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

func (c *WindsurfRulesCommand) printSuccess(tc *terminal.Context) {
	relPath, _ := filepath.Rel(".", c.outputPath)
	if relPath == "" {
		relPath = c.outputPath
	}
	fmt.Fprintf(tc.Stdout, "Generated %s successfully.\n\n", relPath)
	fmt.Fprintln(tc.Stdout, "Next steps:")
	fmt.Fprintln(tc.Stdout, "1. Review .windsurfrules and customize rules as needed")
	fmt.Fprintln(tc.Stdout, "2. Commit to version control (git add .windsurfrules && git commit -m \"Add .windsurfrules\")")
	fmt.Fprintln(tc.Stdout, "3. Windsurf IDE picks up rules from .windsurfrules automatically")
}
