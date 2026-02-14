package config

import (
	"fmt"
	"testing"
)

func BenchmarkDeepMerge_10Keys(b *testing.B) {
	target := make(ConfigObject, 10)
	source := make(ConfigObject, 10)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		target[key] = i
		source[key] = i * 2
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DeepMerge(target, source)
	}
}

func BenchmarkDeepMerge_100Keys(b *testing.B) {
	target := make(ConfigObject, 100)
	source := make(ConfigObject, 100)
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		target[key] = i
		source[key] = i * 2
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DeepMerge(target, source)
	}
}

func BenchmarkDeepMerge_1000Keys(b *testing.B) {
	target := make(ConfigObject, 1000)
	source := make(ConfigObject, 1000)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("key%d", i)
		target[key] = i
		source[key] = i * 2
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DeepMerge(target, source)
	}
}

func BenchmarkGet_Nested5Levels(b *testing.B) {
	cfg := NewConfig(ConfigObject{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": map[string]interface{}{
					"d": map[string]interface{}{
						"e": "value",
					},
				},
			},
		},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.Get("a.b.c.d.e", "fallback")
	}
}

func BenchmarkSet_Nested5Levels(b *testing.B) {
	cfg := NewConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg.Set("a.b.c.d.e", "value")
	}
}

func BenchmarkEnvironment_100Vars(b *testing.B) {
	// Note: This benchmark doesn't actually set env vars,
	// it just measures the Environment() function performance
	// with whatever env vars exist at runtime.
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Environment("CURE_", "_")
	}
}

func BenchmarkNewConfig_MultiSource(b *testing.B) {
	defaults := ConfigObject{
		"timeout": 30,
		"format":  "json",
		"verbose": false,
	}
	override1 := ConfigObject{
		"timeout": 60,
		"database": map[string]interface{}{
			"host": "localhost",
		},
	}
	override2 := ConfigObject{
		"database": map[string]interface{}{
			"port": 5432,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewConfig(defaults, override1, override2)
	}
}
