package test

import (
	"os"
	"testing"

	"github.com/scopweb/mcp-go-context/internal/config"
	"github.com/scopweb/mcp-go-context/internal/memory"
)

func setupTestManager(t *testing.T) *memory.Manager {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    "./testa/testmem.json",
		SessionTTLDays: 1,
		MaxSessions:    2,
	}
	mgr, err := memory.New(cfg)
	if err != nil {
		t.Fatalf("Failed to create memory manager: %v", err)
	}
	return mgr
}

func TestMemoryStoreAndRetrieve(t *testing.T) {
	mgr := setupTestManager(t)
	defer os.RemoveAll("./testa")

	err := mgr.Store("testkey", "testcontent", []string{"tag1"})
	if err != nil {
		t.Errorf("Store failed: %v", err)
	}

	mem, err := mgr.Retrieve("testkey")
	if err != nil {
		t.Errorf("Retrieve failed: %v", err)
	}
	if mem == nil || mem.Content != "testcontent" {
		t.Errorf("Memory content mismatch: got %v", mem)
	}
}

func TestMemorySearch(t *testing.T) {
	mgr := setupTestManager(t)
	defer os.RemoveAll("./testa")
	mgr.Store("key1", "content1", []string{"tagA"})
	mgr.Store("key2", "content2", []string{"tagB"})
	results, err := mgr.Search("content1", nil)
	if err != nil || len(results) == 0 {
		t.Errorf("Search failed or empty: %v", err)
	}
	results, err = mgr.Search("", []string{"tagB"})
	if err != nil || len(results) == 0 {
		t.Errorf("Search by tag failed or empty: %v", err)
	}
}

func TestGetRecentMemories(t *testing.T) {
	mgr := setupTestManager(t)
	defer os.RemoveAll("./testa")
	mgr.Store("key1", "content1", []string{"tagA"})
	mgr.Store("key2", "content2", []string{"tagB"})
	mems, err := mgr.GetRecentMemories(1)
	if err != nil || len(mems) == 0 {
		t.Errorf("GetRecentMemories failed: %v", err)
	}
}

func TestMemoryClear(t *testing.T) {
	mgr := setupTestManager(t)
	defer os.RemoveAll("./testa")
	mgr.Store("key1", "content1", []string{"tagA"})
	err := mgr.Clear()
	if err != nil {
		t.Errorf("Clear failed: %v", err)
	}
	mems, _ := mgr.GetRecentMemories(10)
	if len(mems) != 0 {
		t.Errorf("Clear did not remove memories")
	}
}
