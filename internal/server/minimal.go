package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/transport"
)

// MinimalServer represents a minimal MCP server for testing
type MinimalServer struct {
	config    *config.Config
	transport transport.Transport
}

// NewMinimal creates a minimal MCP server (no analyzer, no memory)
func NewMinimal(cfg *config.Config) (*MinimalServer, error) {
	log.Printf("Creating minimal server with transport: %s", cfg.Transport.Type)
	
	// Initialize transport only
	var trans transport.Transport

	switch cfg.Transport.Type {
	case "stdio":
		log.Printf("Initializing stdio transport...")
		trans = transport.NewStdioTransport()
	case "http":
		log.Printf("Initializing HTTP transport on port %d...", cfg.Transport.Port)
		trans = transport.NewHTTPTransport(cfg.Transport.Port)
	case "sse":
		log.Printf("Initializing SSE transport on port %d...", cfg.Transport.Port)
		trans = transport.NewSSETransport(cfg.Transport.Port)
	default:
		return nil, fmt.Errorf("unknown transport type: %s", cfg.Transport.Type)
	}

	log.Printf("Transport initialized successfully")

	// Create minimal server
	srv := &MinimalServer{
		config:    cfg,
		transport: trans,
	}

	log.Printf("Minimal server created successfully")
	return srv, nil
}

// Start starts the minimal MCP server
func (s *MinimalServer) Start(ctx context.Context) error {
	log.Printf("Starting minimal MCP server...")
	
	// Initialize server info
	info := transport.ServerInfo{
		Name:    "MCP Context Server (Minimal)",
		Version: "1.0.0-minimal",
		Instructions: "Minimal MCP server for debugging connection issues.",
	}

	log.Printf("Starting transport with handler...")
	// Start transport
	return s.transport.Start(ctx, info, s.handleRequest)
}

// handleRequest handles incoming MCP requests (minimal implementation)
func (s *MinimalServer) handleRequest(ctx context.Context, req json.RawMessage) (json.RawMessage, error) {
	start := time.Now()
	log.Printf("=== MINIMAL REQUEST START ===")
	log.Printf("Raw request: %s", string(req))
	
	// Parse base request to get method and ID
	var baseReq struct {
		JSONRPC string      `json:"jsonrpc"`
		Method  string      `json:"method"`
		ID      interface{} `json:"id"`
		Params  interface{} `json:"params,omitempty"`
	}

	if err := json.Unmarshal(req, &baseReq); err != nil {
		log.Printf("Parse error after %v: %v", time.Since(start), err)
		return s.createErrorResponse(nil, -32700, fmt.Sprintf("Parse error: %v", err))
	}

	// Validate JSON-RPC version
	if baseReq.JSONRPC != "2.0" {
		log.Printf("Invalid JSON-RPC version: %s", baseReq.JSONRPC)
		return s.createErrorResponse(baseReq.ID, -32600, "Invalid Request: jsonrpc must be '2.0'")
	}

	log.Printf("Handling request: %s (ID: %v)", baseReq.Method, baseReq.ID)

	var result interface{}
	var err error

	switch baseReq.Method {
	case "initialize":
		log.Printf("=== MINIMAL INITIALIZE REQUEST ===")
		methodStart := time.Now()
		result, err = s.handleInitialize(baseReq.ID)
		if err != nil {
			log.Printf("Initialize error after %v: %v", time.Since(methodStart), err)
		} else {
			log.Printf("Initialize successful after %v", time.Since(methodStart))
		}
	case "tools/list":
		log.Printf("=== MINIMAL TOOLS/LIST REQUEST ===")
		result, err = s.handleToolsList()
	case "notifications/initialized":
		log.Printf("=== INITIALIZED NOTIFICATION ===")
		log.Printf("Received initialized notification")
		return nil, nil
	default:
		log.Printf("Unknown method: %s", baseReq.Method)
		return s.createErrorResponse(baseReq.ID, -32601, fmt.Sprintf("Method not found: %s", baseReq.Method))
	}

	if err != nil {
		log.Printf("Request error after %v: %v", time.Since(start), err)
		response, createErr := s.createErrorResponse(baseReq.ID, -32603, err.Error())
		if createErr != nil {
			log.Printf("Failed to create error response: %v", createErr)
		}
		log.Printf("=== MINIMAL REQUEST END (ERROR) === Total time: %v", time.Since(start))
		return response, createErr
	}

	log.Printf("Request successful for method: %s", baseReq.Method)
	response, createErr := s.createSuccessResponse(baseReq.ID, result)
	if createErr != nil {
		log.Printf("Failed to create success response: %v", createErr)
	} else {
		log.Printf("Response created: %s", string(response))
	}
	log.Printf("=== MINIMAL REQUEST END (SUCCESS) === Total time: %v", time.Since(start))
	return response, createErr
}

// createSuccessResponse creates a JSON-RPC success response
func (s *MinimalServer) createSuccessResponse(id interface{}, result interface{}) (json.RawMessage, error) {
	log.Printf("Creating success response for ID: %v", id)
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal success response: %v", err)
	}
	return data, err
}

// createErrorResponse creates a JSON-RPC error response
func (s *MinimalServer) createErrorResponse(id interface{}, code int, message string) (json.RawMessage, error) {
	log.Printf("Creating error response for ID: %v, code: %d, message: %s", id, code, message)
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal error response: %v", err)
	}
	return data, err
}

// handleInitialize handles the initialize request (minimal)
func (s *MinimalServer) handleInitialize(id interface{}) (interface{}, error) {
	log.Printf("Processing minimal initialize request for ID: %v", id)
	
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    "MCP Context Server (Minimal)",
			"version": "1.0.0-minimal",
		},
	}
	
	log.Printf("Minimal initialize response prepared: %+v", result)
	return result, nil
}

// handleToolsList returns empty tools list
func (s *MinimalServer) handleToolsList() (interface{}, error) {
	log.Printf("Processing minimal tools/list request")
	result := map[string]interface{}{
		"tools": []interface{}{},
	}
	log.Printf("Empty tools list prepared")
	return result, nil
}