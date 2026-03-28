package agent_test

import (
	"context"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
)

func TestRegisterSkillAndLookup(t *testing.T) {
	agent.ResetSkillRegistry()
	t.Cleanup(agent.ResetSkillRegistry)

	echoTool := agent.FuncTool("echo", "echo tool", nil,
		func(_ context.Context, _ map[string]any) (string, error) { return "ok", nil },
	)

	skill := agent.Skill{
		Name:         "greeter",
		Description:  "Greets users",
		SystemPrompt: "You are a friendly greeter.",
		Tools:        []agent.Tool{echoTool},
	}
	agent.RegisterSkill(skill)

	got, ok := agent.LookupSkill("greeter")
	if !ok {
		t.Fatal("LookupSkill: expected true, got false")
	}
	if got.Name != skill.Name {
		t.Errorf("Name = %q, want %q", got.Name, skill.Name)
	}
	if got.Description != skill.Description {
		t.Errorf("Description = %q, want %q", got.Description, skill.Description)
	}
	if got.SystemPrompt != skill.SystemPrompt {
		t.Errorf("SystemPrompt = %q, want %q", got.SystemPrompt, skill.SystemPrompt)
	}
	if len(got.Tools) != 1 {
		t.Errorf("len(Tools) = %d, want 1", len(got.Tools))
	}
}

func TestLookupSkill_Miss(t *testing.T) {
	agent.ResetSkillRegistry()
	t.Cleanup(agent.ResetSkillRegistry)

	got, ok := agent.LookupSkill("nonexistent")
	if ok {
		t.Errorf("LookupSkill: expected false, got true")
	}
	if got.Name != "" {
		t.Errorf("zero Skill.Name = %q, want empty", got.Name)
	}
}

func TestRegisterSkill_PanicOnEmpty(t *testing.T) {
	agent.ResetSkillRegistry()
	t.Cleanup(agent.ResetSkillRegistry)

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty skill name, got none")
		}
	}()
	agent.RegisterSkill(agent.Skill{Name: ""})
}

func TestRegisterSkill_PanicOnDuplicate(t *testing.T) {
	agent.ResetSkillRegistry()
	t.Cleanup(agent.ResetSkillRegistry)

	agent.RegisterSkill(agent.Skill{Name: "dup-skill"})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate skill registration, got none")
		}
	}()
	agent.RegisterSkill(agent.Skill{Name: "dup-skill"})
}

func TestSkills_ReturnsSorted(t *testing.T) {
	agent.ResetSkillRegistry()
	t.Cleanup(agent.ResetSkillRegistry)

	agent.RegisterSkill(agent.Skill{Name: "zebra"})
	agent.RegisterSkill(agent.Skill{Name: "apple"})
	agent.RegisterSkill(agent.Skill{Name: "mango"})

	skills := agent.Skills()
	if len(skills) != 3 {
		t.Fatalf("len(Skills()) = %d, want 3", len(skills))
	}
	if skills[0].Name != "apple" || skills[1].Name != "mango" || skills[2].Name != "zebra" {
		t.Errorf("Skills() order = [%q, %q, %q], want [apple, mango, zebra]",
			skills[0].Name, skills[1].Name, skills[2].Name)
	}
}

func TestResetSkillRegistry_Isolation(t *testing.T) {
	agent.ResetSkillRegistry()
	t.Cleanup(agent.ResetSkillRegistry)

	agent.RegisterSkill(agent.Skill{Name: "temp-skill"})

	// Reset and verify the skill is gone.
	agent.ResetSkillRegistry()
	_, ok := agent.LookupSkill("temp-skill")
	if ok {
		t.Error("skill should be absent after ResetSkillRegistry")
	}
	if got := agent.Skills(); len(got) != 0 {
		t.Errorf("Skills() = %v, want empty after reset", got)
	}
}

func TestSkills_EmptyRegistry(t *testing.T) {
	agent.ResetSkillRegistry()
	t.Cleanup(agent.ResetSkillRegistry)

	skills := agent.Skills()
	if skills == nil {
		// nil slice is acceptable — check length only
		return
	}
	if len(skills) != 0 {
		t.Errorf("Skills() = %v, want empty slice", skills)
	}
}
