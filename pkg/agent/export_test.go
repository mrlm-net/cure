package agent

// resetRegistry clears the global provider registry.
// Only available during testing — not compiled into production builds.
func resetRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = make(map[string]AgentFactory)
}

// ResetSkillRegistry clears the global skill registry. Only available during testing.
func ResetSkillRegistry() {
	skillMu.Lock()
	defer skillMu.Unlock()
	skillRegistry = make(map[string]Skill)
}
