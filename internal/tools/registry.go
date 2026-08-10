package tools

import (
	"encoding/json"
	"fmt"
)

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
	Handler     ToolHandler            `json:"-"`
}

// ToolHandler is a function that handles tool execution
type ToolHandler func(args json.RawMessage, ctx interface{}) (interface{}, error)

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
		tools = append(tools, map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		})
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
