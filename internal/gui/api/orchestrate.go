package api

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/mrlm-net/cure/internal/orchestrator"
	"github.com/mrlm-net/cure/pkg/project"
)

func orchestrateStatusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, _ := os.Getwd()

		orch := orchestrator.New(&project.Project{}, cwd)
		statuses, err := orch.Status(r.Context())
		if err != nil {
			// No containers running or docker not available
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]any{})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(statuses)
	}
}

func orchestrateInitHandler(store project.ProjectStore, projectName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if store == nil || projectName == "" {
			writeError(w, http.StatusBadRequest, "no project detected")
			return
		}

		p, err := store.Load(projectName)
		if err != nil {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}

		cwd, _ := os.Getwd()
		orch := orchestrator.New(p, cwd)
		if err := orch.Init(); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "initialized",
			"message": "docker-compose.cure.yml and Dockerfile.cure generated",
		})
	}
}
