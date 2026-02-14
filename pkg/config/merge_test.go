package config

import (
	"fmt"
	"testing"
)

// configEqual compares two ConfigObjects recursively
func configEqual(a, b ConfigObject) bool {
	if len(a) != len(b) {
		return false
	}
	for key, aVal := range a {
		bVal, exists := b[key]
		if !exists {
			return false
		}
		if !valuesEqual(aVal, bVal) {
			return false
		}
	}
	return true
}

func valuesEqual(a, b interface{}) bool {
	// Handle maps
	aMap, aIsMap := toMap(a)
	bMap, bIsMap := toMap(b)
	if aIsMap && bIsMap {
		return configEqual(aMap, bMap)
	}
	if aIsMap != bIsMap {
		return false
	}

	// Handle slices
	aSlice, aIsSlice := a.([]interface{})
	bSlice, bIsSlice := b.([]interface{})
	if aIsSlice && bIsSlice {
		if len(aSlice) != len(bSlice) {
			return false
		}
		for i := range aSlice {
			if !valuesEqual(aSlice[i], bSlice[i]) {
				return false
			}
		}
		return true
	}
	if aIsSlice != bIsSlice {
		return false
	}

	// Primitives
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func toMap(v interface{}) (ConfigObject, bool) {
	switch m := v.(type) {
	case ConfigObject:
		return m, true
	case map[string]interface{}:
		return ConfigObject(m), true
	default:
		return nil, false
	}
}

func TestDeepMerge(t *testing.T) {
	tests := []struct {
		name   string
		target ConfigObject
		source ConfigObject
		want   ConfigObject
	}{
		{
			name:   "nil source",
			target: ConfigObject{"a": 1},
			source: nil,
			want:   ConfigObject{"a": 1},
		},
		{
			name:   "nil target",
			target: nil,
			source: ConfigObject{"a": 1},
			want:   ConfigObject{"a": 1},
		},
		{
			name:   "both nil",
			target: nil,
			source: nil,
			want:   ConfigObject{},
		},
		{
			name:   "non-overlapping keys",
			target: ConfigObject{"a": 1},
			source: ConfigObject{"b": 2},
			want:   ConfigObject{"a": 1, "b": 2},
		},
		{
			name:   "overlapping primitives",
			target: ConfigObject{"a": 1},
			source: ConfigObject{"a": 2},
			want:   ConfigObject{"a": 2},
		},
		{
			name: "merge maps",
			target: ConfigObject{
				"db": map[string]interface{}{"host": "localhost"},
			},
			source: ConfigObject{
				"db": map[string]interface{}{"port": 5432},
			},
			want: ConfigObject{
				"db": map[string]interface{}{
					"host": "localhost",
					"port": 5432,
				},
			},
		},
		{
			name: "merge nested maps",
			target: ConfigObject{
				"a": map[string]interface{}{
					"b": map[string]interface{}{"c": 1},
				},
			},
			source: ConfigObject{
				"a": map[string]interface{}{
					"b": map[string]interface{}{"d": 2},
				},
			},
			want: ConfigObject{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": 1,
						"d": 2,
					},
				},
			},
		},
		{
			name: "concatenate slices",
			target: ConfigObject{
				"items": []interface{}{1, 2},
			},
			source: ConfigObject{
				"items": []interface{}{3, 4},
			},
			want: ConfigObject{
				"items": []interface{}{1, 2, 3, 4},
			},
		},
		{
			name: "replace primitive with map",
			target: ConfigObject{
				"value": "string",
			},
			source: ConfigObject{
				"value": map[string]interface{}{"nested": "new"},
			},
			want: ConfigObject{
				"value": map[string]interface{}{"nested": "new"},
			},
		},
		{
			name: "replace map with primitive",
			target: ConfigObject{
				"value": map[string]interface{}{"nested": "old"},
			},
			source: ConfigObject{
				"value": "string",
			},
			want: ConfigObject{
				"value": "string",
			},
		},
		{
			name: "replace slice with primitive",
			target: ConfigObject{
				"value": []interface{}{1, 2},
			},
			source: ConfigObject{
				"value": "string",
			},
			want: ConfigObject{
				"value": "string",
			},
		},
		{
			name: "complex merge",
			target: ConfigObject{
				"timeout": 10,
				"database": map[string]interface{}{
					"host": "localhost",
					"port": 5432,
				},
				"tags": []interface{}{"dev"},
			},
			source: ConfigObject{
				"timeout": 30,
				"database": map[string]interface{}{
					"port": 3306,
					"user": "admin",
				},
				"tags":    []interface{}{"prod"},
				"verbose": true,
			},
			want: ConfigObject{
				"timeout": 30,
				"database": map[string]interface{}{
					"host": "localhost",
					"port": 3306,
					"user": "admin",
				},
				"tags":    []interface{}{"dev", "prod"},
				"verbose": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeepMerge(tt.target, tt.source)
			if !configEqual(got, tt.want) {
				t.Errorf("DeepMerge() = %v, want %v", got, tt.want)
			}
		})
	}
}
