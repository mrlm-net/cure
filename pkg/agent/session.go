package agent

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Session holds the full state of a conversation with an AI provider.
// Sessions are persisted via [SessionStore].
type Session struct {
	ID           string    `json:"id"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	SystemPrompt string    `json:"system_prompt,omitempty"`
	History      []Message `json:"history"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ForkOf       string    `json:"fork_of,omitempty"`
	Tags         []string  `json:"tags,omitempty"`
}

// NewSession creates a new Session for the given provider and model.
// Panics if the operating system's cryptographic random source is unavailable.
func NewSession(provider, model string) *Session {
	now := time.Now().UTC()
	return &Session{
		ID:        newSessionID(),
		Provider:  provider,
		Model:     model,
		History:   []Message{},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Fork returns a deep copy of the session with a new ID and ForkOf set to the
// original session's ID. The forked session shares no mutable state with the original.
// Panics if the operating system's cryptographic random source is unavailable.
func (s *Session) Fork() *Session {
	now := time.Now().UTC()
	history := make([]Message, len(s.History))
	copy(history, s.History)

	var tags []string
	if len(s.Tags) > 0 {
		tags = make([]string, len(s.Tags))
		copy(tags, s.Tags)
	}

	return &Session{
		ID:           newSessionID(),
		Provider:     s.Provider,
		Model:        s.Model,
		SystemPrompt: s.SystemPrompt,
		History:      history,
		CreatedAt:    now,
		UpdatedAt:    now,
		ForkOf:       s.ID,
		Tags:         tags,
	}
}

// AppendUserMessage appends a user message to the session history and updates UpdatedAt.
func (s *Session) AppendUserMessage(content string) {
	s.History = append(s.History, Message{Role: RoleUser, Content: content})
	s.UpdatedAt = time.Now().UTC()
}

// AppendAssistantMessage appends an assistant message to the session history and updates UpdatedAt.
func (s *Session) AppendAssistantMessage(content string) {
	s.History = append(s.History, Message{Role: RoleAssistant, Content: content})
	s.UpdatedAt = time.Now().UTC()
}

// newSessionID generates a 32-character hex-encoded session ID using crypto/rand.
// It panics if the random source fails (should never happen on a healthy OS).
func newSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("agent: failed to generate session ID: " + err.Error())
	}
	return hex.EncodeToString(b)
}
