package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/mrlm-net/cure/pkg/agent"
)

type result struct {
	ev  agent.Event
	err error
}

// Run streams a response for the given session as an iter.Seq2[Event, error].
// The caller iterates events with: for ev, err := range a.Run(ctx, session) { ... }
// If the caller breaks out of the range loop before the stream completes, it
// MUST cancel ctx to allow the background streaming goroutine to terminate
// cleanly. Failing to do so will leak the goroutine until the stream ends
// naturally or the process exits.
//
// When the session has Tools registered and the model requests tool calls,
// Run automatically executes the tool loop: it calls each requested tool,
// emits EventKindToolCall and EventKindToolResult events, appends the results
// to the session history, and re-invokes the model. The loop repeats until
// the model returns without requesting tools or until maxToolTurns is reached.
func (a *claudeAdapter) Run(ctx context.Context, sess *agent.Session) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		ch := make(chan result)
		go func() {
			defer close(ch)
			a.executeToolLoop(ctx, sess, ch)
		}()
		for r := range ch {
			if !yield(r.ev, r.err) {
				return
			}
		}
	}
}

// send delivers a result to ch, aborting if ctx is cancelled.
// Returns false if the send was cancelled (goroutine should stop).
func send(ctx context.Context, ch chan<- result, r result) bool {
	select {
	case ch <- r:
		return true
	case <-ctx.Done():
		return false
	}
}

// executeToolLoop orchestrates the multi-turn tool loop.
//
// On each iteration it:
//  1. Calls streamInto to stream the model response and collect content blocks.
//  2. Inspects the returned blocks for ToolUseBlock values.
//  3. If tool-use blocks are present AND sess.Tools is non-empty:
//     a. Appends the assistant turn (with tool_use blocks) to session history.
//     b. Emits EventKindToolCall for each tool request.
//     c. Calls the matching tool.
//     d. Emits EventKindToolResult for each outcome.
//     e. Appends tool results to the session history.
//     f. Loops back to step 1 with the updated session.
//  4. If no tool-use blocks (or no tools registered) the assistant message is
//     appended to history and the loop exits normally.
//
// An error is returned (via a result on ch) when maxToolTurns is exceeded.
func (a *claudeAdapter) executeToolLoop(ctx context.Context, sess *agent.Session, ch chan<- result) {
	// Build a lookup map for fast tool resolution once — sess.Tools is immutable
	// for the lifetime of a Run call.
	toolByName := make(map[string]agent.Tool, len(sess.Tools))
	for _, t := range sess.Tools {
		toolByName[t.Name()] = t
	}

	for turn := 0; turn < maxToolTurns; turn++ {
		// Stream one model response turn; collect all content blocks.
		blocks, ok := a.streamInto(ctx, sess, ch)
		if !ok {
			// Context cancelled or streaming error — streamInto already sent the
			// error event; stop the loop.
			return
		}

		// Separate tool-use requests from the rest of the content.
		toolUseBlocks := collectToolUseBlocks(blocks)

		// If no tools are registered or no tool-use blocks in the response,
		// this is a terminal text response. Append and exit.
		if len(toolUseBlocks) == 0 || len(sess.Tools) == 0 {
			sess.AppendAssistantBlocks(blocks)
			return
		}

		// Append the assistant turn (which includes tool_use blocks) so that
		// subsequent requests include the full conversation history.
		sess.AppendAssistantBlocks(blocks)

		// Execute each tool and collect results.
		for _, tub := range toolUseBlocks {
			// Marshal Input map to JSON string for the event payload.
			inputBytes, err := json.Marshal(tub.Input)
			if err != nil {
				msg := fmt.Sprintf("claude: failed to marshal tool input for %s: %v", tub.Name, err)
				send(ctx, ch, result{
					ev:  agent.Event{Kind: agent.EventKindError, Err: msg},
					err: fmt.Errorf("%s", msg), //nolint:goerr113
				})
				return
			}
			inputJSON := string(inputBytes)

			// Emit tool call event.
			if !send(ctx, ch, result{ev: agent.Event{
				Kind: agent.EventKindToolCall,
				ToolCall: &agent.ToolCallEvent{
					ID:        tub.ID,
					ToolName:  tub.Name,
					InputJSON: inputJSON,
				},
			}}) {
				return
			}

			tool, found := toolByName[tub.Name]
			var (
				toolResult string
				isError    bool
			)
			if !found {
				toolResult = fmt.Sprintf("tool not found: %s", tub.Name)
				isError = true
			} else {
				var callErr error
				toolResult, callErr = tool.Call(ctx, tub.Input)
				if callErr != nil {
					// Treat any context error (Canceled or DeadlineExceeded) as a
					// clean shutdown — don't emit an error event.
					if ctx.Err() != nil {
						return
					}
					toolResult = callErr.Error()
					isError = true
				}
			}

			// Emit tool result event.
			if !send(ctx, ch, result{ev: agent.Event{
				Kind: agent.EventKindToolResult,
				ToolResult: &agent.ToolResultEvent{
					ID:       tub.ID,
					ToolName: tub.Name,
					Result:   toolResult,
					IsError:  isError,
				},
			}}) {
				return
			}

			// Append the tool result to session history so the model sees it.
			sess.AppendToolResult(tub.ID, tub.Name, toolResult, isError)
		}
		// Loop back: re-invoke the model with updated history.
	}

	// Hard cap exceeded.
	msg := fmt.Sprintf("claude: tool loop exceeded %d turns", maxToolTurns)
	send(ctx, ch, result{
		ev:  agent.Event{Kind: agent.EventKindError, Err: msg},
		err: fmt.Errorf("%s", msg), //nolint:goerr113 // intentional string-only error
	})
}

// streamInto drives the Anthropic SDK stream and sends events on ch.
// It translates SDK stream events into agent.Event values:
//   - MessageStartEvent                → EventKindStart  (InputTokens)
//   - ContentBlockDeltaEvent with text → EventKindToken  (Text)
//   - MessageDeltaEvent                → accumulates stop_reason and output tokens
//   - MessageStopEvent                 → EventKindDone   (OutputTokens, StopReason)
//   - stream.Err()                     → EventKindError  (sanitised)
//
// Returns the accumulated content blocks from the completed message and true
// on success. On context cancellation or stream error it returns nil, false.
func (a *claudeAdapter) streamInto(
	ctx context.Context,
	sess *agent.Session,
	ch chan<- result,
) ([]agent.ContentBlock, bool) {
	params := a.buildParams(sess)
	stream := a.client.Messages.NewStreaming(ctx, params)
	defer stream.Close()

	var (
		accumulated  anthropic.Message
		outputTokens int
		stopReason   string
	)

	for stream.Next() {
		event := stream.Current()
		// Accumulate the full message (content blocks + tool-use JSON input)
		// so that messageContentToBlocks can inspect them after the stream ends.
		_ = accumulated.Accumulate(event)

		switch e := event.AsAny().(type) {
		case anthropic.MessageStartEvent:
			inputTokens := int(e.Message.Usage.InputTokens)
			if !send(ctx, ch, result{ev: agent.Event{
				Kind:        agent.EventKindStart,
				InputTokens: inputTokens,
			}}) {
				return nil, false
			}

		case anthropic.ContentBlockDeltaEvent:
			// Only emit token events for text deltas; skip input_json_delta etc.
			if textDelta, ok := e.Delta.AsAny().(anthropic.TextDelta); ok && textDelta.Text != "" {
				if !send(ctx, ch, result{ev: agent.Event{
					Kind: agent.EventKindToken,
					Text: textDelta.Text,
				}}) {
					return nil, false
				}
			}

		case anthropic.MessageDeltaEvent:
			// Collect stop reason and output tokens for the final EventKindDone.
			stopReason = string(e.Delta.StopReason)
			outputTokens = int(e.Usage.OutputTokens)

		case anthropic.MessageStopEvent:
			if !send(ctx, ch, result{ev: agent.Event{
				Kind:         agent.EventKindDone,
				OutputTokens: outputTokens,
				StopReason:   stopReason,
			}}) {
				return nil, false
			}
		}
	}

	if err := stream.Err(); err != nil {
		// Context cancellation is not a protocol error — stop quietly.
		if ctx.Err() != nil {
			return nil, false
		}
		sanitised := a.sanitiseError(err)
		send(ctx, ch, result{
			ev:  agent.Event{Kind: agent.EventKindError, Err: sanitised},
			err: fmt.Errorf("%s", sanitised), //nolint:goerr113 // intentional string-only error
		})
		return nil, false
	}

	// Extract content blocks from the accumulated streaming message.
	blocks := messageContentToBlocks(accumulated.Content)
	return blocks, true
}

// collectToolUseBlocks filters a content-block slice and returns only the
// agent.ToolUseBlock values.
func collectToolUseBlocks(blocks []agent.ContentBlock) []agent.ToolUseBlock {
	var out []agent.ToolUseBlock
	for _, b := range blocks {
		if tub, ok := b.(agent.ToolUseBlock); ok {
			out = append(out, tub)
		}
	}
	return out
}

// messageContentToBlocks converts the Anthropic SDK's ContentBlockUnion slice
// (from a completed streaming message) into agent.ContentBlock values.
//
//   - anthropic.TextBlock    → agent.TextBlock
//   - anthropic.ToolUseBlock → agent.ToolUseBlock (Input decoded to map[string]any)
//   - all other types are discarded
func messageContentToBlocks(content []anthropic.ContentBlockUnion) []agent.ContentBlock {
	out := make([]agent.ContentBlock, 0, len(content))
	for _, cb := range content {
		switch v := cb.AsAny().(type) {
		case anthropic.TextBlock:
			out = append(out, agent.TextBlock{Text: v.Text})
		case anthropic.ToolUseBlock:
			// Unmarshal the raw JSON input into map[string]any so that the
			// agent.ToolUseBlock is provider-agnostic and self-contained.
			var input map[string]any
			if raw := v.JSON.Input.Raw(); raw != "" {
				_ = json.Unmarshal([]byte(raw), &input)
			}
			if input == nil {
				input = make(map[string]any)
			}
			out = append(out, agent.ToolUseBlock{
				ID:    v.ID,
				Name:  v.Name,
				Input: input,
			})
		}
	}
	return out
}
