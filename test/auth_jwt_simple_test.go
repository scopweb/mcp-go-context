package test

import (
	"os"
	"testing"
	"time"

	"github.com/scopweb/mcp-go-context/internal/auth"
	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/server"
)

func TestJWTAuthSimple(t *testing.T) {
	// Set JWT secret for testing
	jwtSecret := "test-secret-key-for-jwt-authentication-testing"
	os.Setenv("MCP_JWT_SECRET", jwtSecret)
	defer os.Unsetenv("MCP_JWT_SECRET")

	t.Run("JWT Manager Creation", func(t *testing.T) {
		jwtManager := auth.NewJWTManager(auth.JWTConfig{
			Secret:    jwtSecret,
			Expiry:    time.Hour,
			Issuer:    "mcp-go-context",
			Algorithm: "HS256",
		})

		if !jwtManager.IsEnabled() {
			t.Error("JWT manager should be enabled with secret")
		}
	})

	t.Run("Token Generation and Validation", func(t *testing.T) {
		jwtManager := auth.NewJWTManager(auth.JWTConfig{
			Secret:    jwtSecret,
			Expiry:    time.Hour,
			Issuer:    "mcp-go-context",
			Algorithm: "HS256",
		})

		// Generate token
		token, err := jwtManager.GenerateToken("test-user")
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		if token == "" {
			t.Error("Generated token should not be empty")
		}

		// Validate token
		claims, err := jwtManager.ValidateToken(token)
		if err != nil {
			t.Fatalf("Failed to validate token: %v", err)
		}

		if claims.Subject != "test-user" {
			t.Errorf("Expected subject 'test-user', got '%s'", claims.Subject)
		}

		if claims.Issuer != "mcp-go-context" {
			t.Errorf("Expected issuer 'mcp-go-context', got '%s'", claims.Issuer)
		}

		// Test invalid token
		_, err = jwtManager.ValidateToken("invalid.token.format")
		if err == nil {
			t.Error("Invalid token should be rejected")
		}
	})

	t.Run("Server Configuration with JWT", func(t *testing.T) {
		// Create test configuration with JWT enabled
		cfg := config.DefaultConfig()
		cfg.Security.Auth.Enabled = true
		cfg.Security.Auth.Method = "jwt"
		cfg.Security.Auth.Secret = jwtSecret
		cfg.Security.Auth.Expiry = time.Hour
		cfg.Transport.Type = "stdio" // Use stdio to avoid port conflicts

		// Create server
		srv, err := server.New(cfg)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		if srv == nil {
			t.Error("Server should not be nil")
		}
	})

	t.Run("Header Token Extraction", func(t *testing.T) {
		testCases := []struct {
			name        string
			authHeader  string
			expectError bool
		}{
			{"Valid Bearer Token", "Bearer abc123", false},
			{"Missing Bearer Prefix", "abc123", true},
			{"Empty Header", "", true},
			{"Bearer Without Token", "Bearer ", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				token, err := auth.ExtractTokenFromHeader(tc.authHeader)
				
				if tc.expectError && err == nil {
					t.Error("Expected error but got none")
				} else if !tc.expectError && err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else if !tc.expectError && token == "" {
					t.Error("Expected token but got empty string")
				}
			})
		}
	})

	t.Run("Expired Token", func(t *testing.T) {
		expiredManager := auth.NewJWTManager(auth.JWTConfig{
			Secret:    jwtSecret,
			Expiry:    -time.Hour, // Already expired
			Issuer:    "mcp-go-context",
			Algorithm: "HS256",
		})

		// Generate expired token
		expiredToken, err := expiredManager.GenerateToken("test-user")
		if err != nil {
			t.Fatalf("Failed to generate expired token: %v", err)
		}

		// Try to validate with normal manager
		normalManager := auth.NewJWTManager(auth.JWTConfig{
			Secret:    jwtSecret,
			Expiry:    time.Hour,
			Issuer:    "mcp-go-context",
			Algorithm: "HS256",
		})

		_, err = normalManager.ValidateToken(expiredToken)
		if err == nil {
			t.Error("Expected expired token to be rejected")
		}
	})
}