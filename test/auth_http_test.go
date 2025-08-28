package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/scopweb/mcp-go-context/internal/auth"
	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/server"
)

// getFreePort finds an available TCP port for tests
func getFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func waitForHealth(t *testing.T, url string) {
	t.Helper()
	client := &http.Client{Timeout: 500 * time.Millisecond}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server did not become healthy at %s", url)
}

func TestHTTPAuth_Enforcement(t *testing.T) {
	t.Setenv("MCP_JWT_SECRET", "test-secret-key-for-jwt-tests")

	cfg := config.DefaultConfig()
	port := getFreePort(t)
	cfg.Transport.Type = "http"
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
		// Start will block until ctx is cancelled
		_ = srv.Start(ctx)
	}()

	// Wait until /health answers
	healthURL := fmt.Sprintf("http://127.0.0.1:%d/health", port)
	waitForHealth(t, healthURL)

	// Prepare JSON-RPC initialize payload
	payload := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
	}
	body, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 3 * time.Second}

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

	// 1) Without Authorization header -> expect JSON-RPC error Unauthorized
	req1, _ := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/mcp", port), bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	resp1, err := client.Do(req1)
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	var respJSON1 map[string]any
	json.NewDecoder(resp1.Body).Decode(&respJSON1)
	resp1.Body.Close()
	if _, ok := respJSON1["error"]; !ok {
		t.Fatalf("expected error response, got: %v", respJSON1)
	}
	errObj := respJSON1["error"].(map[string]any)
	if !strings.Contains(fmt.Sprint(errObj["message"]), "Unauthorized") {
		t.Fatalf("expected Unauthorized error, got: %v", errObj)
	}

	// 2) With valid JWT Authorization header -> expect success result
	req2, _ := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:%d/mcp", port), bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+validToken)
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("HTTP authorized request failed: %v", err)
	}
	var respJSON2 map[string]any
	json.NewDecoder(resp2.Body).Decode(&respJSON2)
	resp2.Body.Close()
	if _, ok := respJSON2["result"]; !ok {
		t.Fatalf("expected result response, got: %v", respJSON2)
	}
}
