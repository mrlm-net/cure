package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mrlm-net/cure/internal/agent/claude"
	ctxcmd "github.com/mrlm-net/cure/internal/commands/context"
	agentstore "github.com/mrlm-net/cure/pkg/agent/store"
	"github.com/mrlm-net/cure/pkg/terminal"
)

// validSSEStream returns a complete Anthropic SSE streaming response body.
// It matches the format expected by the claude adapter's streamInto.
func validSSEStream(text string) string {
	event := func(typ, data string) string {
		return fmt.Sprintf("event: %s\ndata: %s\n\n", typ, data)
	}
	return strings.Join([]string{
		event("message_start", `{"type":"message_start","message":{"id":"msg_01","type":"message","role":"assistant","content":[],"model":"claude-opus-4-6","stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":10,"output_tokens":0}}}`),
		event("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`),
		event("content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":%q}}`, text)),
		event("content_block_stop", `{"type":"content_block_stop","index":0}`),
		event("message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn","stop_sequence":null},"usage":{"output_tokens":5}}`),
		event("message_stop", `{"type":"message_stop"}`),
	}, "")
}

// mockAnthropicServer creates an httptest.Server that returns a canned SSE
// response for every POST to /v1/messages.
func mockAnthropicServer(t *testing.T, responseText string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, validSSEStream(responseText))
	}))
}

// runContext is a test-only helper that runs `cure context <args>` with a
// controlled session store directory and captured stdout/stderr buffers.
// This lets E2E tests verify output without capturing os.Stdout.
func runContext(t *testing.T, sessionDir string, out, errBuf *bytes.Buffer, args ...string) error {
	t.Helper()
	cfg := loadConfig()
	st, err := agentstore.NewJSONStore(sessionDir)
	if err != nil {
		return fmt.Errorf("runContext: create store: %w", err)
	}
	router := terminal.New(
		terminal.WithConfig(cfg),
		terminal.WithStdout(out),
		terminal.WithStderr(errBuf),
	)
	router.Register(ctxcmd.NewContextCommand(st))
	return router.RunArgs(append([]string{"context"}, args...))
}

// sessionDir returns the path where the store saves sessions under XDG_DATA_HOME.
func sessionDir(xdgHome string) string {
	return filepath.Join(xdgHome, "cure", "sessions")
}

// TestE2E_ContextNew_WithMockServer verifies that `context new --provider claude
// --message "hello"` creates a session file when a mock Anthropic server is used.
func TestE2E_ContextNew_WithMockServer(t *testing.T) {
	ts := mockAnthropicServer(t, "hi there")
	defer ts.Close()

	xdgHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgHome)
	t.Setenv("ANTHROPIC_BASE_URL", ts.URL)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	var out, errBuf bytes.Buffer
	err := runContext(t, sessionDir(xdgHome), &out, &errBuf,
		"new", "--provider", "claude", "--message", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr: %s)", err, errBuf.String())
	}

	// Verify that at least one session JSON file was created.
	entries, readErr := os.ReadDir(sessionDir(xdgHome))
	if readErr != nil {
		t.Fatalf("session dir not created: %v", readErr)
	}
	var jsonFiles []os.DirEntry
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			jsonFiles = append(jsonFiles, e)
		}
	}
	if len(jsonFiles) == 0 {
		t.Fatal("expected at least one session JSON file, found none")
	}

	// Parse the session file and verify its structure.
	data, err := os.ReadFile(filepath.Join(sessionDir(xdgHome), jsonFiles[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile session JSON: %v", err)
	}
	var sess map[string]any
	if err := json.Unmarshal(data, &sess); err != nil {
		t.Fatalf("session file is not valid JSON: %v", err)
	}
	for _, field := range []string{"id", "provider", "model", "history"} {
		if sess[field] == nil {
			t.Errorf("session JSON missing field %q", field)
		}
	}
	if sess["provider"] != "claude" {
		t.Errorf("provider = %q, want %q", sess["provider"], "claude")
	}
}

// TestE2E_ContextList_Empty verifies that `context list` returns exit 0 and
// a "No sessions found" message when the session directory is empty.
func TestE2E_ContextList_Empty(t *testing.T) {
	xdgHome := t.TempDir()

	var out, errBuf bytes.Buffer
	err := runContext(t, sessionDir(xdgHome), &out, &errBuf, "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "No sessions found") {
		t.Errorf("output = %q, want to contain %q", out.String(), "No sessions found")
	}
}

// TestE2E_ContextList_NDJSON verifies that `context list --format ndjson`
// outputs valid NDJSON when sessions exist.
func TestE2E_ContextList_NDJSON(t *testing.T) {
	ts := mockAnthropicServer(t, "ndjson test response")
	defer ts.Close()

	xdgHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgHome)
	t.Setenv("ANTHROPIC_BASE_URL", ts.URL)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	dir := sessionDir(xdgHome)

	// Create a session via context new.
	var out1, errBuf1 bytes.Buffer
	if err := runContext(t, dir, &out1, &errBuf1,
		"new", "--provider", "claude", "--message", "hello ndjson"); err != nil {
		t.Fatalf("context new: %v (stderr: %s)", err, errBuf1.String())
	}

	// Now list in NDJSON format.
	var out, errBuf bytes.Buffer
	if err := runContext(t, dir, &out, &errBuf, "list", "--format", "ndjson"); err != nil {
		t.Fatalf("context list --format ndjson: %v", err)
	}

	// Each non-empty line must be valid JSON.
	for i, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("line %d is not valid JSON: %v — %q", i, err, line)
		}
	}
}

// TestE2E_ContextResume_UnknownSession verifies that resuming a session ID that
// does not exist exits with an error containing a not-found message.
func TestE2E_ContextResume_UnknownSession(t *testing.T) {
	xdgHome := t.TempDir()

	var out, errBuf bytes.Buffer
	// Use a valid hex ID that simply doesn't exist in the store.
	err := runContext(t, sessionDir(xdgHome), &out, &errBuf,
		"resume", "000000000000000000000000deadbeef", "--message", "hi")
	if err == nil {
		t.Fatal("expected error for unknown session, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "not found")
	}
}

// TestE2E_ContextFork verifies that `context fork <id>` creates a new session
// file with a different ID and ForkOf set to the source ID.
func TestE2E_ContextFork(t *testing.T) {
	ts := mockAnthropicServer(t, "fork test")
	defer ts.Close()

	xdgHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgHome)
	t.Setenv("ANTHROPIC_BASE_URL", ts.URL)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	dir := sessionDir(xdgHome)

	// Create source session.
	var out1, errBuf1 bytes.Buffer
	if err := runContext(t, dir, &out1, &errBuf1,
		"new", "--provider", "claude", "--message", "source message"); err != nil {
		t.Fatalf("context new: %v", err)
	}

	// Identify the source session ID from the file.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var sourceID string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			sourceID = strings.TrimSuffix(e.Name(), ".json")
			break
		}
	}
	if sourceID == "" {
		t.Fatal("no source session file found")
	}

	// Fork it.
	var out2, errBuf2 bytes.Buffer
	if err := runContext(t, dir, &out2, &errBuf2, "fork", sourceID); err != nil {
		t.Fatalf("context fork: %v", err)
	}

	forkedID := strings.TrimSpace(out2.String())
	if forkedID == "" {
		t.Fatal("expected forked session ID on stdout, got empty output")
	}
	if forkedID == sourceID {
		t.Errorf("forked ID %q is same as source ID", forkedID)
	}

	// Verify the forked session file exists with ForkOf set.
	forkedPath := filepath.Join(dir, forkedID+".json")
	data, err := os.ReadFile(forkedPath)
	if err != nil {
		t.Fatalf("forked session file not found: %v", err)
	}
	var forked map[string]any
	if err := json.Unmarshal(data, &forked); err != nil {
		t.Fatalf("forked session file is not valid JSON: %v", err)
	}
	if forked["fork_of"] != sourceID {
		t.Errorf("fork_of = %q, want %q", forked["fork_of"], sourceID)
	}
}

// TestE2E_ContextDelete_Yes verifies that `context delete --yes <id>`
// removes the session file and exits 0. Note: --yes must precede the
// positional ID because Go's flag package stops parsing at the first
// non-flag argument.
func TestE2E_ContextDelete_Yes(t *testing.T) {
	ts := mockAnthropicServer(t, "delete test")
	defer ts.Close()

	xdgHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", xdgHome)
	t.Setenv("ANTHROPIC_BASE_URL", ts.URL)
	t.Setenv("ANTHROPIC_API_KEY", "test-key")

	dir := sessionDir(xdgHome)

	// Create a session to delete.
	var out1, errBuf1 bytes.Buffer
	if err := runContext(t, dir, &out1, &errBuf1,
		"new", "--provider", "claude", "--message", "to be deleted"); err != nil {
		t.Fatalf("context new: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var targetID string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			targetID = strings.TrimSuffix(e.Name(), ".json")
			break
		}
	}
	if targetID == "" {
		t.Fatal("no session file found to delete")
	}

	// Delete it with --yes.
	var out2, errBuf2 bytes.Buffer
	if err := runContext(t, dir, &out2, &errBuf2, "delete", "--yes", targetID); err != nil {
		t.Fatalf("context delete: %v", err)
	}
	if !strings.Contains(out2.String(), "deleted") {
		t.Errorf("output = %q, want to contain %q", out2.String(), "deleted")
	}

	// Session file must no longer exist.
	if _, err := os.Stat(filepath.Join(dir, targetID+".json")); !os.IsNotExist(err) {
		t.Errorf("session file still exists after delete (stat err: %v)", err)
	}
}
