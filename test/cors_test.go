package test

import (
	"net/http/httptest"
	"testing"

	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/security"
)

func TestCORSMiddleware(t *testing.T) {
	testCases := []struct {
		name           string
		corsConfig     config.CORSConfig
		origin         string
		method         string
		expectAllow    bool
		expectedOrigin string
	}{
		{
			name: "CORS Disabled - Allow All",
			corsConfig: config.CORSConfig{
				Enabled: false,
			},
			origin:         "https://evil.com",
			method:         "POST",
			expectAllow:    true,
			expectedOrigin: "",
		},
		{
			name: "Allowed Origin",
			corsConfig: config.CORSConfig{
				Enabled: true,
				Origins: []string{"https://localhost:3000", "app://claude-desktop"},
				Methods: []string{"POST", "OPTIONS"},
				Headers: []string{"Content-Type", "Authorization"},
			},
			origin:         "https://localhost:3000",
			method:         "POST",
			expectAllow:    true,
			expectedOrigin: "https://localhost:3000",
		},
		{
			name: "Blocked Origin",
			corsConfig: config.CORSConfig{
				Enabled: true,
				Origins: []string{"https://localhost:3000"},
				Methods: []string{"POST", "OPTIONS"},
				Headers: []string{"Content-Type", "Authorization"},
			},
			origin:         "https://evil.com",
			method:         "POST",
			expectAllow:    false,
			expectedOrigin: "",
		},
		{
			name: "Claude Desktop Origin",
			corsConfig: config.CORSConfig{
				Enabled: true,
				Origins: []string{"app://claude-desktop"},
				Methods: []string{"POST", "OPTIONS"},
				Headers: []string{"Content-Type", "Authorization"},
			},
			origin:         "app://claude-desktop",
			method:         "POST",
			expectAllow:    true,
			expectedOrigin: "app://claude-desktop",
		},
		{
			name: "Wildcard Pattern",
			corsConfig: config.CORSConfig{
				Enabled: true,
				Origins: []string{"*.example.com"},
				Methods: []string{"POST", "OPTIONS"},
				Headers: []string{"Content-Type", "Authorization"},
			},
			origin:         "https://api.example.com",
			method:         "POST",
			expectAllow:    true,
			expectedOrigin: "https://api.example.com",
		},
		{
			name: "Preflight Request - Allowed",
			corsConfig: config.CORSConfig{
				Enabled: true,
				Origins: []string{"https://localhost:3000"},
				Methods: []string{"POST", "OPTIONS"},
				Headers: []string{"Content-Type", "Authorization"},
			},
			origin:         "https://localhost:3000",
			method:         "OPTIONS",
			expectAllow:    true,
			expectedOrigin: "https://localhost:3000",
		},
		{
			name: "Preflight Request - Blocked",
			corsConfig: config.CORSConfig{
				Enabled: true,
				Origins: []string{"https://localhost:3000"},
				Methods: []string{"POST", "OPTIONS"},
				Headers: []string{"Content-Type", "Authorization"},
			},
			origin:         "https://evil.com",
			method:         "OPTIONS",
			expectAllow:    false,
			expectedOrigin: "",
		},
		{
			name: "No Origin Header - Allow",
			corsConfig: config.CORSConfig{
				Enabled: true,
				Origins: []string{"https://localhost:3000"},
				Methods: []string{"POST", "OPTIONS"},
				Headers: []string{"Content-Type", "Authorization"},
			},
			origin:         "", // No Origin header
			method:         "POST",
			expectAllow:    true,
			expectedOrigin: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			corsMiddleware := security.NewCORSMiddleware(tc.corsConfig)
			
			req := httptest.NewRequest(tc.method, "/test", nil)
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			
			w := httptest.NewRecorder()
			
			allowed := corsMiddleware.SetHeaders(w, req)
			
			if allowed != tc.expectAllow {
				t.Errorf("Expected allow=%v, got %v", tc.expectAllow, allowed)
			}
			
			if tc.expectAllow {
				actualOrigin := w.Header().Get("Access-Control-Allow-Origin")
				if actualOrigin != tc.expectedOrigin {
					t.Errorf("Expected origin header '%s', got '%s'", tc.expectedOrigin, actualOrigin)
				}
				
				// Check preflight headers for OPTIONS requests
				if tc.method == "OPTIONS" && tc.expectedOrigin != "" {
					methods := w.Header().Get("Access-Control-Allow-Methods")
					if methods == "" {
						t.Error("Expected Access-Control-Allow-Methods header for preflight")
					}
					
					headers := w.Header().Get("Access-Control-Allow-Headers")
					if headers == "" {
						t.Error("Expected Access-Control-Allow-Headers header for preflight")
					}
				}
			}
		})
	}
}

func TestCORSConfiguration(t *testing.T) {
	t.Run("Default Configuration", func(t *testing.T) {
		cfg := config.DefaultConfig()
		
		// Check default CORS configuration
		if !cfg.Security.CORS.Enabled {
			t.Error("CORS should be enabled by default")
		}
		
		expectedOrigins := []string{"https://localhost:3000", "app://claude-desktop"}
		if len(cfg.Security.CORS.Origins) != len(expectedOrigins) {
			t.Errorf("Expected %d origins, got %d", len(expectedOrigins), len(cfg.Security.CORS.Origins))
		}
		
		for i, expected := range expectedOrigins {
			if i >= len(cfg.Security.CORS.Origins) || cfg.Security.CORS.Origins[i] != expected {
				t.Errorf("Expected origin %d to be '%s', got '%s'", i, expected, cfg.Security.CORS.Origins[i])
			}
		}
	})

	t.Run("CORS Methods and Headers", func(t *testing.T) {
		cfg := config.DefaultConfig()
		
		expectedMethods := []string{"POST", "OPTIONS"}
		if len(cfg.Security.CORS.Methods) != len(expectedMethods) {
			t.Errorf("Expected %d methods, got %d", len(expectedMethods), len(cfg.Security.CORS.Methods))
		}
		
		expectedHeaders := []string{"Content-Type", "Authorization"}
		if len(cfg.Security.CORS.Headers) != len(expectedHeaders) {
			t.Errorf("Expected %d headers, got %d", len(expectedHeaders), len(cfg.Security.CORS.Headers))
		}
	})
}

func TestCORSWildcardPatterns(t *testing.T) {
	corsConfig := config.CORSConfig{
		Enabled: true,
		Origins: []string{"*.localhost", "*.example.com"},
		Methods: []string{"POST", "OPTIONS"},
		Headers: []string{"Content-Type"},
	}

	corsMiddleware := security.NewCORSMiddleware(corsConfig)

	testCases := []struct {
		origin      string
		expectAllow bool
	}{
		{"https://api.localhost", true},
		{"https://web.localhost", true},
		{"https://sub.example.com", true},
		{"https://api.example.com", true},
		{"https://localhost", false},        // Exact match needed for wildcard
		{"https://example.com", false},      // Exact match needed for wildcard
		{"https://evil.com", false},         // Not matching pattern
		{"https://fakeexample.com", false},  // Not matching pattern
	}

	for _, tc := range testCases {
		t.Run(tc.origin, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", nil)
			req.Header.Set("Origin", tc.origin)
			
			w := httptest.NewRecorder()
			allowed := corsMiddleware.SetHeaders(w, req)
			
			if allowed != tc.expectAllow {
				t.Errorf("Origin %s: expected allow=%v, got %v", tc.origin, tc.expectAllow, allowed)
			}
		})
	}
}