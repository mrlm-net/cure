// Package tools provides the bridge between pkg/mcp and pkg/agent.
// It is the ONLY place in the codebase where these two packages are connected;
// neither pkg/agent nor pkg/mcp may import the other.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/mcp"
)

// ToolsFromMCPServer converts the tools registered on an MCP server into
// pkg/agent Tool values. Each tool's Call delegates to the underlying mcp.Tool.
// Returns an empty slice when the server has no registered tools.
func ToolsFromMCPServer(srv *mcp.Server) []agent.Tool {
	mcpTools := srv.Tools()
	if len(mcpTools) == 0 {
		return nil
	}

	out := make([]agent.Tool, len(mcpTools))
	for i, t := range mcpTools {
		// Capture loop variable for the closure.
		mcpTool := t
		schema := schemaFromMCP(mcpTool.Schema())
		callFn := func(ctx context.Context, args map[string]any) (string, error) {
			contents, err := mcpTool.Call(ctx, args)
			if err != nil {
				return "", fmt.Errorf("tools: mcp tool %q failed: %w", mcpTool.Name(), err)
			}
			return contentsToString(contents), nil
		}
		out[i] = agent.FuncTool(mcpTool.Name(), mcpTool.Description(), schema, callFn)
	}
	return out
}

// schemaFromMCP converts an mcp.InputSchema into the map[string]any form
// expected by agent.Tool.Schema(). A round-trip through JSON is used to
// ensure that any future additions to InputSchema are preserved without
// requiring code changes here.
func schemaFromMCP(s mcp.InputSchema) map[string]any {
	b, err := json.Marshal(s)
	if err != nil {
		// InputSchema is a simple value type — marshalling should never fail.
		return map[string]any{"type": "object"}
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return map[string]any{"type": "object"}
	}
	return m
}

// contentsToString converts an mcp.Content slice into a single string result
// suitable for returning from an agent tool call. Only TextContent values are
// extracted; other content types (image, resource) are omitted.
func contentsToString(contents []mcp.Content) string {
	parts := make([]string, 0, len(contents))
	for _, c := range contents {
		if tc, ok := c.(mcp.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}
	return strings.Join(parts, "\n")
}
