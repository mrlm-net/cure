package api

import (
	"encoding/json"
	"net/http"
)

// healthHandler returns a handler that reports the server's health status
// and the port it is listening on.
func healthHandler(port int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"port":   port,
		})
	}
}
