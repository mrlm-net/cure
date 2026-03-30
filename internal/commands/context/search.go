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

// SearchCommand implements "cure context search <query>".
// It performs case-insensitive substring search across all session message history.
type SearchCommand struct {
	store  agent.SessionStore
	format string
}

func (c *SearchCommand) Name() string        { return "search" }
func (c *SearchCommand) Description() string { return "Search session history by content" }

func (c *SearchCommand) Usage() string {
	return `Usage: cure context search <query> [flags]

Searches all saved sessions for messages containing the query string
(case-insensitive). Reports the session ID, provider, creation time,
number of matching messages, and a short excerpt from the first match.

Flags:
  --format  Output format: "table" (default) or "ndjson"

Examples:
  cure context search "authentication"
  cure context search "bug fix" --format ndjson
`
}

func (c *SearchCommand) Flags() *flag.FlagSet {
	fset := flag.NewFlagSet("context-search", flag.ContinueOnError)
	fset.StringVar(&c.format, "format", "table", `Output format: "table" or "ndjson"`)
	return fset
}

// searchMatch holds a session and its match statistics.
type searchMatch struct {
	session    *agent.Session
	matchCount int
	excerpt    string
}

// Run executes the search command. It loads all sessions, filters to those
// whose history contains the query, and prints the results.
func (c *SearchCommand) Run(ctx context.Context, tc *terminal.Context) error {
	if len(tc.Args) == 0 {
		return fmt.Errorf("context search: missing required <query> argument")
	}
	query := tc.Args[0]

	sessions, err := c.store.List(ctx)
	if err != nil {
		return fmt.Errorf("context search: %w", err)
	}

	matches := searchSessions(sessions, query)

	if len(matches) == 0 {
		fmt.Fprintln(tc.Stdout, "No sessions matched.")
		return nil
	}

	switch c.format {
	case "ndjson":
		return printSearchNDJSON(tc, matches)
	default:
		printSearchTable(tc, matches)
		return nil
	}
}

// searchSessions iterates over sessions and collects those whose history
// contains at least one message with content matching query (case-insensitive).
func searchSessions(sessions []*agent.Session, query string) []searchMatch {
	queryLower := strings.ToLower(query)
	var results []searchMatch
	for _, s := range sessions {
		var count int
		var excerpt string
		for _, msg := range s.History {
			text := agent.TextOf(msg.Content)
			lower := strings.ToLower(text)
			if strings.Contains(lower, queryLower) {
				count++
				if excerpt == "" {
					excerpt = firstExcerpt(text, query, 80)
				}
			}
		}
		if count > 0 {
			results = append(results, searchMatch{s, count, excerpt})
		}
	}
	return results
}

// firstExcerpt returns a short excerpt from content centred around the first
// occurrence of query (case-insensitive), truncated to maxLen runes.
// Leading/trailing ellipses are added when the excerpt does not start/end at
// the content boundaries. Uses rune-based indexing to avoid splitting
// multi-byte UTF-8 sequences (e.g. CJK characters, emoji).
func firstExcerpt(content, query string, maxLen int) string {
	const leadingContext = 20

	runes := []rune(content)
	lowerRunes := []rune(strings.ToLower(content))
	queryRunes := []rune(strings.ToLower(query))

	// Find the first rune-index where the query matches.
	idx := -1
	for i := range runes {
		if i+len(queryRunes) > len(runes) {
			break
		}
		match := true
		for j, qr := range queryRunes {
			if lowerRunes[i+j] != qr {
				match = false
				break
			}
		}
		if match {
			idx = i
			break
		}
	}

	if idx < 0 {
		// query not found (shouldn't happen in normal flow); return prefix
		if len(runes) > maxLen {
			return string(runes[:maxLen]) + "..."
		}
		return content
	}

	start := idx - leadingContext
	if start < 0 {
		start = 0
	}
	end := start + maxLen
	if end > len(runes) {
		end = len(runes)
	}

	result := string(runes[start:end])
	if start > 0 {
		result = "..." + result
	}
	if end < len(runes) {
		result = result + "..."
	}
	return result
}

// printSearchTable writes a fixed-width table of search results to tc.Stdout.
func printSearchTable(tc *terminal.Context, matches []searchMatch) {
	fmt.Fprintf(tc.Stdout, "%-20s  %-10s  %-20s  %7s  %s\n",
		"ID", "PROVIDER", "CREATED", "MATCHES", "EXCERPT")
	fmt.Fprintf(tc.Stdout, "%-20s  %-10s  %-20s  %7s  %s\n",
		strings.Repeat("-", 20), strings.Repeat("-", 10),
		strings.Repeat("-", 20), strings.Repeat("-", 7), strings.Repeat("-", 50))
	for _, m := range matches {
		fmt.Fprintf(tc.Stdout, "%-20s  %-10s  %-20s  %7d  %s\n",
			m.session.ID,
			m.session.Provider,
			m.session.CreatedAt.Format("2006-01-02 15:04:05"),
			m.matchCount,
			m.excerpt,
		)
	}
}

// searchNDJSONRecord is the JSON representation of a single search result.
type searchNDJSONRecord struct {
	ID         string    `json:"id"`
	Provider   string    `json:"provider"`
	CreatedAt  time.Time `json:"created_at"`
	MatchCount int       `json:"match_count"`
	Excerpt    string    `json:"excerpt"`
}

// printSearchNDJSON writes one JSON object per matching session to tc.Stdout.
func printSearchNDJSON(tc *terminal.Context, matches []searchMatch) error {
	enc := json.NewEncoder(tc.Stdout)
	for _, m := range matches {
		rec := searchNDJSONRecord{
			ID:         m.session.ID,
			Provider:   m.session.Provider,
			CreatedAt:  m.session.CreatedAt,
			MatchCount: m.matchCount,
			Excerpt:    m.excerpt,
		}
		if err := enc.Encode(rec); err != nil {
			return fmt.Errorf("context search: encode: %w", err)
		}
	}
	return nil
}
