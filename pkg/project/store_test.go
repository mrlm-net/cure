package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid simple", "myproject", false},
		{"valid with hyphens", "my-project", false},
		{"valid with numbers", "project123", false},
		{"valid single char", "a", false},
		{"valid starts with number", "1project", false},
		{"empty", "", true},
		{"uppercase", "MyProject", true},
		{"spaces", "my project", true},
		{"underscores", "my_project", true},
		{"starts with hyphen", "-project", true},
		{"special chars", "my@project", true},
		{"too long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true}, // 65 chars
		{"max length", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", false},  // 64 chars
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestStoreSaveLoad(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)

	p := &Project{
		Name:        "test-project",
		Description: "A test project",
		Repos: []Repo{
			{Path: "/tmp/repo1", Remote: "git@github.com:org/repo1.git"},
		},
		Defaults: Defaults{
			Provider: "claude",
			Model:    "claude-opus-4-6",
		},
	}

	if err := st.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists on disk
	path := filepath.Join(dir, "test-project", "project.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("project.json not found: %v", err)
	}

	loaded, err := st.Load("test-project")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Name != p.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, p.Name)
	}
	if loaded.Description != p.Description {
		t.Errorf("Description = %q, want %q", loaded.Description, p.Description)
	}
	if len(loaded.Repos) != 1 {
		t.Fatalf("Repos count = %d, want 1", len(loaded.Repos))
	}
	if loaded.Repos[0].Path != "/tmp/repo1" {
		t.Errorf("Repo path = %q, want %q", loaded.Repos[0].Path, "/tmp/repo1")
	}
	if loaded.Defaults.Provider != "claude" {
		t.Errorf("Provider = %q, want %q", loaded.Defaults.Provider, "claude")
	}
	if loaded.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if loaded.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
}

func TestStoreSaveUpdatesTimestamp(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)

	p := &Project{Name: "ts-test"}
	if err := st.Save(p); err != nil {
		t.Fatalf("first Save: %v", err)
	}

	loaded1, _ := st.Load("ts-test")
	firstUpdated := loaded1.UpdatedAt

	p.Description = "updated"
	if err := st.Save(p); err != nil {
		t.Fatalf("second Save: %v", err)
	}

	loaded2, _ := st.Load("ts-test")
	if !loaded2.UpdatedAt.After(firstUpdated) {
		t.Error("UpdatedAt should advance on re-save")
	}
}

func TestStoreLoadNotFound(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)

	_, err := st.Load("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing project")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestStoreList(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)

	// Empty store
	list, err := st.List()
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("empty store: got %d projects, want 0", len(list))
	}

	// Add projects
	for _, name := range []string{"charlie", "alpha", "bravo"} {
		if err := st.Save(&Project{Name: name}); err != nil {
			t.Fatalf("Save %q: %v", name, err)
		}
	}

	list, err = st.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("got %d projects, want 3", len(list))
	}

	// Sorted by name
	if list[0].Name != "alpha" || list[1].Name != "bravo" || list[2].Name != "charlie" {
		t.Errorf("order = [%s, %s, %s], want [alpha, bravo, charlie]",
			list[0].Name, list[1].Name, list[2].Name)
	}
}

func TestStoreListNonexistentDir(t *testing.T) {
	st := NewStore("/nonexistent/path/that/does/not/exist")
	list, err := st.List()
	if err != nil {
		t.Fatalf("List nonexistent dir: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("got %d projects, want 0", len(list))
	}
}

func TestStoreDelete(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)

	p := &Project{Name: "to-delete"}
	if err := st.Save(p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := st.Delete("to-delete"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := st.Load("to-delete")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("after delete, Load should return ErrNotFound, got: %v", err)
	}
}

func TestStoreDeleteNotFound(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)

	err := st.Delete("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestStoreSaveInvalidName(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)

	err := st.Save(&Project{Name: "INVALID"})
	if err == nil {
		t.Fatal("expected error for invalid name")
	}
}

func TestStoreSaveNil(t *testing.T) {
	dir := t.TempDir()
	st := NewStore(dir)

	err := st.Save(nil)
	if err == nil {
		t.Fatal("expected error for nil project")
	}
}

func TestDefaultBaseDir(t *testing.T) {
	dir, err := DefaultBaseDir()
	if err != nil {
		t.Fatalf("DefaultBaseDir: %v", err)
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got %q", dir)
	}
	if !contains(dir, ".cure") || !contains(dir, "projects") {
		t.Errorf("expected path containing .cure/projects, got %q", dir)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
