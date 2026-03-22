package agent

import "errors"

// Sentinel errors returned by agent operations.
var (
	// ErrProviderNotFound is returned by [New] when the requested provider name
	// has not been registered via [Register].
	ErrProviderNotFound = errors.New("agent: provider not found")

	// ErrSessionNotFound is returned by [SessionStore] implementations when
	// a session ID does not exist in the backing store.
	ErrSessionNotFound = errors.New("agent: session not found")

	// ErrCountNotSupported is returned by [Agent.CountTokens] when the provider
	// does not support token counting.
	ErrCountNotSupported = errors.New("agent: token counting not supported by this provider")
)
