package agent_test

import (
	"encoding/json"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
)

func TestEventJSON(t *testing.T) {
	tests := []struct {
		name  string
		event agent.Event
		want  map[string]any
	}{
		{
			name:  "token event serialises text only",
			event: agent.Event{Kind: agent.EventKindToken, Text: "hello"},
			want:  map[string]any{"kind": "token", "text": "hello"},
		},
		{
			name: "done event with token counts",
			event: agent.Event{
				Kind:         agent.EventKindDone,
				InputTokens:  10,
				OutputTokens: 20,
				StopReason:   "end_turn",
			},
			want: map[string]any{
				"kind":          "done",
				"input_tokens":  float64(10),
				"output_tokens": float64(20),
				"stop_reason":   "end_turn",
			},
		},
		{
			name:  "error event",
			event: agent.Event{Kind: agent.EventKindError, Err: "provider timeout"},
			want:  map[string]any{"kind": "error", "error": "provider timeout"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.event)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			var got map[string]any
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			for k, v := range tt.want {
				if got[k] != v {
					t.Errorf("field %q = %v, want %v", k, got[k], v)
				}
			}
			// Verify omitempty: no zero-value fields
			for k, v := range got {
				if _, expected := tt.want[k]; !expected {
					t.Errorf("unexpected field %q = %v in JSON output", k, v)
				}
			}
		})
	}
}

func TestRoleConstants(t *testing.T) {
	if agent.RoleUser != "user" {
		t.Errorf("RoleUser = %q, want %q", agent.RoleUser, "user")
	}
	if agent.RoleAssistant != "assistant" {
		t.Errorf("RoleAssistant = %q, want %q", agent.RoleAssistant, "assistant")
	}
	if agent.RoleSystem != "system" {
		t.Errorf("RoleSystem = %q, want %q", agent.RoleSystem, "system")
	}
}

func TestEventKindConstants(t *testing.T) {
	kinds := map[agent.EventKind]string{
		agent.EventKindToken: "token",
		agent.EventKindStart: "start",
		agent.EventKindDone:  "done",
		agent.EventKindError: "error",
	}
	for kind, want := range kinds {
		if string(kind) != want {
			t.Errorf("EventKind = %q, want %q", kind, want)
		}
	}
}
