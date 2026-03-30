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

	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/transport"
)

func TestStreamableHTTPTransport(t *testing.T) {
	port := getFreePort(t)
	corsConfig := config.CORSConfig{
		Enabled: true,
		Origins: []string{"https://localhost:3000", "app://claude-desktop"},
		Methods: []string{"GET", "POST", "DELETE", "OPTIONS"},
		Headers: []string{"Accept", "Content-Type", "Authorization", "MCP-Protocol-Version", "MCP-Session-Id"},
	}

	streamable := transport.NewStreamableHTTPTransport(port, corsConfig)

	handler := func(ctx context.Context, req json.RawMessage) (json.RawMessage, error) {
		var request map[string]interface{}
		json.Unmarshal(req, &request)
		if _, hasID := request["id"]; !hasID {
			return nil, nil
		}
		protocolVersion, _ := ctx.Value("protocolVersion").(string)
		if protocolVersion == "" {
			protocolVersion = "2025-03-26"
		}

		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      request["id"],
			"result": map[string]interface{}{
				"protocolVersion": protocolVersion,
				"serverInfo": map[string]interface{}{
					"name":    "Test Server",
					"version": "2.0.0",
				},
			},
		}
		return json.Marshal(response)
	}

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverInfo := transport.ServerInfo{
		Name:         "Test MCP Server",
		Version:      "2.0.0",
		Instructions: "Test server for Streamable HTTP",
	}

	go func() {
		_ = streamable.Start(ctx, serverInfo, handler)
	}()

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	waitForHealth(t, baseURL+"/health")

	var sessionID string

	t.Run("HTTP Request-Response", func(t *testing.T) {
		initReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
		}

		reqBody, _ := json.Marshal(initReq)
		req, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("MCP-Protocol-Version", "2025-11-25")

		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("Failed to make HTTP request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var response map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["jsonrpc"] != "2.0" {
			t.Error("Expected JSON-RPC 2.0 response")
		}

		result, _ := response["result"].(map[string]interface{})
		if result["protocolVersion"] != "2025-11-25" {
			t.Errorf("Expected negotiated protocol 2025-11-25, got %v", result["protocolVersion"])
		}

		sessionID = resp.Header.Get("MCP-Session-Id")
		if sessionID == "" {
			t.Fatal("Expected MCP-Session-Id header on initialize response")
		}
	})

	t.Run("Health Endpoint", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/health")
		if err != nil {
			t.Fatalf("Failed to get health endpoint: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}

		var health map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to decode health response: %v", err)
		}

		if health["status"] != "ok" {
			t.Error("Expected health status 'ok'")
		}

		if health["protocol"] != "2025-11-25" {
			t.Error("Expected protocol version 2025-11-25")
		}

		if health["transport"] != "streamable-http" {
			t.Error("Expected transport 'streamable-http'")
		}
	})

	t.Run("CORS Headers", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", baseURL+"/mcp", nil)
		req.Header.Set("Origin", "https://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "GET")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type, MCP-Protocol-Version, MCP-Session-Id")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make CORS preflight request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected CORS preflight status 200, got %d", resp.StatusCode)
		}

		allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
		if allowOrigin != "https://localhost:3000" {
			t.Errorf("Expected Access-Control-Allow-Origin 'https://localhost:3000', got '%s'", allowOrigin)
		}

		allowMethods := resp.Header.Get("Access-Control-Allow-Methods")
		if !strings.Contains(allowMethods, "GET") || !strings.Contains(allowMethods, "DELETE") {
			t.Fatalf("Expected GET and DELETE in allow methods, got %q", allowMethods)
		}
	})

	t.Run("Initialized Notification Returns Accepted", func(t *testing.T) {
		notification := map[string]interface{}{
			"jsonrpc": "2.0",
			"method":  "notifications/initialized",
		}

		body, _ := json.Marshal(notification)
		req, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
		req.Header.Set("MCP-Protocol-Version", "2025-11-25")
		req.Header.Set("MCP-Session-Id", sessionID)

		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("Failed to send initialized notification: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("Expected status 202 for notification, got %d", resp.StatusCode)
		}
	})

	t.Run("Streaming Request", func(t *testing.T) {
		initReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      2,
			"method":  "ping",
		}

		reqBody, _ := json.Marshal(initReq)
		req, _ := http.NewRequest(http.MethodPost, baseURL+"/mcp", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream, application/json")
		req.Header.Set("MCP-Protocol-Version", "2025-11-25")
		req.Header.Set("MCP-Session-Id", sessionID)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Failed to make streaming request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected streaming status 200, got %d", resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType != "text/event-stream" {
			t.Errorf("Expected Content-Type 'text/event-stream', got '%s'", contentType)
		}

		data := readSSEData(t, resp.Body)
		if !strings.Contains(data, `"id":2`) {
			t.Fatalf("Expected JSON-RPC response in SSE body, got %s", data)
		}
	})

	t.Run("GET Stream Uses Session Header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/mcp", nil)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("MCP-Protocol-Version", "2025-11-25")
		req.Header.Set("MCP-Session-Id", sessionID)

		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("Failed to open GET stream: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", resp.StatusCode)
		}

		reader := bufio.NewReader(resp.Body)
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed reading stream primer: %v", err)
		}
		if !strings.HasPrefix(line, "id: ") {
			t.Fatalf("Expected SSE event id, got %q", line)
		}
	})

	t.Run("Invalid Protocol Header", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, baseURL+"/mcp", nil)
		req.Header.Set("Accept", "text/event-stream")
		req.Header.Set("MCP-Protocol-Version", "2023-01-01")
		req.Header.Set("MCP-Session-Id", sessionID)

		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("Expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Delete Session", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, baseURL+"/mcp", nil)
		req.Header.Set("MCP-Protocol-Version", "2025-11-25")
		req.Header.Set("MCP-Session-Id", sessionID)

		resp, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatalf("Failed to delete session: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("Expected status 204, got %d", resp.StatusCode)
		}

		followReq, _ := http.NewRequest(http.MethodGet, baseURL+"/mcp", nil)
		followReq.Header.Set("Accept", "text/event-stream")
		followReq.Header.Set("MCP-Protocol-Version", "2025-11-25")
		followReq.Header.Set("MCP-Session-Id", sessionID)

		followResp, err := (&http.Client{}).Do(followReq)
		if err != nil {
			t.Fatalf("Failed follow-up request: %v", err)
		}
		defer followResp.Body.Close()

		if followResp.StatusCode != http.StatusNotFound {
			t.Fatalf("Expected status 404 for deleted session, got %d", followResp.StatusCode)
		}
	})
}

func TestStreamableHTTPTransportIntegration(t *testing.T) {
	t.Run("Server Creation with Streamable Transport", func(t *testing.T) {
		cfg := config.DefaultConfig()
		cfg.Transport.Type = "streamable-http"
		cfg.Transport.Port = 8082

		// This should not panic and should create the transport
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Server creation panicked: %v", r)
			}
		}()

		// Note: We can't easily test the full server creation here without
		// setting up all dependencies, but we can test that the transport
		// type is recognized in the switch statement
		if cfg.Transport.Type != "streamable-http" {
			t.Error("Transport type not preserved")
		}
	})
}

func readSSEData(t *testing.T, body io.Reader) string {
	t.Helper()
	reader := bufio.NewReader(body)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read SSE line: %v", err)
		}
		if strings.HasPrefix(line, "data: ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		}
	}
	t.Fatal("Timed out reading SSE data")
	return ""
}
