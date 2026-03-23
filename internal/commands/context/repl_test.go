package ctxcmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

func newREPLContext(stdout, stderr *bytes.Buffer) *terminal.Context {
	return &terminal.Context{Stdout: stdout, Stderr: stderr}
}

func TestREPL_ExitCommand(t *testing.T) {
	a := &mockAgent{events: makeTokenEvents("ignored")}
	st := newMockStore()
	sess := agent.NewSession("mock", "test-model")

	var out, errBuf bytes.Buffer
	tc := newREPLContext(&out, &errBuf)

	input := strings.NewReader("/exit\n")
	err := runREPL(context.Background(), tc, a, st, sess, "text", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No messages should have been sent.
	if len(sess.History) != 0 {
		t.Errorf("expected empty history, got %d entries", len(sess.History))
	}
}

func TestREPL_QuitCommand(t *testing.T) {
	a := &mockAgent{events: makeTokenEvents("ignored")}
	st := newMockStore()
	sess := agent.NewSession("mock", "test-model")

	var out, errBuf bytes.Buffer
	tc := newREPLContext(&out, &errBuf)

	input := strings.NewReader("/quit\n")
	err := runREPL(context.Background(), tc, a, st, sess, "text", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sess.History) != 0 {
		t.Errorf("expected empty history, got %d entries", len(sess.History))
	}
}

func TestREPL_ForkCommand(t *testing.T) {
	a := &mockAgent{events: makeTokenEvents("response")}
	st := newMockStore()
	sess := agent.NewSession("mock", "test-model")
	// Pre-save the session so Fork can load it.
	_ = st.Save(context.Background(), sess)

	var out, errBuf bytes.Buffer
	tc := newREPLContext(&out, &errBuf)

	// Fork then exit immediately.
	input := strings.NewReader("/fork\n/exit\n")
	err := runREPL(context.Background(), tc, a, st, sess, "text", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// stdout should contain the new session ID.
	if out.Len() == 0 {
		t.Error("expected forked session ID on stdout")
	}
}

func TestREPL_EmptyLine_Skipped(t *testing.T) {
	a := &mockAgent{events: makeTokenEvents("response")}
	st := newMockStore()
	sess := agent.NewSession("mock", "test-model")

	var out, errBuf bytes.Buffer
	tc := newREPLContext(&out, &errBuf)

	// Empty line followed by exit.
	input := strings.NewReader("\n  \n/exit\n")
	err := runREPL(context.Background(), tc, a, st, sess, "text", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No messages sent.
	if len(sess.History) != 0 {
		t.Errorf("expected empty history, got %d", len(sess.History))
	}
}

func TestREPL_ErrorRollsBackHistory(t *testing.T) {
	providerErr := errors.New("provider failed")
	a := &mockAgent{err: providerErr}
	st := newMockStore()
	sess := agent.NewSession("mock", "test-model")

	var out, errBuf bytes.Buffer
	tc := newREPLContext(&out, &errBuf)

	// Send a message that will fail, then exit.
	input := strings.NewReader("hello\n/exit\n")
	err := runREPL(context.Background(), tc, a, st, sess, "text", input)
	if err != nil {
		t.Fatalf("runREPL should not return error for provider failure: %v", err)
	}
	// History should be rolled back.
	if len(sess.History) != 0 {
		t.Errorf("expected empty history after rollback, got %d", len(sess.History))
	}
	// Error should be printed to stderr.
	if !strings.Contains(errBuf.String(), "error") {
		t.Errorf("expected error message on stderr, got %q", errBuf.String())
	}
}

func TestREPL_EOFCleanExit(t *testing.T) {
	a := &mockAgent{events: makeTokenEvents("ok")}
	st := newMockStore()
	sess := agent.NewSession("mock", "test-model")

	var out, errBuf bytes.Buffer
	tc := newREPLContext(&out, &errBuf)

	// EOF with no content.
	input := strings.NewReader("")
	err := runREPL(context.Background(), tc, a, st, sess, "text", input)
	if err != nil {
		t.Fatalf("unexpected error on clean EOF: %v", err)
	}
}

func TestREPL_EOFWithPendingContent(t *testing.T) {
	a := &mockAgent{events: makeTokenEvents("response")}
	st := newMockStore()
	sess := agent.NewSession("mock", "test-model")

	var out, errBuf bytes.Buffer
	tc := newREPLContext(&out, &errBuf)

	// Content at EOF (no trailing newline).
	input := strings.NewReader("final message")
	err := runREPL(context.Background(), tc, a, st, sess, "text", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The message should have been processed.
	if len(sess.History) != 2 {
		t.Errorf("expected 2 history entries (user+assistant), got %d", len(sess.History))
	}
}

func TestREPL_NormalTurn(t *testing.T) {
	a := &mockAgent{events: makeTokenEvents("reply")}
	st := newMockStore()
	sess := agent.NewSession("mock", "test-model")

	var out, errBuf bytes.Buffer
	tc := newREPLContext(&out, &errBuf)

	input := strings.NewReader("say hello\n/exit\n")
	err := runREPL(context.Background(), tc, a, st, sess, "text", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sess.History) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(sess.History))
	}
	if !strings.Contains(out.String(), "reply") {
		t.Errorf("expected 'reply' in output, got %q", out.String())
	}
}
