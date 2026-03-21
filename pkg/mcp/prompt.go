package mcp

import "context"

// Prompt represents a reusable prompt template exposed to MCP clients via
// prompts/list and prompts/get. Prompts produce [Message] slices that clients
// use to pre-fill conversations.
//
// Example:
//
//	type SummarizePrompt struct{}
//
//	func (p *SummarizePrompt) Name() string        { return "summarize" }
//	func (p *SummarizePrompt) Description() string { return "Summarize the provided text" }
//	func (p *SummarizePrompt) Arguments() []mcp.PromptArgument {
//	    return []mcp.PromptArgument{
//	        {Name: "text", Description: "Text to summarize", Required: true},
//	    }
//	}
//	func (p *SummarizePrompt) Get(ctx context.Context, args map[string]string) ([]mcp.Message, error) {
//	    text := args["text"]
//	    return []mcp.Message{{
//	        Role:    mcp.RoleUser,
//	        Content: mcp.Text("Please summarize: " + text),
//	    }}, nil
//	}
type Prompt interface {
	// Name returns the prompt's unique identifier.
	Name() string

	// Description provides a human-readable explanation of the prompt's purpose.
	Description() string

	// Arguments describes the named parameters the prompt accepts.
	// Clients use this list to render input forms before calling prompts/get.
	Arguments() []PromptArgument

	// Get renders the prompt with the provided argument values.
	// args maps argument names to their string values as supplied by the client.
	// Returns an ordered list of messages that form the prompt conversation.
	Get(ctx context.Context, args map[string]string) ([]Message, error)
}
