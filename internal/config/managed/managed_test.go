package managed

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadManaged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "CLAUDE.md")
	content := "# Project\n\nThis is a test."

	if err := WriteManaged(path, content); err != nil {
		t.Fatalf("WriteManaged: %v", err)
	}

	// File should exist with marker
	if !IsManaged(path) {
		t.Error("file should be managed")
	}

	// Should not have drifted
	drifted, err := HasDrifted(path)
	if err != nil {
		t.Fatalf("HasDrifted: %v", err)
	}
	if drifted {
		t.Error("should not have drifted immediately after write")
	}
}

func TestDriftDetection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	content := "original content"

	WriteManaged(path, content)

	// Modify the content (keep marker intact)
	data, _ := os.ReadFile(path)
	lines := string(data)
	// Replace content after first newline
	idx := 0
	for i, c := range lines {
		if c == '\n' {
			idx = i + 1
			break
		}
	}
	modified := lines[:idx] + "modified content"
	os.WriteFile(path, []byte(modified), 0644)

	drifted, err := HasDrifted(path)
	if err != nil {
		t.Fatalf("HasDrifted: %v", err)
	}
	if !drifted {
		t.Error("should detect drift after content modification")
	}
}

func TestUnmanagedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plain.md")
	os.WriteFile(path, []byte("# Just a file"), 0644)

	if IsManaged(path) {
		t.Error("plain file should not be managed")
	}

	drifted, _ := HasDrifted(path)
	if drifted {
		t.Error("unmanaged file should not report drift")
	}
}

func TestMarkerHashRoundtrip(t *testing.T) {
	content := "hello world"
	marker := GenerateMarker(content)
	hash := ContentHash(content)

	// Marker should contain the hash
	if marker == "" {
		t.Fatal("marker should not be empty")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte(marker+"\n"+content), 0644)

	readHash, err := ReadMarkerHash(path)
	if err != nil {
		t.Fatalf("ReadMarkerHash: %v", err)
	}
	if readHash != hash {
		t.Errorf("hash mismatch: read=%q, computed=%q", readHash, hash)
	}
}
