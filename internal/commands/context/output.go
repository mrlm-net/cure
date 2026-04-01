package ctxcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"os"
	"strings"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/style"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// stdinReader returns tc.Stdin when it is non-nil, otherwise os.Stdin.
// This lets tests inject a custom reader while production code reads the real stdin.
func stdinReader(tc *terminal.Context) io.Reader {
	if tc.Stdin != nil {
		return tc.Stdin
	}
	return os.Stdin
}

// toolResultTruncate truncates s to at most maxLen runes, appending "..." when
// truncation occurs. Used to keep tool result output readable in the REPL.
const toolResultMaxLen = 120

func toolResultTruncate(s string) string {
	runes := []rune(s)
	if len(runes) <= toolResultMaxLen {
		return s
	}
	return string(runes[:toolResultMaxLen]) + "..."
}

// streamText writes token text progressively to stdout (w) as events arrive.
// Tool call and tool result events are rendered as dim annotations to stderr
// (werr) so they do not pollute the response text stream.
// It returns the full accumulated response text.
func streamText(ctx context.Context, w io.Writer, werr io.Writer, events iter.Seq2[agent.Event, error]) (string, error) {
	var sb strings.Builder
	for ev, err := range events {
		if err != nil {
			return sb.String(), fmt.Errorf("context: stream error: %w", err)
		}
		switch ev.Kind {
		case agent.EventKindToken:
			if ev.Text != "" {
				sb.WriteString(ev.Text)
				fmt.Fprint(w, ev.Text)
			}
		case agent.EventKindToolCall:
			if ev.ToolCall != nil {
				// Replace newlines in InputJSON to keep annotation to a single line.
				inputJSON := strings.ReplaceAll(ev.ToolCall.InputJSON, "\n", " ")
				line := fmt.Sprintf("[tool] %s(%s)\n", ev.ToolCall.ToolName, inputJSON)
				fmt.Fprint(werr, style.Dim(line))
			}
		case agent.EventKindToolResult:
			if ev.ToolResult != nil {
				result := toolResultTruncate(ev.ToolResult.Result)
				line := fmt.Sprintf("[tool result] %s: %s\n", ev.ToolResult.ToolName, result)
				if ev.ToolResult.IsError {
					fmt.Fprint(werr, style.Red(line))
				} else {
					fmt.Fprint(werr, style.Dim(line))
				}
			}
		case agent.EventKindError:
			if ev.Err != "" {
				return sb.String(), fmt.Errorf("context: provider error: %s", ev.Err)
			}
		}
	}
	// Ensure the response ends with a newline for readability.
	if sb.Len() > 0 {
		fmt.Fprintln(w)
	}
	return sb.String(), nil
}

// streamNDJSON emits one JSON object per event to w.
// It returns the full accumulated response text (from token events).
func streamNDJSON(ctx context.Context, w io.Writer, events iter.Seq2[agent.Event, error]) (string, error) {
	enc := json.NewEncoder(w)
	var sb strings.Builder
	for ev, err := range events {
		if err != nil {
			return sb.String(), fmt.Errorf("context: stream error: %w", err)
		}
		if encErr := enc.Encode(ev); encErr != nil {
			return sb.String(), fmt.Errorf("context: encode event: %w", encErr)
		}
		if ev.Kind == agent.EventKindToken && ev.Text != "" {
			sb.WriteString(ev.Text)
		}
		if ev.Kind == agent.EventKindError && ev.Err != "" {
			return sb.String(), fmt.Errorf("context: provider error: %s", ev.Err)
		}
	}
	return sb.String(), nil
}

// dispatch routes streaming events to the appropriate output formatter based on
// format ("text" or "ndjson"). It returns the accumulated response text so the
// caller can persist it in the session history.
func dispatch(ctx context.Context, tc *terminal.Context, format string, events iter.Seq2[agent.Event, error]) (string, error) {
	switch format {
	case "ndjson":
		return streamNDJSON(ctx, tc.Stdout, events)
	default:
		// "text" and any unrecognised value fall back to streaming plain text.
		// Tool events are rendered as dim annotations to stderr.
		return streamText(ctx, tc.Stdout, tc.Stderr, events)
	}
}
