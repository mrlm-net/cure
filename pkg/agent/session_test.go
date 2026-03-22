package agent_test

import (
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
)

func TestNewSession(t *testing.T) {
	t.Run("creates session with required fields", func(t *testing.T) {
		s := agent.NewSession("claude", "claude-opus-4-5")
		if s.ID == "" {
			t.Error("ID is empty")
		}
		if len(s.ID) != 32 {
			t.Errorf("ID length = %d, want 32", len(s.ID))
		}
		if s.Provider != "claude" {
			t.Errorf("Provider = %q, want %q", s.Provider, "claude")
		}
		if s.Model != "claude-opus-4-5" {
			t.Errorf("Model = %q, want %q", s.Model, "claude-opus-4-5")
		}
		if s.History == nil {
			t.Error("History is nil, want empty slice")
		}
		if len(s.History) != 0 {
			t.Errorf("History len = %d, want 0", len(s.History))
		}
		if s.CreatedAt.IsZero() {
			t.Error("CreatedAt is zero")
		}
		if s.UpdatedAt.IsZero() {
			t.Error("UpdatedAt is zero")
		}
		if s.ForkOf != "" {
			t.Errorf("ForkOf = %q, want empty", s.ForkOf)
		}
		if s.Tags != nil {
			t.Errorf("Tags = %v, want nil", s.Tags)
		}
	})
	t.Run("generates unique IDs", func(t *testing.T) {
		s1 := agent.NewSession("p", "m")
		s2 := agent.NewSession("p", "m")
		if s1.ID == s2.ID {
			t.Errorf("duplicate session IDs: %q", s1.ID)
		}
	})
}

func TestSessionFork(t *testing.T) {
	t.Run("fork has new ID and ForkOf set", func(t *testing.T) {
		orig := agent.NewSession("claude", "claude-opus-4-5")
		fork := orig.Fork()

		if fork.ID == orig.ID {
			t.Error("fork ID should differ from original")
		}
		if fork.ForkOf != orig.ID {
			t.Errorf("ForkOf = %q, want %q", fork.ForkOf, orig.ID)
		}
	})
	t.Run("fork deep copies history", func(t *testing.T) {
		orig := agent.NewSession("claude", "claude-opus-4-5")
		orig.AppendUserMessage("hello")
		fork := orig.Fork()

		fork.AppendUserMessage("fork-only message")
		if len(orig.History) != 1 {
			t.Errorf("original history modified: len = %d", len(orig.History))
		}
	})
	t.Run("fork with nil tags stays nil", func(t *testing.T) {
		orig := agent.NewSession("claude", "claude-opus-4-5")
		fork := orig.Fork()
		if fork.Tags != nil {
			t.Errorf("Tags = %v, want nil", fork.Tags)
		}
	})
	t.Run("fork deep copies non-nil tags", func(t *testing.T) {
		orig := agent.NewSession("claude", "claude-opus-4-5")
		orig.Tags = []string{"important", "prod"}
		fork := orig.Fork()

		fork.Tags[0] = "mutated"
		if orig.Tags[0] != "important" {
			t.Errorf("original Tags[0] = %q, mutating fork affected original", orig.Tags[0])
		}
	})
	t.Run("fork copies SystemPrompt and Model", func(t *testing.T) {
		orig := agent.NewSession("claude", "claude-opus-4-5")
		orig.SystemPrompt = "be helpful"
		fork := orig.Fork()
		if fork.SystemPrompt != orig.SystemPrompt {
			t.Errorf("SystemPrompt = %q, want %q", fork.SystemPrompt, orig.SystemPrompt)
		}
		if fork.Model != orig.Model {
			t.Errorf("Model = %q, want %q", fork.Model, orig.Model)
		}
	})
}

func TestSessionAppend(t *testing.T) {
	t.Run("AppendUserMessage updates History and UpdatedAt", func(t *testing.T) {
		s := agent.NewSession("p", "m")
		before := s.UpdatedAt
		s.AppendUserMessage("hello")
		if len(s.History) != 1 {
			t.Fatalf("History len = %d, want 1", len(s.History))
		}
		if s.History[0].Role != agent.RoleUser {
			t.Errorf("Role = %q, want %q", s.History[0].Role, agent.RoleUser)
		}
		if s.History[0].Content != "hello" {
			t.Errorf("Content = %q, want %q", s.History[0].Content, "hello")
		}
		// UpdatedAt must be set to at least the pre-call time.
		// We use !Before rather than After to avoid flakiness when both
		// time.Now() calls land within the same clock tick.
		if s.UpdatedAt.Before(before) {
			t.Error("UpdatedAt regressed after AppendUserMessage")
		}
	})
	t.Run("AppendAssistantMessage sets RoleAssistant", func(t *testing.T) {
		s := agent.NewSession("p", "m")
		s.AppendAssistantMessage("hi there")
		if s.History[0].Role != agent.RoleAssistant {
			t.Errorf("Role = %q, want %q", s.History[0].Role, agent.RoleAssistant)
		}
	})
}
