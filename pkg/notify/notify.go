// Package notify provides a channel-based notification system for agent-to-human
// communication. Channels (Teams, OS, GUI) implement the Channel interface.
// The Dispatcher fans out notifications and multiplexes responses.
package notify

import (
	"context"
	"fmt"
	"sync"
)

// EventType classifies notification events.
type EventType string

const (
	EventCompletion     EventType = "completion"
	EventBlocker        EventType = "blocker"
	EventDecisionNeeded EventType = "decision_needed"
	EventError          EventType = "error"
)

// Notification is a message sent from an agent session to the developer.
type Notification struct {
	SessionID   string    `json:"session_id"`
	SessionName string    `json:"session_name"`
	ProjectName string    `json:"project_name"`
	EventType   EventType `json:"event_type"`
	Summary     string    `json:"summary"`
	Details     string    `json:"details,omitempty"`
}

// Response is a message from the developer back to an agent session.
type Response struct {
	SessionID string `json:"session_id"`
	ChannelID string `json:"channel_id"`
	Text      string `json:"text"`
}

// Channel sends notifications and optionally receives responses.
type Channel interface {
	Name() string
	Send(ctx context.Context, n Notification) (string, error)
	Responses() <-chan Response // nil if unidirectional
}

// Dispatcher routes notifications to all enabled channels and
// multiplexes responses back to sessions.
type Dispatcher struct {
	channels []Channel
	mu       sync.Mutex
	waiting  map[string]chan Response // sessionID -> response channel
}

// NewDispatcher creates a dispatcher with the given channels.
func NewDispatcher(channels ...Channel) *Dispatcher {
	return &Dispatcher{
		channels: channels,
		waiting:  make(map[string]chan Response),
	}
}

// Notify sends a notification to all enabled channels.
func (d *Dispatcher) Notify(ctx context.Context, n Notification) error {
	var firstErr error
	for _, ch := range d.channels {
		if _, err := ch.Send(ctx, n); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// WaitResponse blocks until a response arrives for the given session,
// or the context is cancelled. First response from any channel wins.
func (d *Dispatcher) WaitResponse(ctx context.Context, sessionID string) (Response, error) {
	ch := make(chan Response, 1)

	d.mu.Lock()
	d.waiting[sessionID] = ch
	d.mu.Unlock()

	defer func() {
		d.mu.Lock()
		delete(d.waiting, sessionID)
		d.mu.Unlock()
	}()

	select {
	case resp := <-ch:
		return resp, nil
	case <-ctx.Done():
		return Response{}, fmt.Errorf("notify: wait cancelled: %w", ctx.Err())
	}
}

// DeliverResponse routes an incoming response to the waiting session.
// Returns false if no one is waiting for this session.
func (d *Dispatcher) DeliverResponse(resp Response) bool {
	d.mu.Lock()
	ch, ok := d.waiting[resp.SessionID]
	d.mu.Unlock()

	if !ok {
		return false
	}

	select {
	case ch <- resp:
		return true
	default:
		return false // already responded
	}
}

// StartListening spawns goroutines to listen for responses from bidirectional channels.
func (d *Dispatcher) StartListening(ctx context.Context) {
	for _, ch := range d.channels {
		respCh := ch.Responses()
		if respCh == nil {
			continue
		}
		go func(responses <-chan Response) {
			for {
				select {
				case resp, ok := <-responses:
					if !ok {
						return
					}
					d.DeliverResponse(resp)
				case <-ctx.Done():
					return
				}
			}
		}(respCh)
	}
}
