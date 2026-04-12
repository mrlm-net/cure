// Package registry manages AI config source registries — collections of
// templates, skills, agents, MCP configs, and system prompts that cure
// distributes to projects and repos.
//
// Sources are git repos or local directories cloned/linked at ~/.cure/registry/<name>/.
// Resolution follows an overlay model: embedded < registry sources < project < repo.
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ErrNotFound is returned when a source does not exist.
var ErrNotFound = errors.New("registry source not found")

// ArtifactType categorizes registry artifacts.
type ArtifactType string

const (
	ArtifactTemplate ArtifactType = "templates"
	ArtifactSkill    ArtifactType = "skills"
	ArtifactAgent    ArtifactType = "agents"
	ArtifactConfig   ArtifactType = "configs"
	ArtifactMCP      ArtifactType = "mcp"
	ArtifactPrompt   ArtifactType = "prompts"
)

// Source is a registered config source (git clone or local directory).
type Source struct {
	Name      string    `json:"name"`
	URL       string    `json:"url,omitempty"`
	Path      string    `json:"path"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Registry manages AI config sources and resolves artifacts from the overlay stack.
type Registry struct {
	baseDir string
	mu      sync.RWMutex
	sources []Source
}

// New creates a Registry rooted at baseDir (typically ~/.cure/registry/).
func New(baseDir string) *Registry {
	return &Registry{baseDir: baseDir}
}

// DefaultBaseDir returns ~/.cure/registry.
func DefaultBaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("registry: resolve home: %w", err)
	}
	return filepath.Join(home, ".cure", "registry"), nil
}

// indexPath returns the path to the registry index file.
func (r *Registry) indexPath() string {
	return filepath.Join(r.baseDir, "registry.json")
}

// loadIndex reads the source list from disk.
func (r *Registry) loadIndex() ([]Source, error) {
	data, err := os.ReadFile(r.indexPath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []Source{}, nil
		}
		return nil, fmt.Errorf("registry: read index: %w", err)
	}
	var idx struct {
		Sources []Source `json:"sources"`
	}
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("registry: parse index: %w", err)
	}
	return idx.Sources, nil
}

// saveIndex writes the source list to disk.
func (r *Registry) saveIndex(sources []Source) error {
	if err := os.MkdirAll(r.baseDir, 0700); err != nil {
		return fmt.Errorf("registry: create dir: %w", err)
	}
	idx := struct {
		Sources []Source `json:"sources"`
	}{Sources: sources}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return fmt.Errorf("registry: marshal index: %w", err)
	}
	return os.WriteFile(r.indexPath(), data, 0600)
}

// Add registers a new source. The source directory must already exist at
// baseDir/<name>/ (caller is responsible for git clone).
func (r *Registry) Add(name, url string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	sources, err := r.loadIndex()
	if err != nil {
		return err
	}
	for _, s := range sources {
		if s.Name == name {
			return fmt.Errorf("registry: source %q already exists", name)
		}
	}

	srcPath := filepath.Join(r.baseDir, name)
	sources = append(sources, Source{
		Name:      name,
		URL:       url,
		Path:      srcPath,
		UpdatedAt: time.Now().UTC(),
	})
	return r.saveIndex(sources)
}

// Remove deregisters a source. Does NOT delete the directory (caller handles cleanup).
func (r *Registry) Remove(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	sources, err := r.loadIndex()
	if err != nil {
		return err
	}
	filtered := make([]Source, 0, len(sources))
	found := false
	for _, s := range sources {
		if s.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, s)
	}
	if !found {
		return fmt.Errorf("registry: remove %q: %w", name, ErrNotFound)
	}
	return r.saveIndex(filtered)
}

// List returns all registered sources sorted by name.
func (r *Registry) List() ([]Source, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sources, err := r.loadIndex()
	if err != nil {
		return nil, err
	}
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Name < sources[j].Name
	})
	return sources, nil
}

// Load returns a single source by name.
func (r *Registry) Load(name string) (*Source, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sources, err := r.loadIndex()
	if err != nil {
		return nil, err
	}
	for i := range sources {
		if sources[i].Name == name {
			return &sources[i], nil
		}
	}
	return nil, fmt.Errorf("registry: load %q: %w", name, ErrNotFound)
}

// Resolve finds an artifact by type and name across all sources.
// Returns the path to the highest-priority match (last source wins).
// Returns empty string if not found.
func (r *Registry) Resolve(artifactType ArtifactType, name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sources, _ := r.loadIndex()
	var result string
	for _, s := range sources {
		candidate := filepath.Join(s.Path, string(artifactType), name)
		if _, err := os.Stat(candidate); err == nil {
			result = candidate
		}
	}
	return result
}

// ResolveAll finds all instances of an artifact type across all sources.
// Returns paths ordered by priority (lowest first).
func (r *Registry) ResolveAll(artifactType ArtifactType, name string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sources, _ := r.loadIndex()
	var results []string
	for _, s := range sources {
		candidate := filepath.Join(s.Path, string(artifactType), name)
		if _, err := os.Stat(candidate); err == nil {
			results = append(results, candidate)
		}
	}
	return results
}

// ListArtifacts returns all artifact names of a given type across all sources.
func (r *Registry) ListArtifacts(artifactType ArtifactType) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sources, _ := r.loadIndex()
	seen := make(map[string]bool)
	var names []string

	for _, s := range sources {
		dir := filepath.Join(s.Path, string(artifactType))
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".") {
				continue
			}
			if !seen[e.Name()] {
				seen[e.Name()] = true
				names = append(names, e.Name())
			}
		}
	}
	sort.Strings(names)
	return names
}
