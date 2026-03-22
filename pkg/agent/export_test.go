package agent

// resetRegistry clears the global provider registry.
// Only available during testing — not compiled into production builds.
func resetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = make(map[string]AgentFactory)
}
