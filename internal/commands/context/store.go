package ctxcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mrlm-net/cure/pkg/agent"
	agentstore "github.com/mrlm-net/cure/pkg/agent/store"
)

// defaultStoreDir returns the directory used to store cure sessions.
// It respects XDG_DATA_HOME if set; otherwise uses ~/.local/share/cure/sessions.
// Returns an error if the home directory cannot be determined.
func defaultStoreDir() (string, error) {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "cure", "sessions"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("context: cannot determine home directory for session store: %w", err)
	}
	return filepath.Join(home, ".local", "share", "cure", "sessions"), nil
}

// DefaultStoreDir returns the directory used to store cure sessions.
// Exported for use by cmd/cure/main.go.
func DefaultStoreDir() (string, error) {
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
