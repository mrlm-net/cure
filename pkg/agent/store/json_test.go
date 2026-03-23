package store_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/agent/store"
)

func newStore(t *testing.T) *store.JSONStore {
	t.Helper()
	s, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	return s
}

// TestJSONStore runs the shared SessionStore compliance suite.
func TestJSONStore(t *testing.T) {
	agent.RunSessionStoreTests(t, newStore(t))
}

// TestJSONStore_ConcurrentSave verifies concurrent saves do not race.
func TestJSONStore_ConcurrentSave(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()
	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			sess := agent.NewSession("claude", "claude-opus-4-6")
			if err := s.Save(ctx, sess); err != nil {
				t.Errorf("Save: %v", err)
			}
		}()
	}
	wg.Wait()
	sessions, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != n {
		t.Errorf("List returned %d sessions, want %d", len(sessions), n)
	}
}

// TestJSONStore_ListEmpty verifies List returns a non-nil empty slice when the
// directory does not exist.
func TestJSONStore_ListEmpty(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	s, err := store.NewJSONStore(dir)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	sessions, err := s.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if sessions == nil {
		t.Error("List returned nil, want non-nil empty slice")
	}
	if len(sessions) != 0 {
		t.Errorf("List returned %d sessions, want 0", len(sessions))
	}
}

// TestJSONStore_ListSkipsCorrupt verifies that corrupt JSON files are silently
// skipped and valid sessions are still returned.
func TestJSONStore_ListSkipsCorrupt(t *testing.T) {
	dir := t.TempDir()
	s, err := store.NewJSONStore(dir)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	ctx := context.Background()

	// Write a valid session.
	valid := agent.NewSession("claude", "claude-opus-4-6")
	if err := s.Save(ctx, valid); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Write a corrupt JSON file directly.
	corruptPath := filepath.Join(dir, "corrupt-session-id.json")
	if err := os.WriteFile(corruptPath, []byte("{not valid json"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	sessions, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("List returned %d sessions, want 1 (corrupt should be skipped)", len(sessions))
	}
	if sessions[0].ID != valid.ID {
		t.Errorf("List returned session ID %q, want %q", sessions[0].ID, valid.ID)
	}
}

// TestJSONStore_IDValidation verifies that invalid IDs are rejected.
func TestJSONStore_IDValidation(t *testing.T) {
	s := newStore(t)
	ctx := context.Background()

	cases := []struct {
		name string
		id   string
	}{
		{"empty", ""},
		{"slash", "bad/id"},
		{"backslash", "bad\\id"},
		{"null byte", "bad\x00id"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sess := &agent.Session{ID: tc.id}

			if err := s.Save(ctx, sess); err == nil {
				t.Error("Save: expected error for invalid ID, got nil")
			} else if errors.Is(err, agent.ErrSessionNotFound) {
				t.Errorf("Save: expected validation error, got ErrSessionNotFound")
			}

			if _, err := s.Load(ctx, tc.id); err == nil {
				t.Error("Load: expected error for invalid ID, got nil")
			} else if errors.Is(err, agent.ErrSessionNotFound) {
				t.Errorf("Load: expected validation error, got ErrSessionNotFound")
			}

			if err := s.Delete(ctx, tc.id); err == nil {
				t.Error("Delete: expected error for invalid ID, got nil")
			} else if errors.Is(err, agent.ErrSessionNotFound) {
				t.Errorf("Delete: expected validation error, got ErrSessionNotFound")
			}
		})
	}
}

// TestJSONStore_FilePermissions verifies session files are created with 0600.
func TestJSONStore_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	s, err := store.NewJSONStore(dir)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	ctx := context.Background()
	sess := agent.NewSession("claude", "claude-opus-4-6")
	if err := s.Save(ctx, sess); err != nil {
		t.Fatalf("Save: %v", err)
	}
	path := filepath.Join(dir, sess.ID+".json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("file permission = %04o, want 0600", perm)
	}
}

// TestJSONStore_DirPermissions verifies the store directory is created with 0700.
func TestJSONStore_DirPermissions(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "sessions")
	s, err := store.NewJSONStore(dir)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	ctx := context.Background()
	sess := agent.NewSession("claude", "claude-opus-4-6")
	if err := s.Save(ctx, sess); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Stat dir: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0700 {
		t.Errorf("dir permission = %04o, want 0700", perm)
	}
}

func BenchmarkJSONStoreRoundTrip(b *testing.B) {
	s, err := store.NewJSONStore(b.TempDir())
	if err != nil {
		b.Fatalf("NewJSONStore: %v", err)
	}
	ctx := context.Background()

	sess := agent.NewSession("claude", "claude-opus-4-6")
	for i := 0; i < 10; i++ {
		if i%2 == 0 {
			sess.AppendUserMessage("user message content for benchmarking purposes")
		} else {
			sess.AppendAssistantMessage("assistant reply content for benchmarking")
		}
	}

	b.ResetTimer()
	for range b.N {
		if err := s.Save(ctx, sess); err != nil {
			b.Fatalf("Save: %v", err)
		}
		if _, err := s.Load(ctx, sess.ID); err != nil {
			b.Fatalf("Load: %v", err)
		}
	}
}

// TestJSONStore_ListSortOrder verifies sessions are returned newest-first.
func TestJSONStore_ListSortOrder(t *testing.T) {
	dir := t.TempDir()
	s, err := store.NewJSONStore(dir)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}
	ctx := context.Background()

	base := time.Now().UTC()
	sessions := []*agent.Session{
		{ID: "aaa", Provider: "p", Model: "m", UpdatedAt: base.Add(-2 * time.Hour), History: []agent.Message{}},
		{ID: "bbb", Provider: "p", Model: "m", UpdatedAt: base.Add(-1 * time.Hour), History: []agent.Message{}},
		{ID: "ccc", Provider: "p", Model: "m", UpdatedAt: base, History: []agent.Message{}},
	}
	for _, sess := range sessions {
		sess.CreatedAt = sess.UpdatedAt
		if err := s.Save(ctx, sess); err != nil {
			t.Fatalf("Save %q: %v", sess.ID, err)
		}
	}

	got, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("List returned %d sessions, want 3", len(got))
	}
	want := []string{"ccc", "bbb", "aaa"}
	for i, sess := range got {
		if sess.ID != want[i] {
			t.Errorf("got[%d].ID = %q, want %q", i, sess.ID, want[i])
		}
	}
}
