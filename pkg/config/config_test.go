package config

import (
	"testing"
)

func TestNewConfig(t *testing.T) {
	tests := []struct {
		name     string
		objs     []ConfigObject
		wantKeys []string
	}{
		{
			name:     "empty config",
			objs:     nil,
			wantKeys: nil,
		},
		{
			name: "single object",
			objs: []ConfigObject{
				{"timeout": 30, "verbose": false},
			},
			wantKeys: []string{"timeout", "verbose"},
		},
		{
			name: "multiple objects with override",
			objs: []ConfigObject{
				{"timeout": 10, "verbose": false},
				{"timeout": 30},
			},
			wantKeys: []string{"timeout", "verbose"},
		},
		{
			name: "nested merge",
			objs: []ConfigObject{
				{"db": map[string]interface{}{"host": "localhost"}},
				{"db": map[string]interface{}{"port": 5432}},
			},
			wantKeys: []string{"db"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig(tt.objs...)
			if cfg == nil {
				t.Fatal("NewConfig returned nil")
			}
			if len(tt.wantKeys) == 0 {
				if len(cfg.data) != 0 {
					t.Errorf("expected empty config, got %d keys", len(cfg.data))
				}
				return
			}
			for _, key := range tt.wantKeys {
				if _, exists := cfg.data[key]; !exists {
					t.Errorf("expected key %q in config", key)
				}
			}
		})
	}
}

func TestConfig_Get(t *testing.T) {
	cfg := NewConfig(ConfigObject{
		"timeout": 30,
		"verbose": false,
		"database": map[string]interface{}{
			"host": "localhost",
			"port": 5432,
			"nested": map[string]interface{}{
				"deep": "value",
			},
		},
	})

	tests := []struct {
		name     string
		key      string
		fallback interface{}
		want     interface{}
	}{
		{
			name:     "top-level int",
			key:      "timeout",
			fallback: 10,
			want:     30,
		},
		{
			name:     "top-level bool",
			key:      "verbose",
			fallback: true,
			want:     false,
		},
		{
			name:     "nested string",
			key:      "database.host",
			fallback: "default",
			want:     "localhost",
		},
		{
			name:     "nested int",
			key:      "database.port",
			fallback: 3306,
			want:     5432,
		},
		{
			name:     "deep nested",
			key:      "database.nested.deep",
			fallback: "fallback",
			want:     "value",
		},
		{
			name:     "missing top-level",
			key:      "missing",
			fallback: "default",
			want:     "default",
		},
		{
			name:     "missing nested",
			key:      "database.missing",
			fallback: "default",
			want:     "default",
		},
		{
			name:     "missing deep nested",
			key:      "a.b.c.d",
			fallback: "default",
			want:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.Get(tt.key, tt.fallback)
			if got != tt.want {
				t.Errorf("Get(%q, %v) = %v, want %v", tt.key, tt.fallback, got, tt.want)
			}
		})
	}
}

func TestConfig_Get_Nil(t *testing.T) {
	var cfg *Config
	got := cfg.Get("key", "fallback")
	if got != "fallback" {
		t.Errorf("nil Config Get() = %v, want fallback", got)
	}
}

func TestConfig_Set(t *testing.T) {
	tests := []struct {
		name  string
		ops   func(*Config)
		check func(*Config) bool
	}{
		{
			name: "set top-level",
			ops: func(c *Config) {
				c.Set("timeout", 30)
			},
			check: func(c *Config) bool {
				return c.Get("timeout", 0) == 30
			},
		},
		{
			name: "set nested",
			ops: func(c *Config) {
				c.Set("database.host", "localhost")
			},
			check: func(c *Config) bool {
				return c.Get("database.host", "") == "localhost"
			},
		},
		{
			name: "set deep nested",
			ops: func(c *Config) {
				c.Set("a.b.c.d", "value")
			},
			check: func(c *Config) bool {
				return c.Get("a.b.c.d", "") == "value"
			},
		},
		{
			name: "overwrite existing",
			ops: func(c *Config) {
				c.Set("timeout", 10)
				c.Set("timeout", 30)
			},
			check: func(c *Config) bool {
				return c.Get("timeout", 0) == 30
			},
		},
		{
			name: "replace primitive with map",
			ops: func(c *Config) {
				c.Set("value", "string")
				c.Set("value.nested", "new")
			},
			check: func(c *Config) bool {
				return c.Get("value.nested", "") == "new"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig()
			tt.ops(cfg)
			if !tt.check(cfg) {
				t.Errorf("check failed")
			}
		})
	}
}

func TestConfig_Set_Nil(t *testing.T) {
	var cfg *Config
	cfg.Set("key", "value") // Should not panic
}
