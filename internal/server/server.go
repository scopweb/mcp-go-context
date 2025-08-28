package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/scopweb/mcp-go-context/internal/analyzer"
	"github.com/scopweb/mcp-go-context/internal/auth"
	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/memory"
	"github.com/scopweb/mcp-go-context/internal/tools"
	"github.com/scopweb/mcp-go-context/internal/transport"
)

// Server represents the MCP Context Server
type Server struct {
	config      *config.Config
	transport   transport.Transport
	analyzer    *analyzer.ProjectAnalyzer
	memory      *memory.Manager
	tools       *tools.Registry
	jwtManager  *auth.JWTManager
	// future: add sync.Mutex if shared state is added
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
		trans = transport.NewHTTPTransportWithCORS(cfg.Transport.Port, cfg.Security.CORS)
	case "sse":
		trans = transport.NewSSETransportWithCORS(cfg.Transport.Port, cfg.Security.CORS)
	case "streamable-http", "streamable":
		trans = transport.NewStreamableHTTPTransport(cfg.Transport.Port, cfg.Security.CORS)
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

	// Initialize JWT manager
	jwtSecret := os.Getenv("MCP_JWT_SECRET")
	if jwtSecret != "" {
		cfg.Security.Auth.Secret = jwtSecret
		cfg.Security.Auth.Enabled = true
	}

	jwtManager := auth.NewJWTManager(auth.JWTConfig{
		Secret:    cfg.Security.Auth.Secret,
		Expiry:    cfg.Security.Auth.Expiry,
		Issuer:    cfg.Security.Auth.Issuer,
		Algorithm: cfg.Security.Auth.Algorithm,
	})

	// Create server
	srv := &Server{
		config:     cfg,
		transport:  trans,
		analyzer:   projectAnalyzer,
		memory:     memoryManager,
		tools:      tools.NewRegistry(),
		jwtManager: jwtManager,
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
	return s.transport.Start(ctx, info, s.HandleRequest)
}

// HandleRequest handles incoming MCP requests with proper JSON-RPC structure
func (s *Server) HandleRequest(ctx context.Context, req json.RawMessage) (json.RawMessage, error) {
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

	// JWT Authentication for HTTP/SSE (if enabled)
	if s.config.Security.Auth.Enabled && s.jwtManager.IsEnabled() {
		// Only authenticate for HTTP/SSE requests, not stdio
		if r, ok := ctx.Value("httpRequest").(*http.Request); ok {
			authHeader := r.Header.Get("Authorization")
			
			token, err := auth.ExtractTokenFromHeader(authHeader)
			if err != nil {
				log.Printf("Auth header error: %v", err)
				return s.createErrorResponse(baseReq.ID, -32000, "Unauthorized: "+err.Error())
			}

			claims, err := s.jwtManager.ValidateToken(token)
			if err != nil {
				log.Printf("Token validation failed: %v", err)
				return s.createErrorResponse(baseReq.ID, -32000, "Unauthorized: "+err.Error())
			}

			log.Printf("Authenticated request from subject: %s", claims.Subject)
		}
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
		"protocolVersion": "2025-03-26", // Updated to MCP 2025
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
			"sampling": map[string]interface{}{
				"supports": false, // Can be enabled for future LLM sampling features
			},
			"roots": map[string]interface{}{
				"listChanged": false,
			},
			"resources": map[string]interface{}{
				"subscribe":   false,
				"listChanged": false,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    "MCP Context Server",
			"version": "2.0.0", // Updated version
			"protocol": "2025-03-26",
			"features": []string{
				"project-analysis",
				"persistent-memory", 
				"documentation-fetching",
				"jwt-authentication",
				"cors-security",
				"streamable-transport",
			},
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

	// Validar nombre de herramienta: solo letras, números, guiones y guiones bajos
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	if !validName.MatchString(toolReq.Params.Name) {
		log.Printf("Rejected tool call with invalid name: %s", toolReq.Params.Name)
		return nil, fmt.Errorf("invalid tool name")
	}

	// Comprobar que la herramienta existe
	if _, exists := s.tools.Get(toolReq.Params.Name); !exists {
		log.Printf("Attempt to call unregistered tool: %s", toolReq.Params.Name)
		return nil, fmt.Errorf("tool not found")
	}

	// Execute tool
	result, err := s.tools.Execute(toolReq.Params.Name, toolReq.Params.Arguments, s)
	if err != nil {
		log.Printf("Tool execution failed: %v", err)
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

	// memory/get
	s.tools.Register(&tools.Tool{
		Name:        "memory-get",
		Description: "Retrieve a memory item by key",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{"type": "string"},
			},
			"required": []string{"key"},
		},
		Handler: tools.MemoryGetHandler,
	})

	// memory/search
	s.tools.Register(&tools.Tool{
		Name:        "memory-search",
		Description: "Search memories by query or tags",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string"},
				"tags":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				"limit": map[string]interface{}{"type": "integer"},
			},
		},
		Handler: tools.MemorySearchHandler,
	})

	// memory/recent
	s.tools.Register(&tools.Tool{
		Name:        "memory-recent",
		Description: "Get recent memories",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "integer"},
			},
		},
		Handler: tools.MemoryRecentHandler,
	})

	// memory/clear (dangerous)
	s.tools.Register(&tools.Tool{
		Name:        "memory-clear",
		Description: "Clear all memories (dangerous)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"confirm": map[string]interface{}{"type": "string", "description": "Type YES_I_UNDERSTAND to proceed"},
			},
			"required": []string{"confirm"},
		},
		Handler: tools.MemoryClearHandler,
	})

	// config/get-project-paths
	s.tools.Register(&tools.Tool{
		Name:        "config-get-project-paths",
		Description: "Get configured project paths",
		InputSchema: map[string]interface{}{"type": "object"},
		Handler:     tools.ConfigGetProjectPathsHandler,
	})

	// auth/generate-token (for development/testing)
	s.tools.Register(&tools.Tool{
		Name:        "auth-generate-token",
		Description: "Generate JWT token for authentication (development use)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"subject": map[string]interface{}{
					"type":        "string",
					"description": "Subject (user identifier) for the token",
				},
			},
			"required": []string{"subject"},
		},
		Handler: s.generateTokenHandler,
	})
}

// generateTokenHandler generates a JWT token for development/testing
func (s *Server) generateTokenHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Subject string `json:"subject"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Validate subject
	if params.Subject == "" {
		return nil, fmt.Errorf("subject is required")
	}

	// Check if JWT is enabled
	if !s.config.Security.Auth.Enabled || !s.jwtManager.IsEnabled() {
		return nil, fmt.Errorf("JWT authentication is not enabled")
	}

	// Generate token
	token, err := s.jwtManager.GenerateToken(params.Subject)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	return []map[string]interface{}{
		{
			"type": "text",
			"text": fmt.Sprintf("JWT Token Generated:\n\nToken: %s\n\nUsage:\nAuthorization: Bearer %s\n\nExpires: %s", 
				token, token, 
				fmt.Sprintf("in %s", s.config.Security.Auth.Expiry.String())),
		},
	}, nil
}
