package ctxcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/style"
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
			var buf, errBuf bytes.Buffer
			text, err := streamText(context.Background(), &buf, &errBuf, seqFromEvents(tt.events))
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

func TestStreamText_ToolCallRenderedToStderr(t *testing.T) {
	events := []agent.Event{
		{Kind: agent.EventKindToolCall, ToolCall: &agent.ToolCallEvent{
			ID:        "tc_1",
			ToolName:  "search",
			InputJSON: `{"q":"golang"}`,
		}},
	}

	var out, errBuf bytes.Buffer
	_, err := streamText(context.Background(), &out, &errBuf, seqFromEvents(events))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Tool call must appear on stderr, not stdout.
	if out.Len() != 0 {
		t.Errorf("stdout should be empty for tool_call event, got: %q", out.String())
	}
	errStr := errBuf.String()
	// Strip ANSI codes for comparison.
	plain := style.Reset(errStr)
	if !strings.Contains(plain, "[tool] search") {
		t.Errorf("stderr should contain '[tool] search', got: %q", plain)
	}
	if !strings.Contains(plain, `{"q":"golang"}`) {
		t.Errorf("stderr should contain input JSON, got: %q", plain)
	}
}

func TestStreamText_ToolResultRenderedToStderr(t *testing.T) {
	events := []agent.Event{
		{Kind: agent.EventKindToolResult, ToolResult: &agent.ToolResultEvent{
			ID:       "tc_1",
			ToolName: "search",
			Result:   "golang is awesome",
			IsError:  false,
		}},
	}

	var out, errBuf bytes.Buffer
	_, err := streamText(context.Background(), &out, &errBuf, seqFromEvents(events))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Tool result must appear on stderr, not stdout.
	if out.Len() != 0 {
		t.Errorf("stdout should be empty for tool_result event, got: %q", out.String())
	}
	errStr := errBuf.String()
	plain := style.Reset(errStr)
	if !strings.Contains(plain, "[tool result] search: golang is awesome") {
		t.Errorf("stderr should contain '[tool result] search: golang is awesome', got: %q", plain)
	}
}

func TestStreamText_ToolResultTruncated(t *testing.T) {
	longResult := strings.Repeat("x", 200)
	events := []agent.Event{
		{Kind: agent.EventKindToolResult, ToolResult: &agent.ToolResultEvent{
			ID:       "tc_1",
			ToolName: "big",
			Result:   longResult,
			IsError:  false,
		}},
	}

	var out, errBuf bytes.Buffer
	_, err := streamText(context.Background(), &out, &errBuf, seqFromEvents(events))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	plain := style.Reset(errBuf.String())
	// After truncation the result portion should end with "..."
	if !strings.Contains(plain, "...") {
		t.Errorf("expected truncation ellipsis in stderr output, got: %q", plain)
	}
	// The raw 200-char string should not appear verbatim.
	if strings.Contains(plain, longResult) {
		t.Errorf("full long result should be truncated, got: %q", plain)
	}
}

func TestStreamText_ToolResultErrorStyling(t *testing.T) {
	events := []agent.Event{
		{Kind: agent.EventKindToolResult, ToolResult: &agent.ToolResultEvent{
			ID:       "tc_err",
			ToolName: "calculator",
			Result:   "division by zero",
			IsError:  true,
		}},
	}

	// No style.Enable() needed — style.Reset() strips ANSI codes unconditionally,
	// so the assertion on plain-text content holds regardless of styling state.

	var out, errBuf bytes.Buffer
	_, err := streamText(context.Background(), &out, &errBuf, seqFromEvents(events))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.Len() != 0 {
		t.Errorf("stdout should be empty for tool_result event, got: %q", out.String())
	}
	plain := style.Reset(errBuf.String())
	if !strings.Contains(plain, "[tool result] calculator: division by zero") {
		t.Errorf("stderr should contain error result, got: %q", plain)
	}
}

func TestToolResultTruncate(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "short string unchanged",
			input: "short",
			want:  "short",
		},
		{
			name:  "exactly 120 chars unchanged",
			input: strings.Repeat("a", 120),
			want:  strings.Repeat("a", 120),
		},
		{
			name:  "121 chars gets truncated with ellipsis",
			input: strings.Repeat("b", 121),
			want:  strings.Repeat("b", 120) + "...",
		},
		{
			name:  "200 chars gets truncated",
			input: strings.Repeat("c", 200),
			want:  strings.Repeat("c", 120) + "...",
		},
		{
			name:  "empty string unchanged",
			input: "",
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toolResultTruncate(tt.input)
			if got != tt.want {
				t.Errorf("toolResultTruncate(%d chars) = %q, want %q", len(tt.input), got, tt.want)
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
