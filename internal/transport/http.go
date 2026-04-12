package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// HTTPTransport implements MCP over HTTP
type HTTPTransport struct {
	port           int
	server         *http.Server
	routeRegistrar func(*http.ServeMux)
}

// NewHTTPTransport creates a new HTTP transport
func NewHTTPTransport(port int) Transport {
	return &HTTPTransport{
		port: port,
	}
}

// SetRouteRegistrar provides extra HTTP routes to register before the server starts.
func (t *HTTPTransport) SetRouteRegistrar(register func(*http.ServeMux)) {
	t.routeRegistrar = register
}

// Start begins the HTTP server
func (t *HTTPTransport) Start(ctx context.Context, info ServerInfo, handler RequestHandler) error {
	mux := t.newMux(ctx, info, handler)

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

func (t *HTTPTransport) newMux(ctx context.Context, info ServerInfo, handler RequestHandler) *http.ServeMux {
	mux := http.NewServeMux()
	if t.routeRegistrar != nil {
		t.routeRegistrar(mux)
	}

	// MCP endpoint
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		var reqData json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		respData, err := handler(ctx, reqData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Write(respData)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "ok",
			"server":  info.Name,
			"version": info.Version,
		})
	})

	return mux
}

// Stop shuts down the HTTP server
func (t *HTTPTransport) Stop() error {
	if t.server != nil {
		return t.server.Shutdown(context.Background())
	}
	return nil
}
