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
