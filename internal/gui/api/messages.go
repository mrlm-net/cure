package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/mrlm-net/cure/pkg/agent"
)

// SSEEvent is a single server-sent event written to the SSE stream.
// It mirrors [agent.Event] but is serialized independently so the wire
// format is decoupled from the internal event type.
type SSEEvent struct {
	Kind         agent.EventKind        `json:"kind"`
	Text         string                 `json:"text,omitempty"`
	InputTokens  int                    `json:"input_tokens,omitempty"`
	OutputTokens int                    `json:"output_tokens,omitempty"`
	StopReason   string                 `json:"stop_reason,omitempty"`
	Err          string                 `json:"error,omitempty"`
	ToolCall     *agent.ToolCallEvent   `json:"tool_call,omitempty"`
	ToolResult   *agent.ToolResultEvent `json:"tool_result,omitempty"`
}

// AgentRunFunc is the function signature for running an agent turn on a session.
// It mirrors [agent.Agent.Run] without requiring the full Agent interface,
// making it easy to inject a stub for testing.
type AgentRunFunc func(ctx context.Context, session *agent.Session) <-chan AgentResult

// AgentResult is the value produced by an AgentRunFunc for each streamed event.
type AgentResult struct {
	Event agent.Event
	Err   error
}

// messagesHandler handles POST /api/context/sessions/{id}/messages.
// It reads a user message, appends it to the session, runs the agent, and
// streams the response as SSE events. When no real agent is configured
// (runFn is nil), a built-in echo stub splits the user message into words
// and streams them back as tokens.
func messagesHandler(store agent.SessionStore, runFn AgentRunFunc, notifier ...Notifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeError(w, http.StatusNotImplemented, "streaming not supported")
			return
		}

		id := r.PathValue("id")
		sess, err := store.Load(r.Context(), id)
		if err != nil {
			if errors.Is(err, agent.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to load session")
			return
		}

		var req MessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.Message == "" {
			writeError(w, http.StatusBadRequest, "message must not be empty")
			return
		}

		// Append the user message to history.
		sess.AppendUserMessage(req.Message)

		// Set SSE headers before writing the first event.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")

		ctx := r.Context()

		// Select the run function — use the echo stub when no real agent is wired.
		run := runFn
		if run == nil {
			run = echoAgentRun
		}

		ch := run(ctx, sess)

		var assistantText strings.Builder

		for {
			select {
			case <-ctx.Done():
				// Client disconnected — save whatever we have and exit.
				if assistantText.Len() > 0 {
					sess.AppendAssistantMessage(assistantText.String())
					_ = store.Save(context.Background(), sess)
				}
				return
			case result, ok := <-ch:
				if !ok {
					// Channel closed — stream complete.
					return
				}
				if result.Err != nil {
					writeSSE(w, flusher, SSEEvent{Kind: agent.EventKindError, Err: result.Err.Error()})
					return
				}

				ev := result.Event

				// Accumulate assistant text from token events.
				if ev.Kind == agent.EventKindToken {
					assistantText.WriteString(ev.Text)
				}

				// Persist the completed assistant turn before sending the done event.
				if ev.Kind == agent.EventKindDone {
					if assistantText.Len() > 0 {
						sess.AppendAssistantMessage(assistantText.String())
					}
					_ = store.Save(context.Background(), sess)

					// Send OS/channel notification on completion
					if len(notifier) > 0 && notifier[0] != nil {
						summary := assistantText.String()
						if len(summary) > 100 {
							summary = summary[:100] + "..."
						}
						notifier[0].Notify(context.Background(), sess.ID, sess.Name, sess.ProjectName, summary)
					}
				}

				// Notify on errors too
				if ev.Kind == agent.EventKindError && len(notifier) > 0 && notifier[0] != nil {
					notifier[0].Notify(context.Background(), sess.ID, sess.Name, sess.ProjectName, "Error: "+ev.Err)
				}

				writeSSE(w, flusher, SSEEvent{
					Kind:         ev.Kind,
					Text:         ev.Text,
					InputTokens:  ev.InputTokens,
					OutputTokens: ev.OutputTokens,
					StopReason:   ev.StopReason,
					Err:          ev.Err,
					ToolCall:     ev.ToolCall,
					ToolResult:   ev.ToolResult,
				})
			}
		}
	}
}

// writeSSE encodes an SSEEvent as a data: line and flushes.
func writeSSE(w http.ResponseWriter, flusher http.Flusher, ev SSEEvent) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// echoAgentRun is a built-in stub that echoes the user's last message back
// as individual word tokens. This allows the SSE infrastructure to work
// without a real AI provider configured.
func echoAgentRun(ctx context.Context, session *agent.Session) <-chan AgentResult {
	ch := make(chan AgentResult, 16)

	// Extract the last user message from history.
	var userText string
	for i := len(session.History) - 1; i >= 0; i-- {
		if session.History[i].Role == agent.RoleUser {
			userText = agent.TextOf(session.History[i].Content)
			break
		}
	}

	go func() {
		defer close(ch)

		// start event
		select {
		case <-ctx.Done():
			return
		case ch <- AgentResult{Event: agent.Event{Kind: agent.EventKindStart}}:
		}

		// token events — one per word
		words := strings.Fields(userText)
		for i, word := range words {
			if i > 0 {
				word = " " + word
			}
			select {
			case <-ctx.Done():
				return
			case ch <- AgentResult{Event: agent.Event{Kind: agent.EventKindToken, Text: word}}:
			}
		}

		// done event
		select {
		case <-ctx.Done():
			return
		case ch <- AgentResult{Event: agent.Event{Kind: agent.EventKindDone, StopReason: "end_turn"}}:
		}
	}()

	return ch
}
