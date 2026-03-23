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

func newTestContext(stdout, stderr *bytes.Buffer, stdin *strings.Reader) *terminal.Context {
	tc := &terminal.Context{
		Stdout: stdout,
		Stderr: stderr,
	}
	if stdin != nil {
		tc.Stdin = stdin
	}
	return tc
}

// TestRunTurn_WithMessage tests Branch 1: explicit message.
func TestRunTurn_WithMessage(t *testing.T) {
	a := &mockAgent{events: makeTokenEvents("response text")}
	st := newMockStore()
	sess := agent.NewSession("mock", "test-model")

	var out, errBuf bytes.Buffer
	tc := newTestContext(&out, &errBuf, strings.NewReader("")) // non-TTY stdin

	err := runTurn(context.Background(), tc, a, st, sess, "hello", "text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Session should have 2 messages: user + assistant.
	if len(sess.History) != 2 {
		t.Errorf("expected 2 history entries, got %d", len(sess.History))
	}
	if sess.History[0].Role != agent.RoleUser {
		t.Errorf("expected first message role user, got %q", sess.History[0].Role)
	}
	if !strings.Contains(out.String(), "response text") {
		t.Errorf("expected output to contain response text, got %q", out.String())
	}
}

// TestRunTurn_PipedStdin tests Branch 2: non-TTY stdin with piped input.
func TestRunTurn_PipedStdin(t *testing.T) {
	a := &mockAgent{events: makeTokenEvents("piped reply")}
	st := newMockStore()
	sess := agent.NewSession("mock", "test-model")

	var out, errBuf bytes.Buffer
	// Inject a non-empty reader as stdin (simulates piped input).
	tc := newTestContext(&out, &errBuf, strings.NewReader("piped message\n"))

	// In tests isatty will return false for real os.Stdin.Fd() since tests
	// run without a terminal. The injected tc.Stdin is the reader branch.
	// We leave msg="" so the code will attempt to read stdin.
	err := runTurn(context.Background(), tc, a, st, sess, "", "text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sess.History) < 1 {
		t.Error("expected at least one history entry after piped stdin turn")
	}
}

// TestRunTurn_NDJSONWithTTY_ReturnsError tests Branch 3.
// We simulate this by testing executeSingleTurn with ndjson format
// and an error event to ensure error propagation works correctly.
// The actual TTY detection branch (branch 3) requires isatty=true which
// cannot be forced in a test without a real PTY, but we test the error path.
func TestRunTurn_AgentError_RollsBackHistory(t *testing.T) {
	providerErr := errors.New("provider failed")
	a := &mockAgent{err: providerErr}
	st := newMockStore()
	sess := agent.NewSession("mock", "test-model")

	var out, errBuf bytes.Buffer
	tc := newTestContext(&out, &errBuf, nil)

	err := executeSingleTurn(context.Background(), tc, a, st, sess, "hello", "text")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// History must be rolled back — no messages persisted after error.
	if len(sess.History) != 0 {
		t.Errorf("expected empty history after rollback, got %d entries", len(sess.History))
	}
}

// TestRunTurn_SaveError_NonFatal verifies a save failure does not abort the command.
func TestRunTurn_SaveError_NonFatal(t *testing.T) {
	a := &mockAgent{events: makeTokenEvents("ok")}
	st := newMockStore()
	st.saveErr = errors.New("disk full")
	sess := agent.NewSession("mock", "test-model")

	var out, errBuf bytes.Buffer
	tc := newTestContext(&out, &errBuf, nil)

	err := executeSingleTurn(context.Background(), tc, a, st, sess, "hello", "text")
	// Command should succeed even when save fails.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(errBuf.String(), "warning") {
		t.Errorf("expected save warning on stderr, got %q", errBuf.String())
	}
}
