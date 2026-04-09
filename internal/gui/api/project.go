package api

import (
	"encoding/json"
	"errors"
	"net/http"

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
