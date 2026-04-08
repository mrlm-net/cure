package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mrlm-net/cure/pkg/agent"
)

func TestMessagesSSE_EchoStub(t *testing.T) {
	store := newMemStore()
	sess := agent.NewSession("claude", "opus")
	_ = store.Save(context.Background(), sess)

	handler := NewAPIRouter(sessionsDeps(store))

	body := `{"message":"hello world"}`
	req := httptest.NewRequest(http.MethodPost, "/api/context/sessions/"+sess.ID+"/messages", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("Cache-Control = %q, want no-cache", cc)
	}
	if xab := rec.Header().Get("X-Accel-Buffering"); xab != "no" {
		t.Errorf("X-Accel-Buffering = %q, want no", xab)
	}

	// Parse SSE events from the response body.
	events := parseSSEEvents(t, rec.Body.String())

	if len(events) < 3 {
		t.Fatalf("expected at least 3 events (start, token(s), done), got %d", len(events))
	}

	// First event must be start.
	if events[0].Kind != agent.EventKindStart {
		t.Errorf("events[0].kind = %q, want start", events[0].Kind)
	}

	// Last event must be done.
	last := events[len(events)-1]
	if last.Kind != agent.EventKindDone {
		t.Errorf("last event kind = %q, want done", last.Kind)
	}

	// All middle events must be tokens.
	var tokenText strings.Builder
	for _, ev := range events[1 : len(events)-1] {
		if ev.Kind != agent.EventKindToken {
			t.Errorf("middle event kind = %q, want token", ev.Kind)
		}
		tokenText.WriteString(ev.Text)
	}

	// The echo stub should reflect the input message.
	if got := tokenText.String(); got != "hello world" {
		t.Errorf("token text = %q, want %q", got, "hello world")
	}
}

func TestMessagesSSE_SessionNotFound(t *testing.T) {
	store := newMemStore()
	handler := NewAPIRouter(sessionsDeps(store))

	body := `{"message":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/context/sessions/nonexistent/messages", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestMessagesSSE_EmptyMessage(t *testing.T) {
	store := newMemStore()
	sess := agent.NewSession("claude", "opus")
	_ = store.Save(context.Background(), sess)

	handler := NewAPIRouter(sessionsDeps(store))

	body := `{"message":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/context/sessions/"+sess.ID+"/messages", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMessagesSSE_InvalidBody(t *testing.T) {
	store := newMemStore()
	sess := agent.NewSession("claude", "opus")
	_ = store.Save(context.Background(), sess)

	handler := NewAPIRouter(sessionsDeps(store))

	req := httptest.NewRequest(http.MethodPost, "/api/context/sessions/"+sess.ID+"/messages", bytes.NewBufferString("not json"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestMessagesSSE_PersistsAssistantMessage(t *testing.T) {
	store := newMemStore()
	sess := agent.NewSession("claude", "opus")
	_ = store.Save(context.Background(), sess)

	handler := NewAPIRouter(sessionsDeps(store))

	body := `{"message":"ping"}`
	req := httptest.NewRequest(http.MethodPost, "/api/context/sessions/"+sess.ID+"/messages", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	// Reload session from store and verify history was saved.
	updated, err := store.Load(context.Background(), sess.ID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if len(updated.History) != 2 {
		t.Fatalf("history len = %d, want 2 (user + assistant)", len(updated.History))
	}
	if updated.History[0].Role != agent.RoleUser {
		t.Errorf("history[0].role = %q, want user", updated.History[0].Role)
	}
	if updated.History[1].Role != agent.RoleAssistant {
		t.Errorf("history[1].role = %q, want assistant", updated.History[1].Role)
	}
	assistantText := agent.TextOf(updated.History[1].Content)
	if assistantText != "ping" {
		t.Errorf("assistant text = %q, want %q", assistantText, "ping")
	}
}

func TestMessagesSSE_CustomAgentRun(t *testing.T) {
	store := newMemStore()
	sess := agent.NewSession("claude", "opus")
	_ = store.Save(context.Background(), sess)

	// Custom run function that emits a fixed token.
	customRun := func(ctx context.Context, session *agent.Session) <-chan agentResult {
		ch := make(chan agentResult, 4)
		go func() {
			defer close(ch)
			ch <- agentResult{Event: agent.Event{Kind: agent.EventKindStart}}
			ch <- agentResult{Event: agent.Event{Kind: agent.EventKindToken, Text: "custom-response"}}
			ch <- agentResult{Event: agent.Event{Kind: agent.EventKindDone, StopReason: "end_turn"}}
		}()
		return ch
	}

	deps := sessionsDeps(store)
	deps.AgentRun = customRun
	handler := NewAPIRouter(deps)

	body := `{"message":"anything"}`
	req := httptest.NewRequest(http.MethodPost, "/api/context/sessions/"+sess.ID+"/messages", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	events := parseSSEEvents(t, rec.Body.String())
	// Find token events.
	var tokenText strings.Builder
	for _, ev := range events {
		if ev.Kind == agent.EventKindToken {
			tokenText.WriteString(ev.Text)
		}
	}
	if got := tokenText.String(); got != "custom-response" {
		t.Errorf("token text = %q, want %q", got, "custom-response")
	}
}

func TestMessagesSSE_ErrorEvent(t *testing.T) {
	store := newMemStore()
	sess := agent.NewSession("claude", "opus")
	_ = store.Save(context.Background(), sess)

	// Agent that emits start then an error.
	errRun := func(ctx context.Context, session *agent.Session) <-chan agentResult {
		ch := make(chan agentResult, 4)
		go func() {
			defer close(ch)
			ch <- agentResult{Event: agent.Event{Kind: agent.EventKindStart}}
			ch <- agentResult{Event: agent.Event{Kind: agent.EventKindError, Err: "provider failure"}}
		}()
		return ch
	}

	deps := sessionsDeps(store)
	deps.AgentRun = errRun
	handler := NewAPIRouter(deps)

	body := `{"message":"hello"}`
	req := httptest.NewRequest(http.MethodPost, "/api/context/sessions/"+sess.ID+"/messages", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	events := parseSSEEvents(t, rec.Body.String())
	// Should have at least start and error.
	found := false
	for _, ev := range events {
		if ev.Kind == agent.EventKindError {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected an error event in the stream")
	}
}

// parseSSEEvents parses SSE data lines from a response body into SSEEvent structs.
func parseSSEEvents(t *testing.T, body string) []SSEEvent {
	t.Helper()
	var events []SSEEvent
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		var ev SSEEvent
		if err := json.Unmarshal([]byte(data), &ev); err != nil {
			t.Fatalf("unmarshal SSE event %q: %v", data, err)
		}
		events = append(events, ev)
	}
	return events
}
