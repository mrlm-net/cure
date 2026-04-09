package registry

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAddListRemove(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir)

	// Create source directory
	srcDir := filepath.Join(dir, "test-source")
	os.MkdirAll(srcDir, 0755)

	if err := reg.Add("test-source", "https://github.com/org/config.git"); err != nil {
		t.Fatalf("Add: %v", err)
	}

	sources, err := reg.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("got %d sources, want 1", len(sources))
	}
	if sources[0].Name != "test-source" {
		t.Errorf("name = %q, want %q", sources[0].Name, "test-source")
	}

	// Duplicate add
	if err := reg.Add("test-source", ""); err == nil {
		t.Fatal("duplicate add should fail")
	}

	// Remove
	if err := reg.Remove("test-source"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	sources, _ = reg.List()
	if len(sources) != 0 {
		t.Errorf("after remove: got %d, want 0", len(sources))
	}
}

func TestRemoveNotFound(t *testing.T) {
	reg := New(t.TempDir())
	err := reg.Remove("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir)

	os.MkdirAll(filepath.Join(dir, "src1"), 0755)
	reg.Add("src1", "")

	s, err := reg.Load("src1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if s.Name != "src1" {
		t.Errorf("name = %q, want %q", s.Name, "src1")
	}

	_, err = reg.Load("nonexistent")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestResolve(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir)

	// Create two sources with overlapping artifacts
	src1 := filepath.Join(dir, "src1", "templates")
	src2 := filepath.Join(dir, "src2", "templates")
	os.MkdirAll(src1, 0755)
	os.MkdirAll(src2, 0755)

	os.WriteFile(filepath.Join(src1, "claude-md.tmpl"), []byte("v1"), 0644)
	os.WriteFile(filepath.Join(src2, "claude-md.tmpl"), []byte("v2"), 0644)
	os.WriteFile(filepath.Join(src1, "only-in-src1.tmpl"), []byte("x"), 0644)

	reg.Add("src1", "")
	reg.Add("src2", "")

	// Last source wins
	result := reg.Resolve(ArtifactTemplate, "claude-md.tmpl")
	if !filepath.IsAbs(result) || filepath.Base(filepath.Dir(result)) != "templates" {
		t.Errorf("unexpected resolve result: %q", result)
	}

	// Only in src1
	result = reg.Resolve(ArtifactTemplate, "only-in-src1.tmpl")
	if result == "" {
		t.Error("should find only-in-src1.tmpl")
	}

	// Not found
	result = reg.Resolve(ArtifactTemplate, "nonexistent.tmpl")
	if result != "" {
		t.Errorf("should return empty for nonexistent, got %q", result)
	}
}

func TestResolveAll(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir)

	src1 := filepath.Join(dir, "src1", "prompts")
	src2 := filepath.Join(dir, "src2", "prompts")
	os.MkdirAll(src1, 0755)
	os.MkdirAll(src2, 0755)

	os.WriteFile(filepath.Join(src1, "base.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(src2, "base.txt"), []byte("b"), 0644)

	reg.Add("src1", "")
	reg.Add("src2", "")

	results := reg.ResolveAll(ArtifactPrompt, "base.txt")
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

func TestListArtifacts(t *testing.T) {
	dir := t.TempDir()
	reg := New(dir)

	src := filepath.Join(dir, "src1", "skills")
	os.MkdirAll(src, 0755)
	os.WriteFile(filepath.Join(src, "review.json"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(src, "build.json"), []byte("{}"), 0644)

	reg.Add("src1", "")

	names := reg.ListArtifacts(ArtifactSkill)
	if len(names) != 2 {
		t.Fatalf("got %d, want 2", len(names))
	}
	if names[0] != "build.json" || names[1] != "review.json" {
		t.Errorf("names = %v, want [build.json, review.json]", names)
	}
}

func TestListEmpty(t *testing.T) {
	reg := New(filepath.Join(t.TempDir(), "nonexistent"))
	sources, err := reg.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(sources) != 0 {
		t.Errorf("got %d, want 0", len(sources))
	}
}
