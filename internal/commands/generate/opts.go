package generate

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/mrlm-net/cure/pkg/fs"
	"github.com/mrlm-net/cure/pkg/template"
)

// AIFileOpts holds the common options shared by all AI assistant file generators.
// Each subcommand embeds this in its own opts type.
type AIFileOpts struct {
	Name           string
	Description    string
	Language       string
	BuildTool      string // default: "make"
	TestFramework  string // default: language-derived
	Conventions    string // comma-separated; empty is valid
	OutputPath     string // subcommand-specific default
	Force          bool
	DryRun         bool
	NonInteractive bool
}

// Per-command opts types wrapping AIFileOpts.

// ClaudeMDOpts holds generation options for the CLAUDE.md generator.
type ClaudeMDOpts struct{ AIFileOpts }

// AgentsMDOpts holds generation options for the AGENTS.md generator.
type AgentsMDOpts struct{ AIFileOpts }

// CopilotInstructionsOpts holds generation options for the copilot-instructions generator.
type CopilotInstructionsOpts struct{ AIFileOpts }

// CursorRulesOpts holds generation options for the cursor-rules generator.
type CursorRulesOpts struct{ AIFileOpts }

// WindsurfRulesOpts holds generation options for the windsurf-rules generator.
type WindsurfRulesOpts struct{ AIFileOpts }

// GeminiMDOpts holds generation options for the GEMINI.md generator.
type GeminiMDOpts struct{ AIFileOpts }

// defaultTestFramework returns a sensible default test framework name for the
// given language. Language comparisons are case-insensitive.
func defaultTestFramework(language string) string {
	switch strings.ToLower(language) {
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

// buildAIFileTemplateData converts an AIFileOpts into the map[string]interface{}
// data structure expected by all AI-file templates.
func buildAIFileTemplateData(opts AIFileOpts) map[string]interface{} {
	convList := []string{}
	if opts.Conventions != "" {
		for _, conv := range strings.Split(opts.Conventions, ",") {
			convList = append(convList, strings.TrimSpace(conv))
		}
	}
	return map[string]interface{}{
		"Name":          opts.Name,
		"Description":   opts.Description,
		"Language":      opts.Language,
		"BuildTool":     opts.BuildTool,
		"TestFramework": opts.TestFramework,
		"Conventions":   convList,
	}
}

// writeAIFile renders templateName with data derived from opts, then writes the
// output to opts.OutputPath (or prints a dry-run preview to w).
//
// It sets default values for BuildTool, TestFramework, and OutputPath when they
// are empty, using the supplied fallbackPath as the output path default.
func writeAIFile(ctx context.Context, w io.Writer, opts AIFileOpts, templateName, fallbackPath string) error {
	// Apply defaults for optional fields.
	if opts.BuildTool == "" {
		opts.BuildTool = "make"
	}
	if opts.TestFramework == "" {
		opts.TestFramework = defaultTestFramework(opts.Language)
	}
	if opts.OutputPath == "" {
		opts.OutputPath = fallbackPath
	}
	opts.OutputPath = filepath.Clean(opts.OutputPath)

	data := buildAIFileTemplateData(opts)
	output, err := template.Render(templateName, data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	if opts.DryRun {
		fmt.Fprintf(w, "# Dry run mode: would write to %s\n\n", opts.OutputPath)
		fmt.Fprintln(w, output)
		return nil
	}

	exists, err := fs.Exists(opts.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to check if %s exists: %w", opts.OutputPath, err)
	}
	if exists && !opts.Force {
		return fmt.Errorf("%s already exists. Use --force to overwrite", opts.OutputPath)
	}

	// Ensure parent directory exists for files nested under subdirectories.
	if dir := filepath.Dir(opts.OutputPath); dir != "." {
		if err := fs.EnsureDir(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return fs.AtomicWrite(opts.OutputPath, []byte(output), 0644)
}

// GenerateClaudeMD renders the claude-md template and writes it to opts.OutputPath,
// or prints a dry-run preview to w when opts.DryRun is true.
// Defaults: BuildTool→"make", TestFramework→language-derived, OutputPath→"./CLAUDE.md".
func GenerateClaudeMD(ctx context.Context, w io.Writer, opts ClaudeMDOpts) error {
	return writeAIFile(ctx, w, opts.AIFileOpts, "claude-md", "./CLAUDE.md")
}

// GenerateAgentsMD renders the agents-md template and writes it to opts.OutputPath,
// or prints a dry-run preview to w when opts.DryRun is true.
// Defaults: BuildTool→"make", TestFramework→language-derived, OutputPath→"./AGENTS.md".
func GenerateAgentsMD(ctx context.Context, w io.Writer, opts AgentsMDOpts) error {
	return writeAIFile(ctx, w, opts.AIFileOpts, "agents-md", "./AGENTS.md")
}

// GenerateCopilotInstructions renders the copilot-instructions template and writes it
// to opts.OutputPath, or prints a dry-run preview to w when opts.DryRun is true.
// Defaults: BuildTool→"make", TestFramework→language-derived,
// OutputPath→"./.github/copilot-instructions.md".
func GenerateCopilotInstructions(ctx context.Context, w io.Writer, opts CopilotInstructionsOpts) error {
	return writeAIFile(ctx, w, opts.AIFileOpts, "copilot-instructions", "./.github/copilot-instructions.md")
}

// GenerateCursorRules renders the cursor-rules template and writes it to
// opts.OutputPath, or prints a dry-run preview to w when opts.DryRun is true.
// Defaults: BuildTool→"make", TestFramework→language-derived,
// OutputPath→"./.cursor/rules/project.mdc".
func GenerateCursorRules(ctx context.Context, w io.Writer, opts CursorRulesOpts) error {
	return writeAIFile(ctx, w, opts.AIFileOpts, "cursor-rules", "./.cursor/rules/project.mdc")
}

// GenerateWindsurfRules renders the windsurf-rules template and writes it to
// opts.OutputPath, or prints a dry-run preview to w when opts.DryRun is true.
// Defaults: BuildTool→"make", TestFramework→language-derived,
// OutputPath→"./.windsurfrules".
func GenerateWindsurfRules(ctx context.Context, w io.Writer, opts WindsurfRulesOpts) error {
	return writeAIFile(ctx, w, opts.AIFileOpts, "windsurf-rules", "./.windsurfrules")
}

// GenerateGeminiMD renders the gemini-md template and writes it to opts.OutputPath,
// or prints a dry-run preview to w when opts.DryRun is true.
// Defaults: BuildTool→"make", TestFramework→language-derived, OutputPath→"./GEMINI.md".
func GenerateGeminiMD(ctx context.Context, w io.Writer, opts GeminiMDOpts) error {
	return writeAIFile(ctx, w, opts.AIFileOpts, "gemini-md", "./GEMINI.md")
}
