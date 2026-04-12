package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

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
func sessionsCreateHandler(store agent.SessionStore, cfg configDefaults, projectName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateSessionRequest
		// Allow empty body — all fields are optional with defaults.
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
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
		sess.Name = agent.DefaultName(provider, sess.ID)
		// Use project from request body if provided, otherwise fall back to auto-detected.
		if req.ProjectName != "" {
			sess.ProjectName = req.ProjectName
		} else {
			sess.ProjectName = projectName
		}

		// Set container target if specified
		if req.ContainerID != "" {
			sess.ContainerID = req.ContainerID
			sess.Mode = "autonomous"
		} else {
			sess.Mode = "interactive"
		}

		// Auto-populate git context and create session branch
		if cwd, err := os.Getwd(); err == nil {
			if branch, err := gitCurrentBranch(cwd); err == nil {
				sess.BranchName = branch
			}
			if dirty, err := gitIsDirty(cwd); err == nil {
				sess.GitDirty = dirty
			}
			sess.RepoName = repoNameFromCwd(cwd)

			// Create isolated branch for this session if on a main/protected branch
			if sess.BranchName == "main" || sess.BranchName == "master" {
				sessionBranch := fmt.Sprintf("session/%s", sess.ID[:8])
				if err := gitCreateBranch(cwd, sessionBranch); err == nil {
					sess.BranchName = sessionBranch
				}
			}
		}

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
	name := s.Name
	if name == "" {
		name = agent.DefaultName(s.Provider, s.ID)
	}
	return SessionSummary{
		ID:          s.ID,
		Provider:    s.Provider,
		Model:       s.Model,
		Tags:        tags,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
		ForkOf:      s.ForkOf,
		Turns:       len(s.History),
		Name:        name,
		ProjectName: s.ProjectName,
		BranchName:  s.BranchName,
		RepoName:    s.RepoName,
		WorkItems:   s.WorkItems,
		AgentRole:   s.AgentRole,
		SkillName:   s.SkillName,
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

func gitCurrentBranch(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func gitIsDirty(dir string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func gitCreateBranch(dir, name string) error {
	cmd := exec.Command("git", "checkout", "-b", name)
	cmd.Dir = dir
	return cmd.Run()
}

func repoNameFromCwd(cwd string) string {
	parts := strings.Split(cwd, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return cwd
}
