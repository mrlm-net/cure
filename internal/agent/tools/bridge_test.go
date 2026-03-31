package tools_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mrlm-net/cure/internal/agent/tools"
	"github.com/mrlm-net/cure/pkg/mcp"
)

// TestToolsFromMCPServer_Empty verifies that a server with no tools
// returns a nil/empty slice rather than panicking.
func TestToolsFromMCPServer_Empty(t *testing.T) {
	srv := mcp.New()
	result := tools.ToolsFromMCPServer(srv)
	if len(result) != 0 {
		t.Errorf("expected empty slice for empty server, got %d tools", len(result))
	}
}

// TestToolsFromMCPServer_NameAndDescription verifies that the bridge
// preserves the name and description from the MCP tool.
func TestToolsFromMCPServer_NameAndDescription(t *testing.T) {
	srv := mcp.New()
	srv.RegisterTool(mcp.FuncTool(
		"echo",
		"Echoes the input back",
		mcp.Schema().String("message", "The message", mcp.Required()).Build(),
		func(_ context.Context, args map[string]any) ([]mcp.Content, error) {
			msg, _ := args["message"].(string)
			return mcp.Text(msg), nil
		},
	))

	agentTools := tools.ToolsFromMCPServer(srv)
	if len(agentTools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(agentTools))
	}
	t.Run("name preserved", func(t *testing.T) {
		if agentTools[0].Name() != "echo" {
			t.Errorf("Name() = %q, want %q", agentTools[0].Name(), "echo")
		}
	})
	t.Run("description preserved", func(t *testing.T) {
		if agentTools[0].Description() != "Echoes the input back" {
			t.Errorf("Description() = %q, want %q", agentTools[0].Description(), "Echoes the input back")
		}
	})
	t.Run("schema has type object", func(t *testing.T) {
		schema := agentTools[0].Schema()
		if schema["type"] != "object" {
			t.Errorf("schema[type] = %v, want %q", schema["type"], "object")
		}
	})
}

// TestToolsFromMCPServer_CallDelegates verifies that calling the agent tool
// delegates to the underlying MCP tool and returns its text result.
func TestToolsFromMCPServer_CallDelegates(t *testing.T) {
	srv := mcp.New()
	srv.RegisterTool(mcp.FuncTool(
		"greet",
		"Returns a greeting",
		mcp.Schema().String("name", "The name to greet", mcp.Required()).Build(),
		func(_ context.Context, args map[string]any) ([]mcp.Content, error) {
			name, _ := args["name"].(string)
			return mcp.Text("Hello, " + name + "!"), nil
		},
	))

	agentTools := tools.ToolsFromMCPServer(srv)
	if len(agentTools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(agentTools))
	}

	result, err := agentTools[0].Call(context.Background(), map[string]any{"name": "World"})
	if err != nil {
		t.Fatalf("Call() returned unexpected error: %v", err)
	}
	if result != "Hello, World!" {
		t.Errorf("Call() = %q, want %q", result, "Hello, World!")
	}
}

// TestToolsFromMCPServer_CallPropagatesErrors verifies that errors from the
// underlying MCP tool are wrapped and returned through Call().
func TestToolsFromMCPServer_CallPropagatesErrors(t *testing.T) {
	sentinelErr := errors.New("tool exploded")

	srv := mcp.New()
	srv.RegisterTool(mcp.FuncTool(
		"boom",
		"Always fails",
		mcp.Schema().Build(),
		func(_ context.Context, _ map[string]any) ([]mcp.Content, error) {
			return nil, sentinelErr
		},
	))

	agentTools := tools.ToolsFromMCPServer(srv)
	if len(agentTools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(agentTools))
	}

	result, err := agentTools[0].Call(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error, got result %q", result)
	}
	if !errors.Is(err, sentinelErr) {
		t.Errorf("errors.Is(err, sentinelErr) = false; err = %v", err)
	}
}

// TestToolsFromMCPServer_MultipleTools verifies that all registered tools
// are returned in registration order.
func TestToolsFromMCPServer_MultipleTools(t *testing.T) {
	srv := mcp.New()
	names := []string{"alpha", "beta", "gamma"}
	for _, name := range names {
		n := name // capture
		srv.RegisterTool(mcp.FuncTool(
			n, "desc of "+n,
			mcp.Schema().Build(),
			func(_ context.Context, _ map[string]any) ([]mcp.Content, error) {
				return mcp.Text(n), nil
			},
		))
	}

	agentTools := tools.ToolsFromMCPServer(srv)
	if len(agentTools) != len(names) {
		t.Fatalf("expected %d tools, got %d", len(names), len(agentTools))
	}
	for i, wantName := range names {
		if agentTools[i].Name() != wantName {
			t.Errorf("tool[%d].Name() = %q, want %q", i, agentTools[i].Name(), wantName)
		}
	}
}
