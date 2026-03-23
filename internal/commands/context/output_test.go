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

func TestStdinReader_WithInjected(t *testing.T) {
	r := strings.NewReader("injected")
	tc := &terminal.Context{Stdin: r}
	got := stdinReader(tc)
	if got != r {
		t.Error("stdinReader should return tc.Stdin when it is non-nil")
	}
}

func TestStdinReader_NilUsesOSStdin(t *testing.T) {
	tc := &terminal.Context{}
	got := stdinReader(tc)
	if got == nil {
		t.Error("stdinReader should return os.Stdin when tc.Stdin is nil")
	}
}

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

func TestStreamText_WritesTokensInOrder(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	events := makeTokenEvents("Hello", ", ", "world")

	text, err := streamText(ctx, &buf, seqFromEvents(events))
	if err != nil {
		t.Fatalf("streamText error: %v", err)
	}
	if !strings.HasPrefix(buf.String(), "Hello, world") {
		t.Errorf("expected output to contain tokens in order, got %q", buf.String())
	}
	if text != "Hello, world" {
		t.Errorf("returned text = %q, want %q", text, "Hello, world")
	}
}

func TestStreamText_EmptyTokens(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	events := []agent.Event{{Kind: agent.EventKindStart}, {Kind: agent.EventKindDone}}

	text, err := streamText(ctx, &buf, seqFromEvents(events))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "" {
		t.Errorf("expected empty text, got %q", text)
	}
}

func TestStreamNDJSON_EmitsValidJSONPerEvent(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	events := makeTokenEvents("tok1", "tok2")

	_, err := streamNDJSON(ctx, &buf, seqFromEvents(events))
	if err != nil {
		t.Fatalf("streamNDJSON error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) == 0 {
		t.Fatal("expected at least one NDJSON line")
	}
	for i, line := range lines {
		var ev agent.Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Errorf("line %d is not valid JSON: %v — %q", i, err, line)
		}
	}
}

func TestStreamNDJSON_AccumulatesText(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	events := makeTokenEvents("foo", "bar")

	text, err := streamNDJSON(ctx, &buf, seqFromEvents(events))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "foobar" {
		t.Errorf("accumulated text = %q, want %q", text, "foobar")
	}
}

func TestDispatch_TextFormat(t *testing.T) {
	var out bytes.Buffer
	tc := &terminal.Context{Stdout: &out, Stderr: &bytes.Buffer{}}
	ctx := context.Background()
	events := makeTokenEvents("hello")

	text, err := dispatch(ctx, tc, "text", seqFromEvents(events))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "hello" {
		t.Errorf("text = %q, want %q", text, "hello")
	}
}

func TestDispatch_NDJSONFormat(t *testing.T) {
	var out bytes.Buffer
	tc := &terminal.Context{Stdout: &out, Stderr: &bytes.Buffer{}}
	ctx := context.Background()
	events := makeTokenEvents("hello")

	_, err := dispatch(ctx, tc, "ndjson", seqFromEvents(events))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), `"kind"`) {
		t.Errorf("expected NDJSON output containing JSON fields, got %q", out.String())
	}
}
