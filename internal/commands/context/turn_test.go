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

// TestDoRunTurn covers all four dispatch branches via doRunTurn, which accepts
// a tty bool so tests do not require a real PTY.
func TestDoRunTurn(t *testing.T) {
	tests := []struct {
		name      string
		msg       string
		format    string
		tty       bool
		stdinData string
		agentErr  error
		wantErr   bool
		errContains string
		wantHistory int
	}{
		{
			name:        "Branch1: explicit message single turn",
			msg:         "hello",
			format:      "text",
			tty:         false,
			wantHistory: 2,
		},
		{
			name:        "Branch2: piped stdin read as message",
			msg:         "",
			format:      "text",
			tty:         false,
			stdinData:   "piped message\n",
			wantHistory: 2,
		},
		{
			name:        "Branch2: empty piped stdin returns error",
			msg:         "",
			format:      "text",
			tty:         false,
			stdinData:   "",
			wantErr:     true,
			errContains: "no message provided",
		},
		{
			name:        "Branch3: TTY + ndjson + no message returns usage error",
			msg:         "",
			format:      "ndjson",
			tty:         true,
			wantErr:     true,
			errContains: "--format ndjson requires --message",
		},
		{
			name:        "Branch1: agent error rolls back history",
			msg:         "hello",
			format:      "text",
			tty:         false,
			agentErr:    errors.New("provider failed"),
			wantErr:     true,
			wantHistory: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var agentEvents []agent.Event
			if tt.agentErr == nil {
				agentEvents = makeTokenEvents("response text")
			}
			a := &mockAgent{events: agentEvents, err: tt.agentErr}
			st := newMockStore()
			sess := agent.NewSession("mock", "test-model")

			var out, errBuf bytes.Buffer
			var stdin *strings.Reader
			if tt.stdinData != "" {
				stdin = strings.NewReader(tt.stdinData)
			} else if !tt.tty {
				stdin = strings.NewReader("") // inject empty reader for non-TTY
			}
			tc := newTestContext(&out, &errBuf, stdin)

			err := doRunTurn(context.Background(), tc, a, st, sess, tt.msg, tt.format, tt.tty)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			if tt.wantHistory > 0 && len(sess.History) != tt.wantHistory {
				t.Errorf("history len = %d, want %d", len(sess.History), tt.wantHistory)
			}
			if tt.wantHistory == 0 && !tt.wantErr && len(sess.History) != 0 {
				t.Errorf("expected empty history, got %d entries", len(sess.History))
			}
		})
	}
}

// TestExecuteSingleTurn_SaveError verifies a save failure is non-fatal.
func TestExecuteSingleTurn_SaveError(t *testing.T) {
	a := &mockAgent{events: makeTokenEvents("ok")}
	st := newMockStore()
	st.saveErr = errors.New("disk full")
	sess := agent.NewSession("mock", "test-model")

	var out, errBuf bytes.Buffer
	tc := newTestContext(&out, &errBuf, nil)

	if err := executeSingleTurn(context.Background(), tc, a, st, sess, "hello", "text"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(errBuf.String(), "warning") {
		t.Errorf("expected save warning on stderr, got %q", errBuf.String())
	}
}
