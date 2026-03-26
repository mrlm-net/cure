package generate

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mrlm-net/cure/pkg/fs"
	"github.com/mrlm-net/cure/pkg/prompt"
	"github.com/mrlm-net/cure/pkg/template"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// CopilotInstructionsCommand generates .github/copilot-instructions.md for GitHub Copilot.
type CopilotInstructionsCommand struct {
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

func (c *CopilotInstructionsCommand) Name() string { return "copilot-instructions" }
func (c *CopilotInstructionsCommand) Description() string {
	return "Generate .github/copilot-instructions.md for GitHub Copilot"
}
func (c *CopilotInstructionsCommand) Usage() string {
	return `Usage: cure generate copilot-instructions [flags]

Generate .github/copilot-instructions.md with YAML frontmatter (applyTo: "**") for
GitHub Copilot. The .github/ directory is created automatically if it does not exist.

Interactive mode (default):
  cure generate copilot-instructions

Non-interactive mode (for CI/CD):
  cure generate copilot-instructions --non-interactive \
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
  --output            Output file path (default: ./.github/copilot-instructions.md)
  --force             Overwrite existing file without prompting
`
}

func (c *CopilotInstructionsCommand) Flags() *flag.FlagSet {
	fset := flag.NewFlagSet("copilot-instructions", flag.ContinueOnError)
	fset.BoolVar(&c.nonInteractive, "non-interactive", false, "Disable prompts, require all values via flags")
	fset.BoolVar(&c.force, "force", false, "Overwrite existing file without prompting")
	fset.BoolVar(&c.dryRun, "dry-run", false, "Preview output without writing file")
	fset.StringVar(&c.outputPath, "output", "./.github/copilot-instructions.md", "Output file path")
	fset.StringVar(&c.name, "name", "", "Project name")
	fset.StringVar(&c.description, "description", "", "Project description")
	fset.StringVar(&c.language, "language", "", "Primary programming language")
	fset.StringVar(&c.buildTool, "build-tool", "", "Build tool (e.g., make, npm, cargo)")
	fset.StringVar(&c.testFramework, "test-framework", "", "Test framework")
	fset.StringVar(&c.conventions, "conventions", "", "Comma-separated key conventions")
	return fset
}

func (c *CopilotInstructionsCommand) Run(ctx context.Context, tc *terminal.Context) error {
	c.loadDefaults(tc)

	if err := c.gatherInput(tc); err != nil {
		return err
	}

	data := c.buildTemplateData()
	output, err := template.Render("copilot-instructions", data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	if c.dryRun {
		fmt.Fprintf(tc.Stdout, "# Dry run mode: would write to %s\n\n", c.outputPath)
		fmt.Fprintln(tc.Stdout, output)
		return nil
	}

	if err := c.checkOverwrite(tc); err != nil {
		return err
	}

	// Ensure parent directory exists (.github/)
	if dir := filepath.Dir(c.outputPath); dir != "." {
		if err := fs.EnsureDir(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	if err := fs.AtomicWrite(c.outputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", c.outputPath, err)
	}

	c.printSuccess(tc)
	return nil
}

func (c *CopilotInstructionsCommand) loadDefaults(tc *terminal.Context) {
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

func (c *CopilotInstructionsCommand) gatherInput(tc *terminal.Context) error {
	if c.nonInteractive {
		return c.validateFlags()
	}
	return c.promptUser(tc)
}

func (c *CopilotInstructionsCommand) validateFlags() error {
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
		c.testFramework = c.defaultTestFramework()
	}
	return nil
}

func (c *CopilotInstructionsCommand) promptUser(tc *terminal.Context) error {
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
		defaultTest = c.defaultTestFramework()
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

func (c *CopilotInstructionsCommand) defaultTestFramework() string {
	switch strings.ToLower(c.language) {
	case "go":
		return "testing"
	case "python":
		return "pytest"
	case "javascript", "typescript":
		return "jest"
	case "rust":
		return "cargo test"
	case "java":
		return "junit"
	default:
		return "testing"
	}
}

func (c *CopilotInstructionsCommand) checkOverwrite(tc *terminal.Context) error {
	exists, err := fs.Exists(c.outputPath)
	if err != nil {
		return fmt.Errorf("failed to check if %s exists: %w", c.outputPath, err)
	}
	if !exists {
		return nil
	}
	if c.force {
		return nil
	}
	if c.nonInteractive {
		return fmt.Errorf("%s already exists. Use --force to overwrite", c.outputPath)
	}
	prompter := prompt.NewPrompter(tc.Stdout, os.Stdin)
	confirm, err := prompter.Confirm(fmt.Sprintf("%s already exists. Overwrite?", c.outputPath))
	if err != nil {
		return err
	}
	if !confirm {
		return fmt.Errorf("aborted: file exists and overwrite declined")
	}
	return nil
}

func (c *CopilotInstructionsCommand) buildTemplateData() map[string]interface{} {
	convList := []string{}
	if c.conventions != "" {
		for _, conv := range strings.Split(c.conventions, ",") {
			convList = append(convList, strings.TrimSpace(conv))
		}
	}
	return map[string]interface{}{
		"Name":          c.name,
		"Description":   c.description,
		"Language":      c.language,
		"BuildTool":     c.buildTool,
		"TestFramework": c.testFramework,
		"Conventions":   convList,
	}
}

func (c *CopilotInstructionsCommand) printSuccess(tc *terminal.Context) {
	relPath, _ := filepath.Rel(".", c.outputPath)
	if relPath == "" {
		relPath = c.outputPath
	}
	fmt.Fprintf(tc.Stdout, "Generated %s successfully.\n\n", relPath)
	fmt.Fprintln(tc.Stdout, "Next steps:")
	fmt.Fprintln(tc.Stdout, "1. Review the file and customize sections as needed")
	fmt.Fprintln(tc.Stdout, "2. Commit to version control")
	fmt.Fprintln(tc.Stdout, "3. GitHub Copilot picks it up automatically from .github/copilot-instructions.md")
}
