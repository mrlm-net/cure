package agent_test

import (
	"errors"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
)

func TestSentinelErrors(t *testing.T) {
	t.Run("ErrProviderNotFound is distinct", func(t *testing.T) {
		if agent.ErrProviderNotFound == nil {
			t.Fatal("ErrProviderNotFound is nil")
		}
	})
	t.Run("ErrSessionNotFound is distinct", func(t *testing.T) {
		if agent.ErrSessionNotFound == nil {
			t.Fatal("ErrSessionNotFound is nil")
		}
	})
	t.Run("ErrCountNotSupported is distinct", func(t *testing.T) {
		if agent.ErrCountNotSupported == nil {
			t.Fatal("ErrCountNotSupported is nil")
		}
	})
	t.Run("sentinels are distinct from each other", func(t *testing.T) {
		if errors.Is(agent.ErrProviderNotFound, agent.ErrSessionNotFound) {
			t.Error("ErrProviderNotFound matches ErrSessionNotFound")
		}
		if errors.Is(agent.ErrProviderNotFound, agent.ErrCountNotSupported) {
			t.Error("ErrProviderNotFound matches ErrCountNotSupported")
		}
		if errors.Is(agent.ErrSessionNotFound, agent.ErrCountNotSupported) {
			t.Error("ErrSessionNotFound matches ErrCountNotSupported")
		}
	})
	t.Run("New wraps ErrProviderNotFound", func(t *testing.T) {
		_, err := agent.New("nonexistent-provider", nil)
		if !errors.Is(err, agent.ErrProviderNotFound) {
			t.Errorf("expected ErrProviderNotFound, got %v", err)
		}
	})
}
