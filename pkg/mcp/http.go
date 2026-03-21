package mcp

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// session represents a single HTTP Streamable MCP client session. Each session
// has a unique ID, an event channel for server-initiated messages, and a
// last-seen timestamp for idle cleanup.
type session struct {
	id       string
	events   chan []byte
	lastSeen time.Time
	mu       sync.Mutex
}

// touch updates the session's last-seen time to now.
func (sess *session) touch() {
	sess.mu.Lock()
	sess.lastSeen = time.Now()
	sess.mu.Unlock()
}

// sessionStore is a concurrent-safe registry of active sessions.
type sessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*session
}

// create allocates a new session with a cryptographically random ID and
// registers it in the store.
func (ss *sessionStore) create() (*session, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, fmt.Errorf("mcp: generate session ID: %w", err)
	}
	sess := &session{
		id:       hex.EncodeToString(b),
		events:   make(chan []byte, 64),
		lastSeen: time.Now(),
	}
	ss.mu.Lock()
	ss.sessions[sess.id] = sess
	ss.mu.Unlock()
	return sess, nil
}

// get retrieves a session by ID.
func (ss *sessionStore) get(id string) (*session, bool) {
	ss.mu.RLock()
	sess, ok := ss.sessions[id]
	ss.mu.RUnlock()
	return sess, ok
}

// delete removes a session from the store and closes its event channel.
func (ss *sessionStore) delete(id string) {
	ss.mu.Lock()
	sess, ok := ss.sessions[id]
	if ok {
		delete(ss.sessions, id)
	}
	ss.mu.Unlock()
	if ok {
		close(sess.events)
	}
}

// ServeHTTP starts the MCP server in HTTP Streamable transport mode.
// All MCP traffic is served on the /mcp endpoint.
//
// addr overrides the address set by [WithAddr]; pass "" to use the configured
// address. ServeHTTP blocks until ctx is cancelled or a fatal error occurs.
func (s *Server) ServeHTTP(ctx context.Context, addr string) error {
	if addr == "" {
		addr = s.addr
	}

	store := &sessionStore{sessions: make(map[string]*session)}
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.mcpHandler(store))

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Shutdown when the context is cancelled.
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	}()

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("mcp: HTTP server: %w", err)
	}
	return nil
}

// mcpHandler returns the http.HandlerFunc for the /mcp endpoint. It routes GET,
// POST, and DELETE requests to the appropriate sub-handlers.
func (s *Server) mcpHandler(store *sessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.checkOrigin(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		switch r.Method {
		case http.MethodPost:
			s.handleHTTPPost(store, w, r)
		case http.MethodGet:
			s.handleHTTPGet(store, w, r)
		case http.MethodDelete:
			s.handleHTTPDelete(store, w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// checkOrigin validates the request Origin header against the allowedOrigins list.
// Returns true when:
//   - allowedOrigins is empty (allow all), or
//   - the request has no Origin header (non-browser / same-origin), or
//   - the Origin matches one of the allowed values (case-insensitive).
func (s *Server) checkOrigin(r *http.Request) bool {
	if len(s.allowedOrigins) == 0 {
		return true
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	for _, allowed := range s.allowedOrigins {
		if strings.EqualFold(allowed, origin) {
			return true
		}
	}
	return false
}

// handleHTTPPost processes incoming JSON-RPC 2.0 messages from the client.
//
// Session lifecycle:
//   - If no Mcp-Session-Id header is present, the request must be an initialize
//     call. A new session is created and its ID is returned in the response header.
//   - If Mcp-Session-Id is present, the existing session is looked up.
//
// Response mode:
//   - If the client Accept header includes "text/event-stream", the response is
//     sent as a single SSE event ("data: <json>\n\n").
//   - Otherwise, the response is sent as inline application/json.
func (s *Server) handleHTTPPost(store *sessionStore, w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("Mcp-Session-Id")

	var sess *session
	if sessionID == "" {
		// No session yet — create one (caller must send initialize first).
		var err error
		sess, err = store.create()
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Mcp-Session-Id", sess.id)
	} else {
		var ok bool
		sess, ok = store.get(sessionID)
		if !ok {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
	}
	sess.touch()

	// Decode the JSON-RPC request body (limit 4 MiB).
	body := io.LimitReader(r.Body, 4*1024*1024)
	data, err := io.ReadAll(body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	var req jsonrpcRequest
	if err := json.Unmarshal(data, &req); err != nil {
		writeJSONError(w, r, errResponse(nil, codeParseError, "parse error: "+err.Error()))
		return
	}

	resp := s.handleRequest(r.Context(), req)
	if resp == nil {
		// Notification — 204 No Content.
		w.WriteHeader(http.StatusNoContent)
		return
	}

	useSSE := strings.Contains(r.Header.Get("Accept"), "text/event-stream")
	if useSSE {
		writeSSEResponse(w, resp)
	} else {
		writeJSONResponse(w, resp)
	}
}

// handleHTTPGet opens a Server-Sent Events stream for server-initiated
// notifications. The client must supply a valid Mcp-Session-Id header.
func (s *Server) handleHTTPGet(store *sessionStore, w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		http.Error(w, "missing Mcp-Session-Id header", http.StatusBadRequest)
		return
	}

	sess, ok := store.get(sessionID)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, open := <-sess.events:
			if !open {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", event)
			flusher.Flush()
		}
	}
}

// handleHTTPDelete terminates an existing session.
func (s *Server) handleHTTPDelete(store *sessionStore, w http.ResponseWriter, r *http.Request) {
	id := r.Header.Get("Mcp-Session-Id")
	if id != "" {
		store.delete(id)
	}
	w.WriteHeader(http.StatusOK)
}

// ---- HTTP response helpers ----

// writeJSONResponse writes resp as application/json with HTTP 200.
func writeJSONResponse(w http.ResponseWriter, resp *jsonrpcResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// writeSSEResponse wraps resp in a single SSE "data:" event.
func writeSSEResponse(w http.ResponseWriter, resp *jsonrpcResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "data: %s\n\n", data)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// writeJSONError writes an error response. If the client accepts SSE, it sends
// an SSE event; otherwise it sends inline JSON.
func writeJSONError(w http.ResponseWriter, r *http.Request, resp *jsonrpcResponse) {
	if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		writeSSEResponse(w, resp)
	} else {
		writeJSONResponse(w, resp)
	}
}
