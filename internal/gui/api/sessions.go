package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/mrlm-net/cure/pkg/agent"
)

// sessionsListHandler returns all sessions ordered by UpdatedAt descending.
// Returns an empty JSON array (not null) when no sessions exist.
func sessionsListHandler(store agent.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions, err := store.List(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list sessions")
			return
		}

		summaries := make([]SessionSummary, 0, len(sessions))
		for _, s := range sessions {
			summaries = append(summaries, sessionToSummary(s))
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(summaries)
	}
}

// sessionsCreateHandler creates a new session. Provider and model fall back
// to config defaults when omitted from the request body.
func sessionsCreateHandler(store agent.SessionStore, cfg configDefaults) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		provider := req.Provider
		if provider == "" {
			provider = cfg.defaultProvider
		}
		model := req.Model
		if model == "" {
			model = cfg.defaultModel
		}

		sess := agent.NewSession(provider, model)
		if err := store.Save(r.Context(), sess); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save session")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(sessionToDetail(sess))
	}
}

// sessionsGetHandler returns a single session with full history.
func sessionsGetHandler(store agent.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		sess, err := store.Load(r.Context(), id)
		if err != nil {
			if errors.Is(err, agent.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to load session")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(sessionToDetail(sess))
	}
}

// sessionsDeleteHandler deletes a session by ID. Returns 204 on success,
// 404 when the session does not exist.
func sessionsDeleteHandler(store agent.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if err := store.Delete(r.Context(), id); err != nil {
			if errors.Is(err, agent.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to delete session")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// sessionsForkHandler forks a session by ID. Returns 201 with the new session.
func sessionsForkHandler(store agent.SessionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		forked, err := store.Fork(r.Context(), id)
		if err != nil {
			if errors.Is(err, agent.ErrSessionNotFound) {
				writeError(w, http.StatusNotFound, "session not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to fork session")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(sessionToDetail(forked))
	}
}

// sessionToSummary converts an agent.Session to a SessionSummary for list responses.
func sessionToSummary(s *agent.Session) SessionSummary {
	tags := s.Tags
	if tags == nil {
		tags = []string{}
	}
	return SessionSummary{
		ID:        s.ID,
		Provider:  s.Provider,
		Model:     s.Model,
		Tags:      tags,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
		ForkOf:    s.ForkOf,
		Turns:     len(s.History),
	}
}

// sessionToDetail converts an agent.Session to a SessionDetail with full history.
func sessionToDetail(s *agent.Session) SessionDetail {
	history := make([]MessageResponse, 0, len(s.History))
	for _, m := range s.History {
		history = append(history, MessageResponse{
			Role:    string(m.Role),
			Content: agent.TextOf(m.Content),
		})
	}
	return SessionDetail{
		SessionSummary: sessionToSummary(s),
		History:        history,
	}
}

// writeError writes a standard JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}

// configDefaults holds fallback values for session creation.
type configDefaults struct {
	defaultProvider string
	defaultModel    string
}
