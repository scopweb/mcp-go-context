package transport

import (
	"context"
	"encoding/json"
)

// Transport defines the interface for MCP transports
type Transport interface {
	Start(ctx context.Context, info ServerInfo, handler RequestHandler) error
	Stop() error
}

// RequestHandler processes incoming requests
type RequestHandler func(ctx context.Context, request json.RawMessage) (json.RawMessage, error)

// ServerInfo contains server metadata
type ServerInfo struct {
	Name         string
	Version      string
	Instructions string
}

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents a JSON-RPC error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Standard JSON-RPC error codes
const (
	ErrorCodeParse          = -32700 // Parse error
	ErrorCodeInvalidRequest = -32600 // Invalid Request
	ErrorCodeMethodNotFound = -32601 // Method not found
	ErrorCodeInvalidParams  = -32602 // Invalid params
	ErrorCodeInternalError  = -32603 // Internal error
)

// MCP specific capability structures
type Capabilities struct {
	Tools *ToolsCapability `json:"tools,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged"`
}

// Tool definition for MCP
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// ToolCallParams represents parameters for tool calls
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// Content represents tool response content
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}