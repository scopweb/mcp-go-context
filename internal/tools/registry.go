package tools

import (
	"encoding/json"
	"fmt"
)

// Tool represents an MCP tool
type Tool struct {
	Name         string                 `json:"name"`
	Title        string                 `json:"title,omitempty"`
	Description  string                 `json:"description"`
	InputSchema  map[string]interface{} `json:"inputSchema"`
	OutputSchema map[string]interface{} `json:"outputSchema,omitempty"`
	Annotations  map[string]interface{} `json:"annotations,omitempty"`
	Handler      ToolHandler            `json:"-"`
}

// ToolHandler is a function that handles tool execution
type ToolHandler func(args json.RawMessage, ctx interface{}) (interface{}, error)

// CallResult represents a tool execution result with optional error semantics.
type CallResult struct {
	Content           []map[string]interface{}
	StructuredContent map[string]interface{}
	IsError           bool
}

// Registry manages available tools
type Registry struct {
	tools map[string]*Tool
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*Tool),
	}
}

// Register adds a new tool to the registry
func (r *Registry) Register(tool *Tool) error {
	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name)
	}
	r.tools[tool.Name] = tool
	return nil
}

// List returns all registered tools
func (r *Registry) List() []map[string]interface{} {
	var tools []map[string]interface{}

	for _, tool := range r.tools {
		entry := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		}
		if tool.Title != "" {
			entry["title"] = tool.Title
		}
		if len(tool.Annotations) > 0 {
			entry["annotations"] = tool.Annotations
		}
		if len(tool.OutputSchema) > 0 {
			entry["outputSchema"] = tool.OutputSchema
		}
		tools = append(tools, entry)
	}

	return tools
}

// Execute runs a tool by name
func (r *Registry) Execute(name string, args json.RawMessage, ctx interface{}) (interface{}, error) {
	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	return tool.Handler(args, ctx)
}

// Get returns a tool by name
func (r *Registry) Get(name string) (*Tool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}
