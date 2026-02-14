package config

import (
	"strings"
)

// ConfigObject is a hierarchical configuration data structure.
// Keys are case-sensitive. Nested values are accessed via dot notation.
type ConfigObject map[string]interface{}

// Config manages hierarchical configuration with multi-source merging.
// It is safe for concurrent reads after construction but NOT safe for
// concurrent writes via Set.
type Config struct {
	data ConfigObject
}

// NewConfig creates a Config by deep merging zero or more ConfigObjects.
// Later objects override earlier ones. Map keys merge recursively,
// slices concatenate, primitives are replaced.
//
// Example:
//
//	base := config.ConfigObject{"timeout": 10, "verbose": false}
//	override := config.ConfigObject{"timeout": 30}
//	cfg := config.NewConfig(base, override)
//	cfg.Get("timeout", 10) // returns 30
//	cfg.Get("verbose", false) // returns false
func NewConfig(objs ...ConfigObject) *Config {
	result := make(ConfigObject)
	for _, obj := range objs {
		result = DeepMerge(result, obj)
	}
	return &Config{data: result}
}

// Get retrieves a value by key using dot notation.
// Returns fallback if key is missing or if multiple fallbacks are provided,
// returns the first one.
//
// Dot notation examples:
//
//	cfg.Get("timeout", 10)           // top-level key
//	cfg.Get("database.host", "localhost") // nested key
//
// Type assertion is the caller's responsibility:
//
//	timeout := cfg.Get("timeout", 30).(int)
func (c *Config) Get(key string, fallback ...interface{}) interface{} {
	if c == nil || c.data == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return nil
	}

	parts := strings.Split(key, ".")
	current := interface{}(c.data)

	for _, part := range parts {
		// Try to convert current to a map
		var m map[string]interface{}
		switch v := current.(type) {
		case map[string]interface{}:
			m = v
		case ConfigObject:
			m = map[string]interface{}(v)
		default:
			if len(fallback) > 0 {
				return fallback[0]
			}
			return nil
		}
		value, exists := m[part]
		if !exists {
			if len(fallback) > 0 {
				return fallback[0]
			}
			return nil
		}
		current = value
	}

	return current
}

// Set stores a value at the specified key path, creating nested maps
// as needed. Uses dot notation for path segments.
//
// Example:
//
//	cfg.Set("database.host", "localhost")
//	// creates: {"database": {"host": "localhost"}}
func (c *Config) Set(key string, value interface{}) {
	if c == nil {
		return
	}
	if c.data == nil {
		c.data = make(ConfigObject)
	}

	parts := strings.Split(key, ".")
	current := c.data

	// Walk/create nested maps for all segments except the last
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		next, exists := current[part]
		if !exists {
			next = make(map[string]interface{})
			current[part] = next
		}
		nextMap, ok := next.(map[string]interface{})
		if !ok {
			// Type conflict: replace with map
			nextMap = make(map[string]interface{})
			current[part] = nextMap
		}
		current = nextMap
	}

	// Set the final segment
	current[parts[len(parts)-1]] = value
}
