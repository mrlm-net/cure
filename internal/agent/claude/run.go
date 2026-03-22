package claude

import (
	"context"
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
func (a *claudeAdapter) Run(ctx context.Context, sess *agent.Session) iter.Seq2[agent.Event, error] {
	return func(yield func(agent.Event, error) bool) {
		ch := make(chan result)
		go func() {
			defer close(ch)
			a.streamInto(ctx, sess, ch)
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

// streamInto drives the Anthropic SDK stream and sends events on ch.
// It translates SDK stream events into agent.Event values:
//   - MessageStartEvent     → EventKindStart  (InputTokens)
//   - ContentBlockDeltaEvent with text → EventKindToken (Text)
//   - MessageDeltaEvent     → accumulates stop_reason and output tokens
//   - MessageStopEvent      → EventKindDone   (OutputTokens, StopReason)
//   - stream.Err()          → EventKindError  (sanitised)
func (a *claudeAdapter) streamInto(ctx context.Context, sess *agent.Session, ch chan<- result) {
	params := a.buildParams(sess)
	stream := a.client.Messages.NewStreaming(ctx, params)
	defer stream.Close()

	var (
		outputTokens int
		stopReason   string
	)

	for stream.Next() {
		event := stream.Current()

		switch e := event.AsAny().(type) {
		case anthropic.MessageStartEvent:
			inputTokens := int(e.Message.Usage.InputTokens)
			if !send(ctx, ch, result{ev: agent.Event{
				Kind:        agent.EventKindStart,
				InputTokens: inputTokens,
			}}) {
				return
			}

		case anthropic.ContentBlockDeltaEvent:
			// Only emit token events for text deltas; skip input_json_delta etc.
			if textDelta, ok := e.Delta.AsAny().(anthropic.TextDelta); ok && textDelta.Text != "" {
				if !send(ctx, ch, result{ev: agent.Event{
					Kind: agent.EventKindToken,
					Text: textDelta.Text,
				}}) {
					return
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
				return
			}
		}
	}

	if err := stream.Err(); err != nil {
		// Context cancellation is not a protocol error — stop quietly.
		if ctx.Err() != nil {
			return
		}
		// Wrap the error to redact any API key from both the event string and the
		// returned error value so callers never see the key in err.Error().
		sanitised := a.sanitiseError(err)
		send(ctx, ch, result{
			ev:  agent.Event{Kind: agent.EventKindError, Err: sanitised},
			err: fmt.Errorf("%s", sanitised), //nolint:goerr113 // intentional string-only error
		})
	}
}
