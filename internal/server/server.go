package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/scopweb/mcp-go-context/internal/analyzer"
	"github.com/scopweb/mcp-go-context/internal/buildinfo"
	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/continuity"
	"github.com/scopweb/mcp-go-context/internal/dashboard"
	"github.com/scopweb/mcp-go-context/internal/memory"
	"github.com/scopweb/mcp-go-context/internal/tools"
	"github.com/scopweb/mcp-go-context/internal/transport"
)

// Server represents the MCP Context Server
type Server struct {
	config     *config.Config
	transport  transport.Transport
	analyzer   *analyzer.ProjectAnalyzer
	memory     *memory.Manager
	continuity *continuity.Service
	tools      *tools.Registry
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

	// Infer active project slug from cwd; falls back to "default" inside InferProject.
	cwd, _ := os.Getwd()
	activeProject := memory.InferProject(cwd)

	memoryManager, err := memory.New(cfg.Memory, activeProject)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory manager: %w", err)
	}

	continuitySvc := continuity.New(cfg.Memory.StoragePath, memoryManager)

	dashboardHandler, err := dashboard.New(memoryManager, projectAnalyzer)
	if err != nil {
		return nil, fmt.Errorf("failed to create dashboard handler: %w", err)
	}

	// Create server
	srv := &Server{
		config:     cfg,
		transport:  trans,
		analyzer:   projectAnalyzer,
		memory:     memoryManager,
		continuity: continuitySvc,
		tools:      tools.NewRegistry(),
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
It analyzes your project, fetches relevant documentation, and maintains conversation memory.
Memory convergence: use promote-memory to mark high-value items for long-term persistence.
Session memory stays in summary.md; promoted memories persist in MCP memory.`,
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
		// Extract protocol version from params for negotiation
		protoVersion := "2025-11-25" // default
		if paramsMap, ok := baseReq.Params.(map[string]interface{}); ok {
			if v, ok := paramsMap["protocolVersion"].(string); ok {
				protoVersion = v
			}
		}
		result, err = s.handleInitialize(baseReq.ID, protoVersion)
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
func (s *Server) handleInitialize(id interface{}, protocolVersion string) (interface{}, error) {
	// Echo the client's protocol version for compatibility (per MCP spec)
	// Claude Desktop and Claude Code send 2025-11-25
	if protocolVersion == "" {
		protocolVersion = "2025-11-25"
	}

	return map[string]interface{}{
		"protocolVersion": protocolVersion,
		"capabilities": map[string]interface{}{
			"tools": map[string]bool{
				"listChanged": true,
			},
		},
		"serverInfo": map[string]string{
			"name":    "MCP Context Server",
			"version": buildinfo.Version,
		},
		"instructions": `This server provides project context analysis and technical memory shared between coding clients on the same machine.

CONTINUITY:
1. At the start of substantive work, call resume-context with the working path.
2. Treat files and Git as the source of truth. Use the handoff for objective, pending work and next step.
3. Pass the same path to save-decision and remember-conversation so memories stay with that project.
4. After a milestone, before switching clients, or before ending work, call save-handoff with expectedRevision.
5. A CONFLICT result preserves both versions. Reconcile them; do not overwrite either.
6. Promote only durable decisions. Do not promote transient handoffs.

MEMORY WORKFLOW:
1. Record important decisions with save-decision, including reason and alternatives.
2. Before ending complex work, call suggest-promotions.
3. Use promote-memory for items that must survive cleanup.
4. Later clients can retrieve them with resume-context, get-context and search-memory.

Available tools:
- analyze-project: Full project structure, languages, dependencies and key files.
- get-context: Best entry point for most queries. Returns relevant code + memories for a topic (automatically handles cold-start).
- changed-files-context: Gets context from recent git changes (very useful for understanding current work).
- fetch-docs: Library documentation via Context7 with local fallback.
- dependency-analysis: Project dependencies with recommendations.
- remember-conversation: Store free-form important context with tags.
- save-decision: Record a technical decision with structured metadata (decisionType, reason, alternatives). Preferred for architecture, fixes and conventions.
- get-decisions: Retrieve decisions filtered by type or keyword.
- search-memory: Full-text search across all memories with relevance + recency scoring. Use project="*" for cross-project search.
- suggest-promotions: Analyzes existing memories and returns the best candidates to promote to long-term storage. Call this regularly before ending work sessions.
- promote-memory: Marks a memory as high-value for persistent storage (the core of memory convergence).
- get-promoted-memories: Lists only the memories that have been explicitly promoted for long-term retention.`,
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

func (s *Server) Continuity() *continuity.Service {
	return s.continuity
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
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Alias for topic: free-text hint used when resolving the library",
				},
				"libraryId": map[string]interface{}{
					"type":        "string",
					"description": "Context7 library ID (e.g. /org/project). Skips the resolve step when provided",
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
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Working directory used to bind the memory to a stable project",
				},
				"project": map[string]interface{}{
					"type":        "string",
					"description": "Explicit project id. Optional when path is provided",
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
		Description: "Powerful search across all memories with smart ranking (relevance + recency + usage). Use this when get-context doesn't surface what you need. Pass project=\"*\" to search across all projects.",
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
				"project": map[string]interface{}{
					"type":        "string",
					"description": "Project slug to filter by. Omit to use the active project, '*' for all projects, 'unassigned' for legacy memories.",
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
		Description: "Records a technical decision with rich metadata. Preferred tool for architecture choices, important fixes, and team conventions. Include 'reason' and 'alternatives' when possible — this greatly increases the chance it will be suggested for long-term promotion later.",
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
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Working directory used to bind the decision to a stable project",
				},
				"project": map[string]interface{}{
					"type":        "string",
					"description": "Explicit project id. Optional when path is provided",
				},
			},
			"required": []string{"key", "content"},
		},
		Handler: tools.SaveDecisionHandler,
	})

	// get-decisions tool
	s.tools.Register(&tools.Tool{
		Name:        "get-decisions",
		Description: "Retrieves technical decisions with optional filtering by type or keyword. Complements search-memory when you want structured decision history.",
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
				"project": map[string]interface{}{
					"type":        "string",
					"description": "Project id. Omit for the active project, '*' for all, 'unassigned' for legacy memories",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results (default: 10)",
				},
			},
		},
		Handler: tools.GetDecisionsHandler,
	})

	// promote-memory tool - marks a memory as high-value for persistence (Phase 6)
	s.tools.Register(&tools.Tool{
		Name:        "promote-memory",
		Description: "Promotes a memory to long-term persistent storage. This is the core action of memory convergence. After calling suggest-promotions, use this on the best candidates. Promoted memories survive sessions and are prioritized in get-context.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{
					"type":        "string",
					"description": "Memory key to promote",
				},
				"confidence": map[string]interface{}{
					"type":        "string",
					"description": "Confidence level: low, medium, high (default: high)",
				},
			},
			"required": []string{"key"},
		},
		Handler: tools.PromoteMemoryHandler,
	})

	// get-promoted-memories tool - returns only promoted (high-value) memories
	s.tools.Register(&tools.Tool{
		Name:        "get-promoted-memories",
		Description: "Returns only the memories that have been explicitly promoted to long-term storage. These are the highest-value items that should influence future decisions. Use this to review what has been preserved across sessions.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results (default: 10)",
				},
			},
		},
		Handler: tools.GetPromotedMemoriesHandler,
	})

	// suggest-promotions tool (Fase 1) - intelligent suggestions for what to promote
	s.tools.Register(&tools.Tool{
		Name:        "suggest-promotions",
		Description: "The key tool for memory hygiene. Analyzes all memories using multiple signals (structure, usage, recency, decision language) and returns the best candidates worth promoting with promote-memory. Call this at the end of complex sessions or before context switching.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of suggestions to return (default: 5)",
				},
			},
		},
		Handler: tools.SuggestPromotionsHandler,
	})

	// memory-stats tool - aggregate memory statistics (memcached-style)
	s.tools.Register(&tools.Tool{
		Name:        "memory-stats",
		Description: "Returns aggregate memory statistics: session and memory counts, promoted/decision totals, total usage (read hits), on-disk storage size, and configured limits. Useful for checking memory hygiene before running suggest-promotions.",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		Handler: tools.MemoryStatsHandler,
	})


	s.tools.Register(&tools.Tool{
		Name:        "save-handoff",
		Description: "Saves a recoverable work summary for another coding client. Pass the working path, expectedRevision from the last resume, and the confirmed objective, pending work and next step. Does not overwrite a newer revision.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":             map[string]interface{}{"type": "string", "description": "Working directory or repository path"},
				"projectId":        map[string]interface{}{"type": "string", "description": "Stable project id, optional if path is given"},
				"handoffId":        map[string]interface{}{"type": "string", "description": "Work-context id. Defaults to the current branch"},
				"expectedRevision": map[string]interface{}{"type": "integer", "description": "Revision last read. Use 0 only when creating"},
				"sourceClient":     map[string]interface{}{"type": "string", "description": "Client saving the handoff, such as Claude or OpenCode"},
				"objective":        map[string]interface{}{"type": "string", "description": "Current objective"},
				"completed":        map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				"verification":     map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				"pending":          map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				"nextStep":         map[string]interface{}{"type": "string", "description": "Concrete next step"},
				"references":       map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			},
		},
		Handler: tools.SaveHandoffHandler,
	})

	s.tools.Register(&tools.Tool{
		Name:        "resume-context",
		Description: "Recovers the shared project handoff and relevant decisions for a working path. Call this before substantive work. It does not select a handoff from another branch automatically.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path":      map[string]interface{}{"type": "string", "description": "Working directory or repository path"},
				"projectId": map[string]interface{}{"type": "string", "description": "Stable project id, optional if path is given"},
				"handoffId": map[string]interface{}{"type": "string", "description": "Specific handoff to open"},
				"query":     map[string]interface{}{"type": "string", "description": "Optional topic used to retrieve related memories"},
				"depth":     map[string]interface{}{"type": "string", "description": "wake (default, short) or full"},
				"maxTokens": map[string]interface{}{"type": "integer", "description": "Approximate response budget"},
			},
		},
		Handler: tools.ResumeContextHandler,
	})
}
