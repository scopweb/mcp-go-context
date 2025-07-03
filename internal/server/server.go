package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/scopweb/mcp-go-context/internal/analyzer"
	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/memory"
	"github.com/scopweb/mcp-go-context/internal/tools"
	"github.com/scopweb/mcp-go-context/internal/transport"
)

// Server represents the MCP Context Server
type Server struct {
	config    *config.Config
	transport transport.Transport
	analyzer  *analyzer.ProjectAnalyzer
	memory    *memory.Manager
	tools     *tools.Registry
}

// New creates a new MCP Context Server
func New(cfg *config.Config) (*Server, error) {
	// Initialize transport
	var trans transport.Transport
	var err error

	switch cfg.Transport.Type {
	case "stdio":
		trans = transport.NewStdioTransport()
	case "http":
		trans = transport.NewHTTPTransport(cfg.Transport.Port)
	case "sse":
		trans = transport.NewSSETransport(cfg.Transport.Port)
	default:
		return nil, fmt.Errorf("unknown transport type: %s", cfg.Transport.Type)
	}

	// Initialize components
	projectAnalyzer, err := analyzer.New(cfg.Context)
	if err != nil {
		return nil, fmt.Errorf("failed to create analyzer: %w", err)
	}

	memoryManager, err := memory.New(cfg.Memory)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory manager: %w", err)
	}

	// Create server
	srv := &Server{
		config:    cfg,
		transport: trans,
		analyzer:  projectAnalyzer,
		memory:    memoryManager,
		tools:     tools.NewRegistry(),
	}

	// Register tools
	srv.registerTools()

	return srv, nil
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	// Initialize server info
	info := transport.ServerInfo{
		Name:    "MCP Context Server",
		Version: "1.0.0",
		Instructions: `This server provides intelligent context management for coding assistance.
It analyzes your project, fetches relevant documentation, and maintains conversation memory.`,
	}

	// Start transport
	return s.transport.Start(ctx, info, s.handleRequest)
}

// handleRequest handles incoming MCP requests with proper JSON-RPC structure
func (s *Server) handleRequest(ctx context.Context, req json.RawMessage) (json.RawMessage, error) {
	// Parse base request to get method and ID
	var baseReq struct {
		JSONRPC string      `json:"jsonrpc"`
		Method  string      `json:"method"`
		ID      interface{} `json:"id"`
		Params  interface{} `json:"params,omitempty"`
	}

	if err := json.Unmarshal(req, &baseReq); err != nil {
		log.Printf("Parse error: %v", err)
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
		log.Printf("Processing initialize request")
		result, err = s.handleInitialize(baseReq.ID)
		if err != nil {
			log.Printf("Initialize error: %v", err)
		} else {
			log.Printf("Initialize successful")
		}
	case "tools/list":
		log.Printf("Processing tools/list request")
		result, err = s.handleToolsList()
	case "tools/call":
		log.Printf("Processing tools/call request")
		result, err = s.handleToolCall(req)
	case "notifications/initialized":
		// Handle initialization notification (no response needed)
		log.Printf("Received initialized notification")
		return nil, nil
	case "notifications/cancelled":
		// Handle cancellation notification (no response needed)
		log.Printf("Received cancelled notification")
		return nil, nil
	default:
		log.Printf("Unknown method: %s", baseReq.Method)
		return s.createErrorResponse(baseReq.ID, -32601, fmt.Sprintf("Method not found: %s", baseReq.Method))
	}

	if err != nil {
		log.Printf("Request error: %v", err)
		return s.createErrorResponse(baseReq.ID, -32603, err.Error())
	}

	log.Printf("Request successful for method: %s", baseReq.Method)
	return s.createSuccessResponse(baseReq.ID, result)
}

// createSuccessResponse creates a JSON-RPC success response
func (s *Server) createSuccessResponse(id interface{}, result interface{}) (json.RawMessage, error) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	return json.Marshal(response)
}

// createErrorResponse creates a JSON-RPC error response
func (s *Server) createErrorResponse(id interface{}, code int, message string) (json.RawMessage, error) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	return json.Marshal(response)
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(id interface{}) (interface{}, error) {
	return map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    "MCP Context Server",
			"version": "1.0.0",
		},
	}, nil
}

// handleToolsList returns available tools
func (s *Server) handleToolsList() (interface{}, error) {
	toolList := s.tools.List()
	return map[string]interface{}{
		"tools": toolList,
	}, nil
}

// handleToolCall executes a tool
func (s *Server) handleToolCall(req json.RawMessage) (interface{}, error) {
	var toolReq struct {
		Params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		} `json:"params"`
	}

	if err := json.Unmarshal(req, &toolReq); err != nil {
		return nil, fmt.Errorf("invalid tool call request: %w", err)
	}

	// Execute tool
	result, err := s.tools.Execute(toolReq.Params.Name, toolReq.Params.Arguments, s)
	if err != nil {
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	return map[string]interface{}{
		"content": result,
	}, nil
}

// GetAnalyzer returns the project analyzer (implements AnalyzerInterface)
func (s *Server) GetAnalyzer() tools.AnalyzerInterface {
	return s.analyzer
}

// GetMemory returns the memory manager (implements MemoryInterface)
func (s *Server) GetMemory() tools.MemoryInterface {
	return s.memory
}

// GetConfig returns the server configuration (implements ConfigInterface)
func (s *Server) GetConfig() tools.ConfigInterface {
	return s.config
}

// registerTools registers all available tools to the server
func (s *Server) registerTools() {
	// analyze-project tool
	s.tools.Register(&tools.Tool{
		Name:        "analyze-project",
		Description: "Analyzes the project structure, dependencies, and provides comprehensive context",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Project path to analyze (default: current directory)",
				},
				"depth": map[string]interface{}{
					"type":        "integer",
					"description": "Analysis depth (default: 3)",
				},
			},
		},
		Handler: tools.AnalyzeProjectHandler,
	})

	// get-context tool
	s.tools.Register(&tools.Tool{
		Name:        "get-context",
		Description: "Retrieves relevant context for the current task based on files, dependencies, and conversation history",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Context query or topic",
				},
				"files": map[string]interface{}{
					"type":        "array",
					"description": "Specific files to include in context",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"maxTokens": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum tokens to return",
				},
			},
			"required": []string{"query"},
		},
		Handler: tools.GetContextHandler,
	})

	// fetch-docs tool
	s.tools.Register(&tools.Tool{
		Name:        "fetch-docs",
		Description: "Fetches documentation for libraries and dependencies",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"library": map[string]interface{}{
					"type":        "string",
					"description": "Library name to fetch docs for",
				},
				"version": map[string]interface{}{
					"type":        "string",
					"description": "Specific version (optional)",
				},
				"topic": map[string]interface{}{
					"type":        "string",
					"description": "Specific topic within the docs",
				},
			},
			"required": []string{"library"},
		},
		Handler: tools.FetchDocsHandler,
	})

	// remember-conversation tool
	s.tools.Register(&tools.Tool{
		Name:        "remember-conversation",
		Description: "Stores important context from the current conversation for future reference",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{
					"type":        "string",
					"description": "Key to store the memory under",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to remember",
				},
				"tags": map[string]interface{}{
					"type":        "array",
					"description": "Tags for categorization",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required": []string{"key", "content"},
		},
		Handler: tools.RememberConversationHandler,
	})

	// dependency-analysis tool
	s.tools.Register(&tools.Tool{
		Name:        "dependency-analysis",
		Description: "Analyzes project dependencies and suggests relevant documentation",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"includeTransitive": map[string]interface{}{
					"type":        "boolean",
					"description": "Include transitive dependencies",
				},
				"onlyDirect": map[string]interface{}{
					"type":        "boolean",
					"description": "Only analyze direct dependencies",
				},
			},
		},
		Handler: tools.DependencyAnalysisHandler,
	})
}