package agent

import (
	"context"
	"fmt"
)

// Tool is a callable function that an Agent may invoke during an agentic session.
// Use [FuncTool] to create a Tool from a plain function without defining a struct.
type Tool interface {
	// Name returns the unique name of the tool.
	Name() string
	// Description returns a human-readable description of what the tool does.
	Description() string
	// Schema returns a JSON Schema map describing the tool's input parameters.
	Schema() map[string]any
	// Call invokes the tool with the given arguments and returns a string result.
	Call(ctx context.Context, args map[string]any) (string, error)
}

// FuncTool creates a [Tool] from a name, description, JSON Schema map, and function.
//
// This is a convenience constructor to avoid defining a new struct for simple tools.
//
// Schema contract: schema must not be nil, and schema["type"] must equal "object".
// Violating either constraint causes a panic at construction time with a descriptive
// message that includes the tool name. Example of a valid minimal schema:
//
//	map[string]any{
//	    "type": "object",
//	    "properties": map[string]any{},
//	}
func FuncTool(
	name, desc string,
	schema map[string]any,
	fn func(context.Context, map[string]any) (string, error),
) Tool {
	if schema == nil {
		panic("agent.FuncTool: schema must not be nil for tool " + name +
			`; provide a JSON Schema map with "type": "object"`)
	}
	if typ, _ := schema["type"].(string); typ != "object" {
		panic("agent.FuncTool: schema[\"type\"] must be \"object\" for tool " + name +
			"; got " + fmt.Sprintf("%v", schema["type"]))
	}
	return &funcTool{name: name, desc: desc, schema: schema, fn: fn}
}

type funcTool struct {
	name   string
	desc   string
	schema map[string]any
	fn     func(context.Context, map[string]any) (string, error)
}

func (t *funcTool) Name() string               { return t.name }
func (t *funcTool) Description() string        { return t.desc }
func (t *funcTool) Schema() map[string]any     { return t.schema }
func (t *funcTool) Call(ctx context.Context, args map[string]any) (string, error) {
	return t.fn(ctx, args)
}
