package config

import (
	"os"
	"strings"
)

// Environment loads configuration from environment variables.
// Filters variables by prefix and converts them to nested ConfigObject
// using separator as the nesting delimiter.
//
// Example:
//
//	os.Setenv("CURE_DATABASE_HOST", "localhost")
//	os.Setenv("CURE_TIMEOUT", "30")
//	cfg := Environment("CURE_", "_")
//	// returns: {"database": {"host": "localhost"}, "timeout": "30"}
//
// Keys are normalized to lowercase after prefix stripping.
func Environment(prefix, separator string) ConfigObject {
	result := make(ConfigObject)

	for _, envVar := range os.Environ() {
		// Split on first '='
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Filter by prefix
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		// Strip prefix and convert to lowercase
		key = strings.TrimPrefix(key, prefix)
		key = strings.ToLower(key)

		// Convert separator to dot notation
		if separator != "" && separator != "." {
			key = strings.ReplaceAll(key, separator, ".")
		}

		// Build nested structure
		if key == "" {
			continue
		}

		// Use a temporary Config to leverage Set logic
		temp := &Config{data: result}
		temp.Set(key, value)
		result = temp.data
	}

	return result
}
