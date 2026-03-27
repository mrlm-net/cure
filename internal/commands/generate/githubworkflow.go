package generate

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/mrlm-net/cure/pkg/fs"
	"github.com/mrlm-net/cure/pkg/prompt"
	"github.com/mrlm-net/cure/pkg/template"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// goVersionPattern matches valid Go version strings like "1.25" or "1.22".
var goVersionPattern = regexp.MustCompile(`^\d+\.\d+$`)

// GithubWorkflowOpts holds configuration for the GitHub Actions CI workflow generator.
type GithubWorkflowOpts struct {
	// GoVersion is the Go toolchain version to use in the workflow (default: "1.25").
	GoVersion string
	// IncludeLint adds a `go vet` step when true.
	IncludeLint bool
	// IncludeCoverage adds test coverage upload via codecov when true.
	IncludeCoverage bool
	// OutputPath is the destination file path (default: "./.github/workflows/ci.yml").
	OutputPath string
	// Force overwrites an existing file without prompting.
	Force bool
	// DryRun renders the template to w and returns without writing any file.
	DryRun bool
	// NonInteractive skips all prompts and uses flag values with defaults.
	NonInteractive bool
}

// GenerateGithubWorkflow renders a GitHub Actions CI workflow for a Go project and
// writes it to the configured OutputPath. When DryRun is true the rendered
// content is written to w instead and no file is created.
//
// Defaults applied when fields are empty:
//   - GoVersion  → "1.25"
//   - OutputPath → "./.github/workflows/ci.yml"
func GenerateGithubWorkflow(ctx context.Context, w io.Writer, opts GithubWorkflowOpts) error {
	// Apply defaults.
	if opts.GoVersion == "" {
		opts.GoVersion = "1.25"
	}
	if opts.OutputPath == "" {
		opts.OutputPath = "./.github/workflows/ci.yml"
	}

	// Validate Go version format.
	if !goVersionPattern.MatchString(opts.GoVersion) {
		return fmt.Errorf("invalid --go-version %q: must match MAJOR.MINOR (e.g. \"1.25\")", opts.GoVersion)
	}

	// Ensure the target directory exists before rendering so that AtomicWrite
	// can create the temp file in the same directory (required for atomic rename).
	if !opts.DryRun {
		dir := filepath.Dir(opts.OutputPath)
		if err := fs.EnsureDir(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %s: %w", dir, err)
		}
	}

	// Render template.
	data := map[string]interface{}{
		"GoVersion":       opts.GoVersion,
		"IncludeLint":     opts.IncludeLint,
		"IncludeCoverage": opts.IncludeCoverage,
	}
	output, err := template.Render("github-workflow-go", data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	// Dry-run: write to the provided writer and return without touching the filesystem.
	if opts.DryRun {
		fmt.Fprintf(w, "# Dry run mode: would write to %s\n\n", opts.OutputPath)
		fmt.Fprintln(w, output)
		return nil
	}

	// Overwrite check.
	exists, err := fs.Exists(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to check if %s exists: %w", opts.OutputPath, err)
	}
	if exists && !opts.Force {
		return fmt.Errorf("%s already exists. Use --force to overwrite", opts.OutputPath)
	}

	// Persist to disk atomically.
	if err := fs.AtomicWrite(opts.OutputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", opts.OutputPath, err)
	}

	return nil
}

// GithubWorkflowCommand implements terminal.Command for `cure generate github-workflow`.
type GithubWorkflowCommand struct {
	nonInteractive  bool
	force           bool
	dryRun          bool
	goVersion       string
	includeLint     bool
	includeCoverage bool
	outputPath      string
}

// Name returns the subcommand name used in the CLI.
func (c *GithubWorkflowCommand) Name() string { return "github-workflow" }

// Description returns a short description shown in help output.
func (c *GithubWorkflowCommand) Description() string {
	return "Generate .github/workflows/ci.yml for GitHub Actions CI"
}

// Usage returns detailed usage information including flags and examples.
func (c *GithubWorkflowCommand) Usage() string {
	return `Usage: cure generate github-workflow [flags]

Generate a .github/workflows/ci.yml file for GitHub Actions CI targeting Go projects.

Interactive mode (default):
  cure generate github-workflow

Non-interactive mode (for CI/CD):
  cure generate github-workflow --non-interactive \
    --go-version 1.25 \
    --lint \
    --coverage

Flags:
  --non-interactive   Disable prompts, use flag values with defaults
  --dry-run           Preview generated output without writing to disk
  --force             Overwrite existing file without prompting
  --go-version        Go toolchain version (default: 1.25)
  --lint              Include go vet step
  --coverage          Include test coverage upload via codecov
  --output            Output file path (default: ./.github/workflows/ci.yml)

Examples:
  # Interactive wizard
  cure generate github-workflow

  # Non-interactive with defaults
  cure generate github-workflow --non-interactive

  # Preview with lint and coverage
  cure generate github-workflow --non-interactive \
    --go-version 1.22 \
    --lint \
    --coverage \
    --dry-run

  # Custom output path
  cure generate github-workflow --output .github/workflows/test.yml
`
}

// Flags returns the flag set for this command.
func (c *GithubWorkflowCommand) Flags() *flag.FlagSet {
	fset := flag.NewFlagSet("github-workflow", flag.ContinueOnError)
	fset.BoolVar(&c.nonInteractive, "non-interactive", false, "Disable prompts, use flag values with defaults")
	fset.BoolVar(&c.force, "force", false, "Overwrite existing file without prompting")
	fset.BoolVar(&c.dryRun, "dry-run", false, "Preview output without writing file")
	fset.StringVar(&c.goVersion, "go-version", "1.25", "Go toolchain version (e.g. 1.25, 1.22)")
	fset.BoolVar(&c.includeLint, "lint", false, "Include go vet step")
	fset.BoolVar(&c.includeCoverage, "coverage", false, "Include test coverage upload via codecov")
	fset.StringVar(&c.outputPath, "output", "./.github/workflows/ci.yml", "Output file path")
	return fset
}

// Run executes the command, either via interactive prompts or using provided flags.
func (c *GithubWorkflowCommand) Run(ctx context.Context, tc *terminal.Context) error {
	if !c.nonInteractive {
		if err := c.promptUser(tc); err != nil {
			return err
		}
	}

	opts := GithubWorkflowOpts{
		GoVersion:       c.goVersion,
		IncludeLint:     c.includeLint,
		IncludeCoverage: c.includeCoverage,
		OutputPath:      c.outputPath,
		Force:           c.force,
		DryRun:          c.dryRun,
		NonInteractive:  c.nonInteractive,
	}

	if err := GenerateGithubWorkflow(ctx, tc.Stdout, opts); err != nil {
		return err
	}

	// Print success message only when a file was actually written.
	if !c.dryRun {
		c.printSuccess(tc)
	}
	return nil
}

// promptUser runs the interactive wizard to collect values from the user.
func (c *GithubWorkflowCommand) promptUser(tc *terminal.Context) error {
	prompter := prompt.NewPrompter(tc.Stdout, os.Stdin)

	var err error

	// Go version — optional with default.
	defaultVersion := c.goVersion
	if defaultVersion == "" {
		defaultVersion = "1.25"
	}
	c.goVersion, err = prompter.Optional(fmt.Sprintf("Go version [%s]:", defaultVersion), defaultVersion)
	if err != nil {
		return err
	}
	if c.goVersion == "" {
		c.goVersion = defaultVersion
	}

	// Include lint step.
	c.includeLint, err = prompter.Confirm("Include lint step (go vet)?")
	if err != nil {
		return err
	}

	// Include coverage upload.
	c.includeCoverage, err = prompter.Confirm("Include test coverage upload?")
	if err != nil {
		return err
	}

	return nil
}

// printSuccess writes a success message and next steps to stdout.
func (c *GithubWorkflowCommand) printSuccess(tc *terminal.Context) {
	relPath, _ := filepath.Rel(".", c.outputPath)
	if relPath == "" {
		relPath = c.outputPath
	}

	fmt.Fprintf(tc.Stdout, "Generated %s successfully.\n\n", relPath)
	fmt.Fprintln(tc.Stdout, "Next steps:")
	fmt.Fprintln(tc.Stdout, "1. Review the workflow and adjust branches or triggers as needed")
	fmt.Fprintln(tc.Stdout, "2. Commit to version control (git add .github/workflows/ci.yml && git commit -m \"Add CI workflow\")")
	fmt.Fprintln(tc.Stdout, "3. Push to GitHub — the workflow runs automatically on push and pull_request events")
}
