package ctxcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

func TestListCommand(t *testing.T) {
	tests := []struct {
		name         string
		sessions     []*agent.Session
		format       string
		provider     string
		wantContains []string
		wantNot      []string
	}{
		{
			name:         "empty store prints no-sessions message",
			sessions:     nil,
			format:       "text",
			wantContains: []string{"No sessions found"},
		},
		{
			name: "text format shows truncated ID and provider",
			sessions: []*agent.Session{
				{ID: "abcdef123456789", Provider: "claude", Model: "claude-opus-4-6", History: []agent.Message{{}, {}}, UpdatedAt: time.Now()},
			},
			format:       "text",
			wantContains: []string{"abcdef123456", "claude", "2"},
		},
		{
			name: "ndjson format emits valid JSON",
			sessions: []*agent.Session{
				{ID: "abc123", Provider: "claude", Model: "m", History: []agent.Message{}, UpdatedAt: time.Now()},
			},
			format:       "ndjson",
			wantContains: []string{`"id"`},
		},
		{
			name: "provider filter hides non-matching sessions",
			sessions: []*agent.Session{
				{ID: "aaa", Provider: "claude", Model: "m", History: []agent.Message{}, UpdatedAt: time.Now()},
				{ID: "bbb", Provider: "openai", Model: "m", History: []agent.Message{}, UpdatedAt: time.Now()},
			},
			format:       "text",
			provider:     "claude",
			wantContains: []string{"aaa"},
			wantNot:      []string{"bbb"},
		},
		{
			name: "long model name is truncated in text format",
			sessions: []*agent.Session{
				{ID: "abc", Provider: "p", Model: "very-long-model-name-xyz", History: []agent.Message{}, UpdatedAt: time.Now()},
			},
			format:       "text",
			wantContains: []string{"..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newMockStore()
			for _, s := range tt.sessions {
				_ = st.Save(context.Background(), s)
			}

			cmd := &ListCommand{store: st, format: tt.format, provider: tt.provider}
			var out bytes.Buffer
			tc := &terminal.Context{Stdout: &out, Stderr: &bytes.Buffer{}}

			if err := cmd.Run(context.Background(), tc); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := out.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output %q does not contain %q", output, want)
				}
			}
			for _, notWant := range tt.wantNot {
				if strings.Contains(output, notWant) {
					t.Errorf("output %q should not contain %q", output, notWant)
				}
			}
		})
	}
}

func TestListNDJSONValid(t *testing.T) {
	st := newMockStore()
	sess := agent.NewSession("claude", "claude-opus-4-6")
	_ = st.Save(context.Background(), sess)

	cmd := &ListCommand{store: st, format: "ndjson"}
	var out bytes.Buffer
	tc := &terminal.Context{Stdout: &out, Stderr: &bytes.Buffer{}}

	if err := cmd.Run(context.Background(), tc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		var s agent.Session
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			t.Errorf("line %d is not valid JSON: %v — %q", i, err, line)
		}
	}
}

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		name string
		age  time.Duration
		want string
	}{
		{"just now", 10 * time.Second, "just now"},
		{"minutes", 5 * time.Minute, "5m ago"},
		{"hours", 3 * time.Hour, "3h ago"},
		{"days", 50 * time.Hour, "2d ago"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativeTime(time.Now().Add(-tt.age))
			if got != tt.want {
				t.Errorf("relativeTime = %q, want %q", got, tt.want)
			}
		})
	}
}
