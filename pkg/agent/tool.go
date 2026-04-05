package agent

import "context"

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

// FuncTool creates a Tool from a name, description, schema map, and function.
// This is a convenience constructor to avoid defining a new struct for simple tools.
//
// The schema parameter must be a non-nil JSON Schema object with "type": "object".
// The minimal valid schema is:
//
//	map[string]any{"type": "object"}
//
// FuncTool panics at registration time if schema is nil or if schema["type"] is
// not "object". Early panics catch misconfigured tools before any agent session
// starts.
func FuncTool(
	name, desc string,
	schema map[string]any,
	fn func(context.Context, map[string]any) (string, error),
) Tool {
	if schema == nil {
		panic(`agent.FuncTool: tool "` + name + `" has nil schema; schema must be a JSON Schema object (map["type"]="object")`)
	}
	if typ, _ := schema["type"].(string); typ != "object" {
		panic(`agent.FuncTool: tool "` + name + `" schema["type"] is "` + typ + `", expected "object"`)
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
