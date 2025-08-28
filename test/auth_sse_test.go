package test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/scopweb/mcp-go-context/internal/auth"
	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/server"
)

func TestSSEAuth_MessageFlow(t *testing.T) {
	t.Setenv("MCP_JWT_SECRET", "test-secret-key-for-jwt-tests")

	cfg := config.DefaultConfig()
	port := getFreePort(t)
	cfg.Transport.Type = "sse"
	cfg.Transport.Port = port
	// Enable JWT authentication
	cfg.Security.Auth.Enabled = true
	cfg.Security.Auth.Method = "jwt"

	srv, err := server.New(cfg)
	if err != nil {
		t.Fatalf("server.New error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = srv.Start(ctx)
	}()

	// Wait for SSE endpoint to be available by polling /messages with OPTIONS
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest("OPTIONS", fmt.Sprintf("http://127.0.0.1:%d/messages?sessionId=probe", port), nil)
		resp, err := client.Do(req)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Open SSE stream to get a sessionId
	sseResp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/sse", port))
	if err != nil {
		t.Fatalf("failed to connect to SSE: %v", err)
	}
	defer sseResp.Body.Close()

	reader := bufio.NewReader(sseResp.Body)

	// Read first event (session)
	ev := readSSEEventJSON(t, reader, 5*time.Second)
	if typ, _ := ev["type"].(string); typ != "session" {
		t.Fatalf("expected first event type=session, got: %v", ev)
	}
	sessionID, _ := ev["sessionId"].(string)
	if sessionID == "" {
		t.Fatalf("missing sessionId in session event: %v", ev)
	}

	// Read second event (server.info) and ignore
	_ = readSSEEventJSON(t, reader, 5*time.Second)

	// Prepare JSON-RPC initialize payload
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
	}
	body, _ := json.Marshal(payload)

	// Generate JWT token for testing
	jwtConfig := auth.JWTConfig{
		Secret: "test-secret-key-for-jwt-tests",
		Expiry: time.Hour,
	}
	jwtManager := auth.NewJWTManager(jwtConfig)
	validToken, err := jwtManager.GenerateToken("test-user")
	if err != nil {
		t.Fatalf("failed to generate JWT token: %v", err)
	}

	// 1) Send without Authorization -> expect response event containing JSON-RPC error Unauthorized
	req1, _ := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/messages?sessionId=%s", port, sessionID), bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	resp1, err := client.Do(req1)
	if err != nil {
		t.Fatalf("messages POST failed: %v", err)
	}
	if resp1.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on /messages, got %d", resp1.StatusCode)
	}
	resp1.Body.Close()

	ev1 := awaitResponseEvent(t, reader, 5*time.Second)
	if typ, _ := ev1["type"].(string); typ != "response" {
		t.Fatalf("expected type=response, got: %v", ev1)
	}
	data1, _ := ev1["data"].(map[string]any)
	if data1 == nil {
		t.Fatalf("missing data in response event: %v", ev1)
	}
	if _, ok := data1["error"]; !ok {
		t.Fatalf("expected JSON-RPC error in unauthorized response, got: %v", data1)
	}
	errObj, _ := data1["error"].(map[string]any)
	if errObj == nil || !strings.Contains(fmt.Sprint(errObj["message"]), "Unauthorized") {
		t.Fatalf("expected Unauthorized error, got: %v", errObj)
	}

	// 2) Send with valid JWT Authorization -> expect response event with JSON-RPC result
	req2, _ := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/messages?sessionId=%s", port, sessionID), bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+validToken)
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("messages POST failed: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on /messages, got %d", resp2.StatusCode)
	}
	resp2.Body.Close()

	ev2 := awaitResponseEvent(t, reader, 5*time.Second)
	if typ, _ := ev2["type"].(string); typ != "response" {
		t.Fatalf("expected type=response, got: %v", ev2)
	}
	data2, _ := ev2["data"].(map[string]any)
	if data2 == nil {
		t.Fatalf("missing data in response event: %v", ev2)
	}
	if _, ok := data2["result"]; !ok {
		t.Fatalf("expected JSON-RPC result in authorized response, got: %v", data2)
	}
}

// readSSEEventJSON reads a single SSE event object from the stream with a timeout.
func readSSEEventJSON(t *testing.T, r *bufio.Reader, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		// Accumulate lines for one event
		var b strings.Builder
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				t.Fatalf("read error: %v", err)
			}

			// Skip keepalive comments
			if strings.HasPrefix(line, ":") {
				// flush blank line after comment if any; continue outer loop
				break
			}

			if strings.HasPrefix(line, "data: ") {
				b.WriteString(strings.TrimPrefix(line, "data: "))
			}

			// Blank line ends the event
			if strings.TrimSpace(line) == "" && b.Len() > 0 {
				break
			}

			// Continue accumulating lines for the same event
		}

		txt := strings.TrimSpace(b.String())
		if txt == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(txt), &obj); err != nil {
			// Not a JSON event, skip
			continue
		}
		return obj
	}
	t.Fatal("timeout waiting for SSE event")
	return nil
}

// awaitResponseEvent waits until an SSE event with type=response arrives.
func awaitResponseEvent(t *testing.T, r *bufio.Reader, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		ev := readSSEEventJSON(t, r, time.Until(deadline))
		if ev == nil {
			continue
		}
		if typ, _ := ev["type"].(string); typ == "response" {
			return ev
		}
	}
	t.Fatal("timeout waiting for response event")
	return nil
}
