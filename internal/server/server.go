package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/scopweb/mcp-go-context/internal/analyzer"
	"github.com/scopweb/mcp-go-context/internal/auth"
	"github.com/scopweb/mcp-go-context/internal/cache"
	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/memory"
	"github.com/scopweb/mcp-go-context/internal/tools"
	"github.com/scopweb/mcp-go-context/internal/transport"
)

// Server represents the MCP Context Server
type Server struct {
	config       *config.Config
	transport    transport.Transport
	analyzer     *analyzer.ProjectAnalyzer
	memory       *memory.Manager
	tools        *tools.Registry
	jwtManager   *auth.JWTManager
	contextCache *cache.ContextCache
	sessions     map[string]*sessionState
	sessionMu    sync.RWMutex
}

type sessionState struct {
	protocolVersion string
	initialized     bool
	ready           bool
}

type protocolError struct {
	code    int
	message string
}

func (e *protocolError) Error() string {
	return e.message
}

type initializeRequest struct {
	ProtocolVersion string `json:"protocolVersion"`
}

var supportedProtocolVersions = map[string]struct{}{
	"2024-11-05": {},
	"2025-03-26": {},
	"2025-06-18": {},
	"2025-11-25": {},
}

const latestProtocolVersion = "2025-11-25"

const (
	serverName    = "mcp-go-context"
	serverTitle   = "MCP Go Context"
	serverVersion = "2.1.1"
)

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

	// Initialize context cache (1000 items, 30 minute TTL)
	contextCache := cache.NewContextCache(1000, 30*time.Minute)

	// Create server
	srv := &Server{
		config:       cfg,
		transport:    trans,
		analyzer:     projectAnalyzer,
		memory:       memoryManager,
		tools:        tools.NewRegistry(),
		jwtManager:   jwtManager,
		contextCache: contextCache,
		sessions:     make(map[string]*sessionState),
	}

	// Register tools
	srv.registerTools()

	return srv, nil
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	// Initialize server info
	info := transport.ServerInfo{
		Name:         serverName,
		Version:      serverVersion,
		Instructions: s.buildInstructions(),
	}

	// Start transport
	return s.transport.Start(ctx, info, s.HandleRequest)
}

// HandleRequest handles incoming MCP requests with proper JSON-RPC structure
func (s *Server) HandleRequest(ctx context.Context, req json.RawMessage) (json.RawMessage, error) {
	// Parse base request to get method and ID
	var baseReq struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		ID      interface{}     `json:"id"`
		Params  json.RawMessage `json:"params,omitempty"`
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

	if err := validateRequestID(baseReq.Method, baseReq.ID); err != nil {
		return s.createErrorResponse(baseReq.ID, -32600, err.Error())
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

	session := s.getSessionState(ctx)
	if baseReq.Method != "initialize" && baseReq.Method != "ping" && baseReq.Method != "notifications/cancelled" {
		if !session.initialized {
			return s.createErrorResponse(baseReq.ID, -32600, "Invalid Request: initialize must be the first request in the session")
		}
	}
	if isOperationalMethod(baseReq.Method) && !session.ready {
		return s.createErrorResponse(baseReq.ID, -32600, "Invalid Request: session is not ready until notifications/initialized is received")
	}

	var result interface{}
	var err error

	switch baseReq.Method {
	case "initialize":
		log.Printf("Processing initialize request")
		result, err = s.handleInitialize(ctx, baseReq.Params)
		if err != nil {
			log.Printf("Initialize error: %v", err)
		} else {
			log.Printf("Initialize successful")
		}
	case "ping":
		result = map[string]interface{}{}
	case "tools/list":
		log.Printf("Processing tools/list request")
		result, err = s.handleToolsList()
	case "tools/call":
		log.Printf("Processing tools/call request")
		result, err = s.handleToolCall(req)
	case "notifications/initialized":
		log.Printf("Received initialized notification")
		s.markSessionReady(ctx)
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
		var requestErr *protocolError
		if errors.As(err, &requestErr) {
			return s.createErrorResponse(baseReq.ID, requestErr.code, requestErr.message)
		}
		return s.createErrorResponse(baseReq.ID, -32603, err.Error())
	}

	log.Printf("Request successful for method: %s", baseReq.Method)
	return s.createSuccessResponse(baseReq.ID, result)
}

func validateRequestID(method string, id interface{}) error {
	if strings.HasPrefix(method, "notifications/") {
		if id != nil {
			return fmt.Errorf("Invalid Request: notifications must not include id")
		}
		return nil
	}

	if id == nil {
		return fmt.Errorf("Invalid Request: id must be a string or integer")
	}

	switch id.(type) {
	case string, float64, int, int32, int64, uint32, uint64:
		return nil
	default:
		return fmt.Errorf("Invalid Request: id must be a string or integer")
	}
}

func isOperationalMethod(method string) bool {
	return method == "tools/list" || method == "tools/call"
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
func (s *Server) handleInitialize(ctx context.Context, rawParams json.RawMessage) (interface{}, error) {
	var params initializeRequest
	if len(rawParams) > 0 {
		if err := json.Unmarshal(rawParams, &params); err != nil {
			return nil, fmt.Errorf("invalid initialize params: %w", err)
		}
	}

	protocolVersion := negotiateProtocolVersion(params.ProtocolVersion)
	s.markSessionInitialized(ctx, protocolVersion)

	return map[string]interface{}{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":     serverName,
			"title":    serverTitle,
			"version":  serverVersion,
			"protocol": protocolVersion,
			"features": []string{
				"project-analysis",
				"persistent-memory",
				"documentation-fetching",
				"jwt-authentication",
				"cors-security",
				"streamable-transport",
			},
		},
		"instructions": s.buildInstructions(),
	}, nil
}

func negotiateProtocolVersion(clientVersion string) string {
	if clientVersion == "" {
		return latestProtocolVersion
	}
	if _, ok := supportedProtocolVersions[clientVersion]; ok {
		return clientVersion
	}
	return latestProtocolVersion
}

func (s *Server) buildInstructions() string {
	toolList := s.tools.List()
	toolIndex := make(map[string]map[string]interface{}, len(toolList))
	names := make([]string, 0, len(toolList))
	for _, tool := range toolList {
		name, _ := tool["name"].(string)
		if name == "" {
			continue
		}
		names = append(names, name)
		toolIndex[name] = tool
	}
	sort.Strings(names)

	var builder strings.Builder
	builder.WriteString("This server is optimized for Claude Code and Claude Desktop workflows that need reusable repository context, durable decisions, and fast focused retrieval.\n\n")
	builder.WriteString("Selection rules for Claude Code and Claude Desktop:\n")
	builder.WriteString("- Use analyze-project first for a fresh repository, architecture review, onboarding request, or when you need a map of the codebase before drilling in.\n")
	builder.WriteString("- Use config-get-project-paths before analyze-project when workspace roots are unclear or when the session starts without obvious repository context.\n")
	builder.WriteString("- Use get-context for targeted code questions after you know the relevant files, area, bug, symbol, or subsystem. Pass specific files when possible.\n")
	builder.WriteString("- Use dependency-analysis for package review, dependency risk, or deciding what external docs should be consulted next.\n")
	builder.WriteString("- Use fetch-docs only for external library or API documentation. Include library, and version or topic when known.\n")
	builder.WriteString("- Before answering historical or architectural questions, check memory-search or memory-recent so prior decisions are reused across threads and fresh Claude sessions.\n")
	builder.WriteString("- Use memory-get when the user references an exact saved key or when a stable key is already known.\n")
	builder.WriteString("- Use remember-conversation only for durable facts: accepted designs, implementation decisions, debugging outcomes, stable conventions, deployment notes, or reusable task summaries.\n")
	builder.WriteString("- When saving memory, prefer stable keys such as repo/component/topic, team/decision/name, or bugfix/area/symptom. Add concise tags so future searches succeed.\n")
	builder.WriteString("- For Claude Code refactors or long task chains, store the final outcome with remember-conversation if it will matter in another thread.\n")
	builder.WriteString("- For Claude Desktop follow-up sessions, rebuild state with memory-recent or memory-search instead of repeating full project analysis unless the repo changed materially.\n")
	builder.WriteString("- Use auth-generate-token only for development or testing flows when JWT authentication is enabled.\n")
	builder.WriteString("- Do not use memory-clear unless the user explicitly asks to wipe memory and provides the required confirmation string.\n")
	builder.WriteString("- Prefer the narrowest tool that answers the request. Do not repeatedly call analyze-project if a focused tool can answer the next step.\n")
	builder.WriteString("- Inspect tools/list before tools/call if the required arguments are unclear.\n\n")
	builder.WriteString("Available tools:\n")
	for _, name := range names {
		builder.WriteString(formatInstructionToolLine(toolIndex[name]))
	}
	return builder.String()
}

func formatInstructionToolLine(tool map[string]interface{}) string {
	name, _ := tool["name"].(string)
	title, _ := tool["title"].(string)
	description, _ := tool["description"].(string)

	var builder strings.Builder
	builder.WriteString("- ")
	builder.WriteString(name)
	if title != "" {
		builder.WriteString(" (")
		builder.WriteString(title)
		builder.WriteString(")")
	}
	if description != "" {
		builder.WriteString(": ")
		builder.WriteString(description)
	}
	builder.WriteString("\n")
	return builder.String()
}

func toolAnnotations(readOnly, destructive, idempotent, openWorld bool) map[string]interface{} {
	return map[string]interface{}{
		"readOnlyHint":    readOnly,
		"destructiveHint": destructive,
		"idempotentHint":  idempotent,
		"openWorldHint":   openWorld,
	}
}

func dependencyItemSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name":    map[string]interface{}{"type": "string"},
			"version": map[string]interface{}{"type": "string"},
			"type":    map[string]interface{}{"type": "string"},
			"path":    map[string]interface{}{"type": "string"},
		},
		"required": []string{"name", "version", "type", "path"},
	}
}

func fileSummarySchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":     map[string]interface{}{"type": "string"},
			"language": map[string]interface{}{"type": "string"},
			"size":     map[string]interface{}{"type": "integer"},
		},
		"required": []string{"path", "language", "size"},
	}
}

func memoryItemSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"key":       map[string]interface{}{"type": "string"},
			"content":   map[string]interface{}{"type": "string"},
			"tags":      map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"timestamp": map[string]interface{}{"type": "string"},
			"usage":     map[string]interface{}{"type": "integer"},
		},
		"required": []string{"key", "content", "tags", "timestamp", "usage"},
	}
}

func analyzeProjectOutputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"rootPath":             map[string]interface{}{"type": "string"},
			"totalFiles":           map[string]interface{}{"type": "integer"},
			"totalSize":            map[string]interface{}{"type": "integer"},
			"languages":            map[string]interface{}{"type": "object", "additionalProperties": map[string]interface{}{"type": "integer"}},
			"directories":          map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"directDependencies":   map[string]interface{}{"type": "array", "items": dependencyItemSchema()},
			"indirectDependencies": map[string]interface{}{"type": "array", "items": dependencyItemSchema()},
			"keyFiles":             map[string]interface{}{"type": "array", "items": fileSummarySchema()},
		},
		"required": []string{"rootPath", "totalFiles", "totalSize", "languages", "directories", "directDependencies", "indirectDependencies", "keyFiles"},
	}
}

func dependencyAnalysisOutputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"directDependencies":   map[string]interface{}{"type": "array", "items": dependencyItemSchema()},
			"indirectDependencies": map[string]interface{}{"type": "array", "items": dependencyItemSchema()},
			"recommendations":      map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
		},
		"required": []string{"directDependencies", "indirectDependencies", "recommendations"},
	}
}

func memoryGetOutputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"memory": memoryItemSchema(),
		},
		"required": []string{"memory"},
	}
}

func memoryListOutputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"count":    map[string]interface{}{"type": "integer"},
			"memories": map[string]interface{}{"type": "array", "items": memoryItemSchema()},
		},
		"required": []string{"count", "memories"},
	}
}

func getContextOutputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query":     map[string]interface{}{"type": "string"},
			"files":     map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"maxTokens": map[string]interface{}{"type": "integer"},
			"source":    map[string]interface{}{"type": "string"},
			"text":      map[string]interface{}{"type": "string"},
		},
		"required": []string{"query", "files", "maxTokens", "source", "text"},
	}
}

func fetchDocsOutputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"library": map[string]interface{}{"type": "string"},
			"version": map[string]interface{}{"type": "string"},
			"topic":   map[string]interface{}{"type": "string"},
			"source":  map[string]interface{}{"type": "string"},
			"text":    map[string]interface{}{"type": "string"},
		},
		"required": []string{"library", "version", "topic", "source", "text"},
	}
}

func configPathsOutputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"count": map[string]interface{}{"type": "integer"},
			"paths": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
		},
		"required": []string{"count", "paths"},
	}
}

func rememberConversationOutputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"stored": map[string]interface{}{"type": "boolean"},
			"key":    map[string]interface{}{"type": "string"},
			"tags":   map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
		},
		"required": []string{"stored", "key", "tags"},
	}
}

func (s *Server) getSessionState(ctx context.Context) *sessionState {
	key := sessionKeyFromContext(ctx)

	s.sessionMu.RLock()
	state, ok := s.sessions[key]
	s.sessionMu.RUnlock()
	if ok {
		return state
	}

	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	if state, ok = s.sessions[key]; ok {
		return state
	}
	state = &sessionState{}
	s.sessions[key] = state
	return state
}

func (s *Server) markSessionInitialized(ctx context.Context, protocolVersion string) {
	state := s.getSessionState(ctx)
	s.sessionMu.Lock()
	state.initialized = true
	state.ready = false
	state.protocolVersion = protocolVersion
	s.sessionMu.Unlock()
}

func (s *Server) markSessionReady(ctx context.Context) {
	state := s.getSessionState(ctx)
	s.sessionMu.Lock()
	if state.initialized {
		state.ready = true
	}
	s.sessionMu.Unlock()
}

func sessionKeyFromContext(ctx context.Context) string {
	if sessionID, ok := ctx.Value("sessionID").(string); ok && sessionID != "" {
		return sessionID
	}
	if transportType, ok := ctx.Value("transportType").(string); ok && transportType != "" {
		return transportType
	}
	return "default"
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
		return nil, &protocolError{code: transport.ErrorCodeInvalidParams, message: fmt.Sprintf("Invalid tool call request: %v", err)}
	}

	// Validar nombre de herramienta: solo letras, números, guiones y guiones bajos
	validName := regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)
	if !validName.MatchString(toolReq.Params.Name) {
		log.Printf("Rejected tool call with invalid name: %s", toolReq.Params.Name)
		return nil, &protocolError{code: transport.ErrorCodeInvalidParams, message: "Invalid tool name"}
	}

	// Comprobar que la herramienta existe
	if _, exists := s.tools.Get(toolReq.Params.Name); !exists {
		log.Printf("Attempt to call unregistered tool: %s", toolReq.Params.Name)
		return nil, &protocolError{code: transport.ErrorCodeMethodNotFound, message: fmt.Sprintf("Tool not found: %s", toolReq.Params.Name)}
	}

	// Execute tool
	result, err := s.tools.Execute(toolReq.Params.Name, toolReq.Params.Arguments, s)
	if err != nil {
		log.Printf("Tool execution failed: %v", err)
		return s.toolErrorResult(fmt.Sprintf("%v", err)), nil
	}

	if callResult, ok := result.(tools.CallResult); ok {
		response := map[string]interface{}{
			"content": callResult.Content,
			"isError": callResult.IsError,
		}
		if len(callResult.StructuredContent) > 0 {
			response["structuredContent"] = callResult.StructuredContent
		}
		return response, nil
	}

	return map[string]interface{}{
		"content": result,
	}, nil
}

func (s *Server) toolErrorResult(message string) map[string]interface{} {
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("❌ Error: %s", message),
			},
		},
		"isError": true,
	}
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

// GetContextCache returns the context cache
func (s *Server) GetContextCache() *cache.ContextCache {
	return s.contextCache
}

// registerTools registers all available tools to the server
func (s *Server) registerTools() {
	// analyze-project tool
	s.tools.Register(&tools.Tool{
		Name:        "analyze-project",
		Title:       "Analyze Project",
		Description: "Start here for a new repository or broad architecture question; returns structure, key files, and dependencies",
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
		OutputSchema: analyzeProjectOutputSchema(),
		Annotations:  toolAnnotations(true, false, true, false),
		Handler:      tools.AnalyzeProjectHandler,
	})

	// get-context tool
	s.tools.Register(&tools.Tool{
		Name:        "get-context",
		Title:       "Get Relevant Context",
		Description: "Use after repository discovery to retrieve focused code and memory context for a specific bug, symbol, file set, or task",
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
		OutputSchema: getContextOutputSchema(),
		Annotations:  toolAnnotations(true, false, true, false),
		Handler:      tools.GetContextHandler,
	})

	// fetch-docs tool
	s.tools.Register(&tools.Tool{
		Name:        "fetch-docs",
		Title:       "Fetch Library Documentation",
		Description: "Fetch external library or API documentation when the answer depends on behavior outside the local repository",
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
		OutputSchema: fetchDocsOutputSchema(),
		Annotations:  toolAnnotations(true, false, true, true),
		Handler:      tools.FetchDocsHandler,
	})

	// remember-conversation tool
	s.tools.Register(&tools.Tool{
		Name:        "remember-conversation",
		Title:       "Remember Conversation Context",
		Description: "Persist durable project decisions, conventions, and task outcomes under a stable key for reuse across sessions and threads",
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
		OutputSchema: rememberConversationOutputSchema(),
		Annotations:  toolAnnotations(false, false, false, false),
		Handler:      tools.RememberConversationHandler,
	})

	// dependency-analysis tool
	s.tools.Register(&tools.Tool{
		Name:        "dependency-analysis",
		Title:       "Analyze Dependencies",
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
		OutputSchema: dependencyAnalysisOutputSchema(),
		Annotations:  toolAnnotations(true, false, true, false),
		Handler:      tools.DependencyAnalysisHandler,
	})

	// memory/get
	s.tools.Register(&tools.Tool{
		Name:        "memory-get",
		Title:       "Get Memory Item",
		Description: "Retrieve an exact saved memory by key when the key is already known",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{"type": "string"},
			},
			"required": []string{"key"},
		},
		OutputSchema: memoryGetOutputSchema(),
		Annotations:  toolAnnotations(true, false, true, false),
		Handler:      tools.MemoryGetHandler,
	})

	// memory/search
	s.tools.Register(&tools.Tool{
		Name:        "memory-search",
		Title:       "Search Memories",
		Description: "Search saved context by query or tags before re-analyzing the repository or repeating prior decisions",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{"type": "string"},
				"tags":  map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				"limit": map[string]interface{}{"type": "integer"},
			},
		},
		OutputSchema: memoryListOutputSchema(),
		Annotations:  toolAnnotations(true, false, true, false),
		Handler:      tools.MemorySearchHandler,
	})

	// memory/recent
	s.tools.Register(&tools.Tool{
		Name:        "memory-recent",
		Title:       "List Recent Memories",
		Description: "List recent saved context to quickly rebuild state in a fresh Claude session",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{"type": "integer"},
			},
		},
		OutputSchema: memoryListOutputSchema(),
		Annotations:  toolAnnotations(true, false, true, false),
		Handler:      tools.MemoryRecentHandler,
	})

	// memory/clear (dangerous)
	s.tools.Register(&tools.Tool{
		Name:        "memory-clear",
		Title:       "Clear All Memories",
		Description: "Clear all memories (dangerous)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"confirm": map[string]interface{}{"type": "string", "description": "Type YES_I_UNDERSTAND to proceed"},
			},
			"required": []string{"confirm"},
		},
		Annotations: toolAnnotations(false, true, false, false),
		Handler:     tools.MemoryClearHandler,
	})

	// config/get-project-paths
	s.tools.Register(&tools.Tool{
		Name:         "config-get-project-paths",
		Title:        "Get Configured Project Paths",
		Description:  "Return configured workspace and repository roots before analysis or context retrieval",
		InputSchema:  map[string]interface{}{"type": "object"},
		OutputSchema: configPathsOutputSchema(),
		Annotations:  toolAnnotations(true, false, true, false),
		Handler:      tools.ConfigGetProjectPathsHandler,
	})

	// auth/generate-token (for development/testing)
	s.tools.Register(&tools.Tool{
		Name:        "auth-generate-token",
		Title:       "Generate JWT Token",
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
		Annotations: toolAnnotations(true, false, false, false),
		Handler:     s.generateTokenHandler,
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
