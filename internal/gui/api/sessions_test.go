package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/mrlm-net/cure/pkg/agent"
)

// memStore is an in-memory SessionStore for testing.
type memStore struct {
	mu       sync.Mutex
	sessions map[string]*agent.Session
}

func newMemStore() *memStore {
	return &memStore{sessions: make(map[string]*agent.Session)}
}

func (m *memStore) Save(_ context.Context, s *agent.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Deep copy to avoid test mutation issues.
	data, _ := json.Marshal(s)
	var copy agent.Session
	_ = json.Unmarshal(data, &copy)
	m.sessions[s.ID] = &copy
	return nil
}

func (m *memStore) Load(_ context.Context, id string) (*agent.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return nil, agent.ErrSessionNotFound
	}
	// Deep copy to avoid mutation.
	data, _ := json.Marshal(s)
	var copy agent.Session
	_ = json.Unmarshal(data, &copy)
	return &copy, nil
}

func (m *memStore) List(_ context.Context) ([]*agent.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]*agent.Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].UpdatedAt.Equal(result[j].UpdatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result, nil
}

func (m *memStore) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[id]; !ok {
		return agent.ErrSessionNotFound
	}
	delete(m.sessions, id)
	return nil
}

func (m *memStore) Fork(ctx context.Context, id string) (*agent.Session, error) {
	m.mu.Lock()
	s, ok := m.sessions[id]
	m.mu.Unlock()
	if !ok {
		return nil, agent.ErrSessionNotFound
	}
	forked := s.Fork()
	if err := m.Save(ctx, forked); err != nil {
		return nil, err
	}
	return forked, nil
}

// sessionsDeps builds a Deps with the given store for session endpoint testing.
func sessionsDeps(store agent.SessionStore) Deps {
	return Deps{
		Config: nil,
		Checks: nil,
		Port:   9090,
		Store:  store,
	}
}

func TestSessionsList_EmptyStore(t *testing.T) {
	store := newMemStore()
	handler := NewAPIRouter(sessionsDeps(store))

	req := httptest.NewRequest(http.MethodGet, "/api/context/sessions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	// Must be [] not null.
	body := bytes.TrimSpace(rec.Body.Bytes())
	if string(body) != "[]" {
		t.Errorf("body = %s, want []", body)
	}
}

func TestSessionsList_WithSessions(t *testing.T) {
	store := newMemStore()
	s1 := agent.NewSession("claude", "opus")
	s1.UpdatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	s2 := agent.NewSession("openai", "gpt-4")
	s2.UpdatedAt = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	_ = store.Save(context.Background(), s1)
	_ = store.Save(context.Background(), s2)

	handler := NewAPIRouter(sessionsDeps(store))

	req := httptest.NewRequest(http.MethodGet, "/api/context/sessions", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var summaries []SessionSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &summaries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("len = %d, want 2", len(summaries))
	}
	// s2 has later UpdatedAt, so it should come first.
	if summaries[0].ID != s2.ID {
		t.Errorf("first session ID = %q, want %q", summaries[0].ID, s2.ID)
	}
}

func TestSessionsCreate(t *testing.T) {
	store := newMemStore()
	handler := NewAPIRouter(sessionsDeps(store))

	body := `{"provider":"claude","model":"sonnet"}`
	req := httptest.NewRequest(http.MethodPost, "/api/context/sessions", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var detail SessionDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if detail.Provider != "claude" {
		t.Errorf("provider = %q, want %q", detail.Provider, "claude")
	}
	if detail.Model != "sonnet" {
		t.Errorf("model = %q, want %q", detail.Model, "sonnet")
	}
	if detail.ID == "" {
		t.Error("session ID must not be empty")
	}
	if detail.History == nil {
		t.Error("history must not be nil")
	}
}

func TestSessionsCreate_DefaultsFromConfig(t *testing.T) {
	store := newMemStore()
	deps := sessionsDeps(store)
	deps.Config = map[string]any{
		"agent.provider": "openai",
		"agent.model":    "gpt-4o",
	}
	handler := NewAPIRouter(deps)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/context/sessions", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var detail SessionDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if detail.Provider != "openai" {
		t.Errorf("provider = %q, want %q", detail.Provider, "openai")
	}
	if detail.Model != "gpt-4o" {
		t.Errorf("model = %q, want %q", detail.Model, "gpt-4o")
	}
}

func TestSessionsCreate_InvalidBody(t *testing.T) {
	store := newMemStore()
	handler := NewAPIRouter(sessionsDeps(store))

	req := httptest.NewRequest(http.MethodPost, "/api/context/sessions", bytes.NewBufferString("not json"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestSessionsGet_Found(t *testing.T) {
	store := newMemStore()
	sess := agent.NewSession("claude", "opus")
	sess.AppendUserMessage("hello")
	sess.AppendAssistantMessage("hi there")
	_ = store.Save(context.Background(), sess)

	handler := NewAPIRouter(sessionsDeps(store))

	req := httptest.NewRequest(http.MethodGet, "/api/context/sessions/"+sess.ID, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var detail SessionDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if detail.ID != sess.ID {
		t.Errorf("id = %q, want %q", detail.ID, sess.ID)
	}
	if len(detail.History) != 2 {
		t.Fatalf("history len = %d, want 2", len(detail.History))
	}
	if detail.History[0].Role != "user" {
		t.Errorf("history[0].role = %q, want user", detail.History[0].Role)
	}
	if detail.History[0].Content != "hello" {
		t.Errorf("history[0].content = %q, want hello", detail.History[0].Content)
	}
	if detail.History[1].Role != "assistant" {
		t.Errorf("history[1].role = %q, want assistant", detail.History[1].Role)
	}
	if detail.Turns != 2 {
		t.Errorf("turns = %d, want 2", detail.Turns)
	}
}

func TestSessionsGet_NotFound(t *testing.T) {
	store := newMemStore()
	handler := NewAPIRouter(sessionsDeps(store))

	req := httptest.NewRequest(http.MethodGet, "/api/context/sessions/nonexistent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if errResp.Error != "session not found" {
		t.Errorf("error = %q, want %q", errResp.Error, "session not found")
	}
}

func TestSessionsDelete_Success(t *testing.T) {
	store := newMemStore()
	sess := agent.NewSession("claude", "opus")
	_ = store.Save(context.Background(), sess)

	handler := NewAPIRouter(sessionsDeps(store))

	req := httptest.NewRequest(http.MethodDelete, "/api/context/sessions/"+sess.ID, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}

	// Confirm the session is gone.
	_, err := store.Load(context.Background(), sess.ID)
	if err == nil {
		t.Error("session should have been deleted")
	}
}

func TestSessionsDelete_NotFound(t *testing.T) {
	store := newMemStore()
	handler := NewAPIRouter(sessionsDeps(store))

	req := httptest.NewRequest(http.MethodDelete, "/api/context/sessions/nonexistent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSessionsFork(t *testing.T) {
	store := newMemStore()
	sess := agent.NewSession("claude", "opus")
	sess.AppendUserMessage("hello")
	_ = store.Save(context.Background(), sess)

	handler := NewAPIRouter(sessionsDeps(store))

	req := httptest.NewRequest(http.MethodPost, "/api/context/sessions/"+sess.ID+"/fork", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var detail SessionDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if detail.ID == sess.ID {
		t.Error("forked session must have a different ID")
	}
	if detail.ForkOf != sess.ID {
		t.Errorf("fork_of = %q, want %q", detail.ForkOf, sess.ID)
	}
	if len(detail.History) != 1 {
		t.Errorf("history len = %d, want 1", len(detail.History))
	}
	if detail.Provider != "claude" {
		t.Errorf("provider = %q, want %q", detail.Provider, "claude")
	}
}

func TestSessionsFork_NotFound(t *testing.T) {
	store := newMemStore()
	handler := NewAPIRouter(sessionsDeps(store))

	req := httptest.NewRequest(http.MethodPost, "/api/context/sessions/nonexistent/fork", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestSessionsEndpoints_NilStore(t *testing.T) {
	// When Store is nil, session endpoints should 404 (not registered).
	deps := Deps{Port: 9090}
	handler := NewAPIRouter(deps)

	endpoints := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/context/sessions"},
		{http.MethodPost, "/api/context/sessions"},
		{http.MethodGet, "/api/context/sessions/abc"},
		{http.MethodDelete, "/api/context/sessions/abc"},
		{http.MethodPost, "/api/context/sessions/abc/fork"},
		{http.MethodPost, "/api/context/sessions/abc/messages"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
			}
		})
	}
}
