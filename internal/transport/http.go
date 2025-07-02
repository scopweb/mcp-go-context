package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// HTTPTransport implements MCP over HTTP
type HTTPTransport struct {
	port   int
	server *http.Server
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(port int) Transport {
	return &HTTPTransport{
		port: port,
	}
}

// Start begins the HTTP server
func (t *HTTPTransport) Start(ctx context.Context, info ServerInfo, handler RequestHandler) error {
	mux := http.NewServeMux()

	// MCP endpoint
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "application/json")

		// Read request
		var reqData json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		// Handle request
		respData, err := handler(ctx, reqData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Send response
		w.Write(respData)
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
			"server": info.Name,
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
