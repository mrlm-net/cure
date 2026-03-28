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

func TestForkCommand(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		seedSession *agent.Session
		forkErr     error
		wantErr     bool
		errContains string
		wantForkOf  string // non-empty: check output ID maps to a forked session
	}{
		{
			name:        "missing argument returns error",
			args:        []string{},
			wantErr:     true,
			errContains: "missing <session-id>",
		},
		{
			name:        "unknown session returns not-found error",
			args:        []string{"nonexistent"},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:        "forks session and prints new ID",
			args:        []string{""}, // will be replaced with real ID below
			seedSession: agent.NewSession("claude", "m"),
			wantErr:     false,
		},
		{
			name:    "store fork error is propagated",
			args:    []string{"someid"},
			forkErr: errors.New("storage failure"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newMockStore()

			var seedID string
			if tt.seedSession != nil {
				_ = st.Save(context.Background(), tt.seedSession)
				seedID = tt.seedSession.ID
				if len(tt.args) > 0 && tt.args[0] == "" {
					tt.args[0] = seedID
				}
			}
			if tt.forkErr != nil {
				st.forkErr = tt.forkErr
				// Seed a dummy session so Load doesn't fail before fork.
				dummy := agent.NewSession("p", "m")
				_ = st.Save(context.Background(), dummy)
				tt.args[0] = dummy.ID
			}

			cmd := &ForkCommand{store: st}
			var out bytes.Buffer
			tc := &terminal.Context{
				Stdout: &out,
				Stderr: &bytes.Buffer{},
				Args:   tt.args,
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

			// Output must be a non-empty ID.
			outputID := strings.TrimSpace(out.String())
			if outputID == "" {
				t.Fatal("expected forked session ID on stdout, got empty output")
			}
			// Must differ from source.
			if seedID != "" && outputID == seedID {
				t.Errorf("forked ID %q is same as source ID", outputID)
			}

			// Forked session should be persisted and have ForkOf set.
			forked, loadErr := st.Load(context.Background(), outputID)
			if loadErr != nil {
				t.Fatalf("forked session not persisted: %v", loadErr)
			}
			if forked.ForkOf != seedID {
				t.Errorf("ForkOf = %q, want %q", forked.ForkOf, seedID)
			}
		})
	}
}

func TestForkIsolation(t *testing.T) {
	st := newMockStore()
	src := agent.NewSession("claude", "m")
	src.AppendUserMessage("hello")
	_ = st.Save(context.Background(), src)

	cmd := &ForkCommand{store: st}
	var out bytes.Buffer
	tc := &terminal.Context{
		Stdout: &out,
		Stderr: &bytes.Buffer{},
		Args:   []string{src.ID},
	}

	if err := cmd.Run(context.Background(), tc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	forkedID := strings.TrimSpace(out.String())
	forked, _ := st.Load(context.Background(), forkedID)

	// Mutate forked history — source must be unaffected.
	forked.History[0].Content = agent.MessageContent{agent.TextBlock{Text: "mutated"}}
	reloaded, _ := st.Load(context.Background(), src.ID)
	if agent.TextOf(reloaded.History[0].Content) == "mutated" {
		t.Error("fork shares history slice with source")
	}
}

func TestForkCommand_ErrSessionNotFound(t *testing.T) {
	st := newMockStore()
	cmd := &ForkCommand{store: st}
	var out bytes.Buffer
	tc := &terminal.Context{
		Stdout: &out,
		Stderr: &bytes.Buffer{},
		Args:   []string{"doesnotexist"},
	}

	err := cmd.Run(context.Background(), tc)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, agent.ErrSessionNotFound) {
		// The error is wrapped — check message instead.
		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("error = %q, want 'not found'", err.Error())
		}
	}
}
