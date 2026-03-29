package mcp

import (
	"context"
	"flag"
	"fmt"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mrlm-net/cure/internal/commands/generate"
	pkgdoctor "github.com/mrlm-net/cure/pkg/doctor"
	pkgmcp "github.com/mrlm-net/cure/pkg/mcp"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// ServeCommand implements "cure mcp serve". It starts an MCP server exposing
// cure capabilities (doctor, generate) as MCP tools. The transport is
// auto-detected: stdio when stdin is a pipe, HTTP otherwise.
type ServeCommand struct {
	addr string
}

// NewMCPCommand returns a Router that groups the "mcp" subcommands.
func NewMCPCommand() terminal.Command {
	r := terminal.New(
		terminal.WithName("mcp"),
		terminal.WithDescription("MCP server exposing cure tools to AI agents"),
	)
	r.Register(&ServeCommand{})
	return r
}

// Name returns "serve".
func (c *ServeCommand) Name() string { return "serve" }

// Description returns a short description for help output.
func (c *ServeCommand) Description() string {
	return "Start MCP server exposing cure tools (doctor, generate)"
}

// Usage returns detailed usage information.
func (c *ServeCommand) Usage() string {
	return `Usage: cure mcp serve [--addr <host:port>]

Start an MCP (Model Context Protocol) server that exposes cure tools to AI
agents. The transport is auto-detected:
  - stdio  when stdin is a pipe (e.g. piped from an MCP host)
  - HTTP   when running interactively (binds to --addr)

Exposed tools:
  doctor               Run cure project health checks
  generate_claude_md   Generate CLAUDE.md project context file
  generate_agents_md   Generate AGENTS.md cross-tool context file
  generate_scaffold    Generate multiple AI context files (name, description,
                       language required)

Flags:
  --addr string   TCP address for HTTP transport (default "127.0.0.1:8080")

Examples:
  cure mcp serve
  cure mcp serve --addr 0.0.0.0:9090
`
}

// Flags returns the FlagSet for the serve command.
func (c *ServeCommand) Flags() *flag.FlagSet {
	fset := flag.NewFlagSet("serve", flag.ContinueOnError)
	fset.StringVar(&c.addr, "addr", "127.0.0.1:8080", "TCP address for HTTP transport")
	return fset
}

// Run starts the MCP server and blocks until ctx is cancelled or a fatal error
// occurs. Graceful shutdown is triggered by SIGINT or SIGTERM.
func (c *ServeCommand) Run(ctx context.Context, tc *terminal.Context) error {
	srv := pkgmcp.New(
		pkgmcp.WithName("cure"),
		pkgmcp.WithVersion("0.9.0"),
		pkgmcp.WithAddr(c.addr),
		pkgmcp.WithStdout(tc.Stdout),
		pkgmcp.WithStderr(tc.Stderr),
	)

	registerDoctorTool(srv)
	registerGenerateClaudeMDTool(srv)
	registerGenerateAgentsMDTool(srv)
	registerGenerateScaffoldTool(srv)

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := srv.Serve(ctx); err != nil && err != context.Canceled {
		return fmt.Errorf("mcp server: %w", err)
	}
	return nil
}

// registerDoctorTool registers the "doctor" MCP tool which runs the built-in
// project health checks and returns a text summary.
func registerDoctorTool(srv *pkgmcp.Server) {
	srv.RegisterTool(pkgmcp.FuncTool(
		"doctor",
		"Run cure project health checks against the current directory",
		pkgmcp.Schema().Build(),
		func(ctx context.Context, args map[string]any) ([]pkgmcp.Content, error) {
			var sb strings.Builder
			passed, warned, failed := pkgdoctor.Run(pkgdoctor.BuiltinChecks(), &sb)
			sb.WriteString(fmt.Sprintf("\nSummary: %d passed, %d warned, %d failed", passed, warned, failed))
			result := sb.String()
			if failed > 0 {
				return []pkgmcp.Content{pkgmcp.TextContent{Type: "text", Text: result}},
					fmt.Errorf("doctor: %d check(s) failed", failed)
			}
			return pkgmcp.Text(result), nil
		},
	))
}

// registerGenerateClaudeMDTool registers the "generate_claude_md" MCP tool
// which renders a CLAUDE.md template and returns its content as text.
func registerGenerateClaudeMDTool(srv *pkgmcp.Server) {
	schema := pkgmcp.Schema().
		String("name", "Project name", pkgmcp.Required()).
		String("description", "Project description", pkgmcp.Required()).
		String("language", "Primary programming language", pkgmcp.Required()).
		String("build_tool", "Build tool (default: make)").
		String("test_framework", "Test framework").
		String("conventions", "Comma-separated key conventions").
		Build()

	srv.RegisterTool(pkgmcp.FuncTool(
		"generate_claude_md",
		"Generate a CLAUDE.md project context file for AI assistants",
		schema,
		func(ctx context.Context, args map[string]any) ([]pkgmcp.Content, error) {
			opts, err := aiFileOptsFromArgs(args)
			if err != nil {
				return nil, err
			}

			var sb strings.Builder
			if err := generate.GenerateClaudeMD(ctx, &sb, generate.ClaudeMDOpts{AIFileOpts: opts}); err != nil {
				return nil, fmt.Errorf("generate_claude_md: %w", err)
			}
			return pkgmcp.Text(sb.String()), nil
		},
	))
}

// registerGenerateAgentsMDTool registers the "generate_agents_md" MCP tool
// which renders an AGENTS.md template and returns its content as text.
func registerGenerateAgentsMDTool(srv *pkgmcp.Server) {
	schema := pkgmcp.Schema().
		String("name", "Project name", pkgmcp.Required()).
		String("description", "Project description", pkgmcp.Required()).
		String("language", "Primary programming language", pkgmcp.Required()).
		String("build_tool", "Build tool (default: make)").
		String("test_framework", "Test framework").
		String("conventions", "Comma-separated key conventions").
		Build()

	srv.RegisterTool(pkgmcp.FuncTool(
		"generate_agents_md",
		"Generate an AGENTS.md cross-tool AI assistant context file",
		schema,
		func(ctx context.Context, args map[string]any) ([]pkgmcp.Content, error) {
			opts, err := aiFileOptsFromArgs(args)
			if err != nil {
				return nil, err
			}

			var sb strings.Builder
			if err := generate.GenerateAgentsMD(ctx, &sb, generate.AgentsMDOpts{AIFileOpts: opts}); err != nil {
				return nil, fmt.Errorf("generate_agents_md: %w", err)
			}
			return pkgmcp.Text(sb.String()), nil
		},
	))
}

// registerGenerateScaffoldTool registers the "generate_scaffold" MCP tool which
// generates all AI assistant context files in one pass, returning a combined
// text output. Name, description, and language are required.
func registerGenerateScaffoldTool(srv *pkgmcp.Server) {
	schema := pkgmcp.Schema().
		String("name", "Project name", pkgmcp.Required()).
		String("description", "Project description", pkgmcp.Required()).
		String("language", "Primary programming language", pkgmcp.Required()).
		String("build_tool", "Build tool (default: make)").
		String("test_framework", "Test framework").
		String("conventions", "Comma-separated key conventions").
		Build()

	// scaffoldTarget groups a generator name with its function.
	type scaffoldTarget struct {
		name string
		fn   func(context.Context, *strings.Builder, generate.AIFileOpts) error
	}
	targets := []scaffoldTarget{
		{
			name: "claude-md",
			fn: func(ctx context.Context, sb *strings.Builder, opts generate.AIFileOpts) error {
				return generate.GenerateClaudeMD(ctx, sb, generate.ClaudeMDOpts{AIFileOpts: opts})
			},
		},
		{
			name: "agents-md",
			fn: func(ctx context.Context, sb *strings.Builder, opts generate.AIFileOpts) error {
				return generate.GenerateAgentsMD(ctx, sb, generate.AgentsMDOpts{AIFileOpts: opts})
			},
		},
	}

	srv.RegisterTool(pkgmcp.FuncTool(
		"generate_scaffold",
		"Generate CLAUDE.md and AGENTS.md AI context files in one pass",
		schema,
		func(ctx context.Context, args map[string]any) ([]pkgmcp.Content, error) {
			opts, err := aiFileOptsFromArgs(args)
			if err != nil {
				return nil, err
			}
			// DryRun and NonInteractive are already set by aiFileOptsFromArgs;
			// set them explicitly here to document the intent for this tool.
			opts.DryRun = true
			opts.NonInteractive = true

			var combined strings.Builder
			var errs []string
			for _, tgt := range targets {
				var sb strings.Builder
				if err := tgt.fn(ctx, &sb, opts); err != nil {
					errs = append(errs, fmt.Sprintf("%s: %v", tgt.name, err))
					continue
				}
				if combined.Len() > 0 {
					combined.WriteString("\n\n---\n\n")
				}
				combined.WriteString(fmt.Sprintf("# %s\n\n", tgt.name))
				combined.WriteString(sb.String())
			}

			if len(errs) > 0 {
				return nil, fmt.Errorf("generate_scaffold: %s", strings.Join(errs, "; "))
			}
			return pkgmcp.Text(combined.String()), nil
		},
	))
}

// aiFileOptsFromArgs converts a map[string]any (from an MCP tool call) into
// a generate.AIFileOpts value. The name, description, and language fields are
// required; the rest are optional with sensible defaults.
func aiFileOptsFromArgs(args map[string]any) (generate.AIFileOpts, error) {
	name, _ := args["name"].(string)
	if name == "" {
		return generate.AIFileOpts{}, fmt.Errorf("name is required")
	}
	description, _ := args["description"].(string)
	if description == "" {
		return generate.AIFileOpts{}, fmt.Errorf("description is required")
	}
	language, _ := args["language"].(string)
	if language == "" {
		return generate.AIFileOpts{}, fmt.Errorf("language is required")
	}

	buildTool, _ := args["build_tool"].(string)
	testFramework, _ := args["test_framework"].(string)
	conventions, _ := args["conventions"].(string)

	return generate.AIFileOpts{
		Name:           name,
		Description:    description,
		Language:       language,
		BuildTool:      buildTool,
		TestFramework:  testFramework,
		Conventions:    conventions,
		DryRun:         true,  // always dry-run — return content, don't write to disk
		NonInteractive: true,  // never prompt — args come from MCP client
	}, nil
}
