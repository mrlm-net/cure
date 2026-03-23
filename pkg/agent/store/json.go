package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/mrlm-net/cure/pkg/agent"
)

// Compile-time assertion that JSONStore implements agent.SessionStore.
var _ agent.SessionStore = (*JSONStore)(nil)

// JSONStore is a file-backed SessionStore that persists each session as a JSON
// file under a configurable directory. Writes are atomic via os.CreateTemp +
// os.Rename. JSONStore is safe for concurrent use.
type JSONStore struct {
	dir string
	mu  sync.Mutex
}

// NewJSONStore creates a JSONStore rooted at dir. dir may start with "~/"
// which is expanded to the current user's home directory. The directory is
// not created until the first call to Save.
func NewJSONStore(dir string) (*JSONStore, error) {
	if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("store: expand home directory: %w", err)
		}
		dir = filepath.Join(home, dir[2:])
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("store: resolve directory path: %w", err)
	}
	return &JSONStore{dir: abs}, nil
}

// validSessionID matches the output of newSessionID: 1–64 lowercase hex characters.
// This allow-list is intentionally strict — any ID that doesn't match is rejected
// before it reaches the filesystem, eliminating path-traversal risk entirely.
var validSessionID = regexp.MustCompile(`^[0-9a-f]{1,64}$`)

// validateID rejects session IDs that do not match the allow-list.
// Only lowercase hex strings of length 1–64 are accepted. This matches the
// output of [agent.NewSession] exactly and is stricter than a deny-list.
func validateID(id string) error {
	if !validSessionID.MatchString(id) {
		if id == "" {
			return fmt.Errorf("store: invalid session ID: empty string")
		}
		return fmt.Errorf("store: invalid session ID %q: must match [0-9a-f]{1,64}", id)
	}
	return nil
}

// sessionPath returns the absolute path for a session file.
func (s *JSONStore) sessionPath(id string) string {
	return filepath.Join(s.dir, id+".json")
}

// Save persists the session atomically. The session file receives mode 0600.
// The store directory is created with mode 0700 if it does not exist.
// Save is safe for concurrent use.
func (s *JSONStore) Save(_ context.Context, sess *agent.Session) error {
	if sess == nil {
		return fmt.Errorf("store: session must not be nil")
	}
	if err := validateID(sess.ID); err != nil {
		return err
	}
	data, err := json.Marshal(sess)
	if err != nil {
		return fmt.Errorf("store: marshal session %q: %w", sess.ID, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, 0700); err != nil {
		return fmt.Errorf("store: create directory: %w", err)
	}
	tmp, err := os.CreateTemp(s.dir, ".session-*.tmp")
	if err != nil {
		return fmt.Errorf("store: create temp file: %w", err)
	}
	tmpName := tmp.Name()
	// Clean up the temp file if anything goes wrong before rename.
	ok := false
	defer func() {
		if !ok {
			os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("store: write session %q: %w", sess.ID, err)
	}
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("store: chmod session %q: %w", sess.ID, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("store: close temp file: %w", err)
	}
	if err := os.Rename(tmpName, s.sessionPath(sess.ID)); err != nil {
		return fmt.Errorf("store: rename session %q: %w", sess.ID, err)
	}
	ok = true
	return nil
}

// Load retrieves a session by ID.
// Returns a wrapped [agent.ErrSessionNotFound] when the session does not exist.
func (s *JSONStore) Load(_ context.Context, id string) (*agent.Session, error) {
	if err := validateID(id); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(s.sessionPath(id))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("store: load session %q: %w", id, agent.ErrSessionNotFound)
		}
		return nil, fmt.Errorf("store: read session %q: %w", id, err)
	}
	var sess agent.Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("store: unmarshal session %q: %w", id, err)
	}
	return &sess, nil
}

// List returns all sessions sorted by UpdatedAt descending (newest first).
// Ties are broken by ID ascending. Corrupt or unreadable files are silently
// skipped. Returns a non-nil empty slice when the store directory does not
// exist.
func (s *JSONStore) List(_ context.Context) ([]*agent.Session, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []*agent.Session{}, nil
		}
		return nil, fmt.Errorf("store: list sessions: %w", err)
	}

	var sessions []*agent.Session
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name()))
		if err != nil {
			continue // skip unreadable files
		}
		var sess agent.Session
		if err := json.Unmarshal(data, &sess); err != nil {
			continue // skip corrupt files
		}
		sessions = append(sessions, &sess)
	}

	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].UpdatedAt.Equal(sessions[j].UpdatedAt) {
			return sessions[i].ID < sessions[j].ID
		}
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	if sessions == nil {
		sessions = []*agent.Session{}
	}
	return sessions, nil
}

// Delete removes the session with the given ID from the store.
// Returns a wrapped [agent.ErrSessionNotFound] when the session does not exist.
func (s *JSONStore) Delete(_ context.Context, id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	if err := os.Remove(s.sessionPath(id)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("store: delete session %q: %w", id, agent.ErrSessionNotFound)
		}
		return fmt.Errorf("store: delete session %q: %w", id, err)
	}
	return nil
}

// Fork creates an independent copy of the session identified by id, assigns
// it a new ID, persists the copy, and returns it.
// Returns a wrapped [agent.ErrSessionNotFound] when the source session does not exist.
func (s *JSONStore) Fork(ctx context.Context, id string) (*agent.Session, error) {
	src, err := s.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	forked := src.Fork()
	if err := s.Save(ctx, forked); err != nil {
		return nil, fmt.Errorf("store: fork session %q: %w", id, err)
	}
	return forked, nil
}
