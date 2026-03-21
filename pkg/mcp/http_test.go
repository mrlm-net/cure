package mcp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// postMCP sends a POST request to /mcp with the given JSON body and optional
// session ID header. Returns the response recorder.
func postMCP(t *testing.T, handler http.Handler, body string, sessionID string, accept string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

// buildStore creates a new sessionStore for testing.
func buildStore() *sessionStore {
	return &sessionStore{sessions: make(map[string]*session)}
}

func TestHTTPHandler_Post_Initialize(t *testing.T) {
	srv := New(WithName("http-test"), WithVersion("1.0.0"))
	store := buildStore()
	handler := http.HandlerFunc(srv.mcpHandler(store))

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	rr := postMCP(t, handler, body, "", "")

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	// A session ID must be assigned when no prior session exists.
	sessionID := rr.Header().Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Error("response must include Mcp-Session-Id header")
	}

	var resp map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	result := resp["result"].(map[string]any)
	if result["protocolVersion"] != "2025-03-26" {
		t.Errorf("protocolVersion = %v, want 2025-03-26", result["protocolVersion"])
	}
}

func TestHTTPHandler_Post_WithExistingSession(t *testing.T) {
	srv := New()
	store := buildStore()

	// Create a session manually.
	sess, err := store.create()
	if err != nil {
		t.Fatalf("store.create: %v", err)
	}

	handler := http.HandlerFunc(srv.mcpHandler(store))
	body := `{"jsonrpc":"2.0","id":2,"method":"ping"}`
	rr := postMCP(t, handler, body, sess.id, "")

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHTTPHandler_Post_InvalidSession(t *testing.T) {
	srv := New()
	store := buildStore()
	handler := http.HandlerFunc(srv.mcpHandler(store))

	body := `{"jsonrpc":"2.0","id":1,"method":"ping"}`
	rr := postMCP(t, handler, body, "nonexistent-session-id", "")

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHTTPHandler_Post_SSEResponse(t *testing.T) {
	srv := New()
	store := buildStore()
	handler := http.HandlerFunc(srv.mcpHandler(store))

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
	rr := postMCP(t, handler, body, "", "text/event-stream")

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
	body2 := rr.Body.String()
	if !strings.HasPrefix(body2, "data: ") {
		t.Errorf("SSE body must start with 'data: ', got: %q", body2)
	}
}

func TestHTTPHandler_Post_Notification(t *testing.T) {
	srv := New()
	store := buildStore()

	sess, _ := store.create()
	handler := http.HandlerFunc(srv.mcpHandler(store))

	// notifications/initialized has no ID — server should respond 204.
	body := `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Mcp-Session-Id", sess.id)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d (NoContent)", rr.Code, http.StatusNoContent)
	}
}

func TestHTTPHandler_Post_ParseError(t *testing.T) {
	srv := New()
	store := buildStore()
	handler := http.HandlerFunc(srv.mcpHandler(store))

	rr := postMCP(t, handler, "this is not JSON", "", "")
	// Parse error on new session — still creates a session but returns error.
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var resp map[string]any
	json.Unmarshal(rr.Body.Bytes(), &resp)
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error in response: %v", resp)
	}
	if errObj["code"].(float64) != codeParseError {
		t.Errorf("code = %v, want %d", errObj["code"], codeParseError)
	}
}

func TestHTTPHandler_Delete(t *testing.T) {
	srv := New()
	store := buildStore()

	sess, _ := store.create()
	handler := http.HandlerFunc(srv.mcpHandler(store))

	req := httptest.NewRequest(http.MethodDelete, "/mcp", nil)
	req.Header.Set("Mcp-Session-Id", sess.id)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	// Session should be gone.
	if _, ok := store.get(sess.id); ok {
		t.Error("session must be removed after DELETE")
	}
}

func TestHTTPHandler_Delete_NoSessionID(t *testing.T) {
	srv := New()
	store := buildStore()
	handler := http.HandlerFunc(srv.mcpHandler(store))

	req := httptest.NewRequest(http.MethodDelete, "/mcp", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// DELETE without a session ID must return 400 Bad Request.
	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (BadRequest, missing session ID)", rr.Code, http.StatusBadRequest)
	}
}

func TestHTTPHandler_MethodNotAllowed(t *testing.T) {
	srv := New()
	store := buildStore()
	handler := http.HandlerFunc(srv.mcpHandler(store))

	req := httptest.NewRequest(http.MethodPut, "/mcp", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestHTTPHandler_Get_SSE_NoSession(t *testing.T) {
	srv := New()
	store := buildStore()
	handler := http.HandlerFunc(srv.mcpHandler(store))

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d (BadRequest, missing session)", rr.Code, http.StatusBadRequest)
	}
}

func TestHTTPHandler_Get_SSE_InvalidSession(t *testing.T) {
	srv := New()
	store := buildStore()
	handler := http.HandlerFunc(srv.mcpHandler(store))

	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Mcp-Session-Id", "no-such-session")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestCheckOrigin(t *testing.T) {
	tests := []struct {
		name           string
		allowedOrigins []string
		origin         string
		wantAllowed    bool
	}{
		{
			name:        "no restrictions — all allowed",
			wantAllowed: true,
		},
		{
			name:           "no origin header — allowed (non-browser)",
			allowedOrigins: []string{"https://example.com"},
			origin:         "",
			wantAllowed:    true,
		},
		{
			name:           "matching origin",
			allowedOrigins: []string{"https://example.com"},
			origin:         "https://example.com",
			wantAllowed:    true,
		},
		{
			name:           "matching origin case-insensitive",
			allowedOrigins: []string{"https://Example.Com"},
			origin:         "https://example.com",
			wantAllowed:    true,
		},
		{
			name:           "non-matching origin",
			allowedOrigins: []string{"https://example.com"},
			origin:         "https://evil.com",
			wantAllowed:    false,
		},
		{
			name:           "multiple allowed — one matches",
			allowedOrigins: []string{"https://a.com", "https://b.com"},
			origin:         "https://b.com",
			wantAllowed:    true,
		},
		{
			name:           "null origin with allowlist — rejected (file:// / sandboxed iframe)",
			allowedOrigins: []string{"https://example.com"},
			origin:         "null",
			wantAllowed:    false,
		},
		{
			name:        "null origin with empty allowlist — allowed (no restrictions)",
			origin:      "null",
			wantAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := New(WithAllowedOrigins(tt.allowedOrigins...))
			req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			got := srv.checkOrigin(req)
			if got != tt.wantAllowed {
				t.Errorf("checkOrigin() = %v, want %v", got, tt.wantAllowed)
			}
		})
	}
}

func TestCheckOrigin_Forbidden(t *testing.T) {
	srv := New(WithAllowedOrigins("https://safe.com"))
	store := buildStore()
	handler := http.HandlerFunc(srv.mcpHandler(store))

	req := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader([]byte("{}")))
	req.Header.Set("Origin", "https://evil.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusForbidden)
	}
}

func TestSessionStore(t *testing.T) {
	store := buildStore()

	t.Run("create and get", func(t *testing.T) {
		sess, err := store.create()
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		if sess.id == "" {
			t.Error("session ID must not be empty")
		}
		got, ok := store.get(sess.id)
		if !ok {
			t.Fatal("get must find the session")
		}
		if got != sess {
			t.Error("get must return the same session pointer")
		}
	})

	t.Run("delete", func(t *testing.T) {
		sess, _ := store.create()
		store.delete(sess.id)
		if _, ok := store.get(sess.id); ok {
			t.Error("session must not be found after delete")
		}
	})

	t.Run("get missing", func(t *testing.T) {
		_, ok := store.get("does-not-exist")
		if ok {
			t.Error("get must return false for missing session")
		}
	})

	t.Run("unique IDs", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 100; i++ {
			sess, err := store.create()
			if err != nil {
				t.Fatalf("create[%d]: %v", i, err)
			}
			if ids[sess.id] {
				t.Errorf("duplicate session ID: %s", sess.id)
			}
			ids[sess.id] = true
		}
	})
}

func TestSession_Touch(t *testing.T) {
	sess := &session{
		id:       "test",
		events:   make(chan []byte, 1),
		lastSeen: time.Now().Add(-time.Minute),
	}
	before := sess.lastSeen
	time.Sleep(time.Millisecond)
	sess.touch()
	if !sess.lastSeen.After(before) {
		t.Error("touch() must update lastSeen to a later time")
	}
}
