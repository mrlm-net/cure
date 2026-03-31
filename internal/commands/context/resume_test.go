package ctxcmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

func TestResumeCommand_MissingPositionalArg(t *testing.T) {
	st := newMockStore()
	cmd := &ResumeCommand{store: st, format: "text"}

	var out, errBuf bytes.Buffer
	tc := &terminal.Context{
		Stdout: &out,
		Stderr: &errBuf,
		Args:   []string{}, // no positional args
	}

	err := cmd.Run(context.Background(), tc)
	if err == nil {
		t.Fatal("expected error for missing session-id, got nil")
	}
	if !strings.Contains(err.Error(), "session-id") {
		t.Errorf("error should mention session-id, got: %v", err)
	}
}

func TestResumeCommand_UnknownSessionID(t *testing.T) {
	st := newMockStore()
	cmd := &ResumeCommand{store: st, format: "text"}

	var out, errBuf bytes.Buffer
	tc := &terminal.Context{
		Stdout: &out,
		Stderr: &errBuf,
		Args:   []string{"nonexistent-id"},
	}

	err := cmd.Run(context.Background(), tc)
	if err == nil {
		t.Fatal("expected error for unknown session ID, got nil")
	}
}

func TestResumeCommand_ValidSession(t *testing.T) {
	registerMock(t)

	st := newMockStore()
	// Save a mock session.
	sess := agent.NewSession("mock", "test-model")
	_ = st.Save(context.Background(), sess)

	cmd := &ResumeCommand{
		store:   st,
		message: "hello again",
		format:  "text",
	}

	var out, errBuf bytes.Buffer
	tc := &terminal.Context{
		Stdout: &out,
		Stderr: &errBuf,
		Args:   []string{sess.ID},
	}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResumeCommand_SkillNotRegistered_ReturnsError(t *testing.T) {
	registerMock(t)

	st := newMockStore()
	sess := agent.NewSession("mock", "test-model")
	_ = st.Save(context.Background(), sess)

	cmd := &ResumeCommand{
		store:     st,
		skillName: "resume-skill-that-does-not-exist-xyz",
		format:    "text",
	}

	var out, errBuf bytes.Buffer
	tc := &terminal.Context{
		Stdout: &out,
		Stderr: &errBuf,
		Args:   []string{sess.ID},
	}

	err := cmd.Run(context.Background(), tc)
	if err == nil {
		t.Fatal("expected error for unregistered skill, got nil")
	}
	if !strings.Contains(err.Error(), "resume-skill-that-does-not-exist-xyz") {
		t.Errorf("error should mention the skill name, got: %v", err)
	}
}

func TestResumeCommand_SkillSetsSessionFields(t *testing.T) {
	registerMock(t)

	// unique name prevents double-registration across -count=N runs;
	// ResetSkillRegistry is not exported outside pkg/agent tests.
	agent.RegisterSkill(agent.Skill{
		Name:         "resume-test-skill-sets-fields",
		SystemPrompt: "You are a resume assistant",
	})

	st := newMockStore()
	sess := agent.NewSession("mock", "test-model")
	_ = st.Save(context.Background(), sess)

	cmd := &ResumeCommand{
		store:     st,
		skillName: "resume-test-skill-sets-fields",
		message:   "hello",
		format:    "text",
	}

	var out, errBuf bytes.Buffer
	tc := &terminal.Context{
		Stdout: &out,
		Stderr: &errBuf,
		Args:   []string{sess.ID},
	}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The session should have been saved with the skill's system prompt.
	sessions, _ := st.List(context.Background())
	if len(sessions) == 0 {
		t.Fatal("expected at least one saved session")
	}
	found := false
	for _, s := range sessions {
		if s.SystemPrompt == "You are a resume assistant" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected session SystemPrompt to be set from skill")
	}
}
