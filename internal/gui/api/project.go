package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"

	"github.com/mrlm-net/cure/pkg/project"
)

// projectListHandler returns all registered projects.
func projectListHandler(store project.ProjectStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projects, err := store.List()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to list projects")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(projects)
	}
}

// projectCreateHandler creates a new project.
func projectCreateHandler(store project.ProjectStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var p project.Project
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if err := project.ValidateName(p.Name); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := store.Save(&p); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create project")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(p)
	}
}

// projectGetHandler returns a single project by name.
func projectGetHandler(store project.ProjectStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		p, err := store.Load(name)
		if err != nil {
			if errors.Is(err, project.ErrNotFound) {
				writeError(w, http.StatusNotFound, "project not found")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to load project")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(p)
	}
}

// projectUpdateHandler updates a project's configuration.
func projectUpdateHandler(store project.ProjectStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")

		var p project.Project
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		p.Name = name // ensure name matches URL

		if err := store.Save(&p); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save project")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(p)
	}
}

// configUpdateHandler updates a specific config layer.
func configUpdateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// For now, only support writing the local .cure.json
		body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to read body")
			return
		}

		// Validate it's valid JSON
		var check json.RawMessage
		if err := json.Unmarshal(body, &check); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		if err := os.WriteFile(".cure.json", body, 0644); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to write config")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
