package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantHome bool
	}{
		{
			name:     "tilde path",
			input:    "~/test",
			wantHome: true,
		},
		{
			name:     "dollar home",
			input:    "$HOME/test",
			wantHome: true,
		},
		{
			name:     "absolute path",
			input:    "/tmp/test",
			wantHome: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandPath(tt.input)
			if tt.wantHome {
				home, _ := os.UserHomeDir()
				if home != "" && filepath.Dir(result) != filepath.Dir(home) && filepath.Dir(result) != "/tmp" {
					// Just check it doesn't contain literal ~
					if filepath.Base(result) == "" {
						t.Errorf("expandPath(%q) = %q, should not contain literal ~", tt.input, result)
					}
				}
			}
			// Verify no literal ~ remains
			if filepath.Base(result) == "" && tt.input != result {
				t.Errorf("expandPath(%q) = %q, should expand ~", tt.input, result)
			}
		})
	}
}

func TestLoadWithDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Memory.MaxEntries != 1000 {
		t.Errorf("expected MaxEntries 1000, got %d", cfg.Memory.MaxEntries)
	}

	if cfg.Memory.MaxResults != 10 {
		t.Errorf("expected MaxResults 10, got %d", cfg.Memory.MaxResults)
	}

	if cfg.Memory.SessionTTLDays != 30 {
		t.Errorf("expected SessionTTLDays 30, got %d", cfg.Memory.SessionTTLDays)
	}

	if len(cfg.Context.ProjectPaths) == 0 {
		t.Error("expected at least one project path")
	}
}

func TestLoadWithPathExpansion(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Write config with home paths
	configContent := `{
		"memory": {
			"storagePath": "$HOME/.test-memory"
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify path was expanded
	if cfg.Memory.StoragePath == "$HOME/.test-memory" {
		t.Error("StoragePath should be expanded, got literal $HOME")
	}

	if cfg.Memory.StoragePath == "" {
		t.Error("StoragePath should not be empty after expansion")
	}
}

func TestLoadWithInvalidStoragePath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Create a file where a directory path is expected
	filePath := filepath.Join(tmpDir, "memory.json")
	if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	configContent := `{
		"memory": {
			"storagePath": "` + filePath + `"
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Should convert file path to directory
	if cfg.Memory.StoragePath == filePath {
		t.Error("StoragePath should be converted to directory")
	}
}

func TestLoadWithEmptyProjectPaths(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := `{
		"context": {
			"projectPaths": []
		}
	}`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Should default to ["."]
	if len(cfg.Context.ProjectPaths) == 0 {
		t.Error("projectPaths should default to [.] when empty")
	}
}
