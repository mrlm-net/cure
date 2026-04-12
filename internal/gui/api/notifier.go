package api

import (
	"context"

	"github.com/mrlm-net/cure/pkg/notify"
)

// notifyAdapter wraps a notify.Dispatcher to implement the Notifier interface.
type notifyAdapter struct {
	dispatcher *notify.Dispatcher
}

// NewNotifier creates a Notifier from a notify.Dispatcher.
func NewNotifier(d *notify.Dispatcher) Notifier {
	if d == nil {
		return nil
	}
	return &notifyAdapter{dispatcher: d}
}

func (n *notifyAdapter) Notify(ctx context.Context, sessionID, sessionName, projectName, summary string) error {
	return n.dispatcher.Notify(ctx, notify.Notification{
		SessionID:   sessionID,
		SessionName: sessionName,
		ProjectName: projectName,
		EventType:   notify.EventCompletion,
		Summary:     summary,
	})
}
