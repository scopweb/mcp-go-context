package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/scopweb/mcp-go-context/internal/analyzer"
	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/memory"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()

	mgr, err := memory.New(config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     100,
		SessionTTLDays: 30,
		MaxSessions:    10,
	}, "testproj")
	if err != nil {
		t.Fatalf("memory.New() failed: %v", err)
	}

	cfg := config.ContextConfig{
		ProjectPaths:   []string{"."},
		IgnorePatterns: []string{"*.log", "*.tmp"},
		AutoDetectDeps: false,
	}
	analyzr, err := analyzer.New(cfg)
	if err != nil {
		t.Fatalf("analyzer.New() failed: %v", err)
	}

	handler, err := New(mgr, analyzr)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	return handler
}

func TestHandleDashboardServesHTML(t *testing.T) {
	handler := newTestHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	res := httptest.NewRecorder()

	handler.handleDashboard(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if !strings.Contains(res.Body.String(), "Memory dashboard") {
		t.Fatalf("unexpected dashboard body: %s", res.Body.String())
	}
}

func TestHandleMemoriesReturnsStoredItems(t *testing.T) {
	handler := newTestHandler(t)
	if err := handler.memory.StoreWithType("arch-1", "Use PostgreSQL", []string{"db", "decision"}, "architecture", "Consistency", []string{"MySQL"}); err != nil {
		t.Fatalf("StoreWithType() failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/memories?query=postgresql&limit=10", nil)
	res := httptest.NewRecorder()

	handler.handleMemories(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	var payload memoryListResponse
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(payload.Items) != 1 || payload.Items[0].Key != "arch-1" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.Stats.Total != 1 || payload.Stats.DecisionCount != 1 {
		t.Fatalf("unexpected stats: %+v", payload.Stats)
	}
}

func TestHandleMemoryByKeyDeletesMemory(t *testing.T) {
	handler := newTestHandler(t)
	if err := handler.memory.Store("delete-me", "temporary memory", []string{"cleanup"}); err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/memories/delete-me", nil)
	res := httptest.NewRecorder()

	handler.handleMemoryByKey(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
	if _, err := handler.memory.Retrieve("delete-me"); err == nil {
		t.Fatal("expected memory to be deleted")
	}
}

func TestHandleMemoryByKeyGetDoesNotIncrementUsage(t *testing.T) {
	handler := newTestHandler(t)
	if err := handler.memory.Store("read-only", "inspect without scoring", []string{"dashboard"}); err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/memories/read-only", nil)
	res := httptest.NewRecorder()

	handler.handleMemoryByKey(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	mem, err := handler.memory.Get("read-only")
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}
	if mem.Usage != 0 {
		t.Fatalf("expected usage to remain 0, got %d", mem.Usage)
	}
}
