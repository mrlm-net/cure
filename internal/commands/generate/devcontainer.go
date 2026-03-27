package generate

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mrlm-net/cure/pkg/fs"
	"github.com/mrlm-net/cure/pkg/prompt"
	"github.com/mrlm-net/cure/pkg/template"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// baseImagePattern validates Docker image references.
// Permits registry/namespace/name:tag@digest forms; rejects newlines and quotes.
var baseImagePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/:\-@]*$`)

const (
	devcontainerDefaultName      = "dev"
	devcontainerDefaultBaseImage = "mcr.microsoft.com/devcontainers/base:ubuntu"
	devcontainerDefaultOutputDir = "./.devcontainer"
	devcontainerDockerfileBase   = "ubuntu"
)

// DevcontainerOpts holds all configuration for the devcontainer generator.
// It is exposed so callers can drive generation programmatically without the
// CLI layer.
type DevcontainerOpts struct {
	// Name is the "name" field written to devcontainer.json. Defaults to "dev".
	Name string
	// BaseImage is the Docker image reference used when UseDockerfile is false.
	// Defaults to "mcr.microsoft.com/devcontainers/base:ubuntu".
	BaseImage string
	// UseDockerfile, when true, generates a Dockerfile stub and uses a
	// "build.dockerfile" reference in devcontainer.json instead of "image".
	UseDockerfile bool
	// Extensions is a comma-separated list of VS Code extension IDs. Optional.
	Extensions string
	// PostCreateCommand is an optional shell command executed after container creation.
	PostCreateCommand string
	// OutputDir is the directory where generated files are written.
	// Defaults to "./.devcontainer".
	OutputDir string
	// Force overwrites existing files without prompting.
	Force bool
	// DryRun prints the generated content to w instead of writing files.
	DryRun bool
	// NonInteractive disables interactive prompts and requires all values via opts.
	NonInteractive bool
}

// parseCSV splits a comma-separated string into trimmed, non-empty tokens.
func parseCSV(s string) []string {
	if s == "" {
		return []string{}
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

// devcontainerJSON is the Go representation of .devcontainer/devcontainer.json.
// Using a typed struct instead of text/template ensures encoding/json handles
// all string escaping, preventing JSON injection via user-controlled fields.
type devcontainerJSON struct {
	Name              string                    `json:"name"`
	Build             *devcontainerBuild        `json:"build,omitempty"`
	Image             string                    `json:"image,omitempty"`
	Features          map[string]interface{}    `json:"features"`
	Customizations    devcontainerCustomizations `json:"customizations"`
	PostCreateCommand string                    `json:"postCreateCommand,omitempty"`
}

type devcontainerBuild struct {
	Dockerfile string `json:"dockerfile"`
}

type devcontainerCustomizations struct {
	VSCode devcontainerVSCode `json:"vscode"`
}

type devcontainerVSCode struct {
	Extensions []string `json:"extensions"`
}

// GenerateDevcontainer generates devcontainer configuration files according to
// opts. Output is written to opts.OutputDir; dry-run output is written to w.
func GenerateDevcontainer(ctx context.Context, w io.Writer, opts DevcontainerOpts) error {
	// Apply defaults.
	if opts.Name == "" {
		opts.Name = devcontainerDefaultName
	}
	if opts.OutputDir == "" {
		opts.OutputDir = devcontainerDefaultOutputDir
	}
	if !opts.UseDockerfile && opts.BaseImage == "" {
		opts.BaseImage = devcontainerDefaultBaseImage
	}

	// Validate base image format to prevent Dockerfile injection.
	if opts.BaseImage != "" && !baseImagePattern.MatchString(opts.BaseImage) {
		return fmt.Errorf("invalid --base-image %q: must match registry/name:tag format", opts.BaseImage)
	}

	// Resolve the base image used in the Dockerfile stub.
	dockerfileBaseImage := opts.BaseImage
	if opts.UseDockerfile && dockerfileBaseImage == "" {
		dockerfileBaseImage = devcontainerDockerfileBase
	}

	// Build devcontainer.json using encoding/json for correct string escaping.
	exts := parseCSV(opts.Extensions)
	cfg := devcontainerJSON{
		Name:     opts.Name,
		Features: map[string]interface{}{},
		Customizations: devcontainerCustomizations{
			VSCode: devcontainerVSCode{
				Extensions: exts,
			},
		},
		PostCreateCommand: opts.PostCreateCommand,
	}
	if opts.UseDockerfile {
		cfg.Build = &devcontainerBuild{Dockerfile: "Dockerfile"}
	} else {
		cfg.Image = opts.BaseImage
	}
	jsonBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal devcontainer.json: %w", err)
	}
	devcontainerContent := string(jsonBytes) + "\n"

	devcontainerPath := filepath.Join(opts.OutputDir, "devcontainer.json")

	// Dry-run: write to writer and return without touching disk.
	if opts.DryRun {
		fmt.Fprintf(w, "# Dry run mode: would write to %s\n\n", devcontainerPath)
		fmt.Fprintln(w, devcontainerContent)

		if opts.UseDockerfile {
			dockerfileData := map[string]interface{}{
				"BaseImage": dockerfileBaseImage,
			}
			dockerfileContent, err := template.Render("devcontainer-dockerfile", dockerfileData)
			if err != nil {
				return fmt.Errorf("render dockerfile template: %w", err)
			}
			dockerfilePath := filepath.Join(opts.OutputDir, "Dockerfile")
			fmt.Fprintf(w, "# Dry run mode: would write to %s\n\n", dockerfilePath)
			fmt.Fprintln(w, dockerfileContent)
		}
		return nil
	}

	// Ensure output directory exists (skipped in dry-run — no files will be written).
	if err := fs.EnsureDir(opts.OutputDir, 0755); err != nil {
		return fmt.Errorf("ensure output directory %s: %w", opts.OutputDir, err)
	}

	// Overwrite protection for devcontainer.json.
	if err := checkOverwritePath(devcontainerPath, opts.Force); err != nil {
		return err
	}

	// Write devcontainer.json.
	if err := fs.AtomicWrite(devcontainerPath, []byte(devcontainerContent), 0644); err != nil {
		return fmt.Errorf("write %s: %w", devcontainerPath, err)
	}

	// Optionally render and write Dockerfile.
	if opts.UseDockerfile {
		dockerfileData := map[string]interface{}{
			"BaseImage": dockerfileBaseImage,
		}
		dockerfileContent, err := template.Render("devcontainer-dockerfile", dockerfileData)
		if err != nil {
			return fmt.Errorf("render dockerfile template: %w", err)
		}

		dockerfilePath := filepath.Join(opts.OutputDir, "Dockerfile")
		if err := checkOverwritePath(dockerfilePath, opts.Force); err != nil {
			return err
		}
		if err := fs.AtomicWrite(dockerfilePath, []byte(dockerfileContent), 0644); err != nil {
			return fmt.Errorf("write %s: %w", dockerfilePath, err)
		}
	}

	return nil
}

// checkOverwritePath returns an error if path exists and force is false.
func checkOverwritePath(path string, force bool) error {
	exists, err := fs.Exists(path)
	if err != nil {
		return fmt.Errorf("check %s: %w", path, err)
	}
	if exists && !force {
		return fmt.Errorf("%s already exists; use --force to overwrite", path)
	}
	return nil
}

// DevcontainerCommand is the CLI command that wraps GenerateDevcontainer.
type DevcontainerCommand struct {
	nonInteractive    bool
	force             bool
	dryRun            bool
	name              string
	baseImage         string
	useDockerfile     bool
	extensions        string
	postCreateCommand string
	outputDir         string
}

// Name returns the subcommand name.
func (c *DevcontainerCommand) Name() string { return "devcontainer" }

// Description returns a short description shown in help output.
func (c *DevcontainerCommand) Description() string {
	return "Generate .devcontainer/devcontainer.json for VS Code Dev Containers"
}

// Usage returns detailed usage instructions.
func (c *DevcontainerCommand) Usage() string {
	return `Usage: cure generate devcontainer [flags]

Generate a .devcontainer/devcontainer.json (and optional Dockerfile) for VS Code
Dev Containers / GitHub Codespaces.

Interactive mode (default):
  cure generate devcontainer

Non-interactive mode (for CI/CD):
  cure generate devcontainer --non-interactive \
    --name myproject \
    --base-image mcr.microsoft.com/devcontainers/go:1 \
    --extensions "golang.go,eamodio.gitlens"

Flags:
  --non-interactive       Disable prompts, require --name
  --dry-run               Preview generated output without writing to disk
  --force                 Overwrite existing files without prompting
  --name string           Container name (default "dev")
  --base-image string     Base Docker image (default "mcr.microsoft.com/devcontainers/base:ubuntu")
  --dockerfile            Generate a Dockerfile stub (uses build.dockerfile instead of image)
  --extensions string     Comma-separated VS Code extension IDs (optional)
  --post-create-command   Shell command to run after container creation (optional)
  --output-dir string     Output directory (default "./.devcontainer")

Examples:
  # Interactive wizard
  cure generate devcontainer

  # Non-interactive with Go image
  cure generate devcontainer --non-interactive \
    --name go-service \
    --base-image mcr.microsoft.com/devcontainers/go:1

  # Generate with Dockerfile stub
  cure generate devcontainer --non-interactive \
    --name myapp \
    --dockerfile

  # Preview output without writing
  cure generate devcontainer --non-interactive \
    --name myapp \
    --dry-run
`
}

// Flags returns the flag set for this command.
func (c *DevcontainerCommand) Flags() *flag.FlagSet {
	fset := flag.NewFlagSet("devcontainer", flag.ContinueOnError)
	fset.BoolVar(&c.nonInteractive, "non-interactive", false, "Disable prompts, require all values via flags")
	fset.BoolVar(&c.force, "force", false, "Overwrite existing files without prompting")
	fset.BoolVar(&c.dryRun, "dry-run", false, "Preview output without writing files")
	fset.StringVar(&c.name, "name", devcontainerDefaultName, "Container name")
	fset.StringVar(&c.baseImage, "base-image", devcontainerDefaultBaseImage, "Base Docker image")
	fset.BoolVar(&c.useDockerfile, "dockerfile", false, "Generate a Dockerfile stub")
	fset.StringVar(&c.extensions, "extensions", "", "Comma-separated VS Code extension IDs")
	fset.StringVar(&c.postCreateCommand, "post-create-command", "", "Shell command to run after container creation")
	fset.StringVar(&c.outputDir, "output-dir", devcontainerDefaultOutputDir, "Output directory")
	return fset
}

// Run executes the devcontainer generation command.
func (c *DevcontainerCommand) Run(ctx context.Context, tc *terminal.Context) error {
	opts := DevcontainerOpts{
		Name:              c.name,
		BaseImage:         c.baseImage,
		UseDockerfile:     c.useDockerfile,
		Extensions:        c.extensions,
		PostCreateCommand: c.postCreateCommand,
		OutputDir:         c.outputDir,
		Force:             c.force,
		DryRun:            c.dryRun,
		NonInteractive:    c.nonInteractive,
	}

	// Gather input via prompts or validate flags.
	if err := c.gatherInput(tc, &opts); err != nil {
		return err
	}

	if err := GenerateDevcontainer(ctx, tc.Stdout, opts); err != nil {
		return err
	}

	if !opts.DryRun {
		c.printSuccess(tc, opts)
	}
	return nil
}

// gatherInput collects values via interactive prompts or validates non-interactive flags.
// When stdin is not a TTY (e.g. CI), defaults are used without prompting.
func (c *DevcontainerCommand) gatherInput(tc *terminal.Context, opts *DevcontainerOpts) error {
	if c.nonInteractive || !prompt.IsInteractive(os.Stdin) {
		return c.validateFlags(opts)
	}
	return c.promptUser(tc, opts)
}

// validateFlags ensures required values are present in non-interactive mode.
func (c *DevcontainerCommand) validateFlags(opts *DevcontainerOpts) error {
	if opts.Name == "" {
		return fmt.Errorf("--name is required in non-interactive mode")
	}
	return nil
}

// promptUser runs an interactive wizard to populate opts.
func (c *DevcontainerCommand) promptUser(tc *terminal.Context, opts *DevcontainerOpts) error {
	p := prompt.NewPrompter(tc.Stdout, os.Stdin)

	var err error

	opts.Name, err = p.Required("What is the project name?", opts.Name)
	if err != nil {
		return err
	}

	opts.UseDockerfile, err = p.Confirm("Use a Dockerfile instead of a base image?")
	if err != nil {
		return err
	}

	if !opts.UseDockerfile {
		opts.BaseImage, err = p.Optional(
			fmt.Sprintf("Base image [%s]:", devcontainerDefaultBaseImage),
			opts.BaseImage,
		)
		if err != nil {
			return err
		}
	}

	opts.Extensions, err = p.Optional("VS Code extension IDs (comma-separated, optional):", opts.Extensions)
	if err != nil {
		return err
	}

	opts.PostCreateCommand, err = p.Optional("Post-create command (optional):", opts.PostCreateCommand)
	if err != nil {
		return err
	}

	return nil
}

// printSuccess writes a success message and next steps to stdout.
func (c *DevcontainerCommand) printSuccess(tc *terminal.Context, opts DevcontainerOpts) {
	relDir, _ := filepath.Rel(".", opts.OutputDir)
	if relDir == "" {
		relDir = opts.OutputDir
	}

	fmt.Fprintf(tc.Stdout, "Generated %s/devcontainer.json successfully.\n\n", relDir)
	if opts.UseDockerfile {
		fmt.Fprintf(tc.Stdout, "Generated %s/Dockerfile successfully.\n\n", relDir)
	}
	fmt.Fprintln(tc.Stdout, "Next steps:")
	fmt.Fprintln(tc.Stdout, "1. Review and customize the generated devcontainer configuration")
	fmt.Fprintln(tc.Stdout, "2. Reopen the folder in VS Code: \"Remote-Containers: Reopen in Container\"")
	fmt.Fprintln(tc.Stdout, "3. Commit to version control (git add .devcontainer && git commit)")
}
