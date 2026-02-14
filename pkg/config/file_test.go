package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFile(t *testing.T) {
	// Create temp directory for test files
	tmpDir := t.TempDir()

	// Create valid JSON file
	validPath := filepath.Join(tmpDir, "valid.json")
	validContent := `{"timeout": 30, "verbose": true, "database": {"host": "localhost"}}`
	if err := os.WriteFile(validPath, []byte(validContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create invalid JSON file
	invalidPath := filepath.Join(tmpDir, "invalid.json")
	invalidContent := `{invalid json`
	if err := os.WriteFile(invalidPath, []byte(invalidContent), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
		check   func(ConfigObject) bool
	}{
		{
			name:    "valid JSON",
			path:    validPath,
			wantErr: false,
			check: func(cfg ConfigObject) bool {
				return cfg["timeout"].(float64) == 30 &&
					cfg["verbose"].(bool) == true
			},
		},
		{
			name:    "file not found",
			path:    filepath.Join(tmpDir, "missing.json"),
			wantErr: true,
			check:   nil,
		},
		{
			name:    "invalid JSON",
			path:    invalidPath,
			wantErr: true,
			check:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := File(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("File() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if tt.check != nil && !tt.check(got) {
				t.Errorf("File() check failed for %v", got)
			}
		})
	}
}

func TestFile_TildeExpansion(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	// Create temp file in home directory
	testFile := filepath.Join(homeDir, ".cure_test.json")
	content := `{"test": true}`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(testFile)

	// Test with tilde
	cfg, err := File("~/.cure_test.json")
	if err != nil {
		t.Fatalf("File() error = %v", err)
	}
	if cfg["test"].(bool) != true {
		t.Errorf("tilde expansion failed")
	}
}

func TestFile_NotFoundIsExist(t *testing.T) {
	_, err := File("/nonexistent/path/file.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("error should be IsNotExist, got %v", err)
	}
}
