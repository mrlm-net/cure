package ctxcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

func makeTokenEvents(tokens ...string) []agent.Event {
	evs := []agent.Event{{Kind: agent.EventKindStart}}
	for _, tok := range tokens {
		evs = append(evs, agent.Event{Kind: agent.EventKindToken, Text: tok})
	}
	evs = append(evs, agent.Event{Kind: agent.EventKindDone})
	return evs
}

func seqFromEvents(events []agent.Event) func(yield func(agent.Event, error) bool) {
	return func(yield func(agent.Event, error) bool) {
		for _, ev := range events {
			if !yield(ev, nil) {
				return
			}
		}
	}
}

func TestStdinReader(t *testing.T) {
	tests := []struct {
		name     string
		injected *strings.Reader
		wantNil  bool
	}{
		{
			name:     "returns injected reader when tc.Stdin is set",
			injected: strings.NewReader("injected"),
		},
		{
			name: "returns os.Stdin when tc.Stdin is nil",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := &terminal.Context{}
			if tt.injected != nil {
				tc.Stdin = tt.injected
			}
			got := stdinReader(tc)
			if got == nil {
				t.Fatal("stdinReader returned nil")
			}
			if tt.injected != nil && got != tt.injected {
				t.Error("stdinReader should return tc.Stdin when set")
			}
		})
	}
}

func TestStreamText(t *testing.T) {
	tests := []struct {
		name       string
		events     []agent.Event
		wantText   string
		wantOutput string
	}{
		{
			name:       "writes tokens in order",
			events:     makeTokenEvents("Hello", ", ", "world"),
			wantText:   "Hello, world",
			wantOutput: "Hello, world",
		},
		{
			name:     "empty token list produces empty text",
			events:   []agent.Event{{Kind: agent.EventKindStart}, {Kind: agent.EventKindDone}},
			wantText: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			text, err := streamText(context.Background(), &buf, seqFromEvents(tt.events))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if text != tt.wantText {
				t.Errorf("text = %q, want %q", text, tt.wantText)
			}
			if tt.wantOutput != "" && !strings.HasPrefix(buf.String(), tt.wantOutput) {
				t.Errorf("output = %q, want prefix %q", buf.String(), tt.wantOutput)
			}
		})
	}
}

func TestStreamNDJSON(t *testing.T) {
	tests := []struct {
		name         string
		events       []agent.Event
		wantText     string
		wantValidJSON bool
	}{
		{
			name:          "emits valid JSON per event",
			events:        makeTokenEvents("tok1", "tok2"),
			wantValidJSON: true,
		},
		{
			name:     "accumulates token text",
			events:   makeTokenEvents("foo", "bar"),
			wantText: "foobar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			text, err := streamNDJSON(context.Background(), &buf, seqFromEvents(tt.events))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantText != "" && text != tt.wantText {
				t.Errorf("text = %q, want %q", text, tt.wantText)
			}
			if tt.wantValidJSON {
				for i, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
					if line == "" {
						continue
					}
					var ev agent.Event
					if err := json.Unmarshal([]byte(line), &ev); err != nil {
						t.Errorf("line %d is not valid JSON: %v — %q", i, err, line)
					}
				}
			}
		})
	}
}

func TestDispatch(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		events     []agent.Event
		wantText   string
		wantInOut  string
	}{
		{
			name:     "text format writes tokens and returns text",
			format:   "text",
			events:   makeTokenEvents("hello"),
			wantText: "hello",
		},
		{
			name:      "ndjson format emits JSON and returns text",
			format:    "ndjson",
			events:    makeTokenEvents("hello"),
			wantInOut: `"kind"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			tc := &terminal.Context{Stdout: &out, Stderr: &bytes.Buffer{}}

			text, err := dispatch(context.Background(), tc, tt.format, seqFromEvents(tt.events))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantText != "" && text != tt.wantText {
				t.Errorf("text = %q, want %q", text, tt.wantText)
			}
			if tt.wantInOut != "" && !strings.Contains(out.String(), tt.wantInOut) {
				t.Errorf("output = %q, want to contain %q", out.String(), tt.wantInOut)
			}
		})
	}
}
