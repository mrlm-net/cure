package gui_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	gui "github.com/mrlm-net/cure/internal/gui"
	"github.com/mrlm-net/cure/internal/gui/api"
	"github.com/mrlm-net/cure/pkg/agent"
	"github.com/mrlm-net/cure/pkg/agent/store"
	"github.com/mrlm-net/cure/pkg/config"
	"github.com/mrlm-net/cure/pkg/doctor"
)

// safeBuf is a goroutine-safe bytes.Buffer for capturing server output.
type safeBuf struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *safeBuf) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.Write(p)
}

func (s *safeBuf) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// testServer holds the running test server and its base URL.
type testServer struct {
	baseURL string
	cancel  context.CancelFunc
	errCh   chan error
	store   agent.SessionStore
}

// startTestServer starts a real gui.Server on a free port, waits for it to
// be ready by polling stdout for the URL line, and returns the base URL.
// The caller must call cleanup() when done.
func startTestServer(t *testing.T) *testServer {
	t.Helper()

	storeDir := t.TempDir()
	s, err := store.NewJSONStore(storeDir)
	if err != nil {
		t.Fatalf("NewJSONStore: %v", err)
	}

	cfg := config.NewConfig(config.ConfigObject{
		"version":        "test",
		"agent.provider": "test-provider",
		"agent.model":    "test-model",
	})

	checks := []doctor.CheckFunc{
		func() doctor.CheckResult {
			return doctor.CheckResult{Name: "test-check", Status: doctor.CheckPass, Message: "all good"}
		},
		func() doctor.CheckResult {
			return doctor.CheckResult{Name: "warn-check", Status: doctor.CheckWarn, Message: "something minor"}
		},
	}

	deps := api.Deps{
		Config: cfg.Data(),
		Checks: checks,
		Store:  s,
		Port:   0, // will be overridden when the server binds
	}
	apiRouter := api.NewAPIRouter(deps)

	mux := http.NewServeMux()
	mux.Handle("/api/", apiRouter)

	stdout := &safeBuf{}
	srv := gui.New(mux,
		gui.WithPort(0),
		gui.WithNoBrowser(),
		gui.WithStdout(stdout),
		gui.WithStderr(io.Discard),
	)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx) }()

	// Wait for server to print its URL so we know the port.
	var baseURL string
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			cancel()
			t.Fatal("timeout waiting for test server to start")
		case err := <-errCh:
			cancel()
			t.Fatalf("server exited early: %v", err)
		default:
		}
		line := stdout.String()
		if idx := strings.Index(line, "http://127.0.0.1:"); idx >= 0 {
			// Extract the URL from the output line.
			url := strings.TrimSpace(line[idx:])
			baseURL = url
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	ts := &testServer{
		baseURL: baseURL,
		cancel:  cancel,
		errCh:   errCh,
		store:   s,
	}

	t.Cleanup(func() {
		cancel()
		select {
		case <-errCh:
		case <-time.After(6 * time.Second):
			t.Error("server did not shut down within 6 seconds")
		}
	})

	return ts
}

// get performs a GET request against the test server and returns the response.
func get(t *testing.T, baseURL, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(baseURL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

// postJSON performs a POST request with a JSON body.
func postJSON(t *testing.T, baseURL, path string, body interface{}) *http.Response {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}
	resp, err := http.Post(baseURL+path, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

// doDelete performs a DELETE request.
func doDelete(t *testing.T, baseURL, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, baseURL+path, nil)
	if err != nil {
		t.Fatalf("new DELETE request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE %s: %v", path, err)
	}
	return resp
}

// readBody reads and closes the response body.
func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return data
}

// --- E2E Test Cases ---

func TestGUIServerStartup(t *testing.T) {
	ts := startTestServer(t)

	resp := get(t, ts.baseURL, "/api/health")
	body := readBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/health: status = %d, want %d; body = %s", resp.StatusCode, http.StatusOK, body)
	}

	var health map[string]interface{}
	if err := json.Unmarshal(body, &health); err != nil {
		t.Fatalf("unmarshal health: %v", err)
	}
	if health["status"] != "ok" {
		t.Errorf("health status = %v, want %q", health["status"], "ok")
	}
	// Port is 0 in Deps because we don't know it at Deps construction time,
	// so the health handler reports 0. The important thing is the key exists.
	if _, ok := health["port"]; !ok {
		t.Error("health response missing 'port' field")
	}
}

func TestGUIDoctorEndpoint(t *testing.T) {
	ts := startTestServer(t)

	resp := get(t, ts.baseURL, "/api/doctor")
	body := readBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/doctor: status = %d, want %d; body = %s", resp.StatusCode, http.StatusOK, body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var results []struct {
		Name    string `json:"name"`
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &results); err != nil {
		t.Fatalf("unmarshal doctor results: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	// Verify first check.
	if results[0].Name != "test-check" {
		t.Errorf("results[0].name = %q, want %q", results[0].Name, "test-check")
	}
	if results[0].Status != "pass" {
		t.Errorf("results[0].status = %q, want %q", results[0].Status, "pass")
	}
	if results[0].Message != "all good" {
		t.Errorf("results[0].message = %q, want %q", results[0].Message, "all good")
	}

	// Verify second check.
	if results[1].Name != "warn-check" {
		t.Errorf("results[1].name = %q, want %q", results[1].Name, "warn-check")
	}
	if results[1].Status != "warn" {
		t.Errorf("results[1].status = %q, want %q", results[1].Status, "warn")
	}
}

func TestGUIConfigEndpoint(t *testing.T) {
	ts := startTestServer(t)

	resp := get(t, ts.baseURL, "/api/config")
	body := readBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/config: status = %d, want %d; body = %s", resp.StatusCode, http.StatusOK, body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(body, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}
	if cfg["version"] != "test" {
		t.Errorf("config.version = %v, want %q", cfg["version"], "test")
	}
}

func TestGUIGenerateStubs(t *testing.T) {
	ts := startTestServer(t)

	t.Run("GET /api/generate/list returns 501", func(t *testing.T) {
		resp := get(t, ts.baseURL, "/api/generate/list")
		body := readBody(t, resp)

		if resp.StatusCode != http.StatusNotImplemented {
			t.Fatalf("status = %d, want %d; body = %s", resp.StatusCode, http.StatusNotImplemented, body)
		}

		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(body, &errResp); err != nil {
			t.Fatalf("unmarshal error response: %v", err)
		}
		if errResp.Error == "" {
			t.Error("error field is empty, want non-empty message")
		}
	})

	t.Run("POST /api/generate/test returns 501", func(t *testing.T) {
		resp := postJSON(t, ts.baseURL, "/api/generate/test", map[string]string{"name": "test"})
		body := readBody(t, resp)

		if resp.StatusCode != http.StatusNotImplemented {
			t.Fatalf("status = %d, want %d; body = %s", resp.StatusCode, http.StatusNotImplemented, body)
		}

		var errResp struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(body, &errResp); err != nil {
			t.Fatalf("unmarshal error response: %v", err)
		}
		if errResp.Error == "" {
			t.Error("error field is empty, want non-empty message")
		}
	})
}

func TestGUISessionCRUD(t *testing.T) {
	ts := startTestServer(t)

	// Step 1: Create a session.
	createResp := postJSON(t, ts.baseURL, "/api/context/sessions", map[string]string{
		"provider": "test-provider",
		"model":    "test-model",
	})
	createBody := readBody(t, createResp)

	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/context/sessions: status = %d, want %d; body = %s",
			createResp.StatusCode, http.StatusCreated, createBody)
	}

	var created struct {
		ID       string `json:"id"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
		ForkOf   string `json:"fork_of"`
		Turns    int    `json:"turns"`
		History  []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"history"`
	}
	if err := json.Unmarshal(createBody, &created); err != nil {
		t.Fatalf("unmarshal created session: %v", err)
	}
	if created.ID == "" {
		t.Fatal("created session ID is empty")
	}
	if created.Provider != "test-provider" {
		t.Errorf("provider = %q, want %q", created.Provider, "test-provider")
	}
	if created.Model != "test-model" {
		t.Errorf("model = %q, want %q", created.Model, "test-model")
	}
	sessionID := created.ID

	// Step 2: List sessions and verify the new session is present.
	listResp := get(t, ts.baseURL, "/api/context/sessions")
	listBody := readBody(t, listResp)

	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/context/sessions: status = %d, want %d; body = %s",
			listResp.StatusCode, http.StatusOK, listBody)
	}

	var sessions []struct {
		ID       string `json:"id"`
		Provider string `json:"provider"`
	}
	if err := json.Unmarshal(listBody, &sessions); err != nil {
		t.Fatalf("unmarshal sessions list: %v", err)
	}
	found := false
	for _, s := range sessions {
		if s.ID == sessionID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("session %s not found in list response", sessionID)
	}

	// Step 3: Get the single session by ID.
	getResp := get(t, ts.baseURL, "/api/context/sessions/"+sessionID)
	getBody := readBody(t, getResp)

	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/context/sessions/%s: status = %d, want %d; body = %s",
			sessionID, getResp.StatusCode, http.StatusOK, getBody)
	}

	var detail struct {
		ID       string `json:"id"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}
	if err := json.Unmarshal(getBody, &detail); err != nil {
		t.Fatalf("unmarshal session detail: %v", err)
	}
	if detail.ID != sessionID {
		t.Errorf("session detail ID = %q, want %q", detail.ID, sessionID)
	}

	// Step 4: Fork the session.
	forkResp := postJSON(t, ts.baseURL, fmt.Sprintf("/api/context/sessions/%s/fork", sessionID), nil)
	forkBody := readBody(t, forkResp)

	if forkResp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /api/context/sessions/%s/fork: status = %d, want %d; body = %s",
			sessionID, forkResp.StatusCode, http.StatusCreated, forkBody)
	}

	var forked struct {
		ID     string `json:"id"`
		ForkOf string `json:"fork_of"`
	}
	if err := json.Unmarshal(forkBody, &forked); err != nil {
		t.Fatalf("unmarshal forked session: %v", err)
	}
	if forked.ID == sessionID {
		t.Error("forked session must have a different ID")
	}
	if forked.ID == "" {
		t.Error("forked session ID is empty")
	}
	if forked.ForkOf != sessionID {
		t.Errorf("fork_of = %q, want %q", forked.ForkOf, sessionID)
	}

	// Step 5: Delete the original session.
	deleteResp := doDelete(t, ts.baseURL, "/api/context/sessions/"+sessionID)
	readBody(t, deleteResp) // consume body

	if deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE /api/context/sessions/%s: status = %d, want %d",
			sessionID, deleteResp.StatusCode, http.StatusNoContent)
	}

	// Step 6: Verify the deleted session returns 404.
	getDeletedResp := get(t, ts.baseURL, "/api/context/sessions/"+sessionID)
	getDeletedBody := readBody(t, getDeletedResp)

	if getDeletedResp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET deleted session: status = %d, want %d; body = %s",
			getDeletedResp.StatusCode, http.StatusNotFound, getDeletedBody)
	}

	// Step 7: Verify the forked session still exists.
	getForkedResp := get(t, ts.baseURL, "/api/context/sessions/"+forked.ID)
	getForkedBody := readBody(t, getForkedResp)

	if getForkedResp.StatusCode != http.StatusOK {
		t.Fatalf("GET forked session: status = %d, want %d; body = %s",
			getForkedResp.StatusCode, http.StatusOK, getForkedBody)
	}
}

func TestGUISessionCreate_DefaultsFromConfig(t *testing.T) {
	ts := startTestServer(t)

	// Create a session with empty body -- should use config defaults.
	resp := postJSON(t, ts.baseURL, "/api/context/sessions", map[string]string{})
	body := readBody(t, resp)

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body = %s", resp.StatusCode, http.StatusCreated, body)
	}

	var created struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Config has "agent.provider" and "agent.model" at the top level of ConfigObject.
	// The configString helper reads them via flat key lookup.
	if created.Provider != "test-provider" {
		t.Errorf("provider = %q, want %q", created.Provider, "test-provider")
	}
	if created.Model != "test-model" {
		t.Errorf("model = %q, want %q", created.Model, "test-model")
	}
}

func TestGUISSE(t *testing.T) {
	ts := startTestServer(t)

	// Create a session first.
	createResp := postJSON(t, ts.baseURL, "/api/context/sessions", map[string]string{
		"provider": "test-provider",
		"model":    "test-model",
	})
	createBody := readBody(t, createResp)
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("create session: status = %d, want %d; body = %s",
			createResp.StatusCode, http.StatusCreated, createBody)
	}
	var session struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createBody, &session); err != nil {
		t.Fatalf("unmarshal session: %v", err)
	}

	// Send a message to trigger SSE streaming (echo stub).
	msgBody, _ := json.Marshal(map[string]string{"message": "hello world"})
	resp, err := http.Post(
		ts.baseURL+"/api/context/sessions/"+session.ID+"/messages",
		"application/json",
		bytes.NewReader(msgBody),
	)
	if err != nil {
		t.Fatalf("POST messages: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want %d; body = %s", resp.StatusCode, http.StatusOK, body)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cc)
	}

	// Read the full SSE stream.
	sseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read SSE body: %v", err)
	}

	// Parse SSE events.
	type sseEvent struct {
		Kind       string `json:"kind"`
		Text       string `json:"text,omitempty"`
		StopReason string `json:"stop_reason,omitempty"`
	}
	var events []sseEvent
	for _, line := range strings.Split(string(sseBody), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		var ev sseEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			t.Fatalf("unmarshal SSE event %q: %v", data, err)
		}
		events = append(events, ev)
	}

	if len(events) < 3 {
		t.Fatalf("expected at least 3 SSE events (start, token(s), done), got %d: %v", len(events), events)
	}

	// First event must be "start".
	if events[0].Kind != "start" {
		t.Errorf("events[0].kind = %q, want %q", events[0].Kind, "start")
	}

	// Last event must be "done".
	last := events[len(events)-1]
	if last.Kind != "done" {
		t.Errorf("last event kind = %q, want %q", last.Kind, "done")
	}
	if last.StopReason != "end_turn" {
		t.Errorf("last event stop_reason = %q, want %q", last.StopReason, "end_turn")
	}

	// Middle events must all be "token" events.
	var tokenText strings.Builder
	for _, ev := range events[1 : len(events)-1] {
		if ev.Kind != "token" {
			t.Errorf("middle event kind = %q, want %q", ev.Kind, "token")
		}
		tokenText.WriteString(ev.Text)
	}

	// The echo stub reflects the input message back as tokens.
	if got := tokenText.String(); got != "hello world" {
		t.Errorf("accumulated token text = %q, want %q", got, "hello world")
	}

	// Verify that the session history was persisted with both user and assistant messages.
	getResp := get(t, ts.baseURL, "/api/context/sessions/"+session.ID)
	getBody := readBody(t, getResp)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("GET session after SSE: status = %d; body = %s", getResp.StatusCode, getBody)
	}

	var updated struct {
		History []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"history"`
		Turns int `json:"turns"`
	}
	if err := json.Unmarshal(getBody, &updated); err != nil {
		t.Fatalf("unmarshal updated session: %v", err)
	}
	if len(updated.History) != 2 {
		t.Fatalf("history len = %d, want 2 (user + assistant)", len(updated.History))
	}
	if updated.History[0].Role != "user" {
		t.Errorf("history[0].role = %q, want %q", updated.History[0].Role, "user")
	}
	if updated.History[0].Content != "hello world" {
		t.Errorf("history[0].content = %q, want %q", updated.History[0].Content, "hello world")
	}
	if updated.History[1].Role != "assistant" {
		t.Errorf("history[1].role = %q, want %q", updated.History[1].Role, "assistant")
	}
	if updated.History[1].Content != "hello world" {
		t.Errorf("history[1].content = %q, want %q (echo stub)", updated.History[1].Content, "hello world")
	}
}

func TestGUISSE_SessionNotFound(t *testing.T) {
	ts := startTestServer(t)

	msgBody, _ := json.Marshal(map[string]string{"message": "hello"})
	resp, err := http.Post(
		ts.baseURL+"/api/context/sessions/deadbeefdeadbeefdeadbeefdeadbeef/messages",
		"application/json",
		bytes.NewReader(msgBody),
	)
	if err != nil {
		t.Fatalf("POST messages: %v", err)
	}
	body := readBody(t, resp)

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", resp.StatusCode, http.StatusNotFound, body)
	}
}

func TestGUISSE_EmptyMessage(t *testing.T) {
	ts := startTestServer(t)

	// Create a session.
	createResp := postJSON(t, ts.baseURL, "/api/context/sessions", map[string]string{
		"provider": "test-provider",
		"model":    "test-model",
	})
	createBody := readBody(t, createResp)
	var session struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createBody, &session); err != nil {
		t.Fatalf("unmarshal session: %v", err)
	}

	// Send empty message.
	msgBody, _ := json.Marshal(map[string]string{"message": ""})
	resp, err := http.Post(
		ts.baseURL+"/api/context/sessions/"+session.ID+"/messages",
		"application/json",
		bytes.NewReader(msgBody),
	)
	if err != nil {
		t.Fatalf("POST messages: %v", err)
	}
	body := readBody(t, resp)

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", resp.StatusCode, http.StatusBadRequest, body)
	}
}

func TestGUIDeleteThenFork_NotFound(t *testing.T) {
	ts := startTestServer(t)

	// Create and immediately delete a session.
	createResp := postJSON(t, ts.baseURL, "/api/context/sessions", map[string]string{
		"provider": "test-provider",
		"model":    "test-model",
	})
	createBody := readBody(t, createResp)
	var session struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(createBody, &session); err != nil {
		t.Fatalf("unmarshal session: %v", err)
	}

	deleteResp := doDelete(t, ts.baseURL, "/api/context/sessions/"+session.ID)
	readBody(t, deleteResp)

	// Try to fork a deleted session.
	forkResp := postJSON(t, ts.baseURL, fmt.Sprintf("/api/context/sessions/%s/fork", session.ID), nil)
	forkBody := readBody(t, forkResp)

	if forkResp.StatusCode != http.StatusNotFound {
		t.Fatalf("fork deleted session: status = %d, want %d; body = %s",
			forkResp.StatusCode, http.StatusNotFound, forkBody)
	}
}

func TestGUIMethodNotAllowed(t *testing.T) {
	ts := startTestServer(t)

	// POST to /api/health should be 405.
	resp, err := http.Post(ts.baseURL+"/api/health", "application/json", nil)
	if err != nil {
		t.Fatalf("POST /api/health: %v", err)
	}
	readBody(t, resp)

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/health: status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}
