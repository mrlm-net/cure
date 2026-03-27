package ctxcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/terminal"
)

func TestExportCommand(t *testing.T) {
	// Build a fixed-time session so the expected output is deterministic.
	fixedTime, _ := time.Parse("2006-01-02 15:04:05", "2024-01-15 10:30:00")

	sessionWithHistory := &agent.Session{
		ID:        "abc123def456",
		Provider:  "claude",
		Model:     "claude-opus-4-6",
		History: []agent.Message{
			{Role: agent.RoleUser, Content: "Hello, world!"},
			{Role: agent.RoleAssistant, Content: "Hi there! How can I help you today?"},
		},
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
	}

	sessionEmpty := &agent.Session{
		ID:        "empty456session",
		Provider:  "openai",
		Model:     "gpt-4",
		History:   []agent.Message{},
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
	}

	sessionWithFork := &agent.Session{
		ID:        "forkedsession1",
		Provider:  "claude",
		Model:     "claude-opus-4-6",
		History:   []agent.Message{{Role: agent.RoleUser, Content: "forked message"}},
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
		ForkOf:    "originalsession",
	}

	tests := []struct {
		name        string
		session     *agent.Session
		args        []string
		format      string
		output      string
		wantErr     bool
		errContains string
		wantContains []string
		wantNot     []string
	}{
		{
			name:    "markdown export to stdout — H1 heading and metadata",
			session: sessionWithHistory,
			args:    []string{sessionWithHistory.ID},
			format:  "markdown",
			wantContains: []string{
				"# abc123def456",
				"| Provider | claude |",
				"| Model    | claude-opus-4-6 |",
				"2024-01-15 10:30:00 UTC",
				"## User",
				"Hello, world!",
				"## Assistant",
				"Hi there! How can I help you today?",
			},
		},
		{
			name:    "empty history renders no-messages placeholder",
			session: sessionEmpty,
			args:    []string{sessionEmpty.ID},
			format:  "markdown",
			wantContains: []string{
				"# empty456session",
				"_No messages in this session._",
			},
			wantNot: []string{"## User", "## Assistant"},
		},
		{
			name:    "session with fork_of appears in metadata table",
			session: sessionWithFork,
			args:    []string{sessionWithFork.ID},
			format:  "markdown",
			wantContains: []string{
				"# forkedsession1",
				"| Fork of  | originalsession |",
			},
		},
		{
			name:    "session without fork_of does not show fork row",
			session: sessionWithHistory,
			args:    []string{sessionWithHistory.ID},
			format:  "markdown",
			wantNot: []string{"Fork of"},
		},
		{
			name:    "ndjson export produces valid JSON with expected fields",
			session: sessionWithHistory,
			args:    []string{sessionWithHistory.ID},
			format:  "ndjson",
			wantContains: []string{
				`"id"`,
				`"provider"`,
				`"model"`,
				`"history"`,
				`"abc123def456"`,
				`"claude"`,
			},
		},
		{
			name:        "unknown session ID returns session not found error",
			session:     nil,
			args:        []string{"ghostsession"},
			format:      "markdown",
			wantErr:     true,
			errContains: "session not found: ghostsession",
		},
		{
			name:        "missing session-id argument returns error",
			session:     nil,
			args:        []string{},
			format:      "markdown",
			wantErr:     true,
			errContains: "missing required <session-id>",
		},
		{
			name:        "unsupported format returns error",
			session:     sessionWithHistory,
			args:        []string{sessionWithHistory.ID},
			format:      "xml",
			wantErr:     true,
			errContains: "unsupported format",
		},
		{
			name:    "default format is markdown when empty string",
			session: sessionWithHistory,
			args:    []string{sessionWithHistory.ID},
			format:  "",
			wantContains: []string{
				"# abc123def456",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := newMockStore()
			if tt.session != nil {
				_ = st.Save(context.Background(), tt.session)
			}

			cmd := &ExportCommand{store: st, format: tt.format, output: tt.output}
			var out bytes.Buffer
			var errOut bytes.Buffer
			tc := &terminal.Context{
				Stdout: &out,
				Stderr: &errOut,
				Args:   tt.args,
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
					t.Errorf("output does not contain %q\nfull output:\n%s", want, output)
				}
			}
			for _, notWant := range tt.wantNot {
				if strings.Contains(output, notWant) {
					t.Errorf("output should not contain %q\nfull output:\n%s", notWant, output)
				}
			}
		})
	}
}

func TestExportCommandOutputFile(t *testing.T) {
	fixedTime, _ := time.Parse("2006-01-02 15:04:05", "2024-06-01 08:00:00")
	sess := &agent.Session{
		ID:        "outputtest1234",
		Provider:  "claude",
		Model:     "claude-opus-4-6",
		History:   []agent.Message{{Role: agent.RoleUser, Content: "write to file"}},
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
	}

	t.Run("output to file creates file with correct content", func(t *testing.T) {
		st := newMockStore()
		_ = st.Save(context.Background(), sess)

		tmpDir := t.TempDir()
		outPath := filepath.Join(tmpDir, "session.md")

		cmd := &ExportCommand{store: st, format: "markdown", output: outPath}
		var out bytes.Buffer
		tc := &terminal.Context{
			Stdout: &out,
			Stderr: &bytes.Buffer{},
			Args:   []string{sess.ID},
		}

		if err := cmd.Run(context.Background(), tc); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// stdout should be empty when --output is used
		if out.Len() != 0 {
			t.Errorf("stdout should be empty when --output is set, got: %q", out.String())
		}

		data, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("output file not created: %v", err)
		}
		content := string(data)
		if !strings.Contains(content, "# outputtest1234") {
			t.Errorf("file content does not contain expected H1, got: %q", content)
		}
		if !strings.Contains(content, "write to file") {
			t.Errorf("file content does not contain message text, got: %q", content)
		}
	})

	t.Run("output path with non-existent parent creates parent directory", func(t *testing.T) {
		st := newMockStore()
		_ = st.Save(context.Background(), sess)

		tmpDir := t.TempDir()
		outPath := filepath.Join(tmpDir, "subdir", "nested", "session.md")

		cmd := &ExportCommand{store: st, format: "markdown", output: outPath}
		tc := &terminal.Context{
			Stdout: &bytes.Buffer{},
			Stderr: &bytes.Buffer{},
			Args:   []string{sess.ID},
		}

		if err := cmd.Run(context.Background(), tc); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if _, err := os.Stat(outPath); err != nil {
			t.Errorf("expected output file at %s, got error: %v", outPath, err)
		}
	})

	t.Run("ndjson output to file round-trips to Session", func(t *testing.T) {
		st := newMockStore()
		_ = st.Save(context.Background(), sess)

		tmpDir := t.TempDir()
		outPath := filepath.Join(tmpDir, "session.json")

		cmd := &ExportCommand{store: st, format: "ndjson", output: outPath}
		tc := &terminal.Context{
			Stdout: &bytes.Buffer{},
			Stderr: &bytes.Buffer{},
			Args:   []string{sess.ID},
		}

		if err := cmd.Run(context.Background(), tc); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		data, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("output file not created: %v", err)
		}

		var loaded agent.Session
		if err := json.Unmarshal(bytes.TrimSpace(data), &loaded); err != nil {
			t.Fatalf("output is not valid JSON: %v\ncontent:\n%s", err, string(data))
		}
		if loaded.ID != sess.ID {
			t.Errorf("round-trip ID = %q, want %q", loaded.ID, sess.ID)
		}
		if loaded.Provider != sess.Provider {
			t.Errorf("round-trip Provider = %q, want %q", loaded.Provider, sess.Provider)
		}
		if loaded.Model != sess.Model {
			t.Errorf("round-trip Model = %q, want %q", loaded.Model, sess.Model)
		}
		if len(loaded.History) != len(sess.History) {
			t.Errorf("round-trip History len = %d, want %d", len(loaded.History), len(sess.History))
		}
	})
}

func TestRenderMarkdown(t *testing.T) {
	fixedTime, _ := time.Parse("2006-01-02 15:04:05", "2025-03-27 12:00:00")

	t.Run("renders metadata fields correctly", func(t *testing.T) {
		s := &agent.Session{
			ID:        "testid1",
			Provider:  "openai",
			Model:     "gpt-4o",
			History:   []agent.Message{},
			CreatedAt: fixedTime,
			UpdatedAt: fixedTime,
		}
		content, err := renderMarkdown(s)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := string(content)
		for _, want := range []string{
			"# testid1",
			"| Provider | openai |",
			"| Model    | gpt-4o |",
			"2025-03-27 12:00:00 UTC",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("markdown does not contain %q\nfull output:\n%s", want, got)
			}
		}
	})

	t.Run("role names are title-cased", func(t *testing.T) {
		s := &agent.Session{
			ID:    "roletest",
			History: []agent.Message{
				{Role: agent.RoleUser, Content: "hi"},
				{Role: agent.RoleAssistant, Content: "hello"},
			},
			CreatedAt: fixedTime,
			UpdatedAt: fixedTime,
		}
		content, err := renderMarkdown(s)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got := string(content)
		if !strings.Contains(got, "## User") {
			t.Errorf("expected '## User' in output, got:\n%s", got)
		}
		if !strings.Contains(got, "## Assistant") {
			t.Errorf("expected '## Assistant' in output, got:\n%s", got)
		}
	})
}

func TestRenderNDJSON(t *testing.T) {
	fixedTime, _ := time.Parse("2006-01-02 15:04:05", "2025-01-10 09:00:00")
	s := &agent.Session{
		ID:        "ndjsontest",
		Provider:  "claude",
		Model:     "claude-opus-4-6",
		History:   []agent.Message{{Role: agent.RoleUser, Content: "ndjson test"}},
		CreatedAt: fixedTime,
		UpdatedAt: fixedTime,
	}

	content, err := renderNDJSON(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Must parse as valid JSON.
	var loaded agent.Session
	if err := json.Unmarshal(bytes.TrimSpace(content), &loaded); err != nil {
		t.Fatalf("output is not valid JSON: %v\ncontent:\n%s", err, string(content))
	}

	if loaded.ID != s.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, s.ID)
	}
	if loaded.Provider != s.Provider {
		t.Errorf("Provider = %q, want %q", loaded.Provider, s.Provider)
	}
	if len(loaded.History) != len(s.History) {
		t.Errorf("History len = %d, want %d", len(loaded.History), len(s.History))
	}
}
