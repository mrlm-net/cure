package agent

import (
	"context"
	"iter"
)

// Agent is a provider-agnostic interface for running AI inference.
// Concrete implementations are registered in internal/agent/<provider>/ and
// self-register via init() using the blank-import driver pattern.
type Agent interface {
	// Run streams a response for the given session.
	// The caller iterates events with: for ev, err := range a.Run(ctx, session) { ... }
	// Cancelling ctx terminates the stream.
	Run(ctx context.Context, session *Session) iter.Seq2[Event, error]

	// CountTokens returns the token count for the session's messages.
	// Returns [ErrCountNotSupported] if the provider does not implement this operation.
	CountTokens(ctx context.Context, session *Session) (int, error)

	// Provider returns the provider name this agent was created with (e.g. "claude").
	Provider() string
}

// AgentFactory is a constructor function for an [Agent] implementation.
// cfg is provider-specific configuration (API keys, model, temperature, etc.).
type AgentFactory func(cfg map[string]any) (Agent, error)

// EventKind classifies a streaming event from [Agent.Run].
type EventKind string

const (
	// EventKindToken carries a partial text token from the model.
	EventKindToken EventKind = "token"
	// EventKindStart marks the beginning of a model response turn.
	EventKindStart EventKind = "start"
	// EventKindDone marks the successful end of a model response turn.
	EventKindDone EventKind = "done"
	// EventKindError carries a terminal error from the provider.
	EventKindError EventKind = "error"
)

// Event is a single item in an [Agent.Run] stream.
// Only fields relevant to the [EventKind] are populated.
type Event struct {
	Kind         EventKind `json:"kind"`
	Text         string    `json:"text,omitempty"`
	InputTokens  int       `json:"input_tokens,omitempty"`
	OutputTokens int       `json:"output_tokens,omitempty"`
	StopReason   string    `json:"stop_reason,omitempty"`
	Err          string    `json:"error,omitempty"`
}

// Role is the participant role in a conversation message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// Message is a single turn in a conversation history.
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}
