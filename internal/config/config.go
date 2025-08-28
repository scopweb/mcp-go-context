package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Config represents the server configuration
type Config struct {
	Transport TransportConfig `json:"transport"`
	Context   ContextConfig   `json:"context"`
	Cache     CacheConfig     `json:"cache"`
	Memory    MemoryConfig    `json:"memory"`
	Security  SecurityConfig  `json:"security"`
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

// SecurityConfig defines security settings
type SecurityConfig struct {
	Auth      AuthConfig      `json:"auth"`
	CORS      CORSConfig      `json:"cors"`
	RateLimit RateLimitConfig `json:"rateLimit"`
	TLS       TLSConfig       `json:"tls"`
}

// AuthConfig defines authentication settings
type AuthConfig struct {
	Enabled   bool          `json:"enabled"`
	Method    string        `json:"method"` // "jwt" or "token"
	Secret    string        `json:"secret"`
	Expiry    time.Duration `json:"expiry"`
	Issuer    string        `json:"issuer"`
	Algorithm string        `json:"algorithm"`
}

// CORSConfig defines CORS settings
type CORSConfig struct {
	Enabled bool     `json:"enabled"`
	Origins []string `json:"origins"`
	Methods []string `json:"methods"`
	Headers []string `json:"headers"`
}

// RateLimitConfig defines rate limiting settings
type RateLimitConfig struct {
	Enabled  bool          `json:"enabled"`
	Requests int           `json:"requests"`
	Window   time.Duration `json:"window"`
}

// TLSConfig defines TLS settings
type TLSConfig struct {
	Enabled  bool   `json:"enabled"`
	CertFile string `json:"certFile"`
	KeyFile  string `json:"keyFile"`
	Required bool   `json:"required"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	currentDir, _ := os.Getwd()

	return &Config{
		Transport: TransportConfig{
			Type: "stdio",
			Port: 3000,
		},
		Context: ContextConfig{
			MaxTokens:         10000,
			DefaultLibraries:  []string{},
			ProjectPaths:      []string{currentDir},
			IgnorePatterns:    []string{"*.log", "*.tmp", "node_modules", ".git", "vendor", "bin", "dist", "target"},
			AutoDetectDeps:    true,
			ContextWindowSize: 5000,
		},
		Cache: CacheConfig{
			Enabled:    true,
			Directory:  filepath.Join(homeDir, ".mcp-context", "cache"),
			MaxSizeMB:  500,
			TTLMinutes: 1440, // 24 hours
		},
		Memory: MemoryConfig{
			Enabled:        true,
			Persistent:     true,
			StoragePath:    filepath.Join(homeDir, ".mcp-context", "memory.json"),
			MaxEntries:     1000,
			MaxResults:     10,
			SessionTTLDays: 30,
			MaxSessions:    100,
		},
		Security: SecurityConfig{
			Auth: AuthConfig{
				Enabled:   false,
				Method:    "jwt",
				Secret:    "", // Must be set via environment variable
				Expiry:    time.Hour,
				Issuer:    "mcp-go-context",
				Algorithm: "HS256",
			},
			CORS: CORSConfig{
				Enabled: true,
				Origins: []string{"https://localhost:3000", "app://claude-desktop"},
				Methods: []string{"POST", "OPTIONS"},
				Headers: []string{"Content-Type", "Authorization"},
			},
			RateLimit: RateLimitConfig{
				Enabled:  false,
				Requests: 100,
				Window:   time.Minute,
			},
			TLS: TLSConfig{
				Enabled:  false,
				CertFile: "cert.pem",
				KeyFile:  "key.pem",
				Required: false,
			},
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

	return cfg, nil
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
