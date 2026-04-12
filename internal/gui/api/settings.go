package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
)

// Settings represents the global cure user settings.
// Project-specific settings (tokens, timeout, tracker) belong in project config.
type Settings struct {
	WorkDir         string `json:"workdir"`
	DefaultProvider string `json:"default_provider"`
	DefaultModel    string `json:"default_model"`
}

func settingsPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cure", "config.json")
}

func settingsGetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Defaults
		s := Settings{
			DefaultProvider: "claude-code",
			DefaultModel:   "claude-sonnet-4-6",
		}

		home, _ := os.UserHomeDir()
		s.WorkDir = filepath.Join(home, ".cure", "workdir")

		// Try loading saved settings
		path := settingsPath()
		if data, err := os.ReadFile(path); err == nil {
			json.Unmarshal(data, &s)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(s)
	}
}

func settingsPutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var s Settings
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			writeError(w, http.StatusBadRequest, "invalid settings")
			return
		}

		path := settingsPath()
		dir := filepath.Dir(path)
		os.MkdirAll(dir, 0700)

		data, _ := json.MarshalIndent(s, "", "  ")
		if err := os.WriteFile(path, data, 0600); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save settings")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
