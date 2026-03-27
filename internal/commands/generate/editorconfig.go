package generate

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mrlm-net/cure/pkg/fs"
	"github.com/mrlm-net/cure/pkg/prompt"
	"github.com/mrlm-net/cure/pkg/template"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// EditorconfigOpts holds all configuration for generating an .editorconfig file.
type EditorconfigOpts struct {
	// Languages is the list of language keys to include (e.g. ["go","python"]).
	// An empty slice generates the [*] root section only.
	Languages []string
	// OutputPath is the destination file path. Defaults to "./.editorconfig".
	OutputPath string
	// Force overwrites an existing file without prompting.
	Force bool
	// DryRun writes rendered content to w instead of writing to disk.
	DryRun bool
	// NonInteractive disables prompts and uses Languages directly.
	NonInteractive bool
}

// EditorconfigCommand implements the `cure generate editorconfig` subcommand.
type EditorconfigCommand struct {
	nonInteractive bool
	force          bool
	dryRun         bool
	outputPath     string
	languages      string // comma-separated language keys from --languages flag
}

func (c *EditorconfigCommand) Name() string { return "editorconfig" }
func (c *EditorconfigCommand) Description() string {
	return "Generate .editorconfig with language-aware indent rules"
}
func (c *EditorconfigCommand) Usage() string {
	return `Usage: cure generate editorconfig [flags]

Generate an .editorconfig file with per-language indent rules.

Interactive mode (default):
  cure generate editorconfig

Non-interactive mode (for CI/CD):
  cure generate editorconfig --non-interactive --languages go,python

Supported languages:
  go, javascript, python, rust, java, shell, markdown, yaml, generic

Flags:
  --non-interactive   Disable prompts; with --languages generates those sections,
                      without --languages generates [*] section only
  --dry-run           Preview generated output without writing to disk
  --languages         Comma-separated language keys (non-interactive)
  --output            Output file path (default: ./.editorconfig)
  --force             Overwrite existing file without prompting

Examples:
  # Interactive language selection
  cure generate editorconfig

  # Non-interactive: [*] section only
  cure generate editorconfig --non-interactive

  # Non-interactive with language sections
  cure generate editorconfig --non-interactive --languages go,python,yaml

  # Preview output without writing to disk
  cure generate editorconfig --non-interactive --languages go --dry-run

  # Custom output path
  cure generate editorconfig --output configs/.editorconfig
`
}

func (c *EditorconfigCommand) Flags() *flag.FlagSet {
	fset := flag.NewFlagSet("editorconfig", flag.ContinueOnError)
	fset.BoolVar(&c.nonInteractive, "non-interactive", false, "Disable prompts, use flags only")
	fset.BoolVar(&c.force, "force", false, "Overwrite existing file without prompting")
	fset.BoolVar(&c.dryRun, "dry-run", false, "Preview output without writing file")
	fset.StringVar(&c.outputPath, "output", "./.editorconfig", "Output file path")
	fset.StringVar(&c.languages, "languages", "", "Comma-separated language keys (non-interactive)")
	return fset
}

func (c *EditorconfigCommand) Run(ctx context.Context, tc *terminal.Context) error {
	opts := EditorconfigOpts{
		OutputPath:     c.outputPath,
		Force:          c.force,
		DryRun:         c.dryRun,
		NonInteractive: c.nonInteractive,
	}

	// Parse --languages flag into slice
	if c.languages != "" {
		for _, lang := range strings.Split(c.languages, ",") {
			lang = strings.TrimSpace(lang)
			if lang != "" {
				opts.Languages = append(opts.Languages, lang)
			}
		}
	}

	// Interactive mode: show MultiSelect menu
	if !c.nonInteractive {
		selected, err := c.promptLanguages(tc)
		if err != nil {
			return err
		}
		opts.Languages = selected
	}

	return GenerateEditorconfig(ctx, tc.Stdout, opts)
}

// promptLanguages shows an interactive MultiSelect menu and returns the chosen language keys.
func (c *EditorconfigCommand) promptLanguages(tc *terminal.Context) ([]string, error) {
	options := []prompt.Option{
		{Label: "Go", Value: "go"},
		{Label: "JavaScript/TypeScript", Value: "javascript"},
		{Label: "Python", Value: "python"},
		{Label: "Rust", Value: "rust"},
		{Label: "Java", Value: "java"},
		{Label: "Shell", Value: "shell"},
		{Label: "Markdown", Value: "markdown"},
		{Label: "YAML", Value: "yaml"},
		{Label: "Generic (catch-all)", Value: "generic"},
	}

	prompter := prompt.NewPrompter(tc.Stdout, os.Stdin)
	chosen, err := prompter.MultiSelect("Select languages for .editorconfig sections", options)
	if err != nil {
		return nil, err
	}

	langs := make([]string, 0, len(chosen))
	for _, opt := range chosen {
		langs = append(langs, opt.Value)
	}
	return langs, nil
}

// GenerateEditorconfig renders an .editorconfig file from opts and either writes
// it to w (dry-run) or persists it to opts.OutputPath on disk.
func GenerateEditorconfig(ctx context.Context, w io.Writer, opts EditorconfigOpts) error {
	if opts.OutputPath == "" {
		opts.OutputPath = "./.editorconfig"
	}

	// Build sections in canonical order for deterministic output.
	sections := buildEditorSections(opts.Languages)

	data := map[string]interface{}{
		"Sections": sections,
	}

	output, err := template.Render("editorconfig", data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	if opts.DryRun {
		fmt.Fprintf(w, "# Dry run mode: would write to %s\n\n", opts.OutputPath)
		fmt.Fprintln(w, output)
		return nil
	}

	// Check for existing file before writing.
	exists, err := fs.Exists(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to check if %s exists: %w", opts.OutputPath, err)
	}
	if exists && !opts.Force {
		return fmt.Errorf("%s already exists. Use --force to overwrite", opts.OutputPath)
	}

	if err := fs.AtomicWrite(opts.OutputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", opts.OutputPath, err)
	}

	printEditorconfigSuccess(w, opts.OutputPath)
	return nil
}

// buildEditorSections returns the EditorSection list for the given language keys,
// preserving editorConfigLanguageOrder for deterministic section ordering.
// Unknown language keys are silently skipped.
func buildEditorSections(languages []string) []EditorSection {
	if len(languages) == 0 {
		return nil
	}

	// Build lookup set from requested languages.
	requested := make(map[string]bool, len(languages))
	for _, lang := range languages {
		requested[lang] = true
	}

	// Emit sections in canonical order.
	sections := make([]EditorSection, 0, len(languages))
	for _, key := range editorConfigLanguageOrder {
		if requested[key] {
			if section, ok := editorConfigRules[key]; ok {
				sections = append(sections, section)
			}
		}
	}
	return sections
}

// printEditorconfigSuccess writes the post-generation success message to w.
func printEditorconfigSuccess(w io.Writer, outputPath string) {
	relPath, _ := filepath.Rel(".", outputPath)
	if relPath == "" {
		relPath = outputPath
	}
	fmt.Fprintf(w, "Generated %s successfully.\n\n", relPath)
	fmt.Fprintln(w, "Next steps:")
	fmt.Fprintln(w, "1. Review .editorconfig and add any project-specific overrides")
	fmt.Fprintln(w, "2. Commit to version control (git add .editorconfig && git commit -m \"Add .editorconfig\")")
	fmt.Fprintln(w, "3. Ensure your editor has EditorConfig support (https://editorconfig.org/#download)")
}
