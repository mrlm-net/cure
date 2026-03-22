package agent

import "context"

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
}
