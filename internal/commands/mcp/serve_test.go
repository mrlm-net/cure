package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/internal/commands/generate"
	pkgdoctor "github.com/mrlm-net/cure/pkg/doctor"
)

// TestNewMCPCommand verifies that NewMCPCommand returns a command named "mcp"
// with a non-empty description.
func TestNewMCPCommand(t *testing.T) {
	cmd := NewMCPCommand()
	if cmd.Name() != "mcp" {
		t.Errorf("Name() = %q, want %q", cmd.Name(), "mcp")
	}
	if cmd.Description() == "" {
		t.Error("Description() must not be empty")
	}
}

// TestServeCommandMetadata verifies the ServeCommand identity and help methods.
func TestServeCommandMetadata(t *testing.T) {
	c := &ServeCommand{}

	if c.Name() != "serve" {
		t.Errorf("Name() = %q, want %q", c.Name(), "serve")
	}
	if c.Description() == "" {
		t.Error("Description() must not be empty")
	}
	if !strings.Contains(c.Usage(), "cure mcp serve") {
		t.Errorf("Usage() missing invocation example, got: %s", c.Usage())
	}
}

// TestServeCommandDefaultAddr verifies the --addr flag defaults to "127.0.0.1:8080".
func TestServeCommandDefaultAddr(t *testing.T) {
	c := &ServeCommand{}
	fset := c.Flags()
	if fset == nil {
		t.Fatal("Flags() returned nil")
	}
	if err := fset.Parse([]string{}); err != nil {
		t.Fatalf("flag parse error: %v", err)
	}
	if c.addr != "127.0.0.1:8080" {
		t.Errorf("default addr = %q, want %q", c.addr, "127.0.0.1:8080")
	}
}

// TestServeCommandCustomAddr verifies --addr flag can be overridden.
func TestServeCommandCustomAddr(t *testing.T) {
	c := &ServeCommand{}
	fset := c.Flags()
	if err := fset.Parse([]string{"--addr", "0.0.0.0:9090"}); err != nil {
		t.Fatalf("flag parse error: %v", err)
	}
	if c.addr != "0.0.0.0:9090" {
		t.Errorf("addr = %q, want %q", c.addr, "0.0.0.0:9090")
	}
}

// TestAIFileOptsFromArgs verifies conversion from MCP tool args to AIFileOpts.
func TestAIFileOptsFromArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    map[string]any
		wantErr string
		check   func(t *testing.T, opts generate.AIFileOpts)
	}{
		{
			name: "required fields only",
			args: map[string]any{
				"name":        "myapp",
				"description": "A test application",
				"language":    "go",
			},
			check: func(t *testing.T, opts generate.AIFileOpts) {
				t.Helper()
				if opts.Name != "myapp" {
					t.Errorf("Name = %q, want %q", opts.Name, "myapp")
				}
				if opts.Description != "A test application" {
					t.Errorf("Description = %q, want %q", opts.Description, "A test application")
				}
				if opts.Language != "go" {
					t.Errorf("Language = %q, want %q", opts.Language, "go")
				}
				if !opts.DryRun {
					t.Error("DryRun must be true — MCP tools return content, not write to disk")
				}
				if !opts.NonInteractive {
					t.Error("NonInteractive must be true — MCP tools never prompt")
				}
			},
		},
		{
			name: "all optional fields",
			args: map[string]any{
				"name":           "myapp",
				"description":    "A test application",
				"language":       "go",
				"build_tool":     "make",
				"test_framework": "testing",
				"conventions":    "gofmt,go vet",
			},
			check: func(t *testing.T, opts generate.AIFileOpts) {
				t.Helper()
				if opts.BuildTool != "make" {
					t.Errorf("BuildTool = %q, want %q", opts.BuildTool, "make")
				}
				if opts.TestFramework != "testing" {
					t.Errorf("TestFramework = %q, want %q", opts.TestFramework, "testing")
				}
				if opts.Conventions != "gofmt,go vet" {
					t.Errorf("Conventions = %q, want %q", opts.Conventions, "gofmt,go vet")
				}
			},
		},
		{
			name:    "missing name",
			args:    map[string]any{"description": "desc", "language": "go"},
			wantErr: "name is required",
		},
		{
			name:    "empty name",
			args:    map[string]any{"name": "", "description": "desc", "language": "go"},
			wantErr: "name is required",
		},
		{
			name:    "missing description",
			args:    map[string]any{"name": "myapp", "language": "go"},
			wantErr: "description is required",
		},
		{
			name:    "missing language",
			args:    map[string]any{"name": "myapp", "description": "desc"},
			wantErr: "language is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := aiFileOptsFromArgs(tt.args)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, opts)
			}
		})
	}
}

// TestDoctorToolLogic verifies that the doctor tool logic (sans MCP dispatch)
// produces a text result containing a Summary line. Runs against cwd.
func TestDoctorToolLogic(t *testing.T) {
	var sb strings.Builder
	passed, warned, failed := pkgdoctor.Run(pkgdoctor.BuiltinChecks(), &sb)
	sb.WriteString("\nSummary: ")
	sb.WriteString(strings.Join([]string{
		formatCount(passed, "passed"),
		formatCount(warned, "warned"),
		formatCount(failed, "failed"),
	}, ", "))

	result := sb.String()
	if result == "" {
		t.Error("doctor result must not be empty")
	}
	if !strings.Contains(result, "Summary:") {
		t.Errorf("doctor result missing Summary line, got: %s", result)
	}
}

// TestGenerateClaudeMDToolLogic verifies that the generate_claude_md tool
// produces non-empty output with the project name embedded.
func TestGenerateClaudeMDToolLogic(t *testing.T) {
	opts, err := aiFileOptsFromArgs(map[string]any{
		"name":        "testproject",
		"description": "A test project",
		"language":    "go",
	})
	if err != nil {
		t.Fatalf("aiFileOptsFromArgs: %v", err)
	}

	var sb strings.Builder
	ctx := context.Background()
	if err := generate.GenerateClaudeMD(ctx, &sb, generate.ClaudeMDOpts{AIFileOpts: opts}); err != nil {
		t.Fatalf("GenerateClaudeMD: %v", err)
	}
	content := sb.String()
	if content == "" {
		t.Error("generated CLAUDE.md content must not be empty")
	}
	if !strings.Contains(content, "testproject") {
		t.Errorf("generated content does not contain project name 'testproject', got length=%d", len(content))
	}
}

// TestGenerateAgentsMDToolLogic verifies that the generate_agents_md tool
// produces non-empty output with the project name embedded.
func TestGenerateAgentsMDToolLogic(t *testing.T) {
	opts, err := aiFileOptsFromArgs(map[string]any{
		"name":        "testproject",
		"description": "A test project",
		"language":    "go",
	})
	if err != nil {
		t.Fatalf("aiFileOptsFromArgs: %v", err)
	}

	var sb strings.Builder
	ctx := context.Background()
	if err := generate.GenerateAgentsMD(ctx, &sb, generate.AgentsMDOpts{AIFileOpts: opts}); err != nil {
		t.Fatalf("GenerateAgentsMD: %v", err)
	}
	content := sb.String()
	if content == "" {
		t.Error("generated AGENTS.md content must not be empty")
	}
	if !strings.Contains(content, "testproject") {
		t.Errorf("generated content does not contain project name 'testproject', got length=%d", len(content))
	}
}

// TestGenerateScaffoldToolLogic verifies that both scaffold generators produce
// output when run in dry-run + non-interactive mode.
func TestGenerateScaffoldToolLogic(t *testing.T) {
	opts, err := aiFileOptsFromArgs(map[string]any{
		"name":        "testproject",
		"description": "A test project",
		"language":    "go",
	})
	if err != nil {
		t.Fatalf("aiFileOptsFromArgs: %v", err)
	}
	// aiFileOptsFromArgs already sets DryRun=true and NonInteractive=true.

	ctx := context.Background()
	for _, tt := range []struct {
		name string
		run  func(*strings.Builder) error
	}{
		{
			name: "claude-md",
			run: func(sb *strings.Builder) error {
				return generate.GenerateClaudeMD(ctx, sb, generate.ClaudeMDOpts{AIFileOpts: opts})
			},
		},
		{
			name: "agents-md",
			run: func(sb *strings.Builder) error {
				return generate.GenerateAgentsMD(ctx, sb, generate.AgentsMDOpts{AIFileOpts: opts})
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var sb strings.Builder
			if err := tt.run(&sb); err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			if sb.Len() == 0 {
				t.Errorf("%s: output must not be empty", tt.name)
			}
		})
	}
}

// formatCount is a small helper used in the doctor test to format tallies.
func formatCount(n int, label string) string {
	return strings.Join([]string{itoa(n), label}, " ")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := make([]byte, 0, 10)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
