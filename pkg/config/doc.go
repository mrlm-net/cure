// Package config provides hierarchical configuration management with support
// for multiple sources (files, environment variables, defaults) and deep merging.
//
// # Basic Usage
//
// Create a Config by merging ConfigObjects from multiple sources:
//
//	defaults := config.ConfigObject{"timeout": 30}
//	envCfg := config.Environment("MYAPP_", "_")
//	fileCfg, _ := config.File("~/.myapp.json")
//	cfg := config.NewConfig(defaults, fileCfg, envCfg)
//
// Access values with dot notation:
//
//	timeout := cfg.Get("timeout", 30).(int)
//	dbHost := cfg.Get("database.host", "localhost").(string)
//
// Set values:
//
//	cfg.Set("verbose", true)
//	cfg.Set("database.port", 5432)
//
// # Deep Merge Semantics
//
// Maps are merged recursively, slices are concatenated, primitives are replaced:
//
//	target := ConfigObject{"a": map[string]interface{}{"b": 1}}
//	source := ConfigObject{"a": map[string]interface{}{"c": 2}}
//	result := DeepMerge(target, source)
//	// result: {"a": {"b": 1, "c": 2}}
package config
