package ctxcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultStoreDir_WithoutXDG(t *testing.T) {
	// Ensure XDG_DATA_HOME is unset for this test.
	t.Setenv("XDG_DATA_HOME", "")

	dir, err := defaultStoreDir()
	if err != nil {
		t.Fatalf("defaultStoreDir returned error: %v", err)
	}
	if dir == "" {
		t.Fatal("defaultStoreDir returned empty string")
	}
	// Must end with the canonical suffix.
	if !strings.HasSuffix(dir, filepath.Join("cure", "sessions")) {
		t.Errorf("expected suffix %q, got %q", filepath.Join("cure", "sessions"), dir)
	}
	// Must be an absolute path (home dir expansion applied).
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got %q", dir)
	}
}

func TestDefaultStoreDir_WithXDG(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)

	dir, err := defaultStoreDir()
	if err != nil {
		t.Fatalf("defaultStoreDir returned error: %v", err)
	}
	want := filepath.Join(tmp, "cure", "sessions")
	if dir != want {
		t.Errorf("defaultStoreDir() = %q, want %q", dir, want)
	}
}

func TestDefaultStoreDir_XDGEmpty(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory:", err)
	}

	dir, dirErr := defaultStoreDir()
	if dirErr != nil {
		t.Fatalf("defaultStoreDir returned error: %v", dirErr)
	}
	want := filepath.Join(home, ".local", "share", "cure", "sessions")
	if dir != want {
		t.Errorf("defaultStoreDir() = %q, want %q", dir, want)
	}
}

func TestDefaultModel(t *testing.T) {
	m := defaultModel()
	if m == "" {
		t.Fatal("defaultModel returned empty string")
	}
}
