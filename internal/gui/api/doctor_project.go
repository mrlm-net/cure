package api

import (
	"encoding/json"
	"net/http"

	"github.com/mrlm-net/cure/pkg/doctor"
	"github.com/mrlm-net/cure/pkg/doctor/stack"
	"github.com/mrlm-net/cure/pkg/project"
)

// projectDoctorHandler runs stack-detected health checks across all project repos.
func projectDoctorHandler(store project.ProjectStore, projectName string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p, err := store.Load(projectName)
		if err != nil {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}

		var allResults []CheckResultResponse
		for _, repo := range p.Repos {
			dir := repo.EffectivePath()
			stacks := stack.DetectStacks(dir)
			for _, s := range stacks {
				for _, check := range s.Checks() {
					result := check()
					allResults = append(allResults, CheckResultResponse{
						Name:    "[" + s.Name + "] " + result.Name,
						Status:  statusString(result.Status),
						Message: result.Message,
					})
				}
			}
		}

		if allResults == nil {
			allResults = []CheckResultResponse{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(allResults)
	}
}

func statusString(s doctor.CheckStatus) string {
	switch s {
	case doctor.CheckPass:
		return "pass"
	case doctor.CheckWarn:
		return "warn"
	case doctor.CheckFail:
		return "fail"
	default:
		return "unknown"
	}
}
