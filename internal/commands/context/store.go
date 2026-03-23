package ctxcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mrlm-net/cure/pkg/agent"
	agentstore "github.com/mrlm-net/cure/pkg/agent/store"
)

// defaultStoreDir returns the directory used to store cure sessions.
// It respects XDG_DATA_HOME if set; otherwise falls back to
// ~/.local/share/cure/sessions.
func defaultStoreDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "cure", "sessions")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback: use a relative path so the caller still gets something usable.
		return filepath.Join(".local", "share", "cure", "sessions")
	}
	return filepath.Join(home, ".local", "share", "cure", "sessions")
}

// DefaultStoreDir returns the directory used to store cure sessions.
// Exported for use by cmd/cure/main.go.
func DefaultStoreDir() string {
	return defaultStoreDir()
}

// defaultModel returns the model name used when creating new sessions.
func defaultModel() string {
	return "claude-opus-4-6"
}

// newStore creates a new JSONStore rooted at dir.
func newStore(dir string) (agent.SessionStore, error) {
	st, err := agentstore.NewJSONStore(dir)
	if err != nil {
		return nil, fmt.Errorf("context: failed to open session store at %s: %w", dir, err)
	}
	return st, nil
}
