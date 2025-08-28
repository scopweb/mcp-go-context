package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/security"
)

// StreamableHTTPTransport implements MCP Streamable HTTP Transport (2025-03-26)
// Combines HTTP request-response with Server-Sent Events for bidirectional communication
type StreamableHTTPTransport struct {
	port       int
	server     *http.Server
	corsConfig config.CORSConfig
	sessions   map[string]*streamableSession
	mu         sync.RWMutex
}

type streamableSession struct {
	id         string
	writer     http.ResponseWriter
	flusher    http.Flusher
	messages   chan json.RawMessage
	done       chan struct{}
	lastActive time.Time
}

// NewStreamableHTTPTransport creates a new Streamable HTTP transport
func NewStreamableHTTPTransport(port int, corsConfig config.CORSConfig) Transport {
	return &StreamableHTTPTransport{
		port:       port,
		corsConfig: corsConfig,
		sessions:   make(map[string]*streamableSession),
	}
}

// Start begins the Streamable HTTP server
func (t *StreamableHTTPTransport) Start(ctx context.Context, info ServerInfo, handler RequestHandler) error {
	mux := http.NewServeMux()
	corsMiddleware := security.NewCORSMiddleware(t.corsConfig)

	// Main MCP endpoint - handles both HTTP and stream requests
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS
		if !corsMiddleware.SetHeaders(w, r) {
			log.Printf("CORS rejected origin: %s", r.Header.Get("Origin"))
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if r.Method == http.MethodOptions {
			return // Already handled by CORS
		}

		// Check if this is a streaming request
		acceptHeader := r.Header.Get("Accept")
		if acceptHeader == "text/event-stream" || r.Header.Get("Connection") == "keep-alive" {
			t.handleStreamingRequest(w, r, handler, ctx)
		} else {
			t.handleHTTPRequest(w, r, handler, ctx)
		}
	})

	// Stream endpoint - establishes SSE connection for bidirectional communication
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS
		if !corsMiddleware.SetHeaders(w, r) {
			log.Printf("CORS rejected origin for stream: %s", r.Header.Get("Origin"))
			w.WriteHeader(http.StatusForbidden)
			return
		}

		t.handleStreamConnection(w, r, info)
	})

	// Messages endpoint - receives messages for active streams
	mux.HandleFunc("/messages", func(w http.ResponseWriter, r *http.Request) {
		// Handle CORS
		if !corsMiddleware.SetHeaders(w, r) {
			log.Printf("CORS rejected origin for messages: %s", r.Header.Get("Origin"))
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if r.Method == http.MethodOptions {
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		t.handleStreamMessage(w, r, handler, ctx)
	})

	// Health and capabilities endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		corsMiddleware.SetHeaders(w, r)
		
		if r.Method == http.MethodOptions {
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "ok",
			"server":      info.Name,
			"version":     info.Version,
			"protocol":    "2025-03-26",
			"transport":   "streamable-http",
			"capabilities": map[string]interface{}{
				"streaming": true,
				"http":      true,
				"sse":       true,
			},
		})
	})

	t.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", t.port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Cleanup routine for expired sessions
	go t.cleanupSessions(ctx)

	// Start server
	errChan := make(chan error, 1)
	go func() {
		if err := t.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	log.Printf("Streamable HTTP Transport started on port %d", t.port)

	select {
	case <-ctx.Done():
		return t.Stop()
	case err := <-errChan:
		return err
	}
}

// handleHTTPRequest handles standard HTTP request-response
func (t *StreamableHTTPTransport) handleHTTPRequest(w http.ResponseWriter, r *http.Request, handler RequestHandler, ctx context.Context) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	var reqData json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Process request
	ctxWithReq := context.WithValue(ctx, "httpRequest", r)
	respData, err := handler(ctxWithReq, reqData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}

// handleStreamingRequest handles streaming HTTP requests (hybrid mode)
func (t *StreamableHTTPTransport) handleStreamingRequest(w http.ResponseWriter, r *http.Request, handler RequestHandler, ctx context.Context) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Process the request normally but send response as SSE
	if r.Method != http.MethodPost {
		fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Method not allowed\"}\n\n")
		flusher.Flush()
		return
	}

	var reqData json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Invalid JSON\"}\n\n")
		flusher.Flush()
		return
	}
	defer r.Body.Close()

	// Process request
	ctxWithReq := context.WithValue(ctx, "httpRequest", r)
	respData, err := handler(ctxWithReq, reqData)
	if err != nil {
		errorData, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", errorData)
		flusher.Flush()
		return
	}

	// Send response as SSE
	fmt.Fprintf(w, "event: response\ndata: %s\n\n", respData)
	flusher.Flush()
}

// handleStreamConnection establishes persistent SSE connection
func (t *StreamableHTTPTransport) handleStreamConnection(w http.ResponseWriter, r *http.Request, info ServerInfo) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// Create session
	sessionID := generateSessionID()
	session := &streamableSession{
		id:         sessionID,
		writer:     w,
		flusher:    flusher,
		messages:   make(chan json.RawMessage, 100),
		done:       make(chan struct{}),
		lastActive: time.Now(),
	}

	// Store session
	t.mu.Lock()
	t.sessions[sessionID] = session
	t.mu.Unlock()

	// Send initial connection info
	initData, _ := json.Marshal(map[string]interface{}{
		"type":      "connection",
		"sessionId": sessionID,
		"server":    info.Name,
		"version":   info.Version,
		"protocol":  "2025-03-26",
	})
	fmt.Fprintf(w, "event: init\ndata: %s\n\n", initData)
	flusher.Flush()

	// Keep connection alive
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			t.removeSession(sessionID)
			return
		case <-session.done:
			return
		case msg := <-session.messages:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", msg)
			flusher.Flush()
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

// handleStreamMessage processes messages sent to active streams
func (t *StreamableHTTPTransport) handleStreamMessage(w http.ResponseWriter, r *http.Request, handler RequestHandler, ctx context.Context) {
	// Get session ID
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId", http.StatusBadRequest)
		return
	}

	// Get session
	t.mu.RLock()
	session, exists := t.sessions[sessionID]
	t.mu.RUnlock()

	if !exists {
		http.Error(w, "Invalid session", http.StatusBadRequest)
		return
	}

	// Read message
	var reqData json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Process message asynchronously
	go func() {
		ctxWithReq := context.WithValue(ctx, "httpRequest", r)
		respData, err := handler(ctxWithReq, reqData)
		
		// Update session activity
		session.lastActive = time.Now()

		var response json.RawMessage
		if err != nil {
			errorResp, _ := json.Marshal(map[string]interface{}{
				"type":  "error", 
				"error": err.Error(),
			})
			response = errorResp
		} else {
			successResp, _ := json.Marshal(map[string]interface{}{
				"type": "response",
				"data": json.RawMessage(respData),
			})
			response = successResp
		}

		// Send to session
		select {
		case session.messages <- response:
		case <-session.done:
		default:
			log.Printf("Session %s message queue full", sessionID)
		}
	}()

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

// cleanupSessions removes inactive sessions
func (t *StreamableHTTPTransport) cleanupSessions(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.mu.Lock()
			now := time.Now()
			for id, session := range t.sessions {
				if now.Sub(session.lastActive) > 10*time.Minute {
					close(session.done)
					delete(t.sessions, id)
					log.Printf("Cleaned up inactive session: %s", id)
				}
			}
			t.mu.Unlock()
		}
	}
}

// removeSession removes a session
func (t *StreamableHTTPTransport) removeSession(sessionID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if session, exists := t.sessions[sessionID]; exists {
		close(session.done)
		delete(t.sessions, sessionID)
	}
}

// Stop shuts down the server
func (t *StreamableHTTPTransport) Stop() error {
	// Close all sessions
	t.mu.Lock()
	for _, session := range t.sessions {
		close(session.done)
	}
	t.sessions = make(map[string]*streamableSession)
	t.mu.Unlock()

	if t.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return t.server.Shutdown(ctx)
}