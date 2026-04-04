package claudecode

// ndjsonEnvelope is the outer envelope for all events emitted by
// `claude -p --output-format stream-json --verbose`.
//
// Event type matrix:
//
//	type:"system",  subtype:"init"    → session start, carries session_id
//	type:"assistant"                  → model output; message.content[] holds text/tool_use blocks
//	type:"user"                       → tool results fed back by Claude Code; message.content[] holds tool_result blocks
//	type:"result",  subtype:"success" → final summary; carries usage and cost
//	type:"result",  subtype:"error"   → terminal error from the CLI
type ndjsonEnvelope struct {
	Type      string         `json:"type"`
	Subtype   string         `json:"subtype,omitempty"`
	SessionID string         `json:"session_id,omitempty"`
	Message   *ndjsonMessage `json:"message,omitempty"`
	Usage     *ndjsonUsage   `json:"usage,omitempty"`
	CostUSD   float64        `json:"cost_usd,omitempty"`
	Error     string         `json:"error,omitempty"`
}

// ndjsonMessage is the message object carried by assistant and user events.
type ndjsonMessage struct {
	ID      string         `json:"id"`
	Role    string         `json:"role"`
	Content []ndjsonBlock  `json:"content"`
	Usage   *ndjsonUsage   `json:"usage,omitempty"`
}

// ndjsonBlock represents a single content block within an ndjsonMessage.
// The discriminator is the Type field:
//
//   - "text"        — plain text from the model
//   - "tool_use"    — tool invocation request from the model
//   - "tool_result" — tool result returned by Claude Code
type ndjsonBlock struct {
	// Common
	Type string `json:"type"`

	// text
	Text string `json:"text,omitempty"`

	// tool_use
	ID    string         `json:"id,omitempty"`
	Name  string         `json:"name,omitempty"`
	Input map[string]any `json:"input,omitempty"`

	// tool_result
	ToolUseID string `json:"tool_use_id,omitempty"`
	// ToolName is not part of the CLI wire format; it is left empty for
	// tool_result blocks since the CLI does not echo the tool name back.
	ToolName string `json:"-"`
	// Content for tool_result can be a string or an array. The adapter
	// only reads the string form; complex content is serialised to JSON.
	Content string `json:"content,omitempty"`
	IsError bool   `json:"is_error,omitempty"`
}

// ndjsonUsage contains token accounting from the result event.
type ndjsonUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}
