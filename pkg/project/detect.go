package project

import (
	"path/filepath"
)

// Detector finds the project associated with a working directory by matching
// the cwd against registered repository paths.
type Detector interface {
	Detect(cwd string) (*Project, error)
}

// StoreDetector implements Detector by scanning all projects in a ProjectStore
// and matching the given working directory against each project's repo paths.
type StoreDetector struct {
	store ProjectStore
}

// NewDetector creates a Detector backed by the given ProjectStore.
func NewDetector(store ProjectStore) *StoreDetector {
	return &StoreDetector{store: store}
}

// Detect returns the project whose repo list contains a path that is a prefix
// of or equal to cwd. Returns (nil, nil) if no project matches. Both cwd and
// repo paths are resolved to absolute paths and evaluated for symlinks before
// comparison.
func (d *StoreDetector) Detect(cwd string) (*Project, error) {
	absCwd, err := resolveDir(cwd)
	if err != nil {
		return nil, err
	}

	projects, err := d.store.List()
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		for _, r := range p.Repos {
			absRepo, err := resolveDir(r.Path)
			if err != nil {
				continue // skip unresolvable repo paths
			}
			if isSubdir(absRepo, absCwd) {
				return p, nil
			}
		}
	}

	return nil, nil
}

// resolveDir returns the absolute, symlink-resolved form of dir.
func resolveDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		// If the path doesn't exist yet, fall back to the absolute path.
		return abs, nil
	}
	return resolved, nil
}

// isSubdir reports whether child is equal to or a subdirectory of parent.
// Both paths must be absolute and cleaned.
func isSubdir(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if parent == child {
		return true
	}
	// Ensure parent ends with separator so "/foo" doesn't match "/foobar".
	return len(child) > len(parent) && child[:len(parent)] == parent && child[len(parent)] == filepath.Separator
}
