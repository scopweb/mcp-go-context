package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/scopweb/mcp-go-context/internal/analyzer"
	"github.com/scopweb/mcp-go-context/internal/buildinfo"
	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/dashboard"
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

	dashboardHandler, err := dashboard.New(memoryManager)
	if err != nil {
		return nil, fmt.Errorf("failed to create dashboard handler: %w", err)
	}

	// Create server
	srv := &Server{
		config:    cfg,
		transport: trans,
		analyzer:  projectAnalyzer,
		memory:    memoryManager,
		tools:     tools.NewRegistry(),
	}

	if routeConfigurer, ok := trans.(transport.HTTPRouteConfigurer); ok {
		routeConfigurer.SetRouteRegistrar(dashboardHandler.RegisterRoutes)
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
		Version: buildinfo.Version,
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
		return s.createErrorResponse(nil, -32700, fmt.Sprintf("Parse error: %v", err))
	}

	// Validate JSON-RPC version
	if baseReq.JSONRPC != "2.0" {
		return s.createErrorResponse(baseReq.ID, -32600, "Invalid Request: jsonrpc must be '2.0'")
	}

	log.Printf("Handling request: %s (ID: %v)", baseReq.Method, baseReq.ID)

	var result interface{}
	var err error

	switch baseReq.Method {
	case "initialize":
		result, err = s.handleInitialize(baseReq.ID)
	case "tools/list":
		result, err = s.handleToolsList()
	case "tools/call":
		result, err = s.handleToolCall(req)
	case "notifications/initialized":
		// Handle initialization notification (no response needed)
		return nil, nil
	default:
		return s.createErrorResponse(baseReq.ID, -32601, fmt.Sprintf("Method not found: %s", baseReq.Method))
	}

	if err != nil {
		return s.createErrorResponse(baseReq.ID, -32603, err.Error())
	}

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
			"tools": map[string]bool{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]string{
			"name":    "MCP Context Server",
			"version": buildinfo.Version,
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

// GetAnalyzer returns the project analyzer
func (s *Server) GetAnalyzer() tools.AnalyzerInterface {
	return s.analyzer
}

// GetMemory returns the memory manager
func (s *Server) GetMemory() tools.MemoryInterface {
	return s.memory
}

// GetConfig returns the server configuration
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

	// changed-files-context tool
	s.tools.Register(&tools.Tool{
		Name:        "changed-files-context",
		Description: "Gets context from files changed in recent git commits",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Project path (default: current directory)",
				},
				"commitCount": map[string]interface{}{
					"type":        "integer",
					"description": "Number of recent commits to analyze (default: 5)",
				},
				"maxFiles": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of files to include (default: 10)",
				},
			},
		},
		Handler: tools.ChangedFilesContextHandler,
	})

	// search-memory tool
	s.tools.Register(&tools.Tool{
		Name:        "search-memory",
		Description: "Advanced search through conversation memory with ranking by relevance, recency, and usage",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query to match against memory content",
				},
				"tags": map[string]interface{}{
					"type":        "array",
					"description": "Filter by tags",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results (default: 10)",
				},
			},
		},
		Handler: tools.SearchMemoryHandler,
	})

	// save-decision tool
	s.tools.Register(&tools.Tool{
		Name:        "save-decision",
		Description: "Records a technical decision with structured metadata for future reference",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{
					"type":        "string",
					"description": "Unique identifier for this decision",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Description of the decision made",
				},
				"decisionType": map[string]interface{}{
					"type":        "string",
					"description": "Type: architecture, fix, approach, tech-debt, security, performance",
				},
				"reason": map[string]interface{}{
					"type":        "string",
					"description": "Why this decision was made",
				},
				"alternatives": map[string]interface{}{
					"type":        "array",
					"description": "Alternative approaches that were considered",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"tags": map[string]interface{}{
					"type":        "array",
					"description": "Additional tags",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required": []string{"key", "content"},
		},
		Handler: tools.SaveDecisionHandler,
	})

	// get-decisions tool
	s.tools.Register(&tools.Tool{
		Name:        "get-decisions",
		Description: "Retrieves technical decisions, optionally filtered by type or keyword",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"decisionType": map[string]interface{}{
					"type":        "string",
					"description": "Filter by decision type",
				},
				"keyword": map[string]interface{}{
					"type":        "string",
					"description": "Search keyword in decision content",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results (default: 10)",
				},
			},
		},
		Handler: tools.GetDecisionsHandler,
	})
}
