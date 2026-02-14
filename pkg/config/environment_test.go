package config

import (
	"os"
	"reflect"
	"testing"
)

func TestEnvironment(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		separator string
		envVars   map[string]string
		want      ConfigObject
	}{
		{
			name:      "simple prefix filter",
			prefix:    "CURE_",
			separator: "_",
			envVars: map[string]string{
				"CURE_TIMEOUT": "30",
				"CURE_VERBOSE": "true",
				"OTHER_VAR":    "ignored",
			},
			want: ConfigObject{
				"timeout": "30",
				"verbose": "true",
			},
		},
		{
			name:      "nested keys",
			prefix:    "CURE_",
			separator: "_",
			envVars: map[string]string{
				"CURE_DATABASE_HOST": "localhost",
				"CURE_DATABASE_PORT": "5432",
			},
			want: ConfigObject{
				"database": map[string]interface{}{
					"host": "localhost",
					"port": "5432",
				},
			},
		},
		{
			name:      "deep nested",
			prefix:    "APP_",
			separator: "_",
			envVars: map[string]string{
				"APP_A_B_C_D": "value",
			},
			want: ConfigObject{
				"a": map[string]interface{}{
					"b": map[string]interface{}{
						"c": map[string]interface{}{
							"d": "value",
						},
					},
				},
			},
		},
		{
			name:      "no matching vars",
			prefix:    "CURE_",
			separator: "_",
			envVars: map[string]string{
				"OTHER_VAR": "value",
			},
			want: ConfigObject{},
		},
		{
			name:      "empty prefix",
			prefix:    "",
			separator: "_",
			envVars: map[string]string{
				"VAR": "value",
			},
			want: ConfigObject{
				"var": "value",
			},
		},
		{
			name:      "lowercase normalization",
			prefix:    "CURE_",
			separator: "_",
			envVars: map[string]string{
				"CURE_UPPER_CASE": "value",
			},
			want: ConfigObject{
				"upper": map[string]interface{}{
					"case": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer os.Clearenv()

			got := Environment(tt.prefix, tt.separator)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Environment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnvironment_Integration(t *testing.T) {
	os.Clearenv()
	defer os.Clearenv()

	os.Setenv("CURE_TIMEOUT", "60")
	os.Setenv("CURE_FORMAT", "json")
	os.Setenv("CURE_TRACE_REDACT", "false")

	cfg := Environment("CURE_", "_")

	if cfg["timeout"] != "60" {
		t.Errorf("timeout = %v, want 60", cfg["timeout"])
	}
	if cfg["format"] != "json" {
		t.Errorf("format = %v, want json", cfg["format"])
	}
	trace, ok := cfg["trace"].(map[string]interface{})
	if !ok {
		t.Fatal("trace is not a map")
	}
	if trace["redact"] != "false" {
		t.Errorf("trace.redact = %v, want false", trace["redact"])
	}
}
