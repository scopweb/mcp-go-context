package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/security"
)

// HTTPTransport implements MCP over HTTP
type HTTPTransport struct {
	port       int
	server     *http.Server
	corsConfig config.CORSConfig
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(port int) Transport {
	return &HTTPTransport{
		port: port,
		corsConfig: config.CORSConfig{
			Enabled: false, // Default to disabled for backward compatibility
		},
	}
}

// NewHTTPTransportWithCORS creates a new HTTP transport with CORS configuration
func NewHTTPTransportWithCORS(port int, corsConfig config.CORSConfig) Transport {
	return &HTTPTransport{
		port:       port,
		corsConfig: corsConfig,
	}
}

// Start begins the HTTP server
func (t *HTTPTransport) Start(ctx context.Context, info ServerInfo, handler RequestHandler) error {
	mux := http.NewServeMux()
	corsMiddleware := security.NewCORSMiddleware(t.corsConfig)

	// MCP endpoint
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS
		if !corsMiddleware.SetHeaders(w, r) {
			log.Printf("CORS rejected origin: %s", r.Header.Get("Origin"))
			w.WriteHeader(http.StatusForbidden)
			return
		}

		// Handle preflight
		if r.Method == http.MethodOptions {
			return // Already handled by CORS middleware
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// Read request
		var reqData json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Inyectar request en el contexto para auth
		ctxWithReq := context.WithValue(ctx, "httpRequest", r)
		// Handle request
		respData, err := handler(ctxWithReq, reqData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Send raw JSON response without HTTP headers
		w.Header().Del("Content-Type")
		w.Write(respData)
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS for health check too
		corsMiddleware.SetHeaders(w, r)
		
		if r.Method == http.MethodOptions {
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"server":  info.Name,
			"version": info.Version,
		})
	})

	t.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", t.port),
		Handler: mux,
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- t.server.ListenAndServe()
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		return t.server.Shutdown(context.Background())
	case err := <-errChan:
		return err
	}
}

// Stop shuts down the HTTP server
func (t *HTTPTransport) Stop() error {
	if t.server != nil {
		return t.server.Shutdown(context.Background())
	}
	return nil
}
