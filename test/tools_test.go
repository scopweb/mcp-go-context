package test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/server"
	"github.com/scopweb/mcp-go-context/internal/tools"
)

type dummyServer struct{}

func (d *dummyServer) GetAnalyzer() tools.AnalyzerInterface { return nil }
func (d *dummyServer) GetMemory() tools.MemoryInterface     { return nil }
func (d *dummyServer) GetConfig() tools.ConfigInterface     { return nil }

func TestAnalyzeProjectHandler_InvalidParams(t *testing.T) {
	_, err := tools.AnalyzeProjectHandler(json.RawMessage(`{"path":123}`), &dummyServer{})
	if err == nil {
		t.Error("Expected error for invalid params")
	}
}

func TestGetContextHandler_InvalidParams(t *testing.T) {
	_, err := tools.GetContextHandler(json.RawMessage(`{"query":123}`), &dummyServer{})
	if err == nil {
		t.Error("Expected error for invalid params")
	}
}

func TestFetchDocsHandler_InvalidParams(t *testing.T) {
	_, err := tools.FetchDocsHandler(json.RawMessage(`{"library":123}`), &dummyServer{})
	if err == nil {
		t.Error("Expected error for invalid params")
	}
	resp, err := tools.FetchDocsHandler(json.RawMessage(`{"library":"!@#"}`), &dummyServer{})
	if err == nil {
		result, ok := resp.(tools.CallResult)
		if !ok || !result.IsError || len(result.Content) == 0 || !strings.Contains(fmt.Sprint(result.Content[0]["text"]), "Error") {
			t.Error("Expected error for invalid library name")
		}
	}
}

func TestRememberConversationHandler_InvalidParams(t *testing.T) {
	_, err := tools.RememberConversationHandler(json.RawMessage(`{"key":123}`), &dummyServer{})
	if err == nil {
		t.Error("Expected error for invalid params")
	}
}

func TestDependencyAnalysisHandler_InvalidParams(t *testing.T) {
	_, err := tools.DependencyAnalysisHandler(json.RawMessage(`{"includeTransitive":"yes"}`), &dummyServer{})
	if err == nil {
		t.Error("Expected error for invalid params")
	}
}

func TestToolCallErrorsFollowMCPConventions(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.WithValue(context.Background(), "transportType", "tool-test")

	mustInitializeSession(t, srv, ctx)

	t.Run("Invalid tool input becomes isError result", func(t *testing.T) {
		response := callJSONRPC(t, srv, ctx, map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      10,
			"method":  "tools/call",
			"params": map[string]interface{}{
				"name": "fetch-docs",
				"arguments": map[string]interface{}{
					"library": 123,
				},
			},
		})

		if _, hasProtocolError := response["error"]; hasProtocolError {
			t.Fatalf("Expected tool execution error result, got protocol error: %v", response)
		}

		result, _ := response["result"].(map[string]interface{})
		if result == nil || result["isError"] != true {
			t.Fatalf("Expected isError=true result, got %v", response)
		}
	})

	t.Run("Unknown tool stays protocol error", func(t *testing.T) {
		response := callJSONRPC(t, srv, ctx, map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      11,
			"method":  "tools/call",
			"params": map[string]interface{}{
				"name":      "does-not-exist",
				"arguments": map[string]interface{}{},
			},
		})

		errorObj, _ := response["error"].(map[string]interface{})
		if errorObj == nil {
			t.Fatalf("Expected protocol error, got %v", response)
		}
		if errorObj["code"] != float64(-32601) {
			t.Fatalf("Expected -32601 for unknown tool, got %v", errorObj)
		}
	})

	t.Run("Invalid tool name stays protocol error", func(t *testing.T) {
		response := callJSONRPC(t, srv, ctx, map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      12,
			"method":  "tools/call",
			"params": map[string]interface{}{
				"name":      "bad tool name",
				"arguments": map[string]interface{}{},
			},
		})

		errorObj, _ := response["error"].(map[string]interface{})
		if errorObj == nil {
			t.Fatalf("Expected protocol error, got %v", response)
		}
		if errorObj["code"] != float64(-32602) {
			t.Fatalf("Expected -32602 for invalid tool name, got %v", errorObj)
		}
	})
}

func TestToolsListIncludesMetadata(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.WithValue(context.Background(), "transportType", "tool-metadata-test")

	mustInitializeSession(t, srv, ctx)

	response := callJSONRPC(t, srv, ctx, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      20,
		"method":  "tools/list",
	})

	result, _ := response["result"].(map[string]interface{})
	toolItems, _ := result["tools"].([]interface{})
	if len(toolItems) == 0 {
		t.Fatal("Expected tools/list to return tools")
	}

	toolsByName := make(map[string]map[string]interface{}, len(toolItems))
	for _, item := range toolItems {
		tool, _ := item.(map[string]interface{})
		name, _ := tool["name"].(string)
		if name != "" {
			toolsByName[name] = tool
		}
	}

	analyzeProject := toolsByName["analyze-project"]
	if analyzeProject == nil {
		t.Fatal("Expected analyze-project metadata")
	}
	if analyzeProject["title"] != "Analyze Project" {
		t.Fatalf("Expected title for analyze-project, got %v", analyzeProject["title"])
	}
	outputSchema, _ := analyzeProject["outputSchema"].(map[string]interface{})
	if outputSchema == nil {
		t.Fatal("Expected outputSchema for analyze-project")
	}
	properties, _ := outputSchema["properties"].(map[string]interface{})
	if properties == nil || properties["rootPath"] == nil || properties["keyFiles"] == nil {
		t.Fatalf("Expected structured output properties for analyze-project, got %v", outputSchema)
	}
	annotations, _ := analyzeProject["annotations"].(map[string]interface{})
	if annotations == nil || annotations["readOnlyHint"] != true || annotations["destructiveHint"] != false {
		t.Fatalf("Expected read-only annotations for analyze-project, got %v", analyzeProject["annotations"])
	}

	getContext := toolsByName["get-context"]
	if getContext == nil {
		t.Fatal("Expected get-context metadata")
	}
	contextSchema, _ := getContext["outputSchema"].(map[string]interface{})
	if contextSchema == nil {
		t.Fatal("Expected outputSchema for get-context")
	}
	contextProperties, _ := contextSchema["properties"].(map[string]interface{})
	if contextProperties == nil || contextProperties["source"] == nil || contextProperties["text"] == nil {
		t.Fatalf("Expected structured output properties for get-context, got %v", contextSchema)
	}

	memoryClear := toolsByName["memory-clear"]
	if memoryClear == nil {
		t.Fatal("Expected memory-clear metadata")
	}
	clearAnnotations, _ := memoryClear["annotations"].(map[string]interface{})
	if clearAnnotations == nil || clearAnnotations["destructiveHint"] != true {
		t.Fatalf("Expected destructive annotation for memory-clear, got %v", memoryClear["annotations"])
	}
}

func TestInitializeInstructionsGuideToolSelection(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.WithValue(context.Background(), "transportType", "tool-instructions-test")

	response := callJSONRPC(t, srv, ctx, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      21,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2025-11-25",
		},
	})

	result, _ := response["result"].(map[string]interface{})
	instructions, _ := result["instructions"].(string)
	if instructions == "" {
		t.Fatal("Expected initialize instructions")
	}

	expectedSnippets := []string{
		"Selection rules for Claude Code and Claude Desktop",
		"Use analyze-project first",
		"prefer stable keys such as repo/component/topic",
		"Do not use memory-clear unless the user explicitly asks",
		"analyze-project (Analyze Project)",
		"auth-generate-token (Generate JWT Token)",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(instructions, snippet) {
			t.Fatalf("Expected instructions to contain %q, got:\n%s", snippet, instructions)
		}
	}
}

func TestToolCallIncludesStructuredContentWhenSchemaIsDeclared(t *testing.T) {
	srv := newTestServer(t)
	ctx := context.WithValue(context.Background(), "transportType", "tool-structured-test")

	mustInitializeSession(t, srv, ctx)

	response := callJSONRPC(t, srv, ctx, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      22,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "config-get-project-paths",
			"arguments": map[string]interface{}{},
		},
	})

	result, _ := response["result"].(map[string]interface{})
	structured, _ := result["structuredContent"].(map[string]interface{})
	if structured == nil {
		t.Fatalf("Expected structuredContent in tool result, got %v", response)
	}
	if _, ok := structured["count"].(float64); !ok {
		t.Fatalf("Expected numeric count in structuredContent, got %v", structured)
	}
	paths, _ := structured["paths"].([]interface{})
	if len(paths) == 0 {
		t.Fatalf("Expected paths in structuredContent, got %v", structured)
	}
	content, _ := result["content"].([]interface{})
	if len(content) == 0 {
		t.Fatalf("Expected text content alongside structuredContent, got %v", result)
	}
	if result["isError"] != false {
		t.Fatalf("Expected successful tool result, got %v", result)
	}
}

func newTestServer(t *testing.T) *server.Server {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Transport.Type = "stdio"
	srv, err := server.New(cfg)
	if err != nil {
		t.Fatalf("server.New error: %v", err)
	}
	return srv
}

func mustInitializeSession(t *testing.T, srv *server.Server, ctx context.Context) {
	t.Helper()
	_ = callJSONRPC(t, srv, ctx, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2025-11-25",
		},
	})

	notification, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})
	if _, err := srv.HandleRequest(ctx, notification); err != nil {
		t.Fatalf("initialized notification failed: %v", err)
	}
}

func callJSONRPC(t *testing.T, srv *server.Server, ctx context.Context, payload map[string]interface{}) map[string]interface{} {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	rawResponse, err := srv.HandleRequest(ctx, data)
	if err != nil {
		t.Fatalf("HandleRequest error: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rawResponse, &response); err != nil {
		t.Fatalf("unmarshal response error: %v", err)
	}
	return response
}
