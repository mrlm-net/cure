package api

import (
	"encoding/json"
	"net/http"

	"github.com/mrlm-net/cure/pkg/config"
)

// configHandler returns a handler that serializes the merged application
// configuration as JSON.
func configHandler(cfg config.ConfigObject) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		data := cfg
		if data == nil {
			data = config.ConfigObject{}
		}
		_ = json.NewEncoder(w).Encode(data)
	}
}
