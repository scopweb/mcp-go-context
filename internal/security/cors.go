package security

import (
	"net/http"
	"strings"

	"github.com/scopweb/mcp-go-context/internal/config"
)

// CORSMiddleware handles CORS configuration and validation
type CORSMiddleware struct {
	config config.CORSConfig
}

// NewCORSMiddleware creates a new CORS middleware
func NewCORSMiddleware(config config.CORSConfig) *CORSMiddleware {
	return &CORSMiddleware{
		config: config,
	}
}

// SetHeaders sets CORS headers based on configuration
func (c *CORSMiddleware) SetHeaders(w http.ResponseWriter, r *http.Request) bool {
	if !c.config.Enabled {
		return true // CORS disabled, allow all
	}

	origin := r.Header.Get("Origin")
	
	// Handle preflight requests
	if r.Method == "OPTIONS" {
		return c.handlePreflight(w, r, origin)
	}

	// Set CORS headers for actual requests
	if c.isOriginAllowed(origin) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
	} else if origin != "" {
		// Origin provided but not allowed
		return false
	}
	// If no origin header, allow the request (same-origin or non-browser)

	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Vary", "Origin")

	return true
}

// handlePreflight handles OPTIONS preflight requests
func (c *CORSMiddleware) handlePreflight(w http.ResponseWriter, r *http.Request, origin string) bool {
	if !c.isOriginAllowed(origin) {
		return false
	}

	// Set preflight headers
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	
	// Allow configured methods
	if len(c.config.Methods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(c.config.Methods, ", "))
	} else {
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	}

	// Allow configured headers
	if len(c.config.Headers) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(c.config.Headers, ", "))
	} else {
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	}

	w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
	w.Header().Set("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")
	
	w.WriteHeader(http.StatusOK)
	return true
}

// isOriginAllowed checks if the origin is in the allowed list
func (c *CORSMiddleware) isOriginAllowed(origin string) bool {
	if origin == "" {
		return true // No origin header (same-origin or non-browser request)
	}

	if len(c.config.Origins) == 0 {
		return false // No origins configured, deny all
	}

	// Check exact matches
	for _, allowedOrigin := range c.config.Origins {
		if allowedOrigin == "*" {
			return true // Wildcard allowed (not recommended for production)
		}
		if allowedOrigin == origin {
			return true
		}
		
		// Support for simple wildcard patterns like "*.example.com"
		if strings.HasPrefix(allowedOrigin, "*.") {
			suffix := allowedOrigin[1:] // Remove the "*"
			if strings.HasSuffix(origin, suffix) {
				return true
			}
		}
	}

	return false
}

// IsEnabled returns whether CORS is enabled
func (c *CORSMiddleware) IsEnabled() bool {
	return c.config.Enabled
}

// GetAllowedOrigins returns the list of allowed origins
func (c *CORSMiddleware) GetAllowedOrigins() []string {
	return c.config.Origins
}