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

func TestDeleteCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		seedSession *agent.Session
		yes         bool
		stdinInput  string // empty means no stdin injection (TTY-simulated nil)
		wantErr     bool
		errContains string
		wantOutput  string
		wantDeleted bool
	}{
		{
			name:        "missing argument returns error",
			args:        []string{},
			wantErr:     true,
			errContains: "missing <session-id>",
		},
		{
			name:        "unknown session returns not-found error",
			args:        []string{"ghost"},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:        "--yes skips prompt and deletes session",
			args:        []string{""}, // filled below
			seedSession: agent.NewSession("claude", "m"),
			yes:         true,
			wantOutput:  "deleted",
			wantDeleted: true,
		},
		{
			name:        "confirm y deletes session",
			args:        []string{""},
			seedSession: agent.NewSession("claude", "m"),
			stdinInput:  "y\n",
			wantOutput:  "deleted",
			wantDeleted: true,
		},
		{
			name:        "confirm yes deletes session",
			args:        []string{""},
			seedSession: agent.NewSession("claude", "m"),
			stdinInput:  "yes\n",
			wantOutput:  "deleted",
			wantDeleted: true,
		},
		{
			name:        "confirm n aborts without deleting",
			args:        []string{""},
			seedSession: agent.NewSession("claude", "m"),
			stdinInput:  "n\n",
			wantOutput:  "Aborted",
			wantDeleted: false,
		},
		{
			name:        "empty confirmation line aborts",
			args:        []string{""},
			seedSession: agent.NewSession("claude", "m"),
			stdinInput:  "\n",
			wantOutput:  "Aborted",
			wantDeleted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newMockStore()

			if tt.seedSession != nil {
				_ = st.Save(context.Background(), tt.seedSession)
				if len(tt.args) > 0 && tt.args[0] == "" {
					tt.args[0] = tt.seedSession.ID
				}
			}

			cmd := &DeleteCommand{store: st, yes: tt.yes}
			var out bytes.Buffer
			var stdinReader *strings.Reader
			if tt.stdinInput != "" {
				stdinReader = strings.NewReader(tt.stdinInput)
			}
			tc := &terminal.Context{
				Stdout: &out,
				Stderr: &bytes.Buffer{},
				Args:   tt.args,
				Stdin:  stdinReader,
			}

			err := cmd.Run(context.Background(), tc)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantOutput != "" && !strings.Contains(out.String(), tt.wantOutput) {
				t.Errorf("output = %q, want to contain %q", out.String(), tt.wantOutput)
			}

			if tt.seedSession != nil {
				_, loadErr := st.Load(context.Background(), tt.seedSession.ID)
				deleted := errors.Is(loadErr, agent.ErrSessionNotFound)
				if tt.wantDeleted && !deleted {
					t.Error("expected session to be deleted, but it still exists")
				}
				if !tt.wantDeleted && deleted {
					t.Error("expected session to remain, but it was deleted")
				}
			}
		})
	}
}
