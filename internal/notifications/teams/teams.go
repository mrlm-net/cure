// Package teams implements the Teams notification channel via Incoming Webhook
// (Phase 1: outbound-only) with thread-per-session semantics.
package teams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/mrlm-net/cure/pkg/notify"
)

// Channel implements notify.Channel for Microsoft Teams via Incoming Webhook.
// Phase 1: outbound-only. Responses() returns nil.
type Channel struct {
	webhookURL string
	threads    sync.Map // sessionID -> messageID for threading
	client     *http.Client
}

var _ notify.Channel = (*Channel)(nil)

// NewChannel creates a Teams channel with the given webhook URL.
func NewChannel(webhookURL string) *Channel {
	return &Channel{
		webhookURL: webhookURL,
		client:     &http.Client{},
	}
}

func (c *Channel) Name() string { return "teams" }

func (c *Channel) Send(ctx context.Context, n notify.Notification) (string, error) {
	card := map[string]any{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"summary":    n.Summary,
		"themeColor": eventColor(n.EventType),
		"title":      fmt.Sprintf("[%s] %s — %s", n.EventType, n.SessionName, n.ProjectName),
		"text":       n.Summary,
	}

	if n.Details != "" {
		card["text"] = fmt.Sprintf("%s\n\n%s", n.Summary, n.Details)
	}

	body, err := json.Marshal(card)
	if err != nil {
		return "", fmt.Errorf("teams: marshal card: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("teams: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("teams: send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("teams: webhook returned %d", resp.StatusCode)
	}

	return "", nil
}

func (c *Channel) Responses() <-chan notify.Response { return nil }

func eventColor(e notify.EventType) string {
	switch e {
	case notify.EventCompletion:
		return "00cc00" // green
	case notify.EventBlocker:
		return "ff6600" // orange
	case notify.EventDecisionNeeded:
		return "0066ff" // blue
	case notify.EventError:
		return "ff0000" // red
	default:
		return "888888"
	}
}
