package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the server configuration
type Config struct {
	Transport TransportConfig `json:"transport"`
	Context   ContextConfig   `json:"context"`
	Cache     CacheConfig     `json:"cache"`
	Memory    MemoryConfig    `json:"memory"`
}

// TransportConfig defines transport settings
type TransportConfig struct {
	Type string `json:"type"` // stdio, http, sse
	Port int    `json:"port"`
}

// ContextConfig defines context analysis settings
type ContextConfig struct {
	MaxTokens         int      `json:"maxTokens"`
	DefaultLibraries  []string `json:"defaultLibraries"`
	ProjectPaths      []string `json:"projectPaths"`
	IgnorePatterns    []string `json:"ignorePatterns"`
	AutoDetectDeps    bool     `json:"autoDetectDeps"`
	ContextWindowSize int      `json:"contextWindowSize"`
}

// CacheConfig defines caching settings
type CacheConfig struct {
	Enabled    bool   `json:"enabled"`
	Directory  string `json:"directory"`
	MaxSizeMB  int    `json:"maxSizeMB"`
	TTLMinutes int    `json:"ttlMinutes"`
}

// MemoryConfig defines conversation memory settings
type MemoryConfig struct {
	Enabled        bool   `json:"enabled"`
	Persistent     bool   `json:"persistent"`
	StoragePath    string `json:"storagePath"`
	MaxEntries     int    `json:"maxEntries"`
	MaxResults     int    `json:"maxResults"`
	SessionTTLDays int    `json:"sessionTTLDays"`
	MaxSessions    int    `json:"maxSessions"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()

	return &Config{
		Transport: TransportConfig{
			Type: "stdio",
			Port: 3000,
		},
		Context: ContextConfig{
			MaxTokens:         10000,
			DefaultLibraries:  []string{},
			ProjectPaths:      []string{"."},
			IgnorePatterns:    []string{"*.log", "*.tmp", "node_modules", ".git", "vendor"},
			AutoDetectDeps:    true,
			ContextWindowSize: 5000,
		},
		Cache: CacheConfig{
			Enabled:    true,
			Directory:  filepath.Join(homeDir, ".mcp-go-context", "cache"),
			MaxSizeMB:  500,
			TTLMinutes: 1440, // 24 hours
		},
		Memory: MemoryConfig{
			Enabled:        true,
			Persistent:     true,
			StoragePath:    filepath.Join(homeDir, ".mcp-go-context", "memory"),
			MaxEntries:     1000,
			MaxResults:     10,
			SessionTTLDays: 30,
			MaxSessions:    100,
		},
	}
}

// Load loads configuration from file or returns default
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Expand environment variables and ~ in paths
	cfg.Memory.StoragePath = expandPath(cfg.Memory.StoragePath)
	cfg.Cache.Directory = expandPath(cfg.Cache.Directory)

	// Validate and fix StoragePath if it appears to be a file path
	if cfg.Memory.StoragePath != "" {
		if info, err := os.Stat(cfg.Memory.StoragePath); err == nil && !info.IsDir() {
			// It's a file, not a directory - convert to directory by taking parent
			cfg.Memory.StoragePath = filepath.Dir(cfg.Memory.StoragePath)
		}
	}

	// Ensure at least one project path
	if len(cfg.Context.ProjectPaths) == 0 {
		cfg.Context.ProjectPaths = []string{"."}
	}

	return cfg, nil
}

// expandPath expands ~ to home directory and environment variables
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		if homeDir, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(homeDir, path[2:])
		}
	}
	// Also expand $HOME style variables
	path = os.ExpandEnv(path)
	return path
}

// Save saves configuration to file
func (c *Config) Save(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// GetProjectPaths returns the configured project paths
func (c *Config) GetProjectPaths() []string {
	return c.Context.ProjectPaths
}
