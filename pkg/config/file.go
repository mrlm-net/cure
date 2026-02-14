package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// File loads a JSON configuration file and returns the parsed ConfigObject.
// Returns an error if the file cannot be read or parsed.
//
// Supports tilde expansion for home directory paths.
//
// Example:
//
//	cfg, err := File("~/.cure.json")
//	if err != nil {
//	    // handle error
//	}
func File(path string) (ConfigObject, error) {
	// Expand tilde to home directory
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot determine home directory: %w", err)
		}
		path = filepath.Clean(filepath.Join(homeDir, path[1:]))
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err // Caller can check with os.IsNotExist
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	// Parse JSON
	var result ConfigObject
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", path, err)
	}

	return result, nil
}
