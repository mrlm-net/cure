package agent_test

import (
	"encoding/json"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
)

func TestMessageContent_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		mc      agent.MessageContent
		wantRaw string // expected JSON (string or array)
	}{
		{
			name:    "single TextBlock marshals as plain string",
			mc:      agent.MessageContent{agent.TextBlock{Text: "hello"}},
			wantRaw: `"hello"`,
		},
		{
			name:    "empty MessageContent marshals as empty JSON array",
			mc:      agent.MessageContent{},
			wantRaw: `[]`,
		},
		{
			name: "multiple blocks marshals as typed array",
			mc: agent.MessageContent{
				agent.TextBlock{Text: "before"},
				agent.TextBlock{Text: "after"},
			},
			wantRaw: `[{"type":"text","text":"before"},{"type":"text","text":"after"}]`,
		},
		{
			name: "ToolUseBlock marshals with type discriminator",
			mc: agent.MessageContent{
				agent.ToolUseBlock{ID: "tu_1", Name: "search", Input: map[string]any{"q": "golang"}},
			},
			wantRaw: `[{"type":"tool_use","id":"tu_1","name":"search","input":{"q":"golang"}}]`,
		},
		{
			name: "ToolResultBlock marshals with type discriminator",
			mc: agent.MessageContent{
				agent.ToolResultBlock{ID: "tu_1", ToolName: "search", Result: "found it", IsError: false},
			},
			wantRaw: `[{"type":"tool_result","id":"tu_1","tool_name":"search","result":"found it"}]`,
		},
		{
			name: "ToolResultBlock with IsError marshals is_error field",
			mc: agent.MessageContent{
				agent.ToolResultBlock{ID: "tu_2", ToolName: "run", Result: "failed", IsError: true},
			},
			wantRaw: `[{"type":"tool_result","id":"tu_2","tool_name":"run","result":"failed","is_error":true}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.mc)
			if err != nil {
				t.Fatalf("MarshalJSON: %v", err)
			}
			if string(got) != tt.wantRaw {
				t.Errorf("MarshalJSON = %s, want %s", string(got), tt.wantRaw)
			}
		})
	}
}

func TestMessageContent_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLen int
		check   func(t *testing.T, mc agent.MessageContent)
	}{
		{
			name:    "plain string decodes as single TextBlock (backward compat)",
			input:   `"hello world"`,
			wantLen: 1,
			check: func(t *testing.T, mc agent.MessageContent) {
				t.Helper()
				tb, ok := mc[0].(agent.TextBlock)
				if !ok {
					t.Fatalf("mc[0] is %T, want TextBlock", mc[0])
				}
				if tb.Text != "hello world" {
					t.Errorf("TextBlock.Text = %q, want %q", tb.Text, "hello world")
				}
			},
		},
		{
			name:    "typed text block array",
			input:   `[{"type":"text","text":"hi"}]`,
			wantLen: 1,
			check: func(t *testing.T, mc agent.MessageContent) {
				t.Helper()
				tb, ok := mc[0].(agent.TextBlock)
				if !ok {
					t.Fatalf("mc[0] is %T, want TextBlock", mc[0])
				}
				if tb.Text != "hi" {
					t.Errorf("TextBlock.Text = %q, want %q", tb.Text, "hi")
				}
			},
		},
		{
			name:    "tool_use block decoded correctly",
			input:   `[{"type":"tool_use","id":"tu_1","name":"search","input":{"q":"test"}}]`,
			wantLen: 1,
			check: func(t *testing.T, mc agent.MessageContent) {
				t.Helper()
				tub, ok := mc[0].(agent.ToolUseBlock)
				if !ok {
					t.Fatalf("mc[0] is %T, want ToolUseBlock", mc[0])
				}
				if tub.ID != "tu_1" {
					t.Errorf("ID = %q, want %q", tub.ID, "tu_1")
				}
				if tub.Name != "search" {
					t.Errorf("Name = %q, want %q", tub.Name, "search")
				}
			},
		},
		{
			name:    "tool_result block decoded correctly",
			input:   `[{"type":"tool_result","id":"tu_1","tool_name":"search","result":"found","is_error":false}]`,
			wantLen: 1,
			check: func(t *testing.T, mc agent.MessageContent) {
				t.Helper()
				trb, ok := mc[0].(agent.ToolResultBlock)
				if !ok {
					t.Fatalf("mc[0] is %T, want ToolResultBlock", mc[0])
				}
				if trb.ID != "tu_1" {
					t.Errorf("ID = %q, want %q", trb.ID, "tu_1")
				}
				if trb.Result != "found" {
					t.Errorf("Result = %q, want %q", trb.Result, "found")
				}
				if trb.IsError {
					t.Error("IsError should be false")
				}
			},
		},
		{
			name:    "multi-block array decoded in order",
			input:   `[{"type":"text","text":"first"},{"type":"text","text":"second"}]`,
			wantLen: 2,
			check: func(t *testing.T, mc agent.MessageContent) {
				t.Helper()
				tb0, ok0 := mc[0].(agent.TextBlock)
				tb1, ok1 := mc[1].(agent.TextBlock)
				if !ok0 || !ok1 {
					t.Fatalf("expected two TextBlocks, got %T, %T", mc[0], mc[1])
				}
				if tb0.Text != "first" || tb1.Text != "second" {
					t.Errorf("texts = %q, %q, want %q, %q", tb0.Text, tb1.Text, "first", "second")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mc agent.MessageContent
			if err := json.Unmarshal([]byte(tt.input), &mc); err != nil {
				t.Fatalf("UnmarshalJSON: %v", err)
			}
			if len(mc) != tt.wantLen {
				t.Fatalf("len(mc) = %d, want %d", len(mc), tt.wantLen)
			}
			if tt.check != nil {
				tt.check(t, mc)
			}
		})
	}
}

func TestMessageContent_UnmarshalJSON_Errors(t *testing.T) {
	t.Run("unknown type returns error", func(t *testing.T) {
		var mc agent.MessageContent
		err := json.Unmarshal([]byte(`[{"type":"unknown_block","data":"x"}]`), &mc)
		if err == nil {
			t.Fatal("expected error for unknown block type, got nil")
		}
	})
	t.Run("invalid JSON returns error", func(t *testing.T) {
		var mc agent.MessageContent
		err := json.Unmarshal([]byte(`not json`), &mc)
		if err == nil {
			t.Fatal("expected error for invalid JSON, got nil")
		}
	})
}

func TestMessageContent_RoundTrip(t *testing.T) {
	t.Run("TextBlock round-trip via plain string", func(t *testing.T) {
		orig := agent.MessageContent{agent.TextBlock{Text: "round trip"}}
		b, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		var got agent.MessageContent
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		tb, ok := got[0].(agent.TextBlock)
		if !ok {
			t.Fatalf("got[0] = %T, want TextBlock", got[0])
		}
		if tb.Text != "round trip" {
			t.Errorf("Text = %q, want %q", tb.Text, "round trip")
		}
	})

	t.Run("ToolUseBlock round-trip via typed array", func(t *testing.T) {
		orig := agent.MessageContent{
			agent.ToolUseBlock{ID: "x1", Name: "calc", Input: map[string]any{"expr": "1+1"}},
		}
		b, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		var got agent.MessageContent
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		tub, ok := got[0].(agent.ToolUseBlock)
		if !ok {
			t.Fatalf("got[0] = %T, want ToolUseBlock", got[0])
		}
		if tub.ID != "x1" || tub.Name != "calc" {
			t.Errorf("tub = {%q, %q}, want {%q, %q}", tub.ID, tub.Name, "x1", "calc")
		}
	})

	t.Run("ToolResultBlock round-trip via typed array", func(t *testing.T) {
		orig := agent.MessageContent{
			agent.ToolResultBlock{ID: "r1", ToolName: "calc", Result: "2", IsError: false},
		}
		b, err := json.Marshal(orig)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		var got agent.MessageContent
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		trb, ok := got[0].(agent.ToolResultBlock)
		if !ok {
			t.Fatalf("got[0] = %T, want ToolResultBlock", got[0])
		}
		if trb.ID != "r1" || trb.Result != "2" {
			t.Errorf("trb = {%q, %q}, want {%q, %q}", trb.ID, trb.Result, "r1", "2")
		}
	})
}

func TestTextOf(t *testing.T) {
	tests := []struct {
		name string
		mc   agent.MessageContent
		want string
	}{
		{
			name: "single TextBlock",
			mc:   agent.MessageContent{agent.TextBlock{Text: "hello"}},
			want: "hello",
		},
		{
			name: "multiple TextBlocks joined",
			mc:   agent.MessageContent{agent.TextBlock{Text: "foo"}, agent.TextBlock{Text: "bar"}},
			want: "foobar",
		},
		{
			name: "non-text blocks ignored",
			mc:   agent.MessageContent{agent.ToolUseBlock{ID: "1", Name: "fn"}, agent.TextBlock{Text: "result"}},
			want: "result",
		},
		{
			name: "empty content returns empty string",
			mc:   agent.MessageContent{},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agent.TextOf(tt.mc)
			if got != tt.want {
				t.Errorf("TextOf() = %q, want %q", got, tt.want)
			}
		})
	}
}

func BenchmarkMessageContent_Marshal(b *testing.B) {
	mc := agent.MessageContent{agent.TextBlock{Text: "benchmark content"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(mc)
	}
}

func BenchmarkMessageContent_Unmarshal(b *testing.B) {
	data := []byte(`"benchmark content"`)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var mc agent.MessageContent
		_ = json.Unmarshal(data, &mc)
	}
}
