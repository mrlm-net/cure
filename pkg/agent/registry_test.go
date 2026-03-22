package agent

import (
	"context"
	"errors"
	"iter"
	"testing"
)

// stubAgent is a minimal Agent implementation for registry tests.
type stubAgent struct{ provider string }

func (s *stubAgent) Run(_ context.Context, _ *Session) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {}
}
func (s *stubAgent) CountTokens(_ context.Context, _ *Session) (int, error) {
	return 0, ErrCountNotSupported
}
func (s *stubAgent) Provider() string { return s.provider }

func stubFactory(name string) AgentFactory {
	return func(_ map[string]any) (Agent, error) {
		return &stubAgent{provider: name}, nil
	}
}

func TestRegister(t *testing.T) {
	resetRegistry()
	t.Cleanup(resetRegistry)

	t.Run("registers successfully", func(t *testing.T) {
		Register("alpha", stubFactory("alpha"))
		names := Registered()
		if len(names) != 1 || names[0] != "alpha" {
			t.Errorf("Registered() = %v, want [alpha]", names)
		}
	})
	t.Run("panics on empty name", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for empty name")
			}
		}()
		Register("", stubFactory(""))
	})
	t.Run("panics on duplicate name", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for duplicate registration")
			}
		}()
		Register("alpha", stubFactory("alpha"))
	})
}

func TestNew(t *testing.T) {
	resetRegistry()
	t.Cleanup(resetRegistry)
	Register("beta", stubFactory("beta"))

	t.Run("creates agent for registered provider", func(t *testing.T) {
		a, err := New("beta", nil)
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if a.Provider() != "beta" {
			t.Errorf("Provider() = %q, want %q", a.Provider(), "beta")
		}
	})
	t.Run("returns ErrProviderNotFound for unknown provider", func(t *testing.T) {
		_, err := New("unknown", nil)
		if !errors.Is(err, ErrProviderNotFound) {
			t.Errorf("expected ErrProviderNotFound, got %v", err)
		}
	})
}

func TestRegistered(t *testing.T) {
	resetRegistry()
	t.Cleanup(resetRegistry)

	t.Run("empty when no providers registered", func(t *testing.T) {
		names := Registered()
		if len(names) != 0 {
			t.Errorf("Registered() = %v, want []", names)
		}
	})
	t.Run("returns sorted names", func(t *testing.T) {
		Register("zebra", stubFactory("zebra"))
		Register("alpha", stubFactory("alpha"))
		Register("mango", stubFactory("mango"))
		names := Registered()
		want := []string{"alpha", "mango", "zebra"}
		for i, n := range names {
			if n != want[i] {
				t.Errorf("Registered()[%d] = %q, want %q", i, n, want[i])
			}
		}
	})
}
