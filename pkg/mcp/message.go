package mcp

// Role identifies who authored a message in a prompt conversation.
type Role string

const (
	// RoleUser indicates the message was authored by the human user.
	RoleUser Role = "user"

	// RoleAssistant indicates the message was authored by the AI assistant.
	RoleAssistant Role = "assistant"
)

// Message is a single turn in a prompt conversation, consisting of a role and
// one or more content blocks. MCP prompts return a slice of Messages that clients
// use to pre-fill a conversation.
type Message struct {
	Role    Role      `json:"role"`
	Content []Content `json:"content"`
}

// PromptArgument describes a named parameter accepted by a [Prompt]. Clients use
// the argument list to render input forms before calling prompts/get.
type PromptArgument struct {
	// Name is the argument identifier used as a key in the arguments map.
	Name string `json:"name"`

	// Description is a short human-readable explanation of the argument.
	Description string `json:"description,omitempty"`

	// Required indicates whether the argument must be supplied by the client.
	Required bool `json:"required,omitempty"`
}
