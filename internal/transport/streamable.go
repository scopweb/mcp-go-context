package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/security"
)

const (
	defaultHTTPProtocolVersion = "2025-03-26"
	latestHTTPProtocolVersion  = "2025-11-25"
)

var supportedHTTPProtocolVersions = map[string]struct{}{
	"2024-11-05": {},
	"2025-03-26": {},
	"2025-06-18": {},
	"2025-11-25": {},
}

// StreamableHTTPTransport implements MCP Streamable HTTP Transport.
type StreamableHTTPTransport struct {
	port       int
	server     *http.Server
	corsConfig config.CORSConfig
	sessions   map[string]*streamableSession
	mu         sync.RWMutex
}

type streamableSession struct {
	id         string
	messages   chan json.RawMessage
	done       chan struct{}
	lastActive time.Time
	eventID    uint64
}

// NewStreamableHTTPTransport creates a new Streamable HTTP transport.
func NewStreamableHTTPTransport(port int, corsConfig config.CORSConfig) Transport {
	return &StreamableHTTPTransport{
		port:       port,
		corsConfig: corsConfig,
		sessions:   make(map[string]*streamableSession),
	}
}

// Start begins the Streamable HTTP server.
func (t *StreamableHTTPTransport) Start(ctx context.Context, info ServerInfo, handler RequestHandler) error {
	mux := http.NewServeMux()
	corsMiddleware := security.NewCORSMiddleware(t.corsConfig)

	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		if !corsMiddleware.SetHeaders(w, r) {
			log.Printf("CORS rejected origin: %s", r.Header.Get("Origin"))
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if r.Method == http.MethodOptions {
			return
		}

		protocolVersion, ok := t.validateProtocolVersionHeader(w, r)
		if !ok {
			return
		}

		switch r.Method {
		case http.MethodGet:
			t.handleGetStream(w, r, protocolVersion)
		case http.MethodPost:
			t.handlePostMessage(w, r, handler, ctx, protocolVersion)
		case http.MethodDelete:
			t.handleDeleteSession(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		corsMiddleware.SetHeaders(w, r)
		if r.Method == http.MethodOptions {
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ok",
			"server":    info.Name,
			"version":   info.Version,
			"protocol":  latestHTTPProtocolVersion,
			"transport": "streamable-http",
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

	go t.cleanupSessions(ctx)

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

func (t *StreamableHTTPTransport) validateProtocolVersionHeader(w http.ResponseWriter, r *http.Request) (string, bool) {
	protocolVersion := r.Header.Get("MCP-Protocol-Version")
	if protocolVersion == "" {
		return defaultHTTPProtocolVersion, true
	}
	if _, ok := supportedHTTPProtocolVersions[protocolVersion]; !ok {
		http.Error(w, "Unsupported MCP-Protocol-Version", http.StatusBadRequest)
		return "", false
	}
	return protocolVersion, true
}

func (t *StreamableHTTPTransport) handleGetStream(w http.ResponseWriter, r *http.Request, protocolVersion string) {
	if !accepts(r.Header.Get("Accept"), "text/event-stream") {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.Header.Get("MCP-Session-Id")
	if sessionID == "" {
		http.Error(w, "Missing MCP-Session-Id header", http.StatusBadRequest)
		return
	}

	session, ok := t.getSession(sessionID)
	if !ok {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("MCP-Session-Id", sessionID)
	w.Header().Set("MCP-Protocol-Version", protocolVersion)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	t.writeSSEEvent(w, flusher, session, nil)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-session.done:
			return
		case msg := <-session.messages:
			t.touchSession(session.id)
			t.writeSSEEvent(w, flusher, session, msg)
		case <-ticker.C:
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (t *StreamableHTTPTransport) handlePostMessage(w http.ResponseWriter, r *http.Request, handler RequestHandler, ctx context.Context, protocolVersion string) {
	acceptHeader := r.Header.Get("Accept")
	if !accepts(acceptHeader, "application/json") || !accepts(acceptHeader, "text/event-stream") {
		http.Error(w, "Accept header must include application/json and text/event-stream", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var baseReq struct {
		JSONRPC string      `json:"jsonrpc"`
		ID      interface{} `json:"id,omitempty"`
		Method  string      `json:"method"`
	}
	if err := json.Unmarshal(body, &baseReq); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	sessionID := r.Header.Get("MCP-Session-Id")
	var session *streamableSession
	if baseReq.Method == "initialize" {
		if sessionID == "" {
			sessionID, session = t.createSession()
		} else {
			var ok bool
			session, ok = t.getSession(sessionID)
			if !ok {
				http.Error(w, "Session not found", http.StatusNotFound)
				return
			}
		}
	} else {
		if sessionID == "" {
			http.Error(w, "Missing MCP-Session-Id header", http.StatusBadRequest)
			return
		}
		var ok bool
		session, ok = t.getSession(sessionID)
		if !ok {
			http.Error(w, "Session not found", http.StatusNotFound)
			return
		}
	}

	ctxWithReq := context.WithValue(ctx, "httpRequest", r)
	ctxWithReq = context.WithValue(ctxWithReq, "sessionID", sessionID)
	ctxWithReq = context.WithValue(ctxWithReq, "transportType", "streamable-http")
	ctxWithReq = context.WithValue(ctxWithReq, "protocolVersion", protocolVersion)

	respData, err := handler(ctxWithReq, json.RawMessage(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t.touchSession(sessionID)
	w.Header().Set("MCP-Session-Id", sessionID)
	w.Header().Set("MCP-Protocol-Version", protocolVersion)

	if respData == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	if prefersEventStream(acceptHeader) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}
		t.writeSSEEvent(w, flusher, session, nil)
		t.writeSSEEvent(w, flusher, session, respData)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(respData)
}

func (t *StreamableHTTPTransport) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("MCP-Session-Id")
	if sessionID == "" {
		http.Error(w, "Missing MCP-Session-Id header", http.StatusBadRequest)
		return
	}

	if !t.removeSession(sessionID) {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (t *StreamableHTTPTransport) getSession(sessionID string) (*streamableSession, bool) {
	t.mu.RLock()
	session, exists := t.sessions[sessionID]
	t.mu.RUnlock()
	return session, exists
}

func (t *StreamableHTTPTransport) createSession() (string, *streamableSession) {
	sessionID := generateSessionID()
	session := &streamableSession{
		id:         sessionID,
		messages:   make(chan json.RawMessage, 100),
		done:       make(chan struct{}),
		lastActive: time.Now(),
	}

	t.mu.Lock()
	t.sessions[sessionID] = session
	t.mu.Unlock()

	return sessionID, session
}

func (t *StreamableHTTPTransport) touchSession(sessionID string) {
	t.mu.Lock()
	if session, ok := t.sessions[sessionID]; ok {
		session.lastActive = time.Now()
	}
	t.mu.Unlock()
}

func (t *StreamableHTTPTransport) writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, session *streamableSession, msg json.RawMessage) {
	t.mu.Lock()
	session.eventID++
	eventID := session.eventID
	t.mu.Unlock()

	if len(msg) == 0 {
		fmt.Fprintf(w, "id: %s:%d\ndata:\n\n", session.id, eventID)
		flusher.Flush()
		return
	}

	fmt.Fprintf(w, "id: %s:%d\ndata: %s\n\n", session.id, eventID, msg)
	flusher.Flush()
}

// cleanupSessions removes inactive sessions.
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

// removeSession removes a session.
func (t *StreamableHTTPTransport) removeSession(sessionID string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if session, exists := t.sessions[sessionID]; exists {
		close(session.done)
		delete(t.sessions, sessionID)
		return true
	}

	return false
}

// Stop shuts down the server.
func (t *StreamableHTTPTransport) Stop() error {
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

func accepts(acceptHeader string, contentType string) bool {
	for _, value := range strings.Split(acceptHeader, ",") {
		if strings.TrimSpace(value) == contentType {
			return true
		}
	}
	return false
}

func prefersEventStream(acceptHeader string) bool {
	parts := strings.Split(acceptHeader, ",")
	if len(parts) == 0 {
		return false
	}
	return strings.TrimSpace(parts[0]) == "text/event-stream"
}
