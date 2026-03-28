package agent

import (
	"fmt"
	"sort"
	"sync"
)

// Skill is a named bundle of a system prompt and tool set that can be attached
// to a session at startup via the --skill flag.
type Skill struct {
	Name         string
	Description  string
	SystemPrompt string
	Tools        []Tool
}

var (
	skillMu       sync.RWMutex
	skillRegistry = make(map[string]Skill)
)

// RegisterSkill registers a skill by name. Panics if name is empty or already registered.
// Skills are typically registered from package init() functions.
func RegisterSkill(s Skill) {
	if s.Name == "" {
		panic("agent: RegisterSkill called with empty name")
	}
	skillMu.Lock()
	defer skillMu.Unlock()
	if _, dup := skillRegistry[s.Name]; dup {
		panic(fmt.Sprintf("agent: RegisterSkill called twice for skill %q", s.Name))
	}
	skillRegistry[s.Name] = s
}

// LookupSkill returns the Skill registered under name, and whether it was found.
func LookupSkill(name string) (Skill, bool) {
	skillMu.RLock()
	s, ok := skillRegistry[name]
	skillMu.RUnlock()
	return s, ok
}

// Skills returns a sorted slice of all registered skills.
func Skills() []Skill {
	skillMu.RLock()
	defer skillMu.RUnlock()
	skills := make([]Skill, 0, len(skillRegistry))
	for _, s := range skillRegistry {
		skills = append(skills, s)
	}
	sort.Slice(skills, func(i, j int) bool { return skills[i].Name < skills[j].Name })
	return skills
}
