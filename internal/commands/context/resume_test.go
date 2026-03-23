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
