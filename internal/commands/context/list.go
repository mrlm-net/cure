package ctxcmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// ListCommand implements "cure context list".
// It prints all persisted sessions in a human-readable table or NDJSON format.
type ListCommand struct {
	store agent.SessionStore

	// Flags
	format   string
	provider string
}

func (c *ListCommand) Name() string        { return "list" }
func (c *ListCommand) Description() string { return "List all AI conversation sessions" }

func (c *ListCommand) Usage() string {
	return `Usage: cure context list [options]

List all persisted AI conversation sessions. Output includes the truncated
session ID, provider, model, message count, and relative last-updated time.

Flags:
  --format       Output format: "text" (default) or "ndjson"
  --provider     Filter sessions by provider name

Examples:
  cure context list
  cure context list --provider claude
  cure context list --format ndjson
`
}

func (c *ListCommand) Flags() *flag.FlagSet {
	fs := flag.NewFlagSet("context-list", flag.ContinueOnError)
	fs.StringVar(&c.format, "format", "text", `Output format: "text" or "ndjson"`)
	fs.StringVar(&c.provider, "provider", "", "Filter by provider name")
	return fs
}

func (c *ListCommand) Run(ctx context.Context, tc *terminal.Context) error {
	sessions, err := c.store.List(ctx)
	if err != nil {
		return fmt.Errorf("context list: %w", err)
	}

	// Apply optional provider filter.
	if c.provider != "" {
		filtered := make([]*agent.Session, 0, len(sessions))
		for _, s := range sessions {
			if s.Provider == c.provider {
				filtered = append(filtered, s)
			}
		}
		sessions = filtered
	}

	switch c.format {
	case "ndjson":
		return listNDJSON(tc, sessions)
	case "text", "":
		return listText(tc, sessions)
	default:
		return fmt.Errorf("context list: unknown format %q (want \"text\" or \"ndjson\")", c.format)
	}
}

// listText writes a fixed-width text table to tc.Stdout.
func listText(tc *terminal.Context, sessions []*agent.Session) error {
	if len(sessions) == 0 {
		fmt.Fprintln(tc.Stdout, "No sessions found.")
		return nil
	}

	// Header
	fmt.Fprintf(tc.Stdout, "%-12s  %-10s  %-20s  %8s  %s\n",
		"ID", "PROVIDER", "MODEL", "MESSAGES", "UPDATED")
	fmt.Fprintln(tc.Stdout, strings.Repeat("-", 64))

	for _, s := range sessions {
		msgCount := len(s.History)
		updated := relativeTime(s.UpdatedAt)
		shortID := s.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		provider := s.Provider
		if len(provider) > 10 {
			provider = provider[:7] + "..."
		}
		model := s.Model
		if len(model) > 20 {
			model = model[:17] + "..."
		}
		fmt.Fprintf(tc.Stdout, "%-12s  %-10s  %-20s  %8d  %s\n",
			shortID, provider, model, msgCount, updated)
	}
	return nil
}

// listNDJSON writes one JSON object per session line to tc.Stdout.
func listNDJSON(tc *terminal.Context, sessions []*agent.Session) error {
	enc := json.NewEncoder(tc.Stdout)
	for _, s := range sessions {
		if err := enc.Encode(s); err != nil {
			return fmt.Errorf("context list: encode: %w", err)
		}
	}
	return nil
}

// relativeTime formats a past time as a human-readable relative duration
// (e.g. "just now", "5m ago", "2h ago", "3d ago").
func relativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}
