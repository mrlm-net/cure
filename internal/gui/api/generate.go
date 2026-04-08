package api

import (
	"encoding/json"
	"net/http"
)

// generateListHandler returns 501 until the generate engine is available (issue #89).
func generateListHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Error: "generate engine not yet available",
		})
	}
}

// generateRunHandler returns 501 until the generate engine is available (issue #89).
func generateRunHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		_ = json.NewEncoder(w).Encode(ErrorResponse{
			Error: "generate engine not yet available",
		})
	}
}
