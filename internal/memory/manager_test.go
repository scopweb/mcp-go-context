package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scopweb/mcp-go-context/internal/config"
)

func TestMemoryStoreAndRetrieve(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}

	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Store a memory
	err = m.Store("test-key", "test content", []string{"test"})
	if err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	// Retrieve it
	mem, err := m.Retrieve("test-key")
	if err != nil {
		t.Fatalf("Retrieve() failed: %v", err)
	}

	if mem.Content != "test content" {
		t.Errorf("expected content 'test content', got %q", mem.Content)
	}

	if len(mem.Tags) != 1 || mem.Tags[0] != "test" {
		t.Errorf("expected tag 'test', got %v", mem.Tags)
	}
}

func TestMemorySearch(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}

	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Store multiple memories
	m.Store("key1", "golang is fast", []string{"go", "language"})
	m.Store("key2", "python is easy", []string{"python", "language"})
	m.Store("key3", "rust is safe", []string{"rust", "language"})

	// Search for language
	results, err := m.Search("golang", nil)
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least one result for 'golang' query")
	}

	if results[0].Content != "golang is fast" {
		t.Errorf("expected 'golang is fast', got %q", results[0].Content)
	}
}

func TestMemorySearchByTags(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}

	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	m.Store("key1", "content1", []string{"tag1", "common"})
	m.Store("key2", "content2", []string{"tag2", "common"})
	m.Store("key3", "content3", []string{"tag3"})

	// Search by tag
	results, err := m.Search("", []string{"tag1"})
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Key != "key1" {
		t.Errorf("expected key1, got %s", results[0].Key)
	}
}

func TestMemoryMaxEntriesEviction(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     3,
		MaxResults:     10,
		SessionTTLDays: 30,
	}

	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Store 5 memories (should evict 2)
	m.Store("key1", "content1", []string{"tag1"})
	m.Store("key2", "content2", []string{"tag2"})
	m.Store("key3", "content3", []string{"tag3"})
	m.Store("key4", "content4", []string{"tag4"})
	m.Store("key5", "content5", []string{"tag5"})

	// Check that we have 3 entries
	session := m.getCurrentSession()
	if len(session.Memories) != 3 {
		t.Errorf("expected 3 memories after eviction, got %d", len(session.Memories))
	}

	// key1 should have been evicted (oldest)
	if _, err := m.Retrieve("key1"); err == nil {
		t.Error("key1 should have been evicted")
	}
}

func TestMemoryStoreWithType(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}

	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	err = m.StoreWithType(
		"decision1",
		"Use PostgreSQL for user data",
		[]string{"database", "architecture"},
		"architecture",
		"PostgreSQL has better ACID compliance",
		[]string{"MySQL", "MongoDB"},
	)
	if err != nil {
		t.Fatalf("StoreWithType() failed: %v", err)
	}

	mem, err := m.Retrieve("decision1")
	if err != nil {
		t.Fatalf("Retrieve() failed: %v", err)
	}

	if mem.DecisionType != "architecture" {
		t.Errorf("expected DecisionType 'architecture', got %q", mem.DecisionType)
	}

	if mem.Reason != "PostgreSQL has better ACID compliance" {
		t.Errorf("expected Reason 'PostgreSQL has better ACID compliance', got %q", mem.Reason)
	}

	if len(mem.Alternatives) != 2 {
		t.Errorf("expected 2 alternatives, got %d", len(mem.Alternatives))
	}
}

func TestMemorySearchDecisions(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}

	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	m.StoreWithType("dec1", "Use Redis for caching", []string{"cache"}, "architecture", "fast and simple", nil)
	m.StoreWithType("dec2", "Fix memory leak in worker", []string{"bug"}, "fix", "valgrind detected leak", nil)
	m.StoreWithType("dec3", "Use goroutines", []string{"concurrency"}, "approach", "native Go support", nil)

	// Search decisions by type
	results, err := m.SearchDecisions("architecture", "", 10)
	if err != nil {
		t.Fatalf("SearchDecisions() failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 architecture decision, got %d", len(results))
	}

	if results[0].Key != "dec1" {
		t.Errorf("expected dec1, got %s", results[0].Key)
	}

	// Search decisions by keyword in content
	results, err = m.SearchDecisions("", "redis", 10)
	if err != nil {
		t.Fatalf("SearchDecisions() failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 decision about redis, got %d", len(results))
	}

	if results[0].Key != "dec1" {
		t.Errorf("expected dec1, got %s", results[0].Key)
	}

	// Search by reason
	results, err = m.SearchDecisions("", "valgrind", 10)
	if err != nil {
		t.Fatalf("SearchDecisions() failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 decision mentioning valgrind, got %d", len(results))
	}

	if results[0].Key != "dec2" {
		t.Errorf("expected dec2, got %s", results[0].Key)
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input    string
		expected int // minimum number of tokens
	}{
		{"hello world", 2},
		{"golang is fast and simple", 3}, // should filter stop words
		{"the quick brown fox", 3},      // "the" filtered
		{"API gateway design pattern", 4},
	}

	for _, tt := range tests {
		tokens := tokenize(tt.input)
		if len(tokens) < tt.expected {
			t.Errorf("tokenize(%q) returned %d tokens, expected at least %d", tt.input, len(tokens), tt.expected)
		}
	}
}

func TestMemoryClear(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}

	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	m.Store("key1", "content1", []string{"tag1"})
	m.Store("key2", "content2", []string{"tag2"})

	// Clear all
	err = m.Clear()
	if err != nil {
		t.Fatalf("Clear() failed: %v", err)
	}

	// Should not find any memories
	results, err := m.Search("", nil)
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results after Clear, got %d", len(results))
	}
}

func TestMemoryGetRecentMemories(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}

	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Store with delays
	m.Store("key1", "first", []string{"tag1"})
	time.Sleep(10 * time.Millisecond)
	m.Store("key2", "second", []string{"tag2"})
	time.Sleep(10 * time.Millisecond)
	m.Store("key3", "third", []string{"tag3"})

	results, err := m.GetRecentMemories(2)
	if err != nil {
		t.Fatalf("GetRecentMemories() failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Most recent should be key3
	if results[0].Key != "key3" {
		t.Errorf("expected most recent to be key3, got %s", results[0].Key)
	}
}

func TestGetDecisionTypes(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}

	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	m.StoreWithType("dec1", "Use PostgreSQL", []string{"db"}, "architecture", "", nil)
	m.StoreWithType("dec2", "Fix bug", []string{"bug"}, "fix", "", nil)
	m.StoreWithType("dec3", "Use Redis", []string{"cache"}, "architecture", "", nil)

	types, err := m.GetDecisionTypes()
	if err != nil {
		t.Fatalf("GetDecisionTypes() failed: %v", err)
	}

	if len(types) != 2 {
		t.Errorf("expected 2 decision types, got %d: %v", len(types), types)
	}
}

func TestRetrieveIncrementsStoredUsage(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}

	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if err := m.Store("usage-key", "track me", []string{"usage"}); err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	if _, err := m.Retrieve("usage-key"); err != nil {
		t.Fatalf("Retrieve() failed: %v", err)
	}

	stored := m.getCurrentSession().Memories["usage-key"]
	if stored.Usage != 1 {
		t.Fatalf("expected stored usage 1, got %d", stored.Usage)
	}
}

func TestCleanupRemovesIndexEntriesForExpiredSessions(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 1,
	}

	m, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	session := &Session{
		ID:        "expired",
		StartTime: time.Now().AddDate(0, 0, -5),
		LastUsed:  time.Now().AddDate(0, 0, -5),
		Memories: map[string]Memory{
			"decision-key": {
				Key:       "decision-key",
				Content:   "distributed tracing rollout",
				Tags:      []string{"ops", "trace"},
				Timestamp: time.Now().AddDate(0, 0, -5),
			},
		},
	}
	m.sessions[session.ID] = session
	m.addToIndexes("decision-key", session.Memories["decision-key"])

	if err := os.WriteFile(filepath.Join(cfg.StoragePath, session.ID+".json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to seed session file: %v", err)
	}

	m.cleanup()

	if len(m.tagIndex) != 0 || len(m.wordIndex) != 0 {
		t.Fatalf("expected indexes to be empty after cleanup, got tags=%v words=%v", m.tagIndex, m.wordIndex)
	}
}
