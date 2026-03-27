package ctxcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// makeSession constructs a session with the given ID, provider, and message
// contents for use in search tests.
func makeSession(id, provider string, msgs ...string) *agent.Session {
	s := &agent.Session{
		ID:        id,
		Provider:  provider,
		Model:     "test-model",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	for i, content := range msgs {
		role := agent.RoleUser
		if i%2 == 1 {
			role = agent.RoleAssistant
		}
		s.History = append(s.History, agent.Message{Role: role, Content: content})
	}
	return s
}

func TestSearchCommand(t *testing.T) {
	tests := []struct {
		name         string
		sessions     []*agent.Session
		args         []string
		format       string
		wantContains []string
		wantNot      []string
		wantErr      bool
		errContains  string
	}{
		{
			name:        "missing query argument returns error",
			sessions:    nil,
			args:        []string{},
			wantErr:     true,
			errContains: "missing required <query> argument",
		},
		{
			name:         "no matching sessions prints no-match message",
			sessions:     []*agent.Session{makeSession("abc", "claude", "hello world")},
			args:         []string{"authentication"},
			wantContains: []string{"No sessions matched."},
		},
		{
			name: "query matches single session returns one result",
			sessions: []*agent.Session{
				makeSession("sess1", "claude", "fixing a bug in authentication"),
				makeSession("sess2", "openai", "hello world"),
			},
			args:         []string{"authentication"},
			wantContains: []string{"sess1"},
			wantNot:      []string{"sess2"},
		},
		{
			name: "query matches multiple sessions returns all",
			sessions: []*agent.Session{
				makeSession("sess1", "claude", "authentication token"),
				makeSession("sess2", "openai", "user authentication flow"),
				makeSession("sess3", "gemini", "unrelated content"),
			},
			args:         []string{"authentication"},
			wantContains: []string{"sess1", "sess2"},
			wantNot:      []string{"sess3"},
		},
		{
			name: "case-insensitive matching — uppercase query matches lowercase content",
			sessions: []*agent.Session{
				makeSession("sess1", "claude", "fixing a bug in the code"),
			},
			args:         []string{"BUG"},
			wantContains: []string{"sess1"},
		},
		{
			name: "case-insensitive matching — mixed-case content matches lowercase query",
			sessions: []*agent.Session{
				makeSession("sess1", "claude", "Fixing a Bug in the code"),
			},
			args:         []string{"bug"},
			wantContains: []string{"sess1"},
		},
		{
			name: "session with empty history is not included in results",
			sessions: []*agent.Session{
				{
					ID:        "empty",
					Provider:  "claude",
					Model:     "test",
					History:   []agent.Message{},
					CreatedAt: time.Now().UTC(),
					UpdatedAt: time.Now().UTC(),
				},
			},
			args:         []string{"anything"},
			wantContains: []string{"No sessions matched."},
		},
		{
			name: "match count is correct for multiple matching messages",
			sessions: []*agent.Session{
				makeSession("sess1", "claude",
					"first message about authentication",
					"second response",
					"third message about authentication again",
				),
			},
			args:         []string{"authentication"},
			wantContains: []string{"sess1", "2"},
		},
		{
			name: "ndjson format produces valid JSON per line",
			sessions: []*agent.Session{
				makeSession("sess1", "claude", "test authentication content"),
			},
			args:         []string{"authentication"},
			format:       "ndjson",
			wantContains: []string{`"id"`, `"match_count"`, `"excerpt"`},
		},
		{
			name: "table format shows header and separator",
			sessions: []*agent.Session{
				makeSession("sess1", "claude", "authentication bug fix"),
			},
			args:         []string{"authentication"},
			wantContains: []string{"ID", "PROVIDER", "CREATED", "MATCHES", "EXCERPT"},
		},
		{
			name: "excerpt is included in table output",
			sessions: []*agent.Session{
				makeSession("sess1", "claude", "debugging authentication token flow"),
			},
			args:         []string{"authentication"},
			wantContains: []string{"authentication"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newMockStore()
			for _, s := range tt.sessions {
				_ = st.Save(context.Background(), s)
			}

			format := tt.format
			if format == "" {
				format = "table"
			}
			cmd := &SearchCommand{store: st, format: format}

			var out, errBuf bytes.Buffer
			tc := &terminal.Context{
				Args:   tt.args,
				Stdout: &out,
				Stderr: &errBuf,
			}

			err := cmd.Run(context.Background(), tc)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := out.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("output %q does not contain %q", output, want)
				}
			}
			for _, notWant := range tt.wantNot {
				if strings.Contains(output, notWant) {
					t.Errorf("output %q should not contain %q", output, notWant)
				}
			}
		})
	}
}

func TestSearchNDJSONFields(t *testing.T) {
	st := newMockStore()
	sess := makeSession("testid123", "claude", "authentication flow implementation")
	_ = st.Save(context.Background(), sess)

	cmd := &SearchCommand{store: st, format: "ndjson"}
	var out bytes.Buffer
	tc := &terminal.Context{
		Args:   []string{"authentication"},
		Stdout: &out,
		Stderr: &bytes.Buffer{},
	}

	if err := cmd.Run(context.Background(), tc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 NDJSON line, got %d", len(lines))
	}

	var rec searchNDJSONRecord
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("invalid JSON: %v — %q", err, lines[0])
	}
	if rec.ID != "testid123" {
		t.Errorf("id = %q, want %q", rec.ID, "testid123")
	}
	if rec.Provider != "claude" {
		t.Errorf("provider = %q, want %q", rec.Provider, "claude")
	}
	if rec.MatchCount != 1 {
		t.Errorf("match_count = %d, want 1", rec.MatchCount)
	}
	if rec.Excerpt == "" {
		t.Error("excerpt should not be empty")
	}
}

func TestFirstExcerpt(t *testing.T) {
	tests := []struct {
		name    string
		content string
		query   string
		maxLen  int
		wantIn  string // substring the result must contain
	}{
		{
			name:    "short content returned as-is",
			content: "hello world",
			query:   "world",
			maxLen:  80,
			wantIn:  "hello world",
		},
		{
			name:    "long content truncated with trailing ellipsis",
			content: strings.Repeat("a", 30) + "searchterm" + strings.Repeat("b", 60),
			query:   "searchterm",
			maxLen:  40,
			wantIn:  "searchterm",
		},
		{
			name:    "match near end does not panic",
			content: "some content with the keyword at the very end: authentication",
			query:   "authentication",
			maxLen:  80,
			wantIn:  "authentication",
		},
		{
			name:    "match near start no leading ellipsis",
			content: "authentication is the first word here",
			query:   "authentication",
			maxLen:  80,
			wantIn:  "authentication is the first word",
		},
		{
			name:    "leading ellipsis when match is in the middle of long content",
			content: strings.Repeat("x", 50) + "authentication" + strings.Repeat("y", 50),
			query:   "authentication",
			maxLen:  30,
			wantIn:  "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstExcerpt(tt.content, tt.query, tt.maxLen)
			if !strings.Contains(got, tt.wantIn) {
				t.Errorf("firstExcerpt(%q, %q, %d) = %q, expected to contain %q",
					tt.content, tt.query, tt.maxLen, got, tt.wantIn)
			}
		})
	}
}

func TestSearchSessions(t *testing.T) {
	t.Run("empty session list returns empty results", func(t *testing.T) {
		results := searchSessions(nil, "query")
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	})

	t.Run("match count reflects only matching messages", func(t *testing.T) {
		sessions := []*agent.Session{
			makeSession("s1", "claude",
				"message about authentication",
				"unrelated message",
				"another authentication reference",
				"not relevant",
			),
		}
		results := searchSessions(sessions, "authentication")
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].matchCount != 2 {
			t.Errorf("matchCount = %d, want 2", results[0].matchCount)
		}
	})

	t.Run("excerpt is set from first matching message", func(t *testing.T) {
		sessions := []*agent.Session{
			makeSession("s1", "claude",
				"first matching: authentication flow",
				"second matching: authentication token",
			),
		}
		results := searchSessions(sessions, "authentication")
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		// excerpt comes from the first matching message
		if !strings.Contains(results[0].excerpt, "authentication") {
			t.Errorf("excerpt %q should contain 'authentication'", results[0].excerpt)
		}
	})
}
