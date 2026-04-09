package project

import (
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
	"time"
)

// ErrNotFound is returned when a project does not exist in the store.
var ErrNotFound = errors.New("project not found")

// validName matches project names: lowercase alphanumeric, may contain hyphens,
// must start with a letter or digit. Length 1–64.
var validName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// ValidateName checks whether name is a valid project name.
func ValidateName(name string) error {
	if !validName.MatchString(name) {
		if name == "" {
			return fmt.Errorf("project name must not be empty")
		}
		return fmt.Errorf("project name %q: must match [a-z0-9][a-z0-9-]{0,63}", name)
	}
	return nil
}

// ProjectStore persists and retrieves Project entities.
type ProjectStore interface {
	Save(p *Project) error
	Load(name string) (*Project, error)
	List() ([]*Project, error)
	Delete(name string) error
}

// Store is a filesystem-backed ProjectStore that persists each project as a
// JSON file at <baseDir>/<name>/project.json. Store is safe for concurrent use.
type Store struct {
	baseDir string
	mu      sync.Mutex
}

// compile-time assertion
var _ ProjectStore = (*Store)(nil)

// NewStore creates a Store rooted at baseDir. The directory is not created
// until the first call to Save.
func NewStore(baseDir string) *Store {
	return &Store{baseDir: baseDir}
}

// DefaultBaseDir returns the default project store directory (~/.cure/projects).
// Returns an error if the user's home directory cannot be determined.
func DefaultBaseDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("project store: resolve home directory: %w", err)
	}
	return filepath.Join(home, ".cure", "projects"), nil
}

// projectDir returns the directory path for a project.
func (s *Store) projectDir(name string) string {
	return filepath.Join(s.baseDir, name)
}

// projectPath returns the JSON file path for a project.
func (s *Store) projectPath(name string) string {
	return filepath.Join(s.projectDir(name), "project.json")
}

// Save persists the project atomically. The project directory and file are
// created if they do not exist. UpdatedAt is set to the current time.
func (s *Store) Save(p *Project) error {
	if p == nil {
		return fmt.Errorf("project store: project must not be nil")
	}
	if err := ValidateName(p.Name); err != nil {
		return fmt.Errorf("project store: %w", err)
	}

	p.UpdatedAt = time.Now().UTC()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = p.UpdatedAt
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("project store: marshal %q: %w", p.Name, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.projectDir(p.Name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("project store: create directory %q: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".project-*.tmp")
	if err != nil {
		return fmt.Errorf("project store: create temp file: %w", err)
	}
	tmpName := tmp.Name()

	ok := false
	defer func() {
		if !ok {
			os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("project store: write %q: %w", p.Name, err)
	}
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("project store: chmod %q: %w", p.Name, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("project store: close temp file: %w", err)
	}
	if err := os.Rename(tmpName, s.projectPath(p.Name)); err != nil {
		return fmt.Errorf("project store: rename %q: %w", p.Name, err)
	}

	ok = true
	return nil
}

// Load retrieves a project by name.
// Returns a wrapped ErrNotFound when the project does not exist.
func (s *Store) Load(name string) (*Project, error) {
	if err := ValidateName(name); err != nil {
		return nil, fmt.Errorf("project store: %w", err)
	}

	data, err := os.ReadFile(s.projectPath(name))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("project store: load %q: %w", name, ErrNotFound)
		}
		return nil, fmt.Errorf("project store: read %q: %w", name, err)
	}

	var p Project
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("project store: unmarshal %q: %w", name, err)
	}
	return &p, nil
}

// List returns all projects sorted by name ascending. Corrupt or unreadable
// project files are silently skipped.
func (s *Store) List() ([]*Project, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []*Project{}, nil
		}
		return nil, fmt.Errorf("project store: list: %w", err)
	}

	var projects []*Project
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		path := filepath.Join(s.baseDir, e.Name(), "project.json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var p Project
		if err := json.Unmarshal(data, &p); err != nil {
			continue
		}
		projects = append(projects, &p)
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Name < projects[j].Name
	})

	if projects == nil {
		projects = []*Project{}
	}
	return projects, nil
}

// Delete removes the project directory and all its contents.
// Returns a wrapped ErrNotFound when the project does not exist.
func (s *Store) Delete(name string) error {
	if err := ValidateName(name); err != nil {
		return fmt.Errorf("project store: %w", err)
	}

	dir := s.projectDir(name)
	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("project store: delete %q: %w", name, ErrNotFound)
		}
		return fmt.Errorf("project store: delete %q: %w", name, err)
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("project store: delete %q: %w", name, err)
	}
	return nil
}
