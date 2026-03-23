package agent

import (
	"context"
	"errors"
	"testing"
)

// RunSessionStoreTests is a shared test suite for [SessionStore] implementations.
// Call it from each concrete store's test file:
//
//	func TestMyStore(t *testing.T) {
//	    store := NewMyStore(t.TempDir())
//	    agent.RunSessionStoreTests(t, store)
//	}
func RunSessionStoreTests(t *testing.T, store SessionStore) {
	t.Helper()
	ctx := context.Background()

	t.Run("Save and Load round-trip", func(t *testing.T) {
		s := NewSession("claude", "claude-opus-4-5")
		s.AppendUserMessage("hello")

		if err := store.Save(ctx, s); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := store.Load(ctx, s.ID)
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got.ID != s.ID {
			t.Errorf("ID = %q, want %q", got.ID, s.ID)
		}
		if len(got.History) != 1 {
			t.Errorf("History len = %d, want 1", len(got.History))
		}
	})
	t.Run("Load returns ErrSessionNotFound for unknown ID", func(t *testing.T) {
		_, err := store.Load(ctx, "000000000000000000000000deadbeef")
		if !errors.Is(err, ErrSessionNotFound) {
			t.Errorf("expected ErrSessionNotFound, got %v", err)
		}
	})
	t.Run("List returns saved sessions", func(t *testing.T) {
		// Use a fresh context — this test depends on prior Save in same store.
		s := NewSession("openai", "gpt-4o")
		if err := store.Save(ctx, s); err != nil {
			t.Fatalf("Save: %v", err)
		}
		sessions, err := store.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(sessions) == 0 {
			t.Error("List returned 0 sessions, expected at least 1")
		}
	})
	t.Run("Delete removes session", func(t *testing.T) {
		s := NewSession("claude", "claude-opus-4-5")
		if err := store.Save(ctx, s); err != nil {
			t.Fatalf("Save: %v", err)
		}
		if err := store.Delete(ctx, s.ID); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		_, err := store.Load(ctx, s.ID)
		if !errors.Is(err, ErrSessionNotFound) {
			t.Errorf("after Delete, Load expected ErrSessionNotFound, got %v", err)
		}
	})
	t.Run("Delete returns ErrSessionNotFound for unknown ID", func(t *testing.T) {
		err := store.Delete(ctx, "000000000000000000000000deadbeef")
		if !errors.Is(err, ErrSessionNotFound) {
			t.Errorf("expected ErrSessionNotFound, got %v", err)
		}
	})
	t.Run("Fork creates independent copy", func(t *testing.T) {
		orig := NewSession("claude", "claude-opus-4-5")
		orig.AppendUserMessage("original message")
		if err := store.Save(ctx, orig); err != nil {
			t.Fatalf("Save orig: %v", err)
		}
		forked, err := store.Fork(ctx, orig.ID)
		if err != nil {
			t.Fatalf("Fork: %v", err)
		}
		if forked.ID == orig.ID {
			t.Error("forked session has same ID as original")
		}
		if forked.ForkOf != orig.ID {
			t.Errorf("ForkOf = %q, want %q", forked.ForkOf, orig.ID)
		}
		if len(forked.History) != 1 {
			t.Errorf("History len = %d, want 1", len(forked.History))
		}
		// Verify forked session is persisted and loadable
		loaded, err := store.Load(ctx, forked.ID)
		if err != nil {
			t.Fatalf("Load forked: %v", err)
		}
		if loaded.ForkOf != orig.ID {
			t.Errorf("loaded ForkOf = %q, want %q", loaded.ForkOf, orig.ID)
		}
	})
	t.Run("Fork returns ErrSessionNotFound for unknown ID", func(t *testing.T) {
		_, err := store.Fork(ctx, "000000000000000000000000deadbeef")
		if !errors.Is(err, ErrSessionNotFound) {
			t.Errorf("expected ErrSessionNotFound, got %v", err)
		}
	})
}
