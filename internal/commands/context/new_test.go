package ctxcmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// newCommandForTest builds a NewCommand wired to the given store and with the
// mock provider registered so agent.New("mock", nil) works.
func registerMock(t *testing.T) {
	t.Helper()
	// Register a "mock" provider if not already registered.
	registered := agent.Registered()
	for _, p := range registered {
		if p == "mock" {
			return
		}
	}
	agent.Register("mock", func(cfg map[string]any) (agent.Agent, error) {
		return &mockAgent{events: makeTokenEvents("mock response")}, nil
	})
}

func TestNewCommand_MissingProvider(t *testing.T) {
	st := newMockStore()
	cmd := &NewCommand{store: st}
	// Simulate flag parsing leaving provider empty.
	cmd.provider = ""

	var out, errBuf bytes.Buffer
	tc := &terminal.Context{Stdout: &out, Stderr: &errBuf, Stdin: strings.NewReader("")}

	err := cmd.Run(context.Background(), tc)
	if err == nil {
		t.Fatal("expected error for missing --provider, got nil")
	}
	if !strings.Contains(err.Error(), "--provider") {
		t.Errorf("error should mention --provider, got: %v", err)
	}
}

func TestNewCommand_SessionNameSetsTag(t *testing.T) {
	registerMock(t)

	st := newMockStore()
	cmd := &NewCommand{
		store:       st,
		provider:    "mock",
		sessionName: "my-session",
		message:     "hello",
		format:      "text",
	}

	var out, errBuf bytes.Buffer
	tc := &terminal.Context{Stdout: &out, Stderr: &errBuf}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The session should have been saved with the name tag.
	sessions, _ := st.List(context.Background())
	if len(sessions) == 0 {
		t.Fatal("expected at least one saved session")
	}
	sess := sessions[0]
	found := false
	for _, tag := range sess.Tags {
		if tag == "name:my-session" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected tag 'name:my-session', got tags: %v", sess.Tags)
	}
}

func TestNewCommand_TagsPopulated(t *testing.T) {
	registerMock(t)

	st := newMockStore()
	cmd := &NewCommand{
		store:    st,
		provider: "mock",
		tags:     []string{"project:myapp", "sprint:3"},
		message:  "hello",
		format:   "text",
	}

	var out, errBuf bytes.Buffer
	tc := &terminal.Context{Stdout: &out, Stderr: &errBuf}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sessions, _ := st.List(context.Background())
	if len(sessions) == 0 {
		t.Fatal("expected at least one saved session")
	}
	sess := sessions[0]

	wantTags := []string{"project:myapp", "sprint:3"}
	for _, want := range wantTags {
		found := false
		for _, tag := range sess.Tags {
			if tag == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected tag %q in %v", want, sess.Tags)
		}
	}
}

func TestNewCommand_TagAndSessionNameCoexist(t *testing.T) {
	registerMock(t)

	st := newMockStore()
	cmd := &NewCommand{
		store:       st,
		provider:    "mock",
		sessionName: "my-session",
		tags:        []string{"project:myapp"},
		message:     "hello",
		format:      "text",
	}

	var out, errBuf bytes.Buffer
	tc := &terminal.Context{Stdout: &out, Stderr: &errBuf}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sessions, _ := st.List(context.Background())
	if len(sessions) == 0 {
		t.Fatal("expected at least one saved session")
	}
	sess := sessions[0]

	wantTags := []string{"name:my-session", "project:myapp"}
	for _, want := range wantTags {
		found := false
		for _, tag := range sess.Tags {
			if tag == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected tag %q in %v", want, sess.Tags)
		}
	}
}

func TestStringSliceFlag_EmptyValueReturnsError(t *testing.T) {
	var f stringSliceFlag
	err := f.Set("")
	if err == nil {
		t.Fatal("expected error for empty tag value, got nil")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("error should mention 'cannot be empty', got: %v", err)
	}
}

func TestStringSliceFlag_AccumulatesValues(t *testing.T) {
	var f stringSliceFlag
	for _, v := range []string{"a", "b", "c"} {
		if err := f.Set(v); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if len(f) != 3 {
		t.Fatalf("expected 3 values, got %d: %v", len(f), f)
	}
	if f.String() != "a,b,c" {
		t.Errorf("String() = %q, want %q", f.String(), "a,b,c")
	}
}

func TestNewCommand_SkillNotRegistered_ReturnsError(t *testing.T) {
	registerMock(t)

	st := newMockStore()
	cmd := &NewCommand{
		store:     st,
		provider:  "mock",
		skillName: "skill-that-does-not-exist-xyz",
		message:   "hello",
		format:    "text",
	}

	var out, errBuf bytes.Buffer
	tc := &terminal.Context{Stdout: &out, Stderr: &errBuf}

	err := cmd.Run(context.Background(), tc)
	if err == nil {
		t.Fatal("expected error for unknown skill, got nil")
	}
	if !strings.Contains(err.Error(), "skill-that-does-not-exist-xyz") {
		t.Errorf("error should mention skill name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("error should mention 'not registered', got: %v", err)
	}
}

func TestNewCommand_SkillSetsSessionFields(t *testing.T) {
	registerMock(t)

	agent.RegisterSkill(agent.Skill{
		Name:         "ctxcmd-test-skill-sets-fields",
		Description:  "A test skill",
		SystemPrompt: "You are a test assistant",
		Tools:        []agent.Tool{},
	})

	st := newMockStore()
	cmd := &NewCommand{
		store:     st,
		provider:  "mock",
		skillName: "ctxcmd-test-skill-sets-fields",
		message:   "hello",
		format:    "text",
	}

	var out, errBuf bytes.Buffer
	tc := &terminal.Context{Stdout: &out, Stderr: &errBuf}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sessions, _ := st.List(context.Background())
	if len(sessions) == 0 {
		t.Fatal("expected at least one saved session")
	}
	sess := sessions[0]

	if sess.SkillName != "ctxcmd-test-skill-sets-fields" {
		t.Errorf("SkillName = %q, want %q", sess.SkillName, "ctxcmd-test-skill-sets-fields")
	}
	if sess.SystemPrompt != "You are a test assistant" {
		t.Errorf("SystemPrompt = %q, want %q", sess.SystemPrompt, "You are a test assistant")
	}
}

func TestNewCommand_SkillOverridesSystemPrompt(t *testing.T) {
	registerMock(t)

	agent.RegisterSkill(agent.Skill{
		Name:         "ctxcmd-test-skill-override-prompt",
		Description:  "An override skill",
		SystemPrompt: "Skill prompt wins",
		Tools:        nil,
	})

	st := newMockStore()
	cmd := &NewCommand{
		store:        st,
		provider:     "mock",
		skillName:    "ctxcmd-test-skill-override-prompt",
		systemPrompt: "Manual prompt loses",
		message:      "hello",
		format:       "text",
	}

	var out, errBuf bytes.Buffer
	tc := &terminal.Context{Stdout: &out, Stderr: &errBuf}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sessions, _ := st.List(context.Background())
	if len(sessions) == 0 {
		t.Fatal("expected at least one saved session")
	}
	sess := sessions[0]

	if sess.SystemPrompt != "Skill prompt wins" {
		t.Errorf("SystemPrompt = %q, want skill's prompt %q", sess.SystemPrompt, "Skill prompt wins")
	}
}

func TestNewCommand_SystemPromptSet(t *testing.T) {
	registerMock(t)

	st := newMockStore()
	cmd := &NewCommand{
		store:        st,
		provider:     "mock",
		systemPrompt: "You are a Go expert",
		message:      "hello",
		format:       "text",
	}

	var out, errBuf bytes.Buffer
	tc := &terminal.Context{Stdout: &out, Stderr: &errBuf}

	err := cmd.Run(context.Background(), tc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sessions, _ := st.List(context.Background())
	if len(sessions) == 0 {
		t.Fatal("expected at least one saved session")
	}
	sess := sessions[0]
	if sess.SystemPrompt != "You are a Go expert" {
		t.Errorf("SystemPrompt = %q, want %q", sess.SystemPrompt, "You are a Go expert")
	}
}
