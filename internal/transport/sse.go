package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// SSETransport implements MCP over Server-Sent Events
type SSETransport struct {
	port     int
	server   *http.Server
	sessions map[string]*sseSession
	mu       sync.RWMutex
}

type sseSession struct {
	id       string
	writer   http.ResponseWriter
	flusher  http.Flusher
	messages chan json.RawMessage
	done     chan struct{}
}

// NewSSETransport creates a new SSE transport
func NewSSETransport(port int) Transport {
	return &SSETransport{
		port:     port,
		sessions: make(map[string]*sseSession),
	}
}

// Start begins the SSE server
func (s *SSETransport) Start(ctx context.Context, info ServerInfo, handler RequestHandler) error {
	mux := http.NewServeMux()

	// SSE endpoint for establishing connection
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		// Set headers for SSE
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		// Create session
		sessionID := generateSessionID()
		session := &sseSession{
			id:       sessionID,
			writer:   w,
			flusher:  flusher,
			messages: make(chan json.RawMessage, 100),
			done:     make(chan struct{}),
		}

		// Store session
		s.mu.Lock()
		s.sessions[sessionID] = session
		s.mu.Unlock()

		// Send session ID
		fmt.Fprintf(w, "data: {\"type\":\"session\",\"sessionId\":\"%s\"}\n\n", sessionID)
		flusher.Flush()

		// Send server info
		infoData, _ := json.Marshal(map[string]interface{}{
			"type": "server.info",
			"data": info,
		})
		fmt.Fprintf(w, "data: %s\n\n", infoData)
		flusher.Flush()

		// Keep connection alive
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				s.removeSession(sessionID)
				return
			case <-session.done:
				return
			case msg := <-session.messages:
				fmt.Fprintf(w, "data: %s\n\n", msg)
				flusher.Flush()
			case <-ticker.C:
				fmt.Fprintf(w, ": keepalive\n\n")
				flusher.Flush()
			}
		}
	})

	// Message endpoint for receiving commands
	mux.HandleFunc("/messages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Get session ID
		sessionID := r.URL.Query().Get("sessionId")
		if sessionID == "" {
			http.Error(w, "Missing sessionId", http.StatusBadRequest)
			return
		}

		// Get session
		s.mu.RLock()
		session, exists := s.sessions[sessionID]
		s.mu.RUnlock()

		if !exists {
			http.Error(w, "Invalid session", http.StatusBadRequest)
			return
		}

		// Read request
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read request", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Handle request
		go func() {
			response, err := handler(context.Background(), json.RawMessage(body))
			if err != nil {
				errorResp, _ := json.Marshal(map[string]interface{}{
					"type": "error",
					"error": map[string]interface{}{
						"code":    ErrorCodeInternalError,
						"message": err.Error(),
					},
				})
				session.messages <- errorResp
			} else {
				responseMsg, _ := json.Marshal(map[string]interface{}{
					"type": "response",
					"data": json.RawMessage(response),
				})
				session.messages <- responseMsg
			}
		}()

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		return s.Stop()
	case err := <-errChan:
		return err
	}
}

// Stop shuts down the SSE server
func (s *SSETransport) Stop() error {
	// Close all sessions
	s.mu.Lock()
	for _, session := range s.sessions {
		close(session.done)
	}
	s.sessions = make(map[string]*sseSession)
	s.mu.Unlock()

	if s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}

func (s *SSETransport) removeSession(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, exists := s.sessions[sessionID]; exists {
		close(session.done)
		delete(s.sessions, sessionID)
	}
}

func generateSessionID() string {
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}
