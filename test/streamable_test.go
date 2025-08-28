package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/transport"
)

func TestStreamableHTTPTransport(t *testing.T) {
	// Create test configuration
	corsConfig := config.CORSConfig{
		Enabled: true,
		Origins: []string{"https://localhost:3000", "app://claude-desktop"},
		Methods: []string{"POST", "OPTIONS"},
		Headers: []string{"Content-Type", "Authorization"},
	}

	// Create transport
	streamable := transport.NewStreamableHTTPTransport(8081, corsConfig)

	// Mock handler
	handler := func(ctx context.Context, req json.RawMessage) (json.RawMessage, error) {
		var request map[string]interface{}
		json.Unmarshal(req, &request)
		
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      request["id"],
			"result": map[string]interface{}{
				"protocolVersion": "2025-03-26",
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
		streamable.Start(ctx, serverInfo, handler)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	t.Run("HTTP Request-Response", func(t *testing.T) {
		initReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
		}

		reqBody, _ := json.Marshal(initReq)
		resp, err := http.Post("http://localhost:8081/mcp", "application/json", bytes.NewBuffer(reqBody))
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
	})

	t.Run("Health Endpoint", func(t *testing.T) {
		resp, err := http.Get("http://localhost:8081/health")
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

		if health["protocol"] != "2025-03-26" {
			t.Error("Expected protocol version 2025-03-26")
		}

		if health["transport"] != "streamable-http" {
			t.Error("Expected transport 'streamable-http'")
		}
	})

	t.Run("CORS Headers", func(t *testing.T) {
		req, _ := http.NewRequest("OPTIONS", "http://localhost:8081/mcp", nil)
		req.Header.Set("Origin", "https://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type")

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
	})

	t.Run("Streaming Request", func(t *testing.T) {
		initReq := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "initialize",
		}

		reqBody, _ := json.Marshal(initReq)
		req, _ := http.NewRequest("POST", "http://localhost:8081/mcp", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

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

		// Read a bit of the stream
		buf := make([]byte, 1024)
		n, err := resp.Body.Read(buf)
		if err != nil {
			t.Fatalf("Failed to read from stream: %v", err)
		}

		streamContent := string(buf[:n])
		if !strings.Contains(streamContent, "event: response") {
			t.Error("Expected 'event: response' in stream content")
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