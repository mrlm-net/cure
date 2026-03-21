package mcp

import "context"

// Tool represents a callable function exposed to MCP clients via tools/list
// and tools/call. Implementations define the tool's identity, input schema,
// and execution logic.
//
// Use [FuncTool] to create a Tool from a plain function without defining a
// struct, or implement the interface directly for stateful tools.
//
// Example struct implementation:
//
//	type EchoTool struct{}
//
//	func (t *EchoTool) Name() string        { return "echo" }
//	func (t *EchoTool) Description() string { return "Echoes the input message" }
//	func (t *EchoTool) Schema() mcp.InputSchema {
//	    return mcp.Schema().String("message", "Text to echo", mcp.Required()).Build()
//	}
//	func (t *EchoTool) Call(ctx context.Context, args map[string]any) ([]mcp.Content, error) {
//	    msg, _ := args["message"].(string)
//	    return mcp.Text(msg), nil
//	}
type Tool interface {
	// Name returns the tool's unique identifier as presented to MCP clients.
	Name() string

	// Description provides a human-readable explanation of what the tool does.
	Description() string

	// Schema returns the JSON Schema describing the tool's input parameters.
	Schema() InputSchema

	// Call executes the tool with the provided arguments.
	// args is a decoded JSON object where each key corresponds to a property in Schema.
	// Return (nil, err) to signal an error — the server converts this to an isError response.
	Call(ctx context.Context, args map[string]any) ([]Content, error)
}

// FuncTool creates a [Tool] from a name, description, schema, and function literal.
// This is the idiomatic way to register simple, stateless tools without defining
// a full struct.
//
// Example:
//
//	srv.RegisterTool(mcp.FuncTool(
//	    "add",
//	    "Add two integers",
//	    mcp.Schema().Integer("a", "First operand", mcp.Required()).Integer("b", "Second operand", mcp.Required()).Build(),
//	    func(ctx context.Context, args map[string]any) ([]mcp.Content, error) {
//	        a, _ := args["a"].(float64)
//	        b, _ := args["b"].(float64)
//	        return mcp.Textf("%g", a+b), nil
//	    },
//	))
func FuncTool(
	name, desc string,
	schema InputSchema,
	fn func(context.Context, map[string]any) ([]Content, error),
) Tool {
	return &funcTool{name: name, desc: desc, schema: schema, fn: fn}
}

// funcTool is the concrete implementation of Tool returned by FuncTool.
type funcTool struct {
	name   string
	desc   string
	schema InputSchema
	fn     func(context.Context, map[string]any) ([]Content, error)
}

func (t *funcTool) Name() string        { return t.name }
func (t *funcTool) Description() string { return t.desc }
func (t *funcTool) Schema() InputSchema { return t.schema }
func (t *funcTool) Call(ctx context.Context, args map[string]any) ([]Content, error) {
	return t.fn(ctx, args)
}
