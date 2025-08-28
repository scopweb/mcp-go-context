package test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

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
		arr, ok := resp.([]map[string]interface{})
		if !ok || len(arr) == 0 || !strings.Contains(fmt.Sprint(arr[0]["text"]), "Error") {
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
