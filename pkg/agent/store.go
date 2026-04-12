package agent

import (
	"context"
	"strings"
)

// SessionFilter defines criteria for searching sessions. Zero-value fields
// are ignored (no filtering on that dimension).
type SessionFilter struct {
	ProjectName  string // exact match on ProjectName
	Provider     string // exact match on Provider
	BranchName   string // exact match on BranchName
	HasWorkItem  string // session must contain this work item ID
	SkillName    string // exact match on SkillName
	NameContains string // case-insensitive substring match on Name
	Limit        int    // max results (0 = no limit)
}

// MatchSession reports whether the session satisfies all non-empty filter criteria.
func (f SessionFilter) MatchSession(s *Session) bool {
	if f.ProjectName != "" && s.ProjectName != f.ProjectName {
		return false
	}
	if f.Provider != "" && s.Provider != f.Provider {
		return false
	}
	if f.BranchName != "" && s.BranchName != f.BranchName {
		return false
	}
	if f.SkillName != "" && s.SkillName != f.SkillName {
		return false
	}
	if f.NameContains != "" && !strings.Contains(strings.ToLower(s.Name), strings.ToLower(f.NameContains)) {
		return false
	}
	if f.HasWorkItem != "" {
		found := false
		for _, wi := range s.WorkItems {
			if wi == f.HasWorkItem {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// SessionStore is the persistence interface for [Session] objects.
// Implementations must be safe for concurrent use.
type SessionStore interface {
	// Save persists or updates a session. The session's UpdatedAt should be
	// set by the caller before saving.
	Save(ctx context.Context, s *Session) error

	// Load retrieves a session by ID.
	// Returns [ErrSessionNotFound] (or a wrapped form) when the ID is not found.
	Load(ctx context.Context, id string) (*Session, error)

	// List returns all sessions in the store, ordered by UpdatedAt descending.
	List(ctx context.Context) ([]*Session, error)

	// Delete removes the session with the given ID from the store.
	// Returns [ErrSessionNotFound] (or a wrapped form) if the ID does not exist.
	Delete(ctx context.Context, id string) error

	// Fork creates a copy of the session identified by id, assigns it a new ID,
	// and persists the copy. Returns the forked session.
	// Returns [ErrSessionNotFound] (or a wrapped form) if the source ID does not exist.
	Fork(ctx context.Context, id string) (*Session, error)

	// Search returns sessions matching the filter criteria, ordered by
	// UpdatedAt descending. A zero-value filter returns all sessions.
	Search(ctx context.Context, filter SessionFilter) ([]*Session, error)
}
