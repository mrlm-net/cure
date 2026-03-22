package agent

import (
	"fmt"
	"sort"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = make(map[string]AgentFactory)
)

// Register registers a provider factory under the given name.
// It panics if name is empty or already registered.
// Providers call Register from their package init() function.
func Register(name string, factory AgentFactory) {
	if name == "" {
		panic("agent: Register called with empty provider name")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry[name]; dup {
		panic(fmt.Sprintf("agent: Register called twice for provider %q", name))
	}
	registry[name] = factory
}

// New creates an Agent for the named provider using cfg as configuration.
// Returns [ErrProviderNotFound] (wrapped) if the provider has not been registered.
func New(name string, cfg map[string]any) (Agent, error) {
	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrProviderNotFound, name)
	}
	return factory(cfg)
}

// Registered returns a sorted list of registered provider names.
func Registered() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// resetRegistry clears the registry. For use in tests only.
func resetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = make(map[string]AgentFactory)
}
