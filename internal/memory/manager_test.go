package memory

import (
	"fmt"
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

	m, err := New(cfg, "testproj")
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

	m, err := New(cfg, "testproj")
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

	m, err := New(cfg, "testproj")
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

	m, err := New(cfg, "testproj")
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

	m, err := New(cfg, "testproj")
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

	m, err := New(cfg, "testproj")
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
		{"the quick brown fox", 3},       // "the" filtered
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

	m, err := New(cfg, "testproj")
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

	m, err := New(cfg, "testproj")
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

	m, err := New(cfg, "testproj")
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

	m, err := New(cfg, "testproj")
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

func TestRetrievePersistsUsageToDisk(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}

	m, err := New(cfg, "testproj")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if err := m.Store("persisted-usage", "track me", []string{"usage"}); err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	if _, err := m.Retrieve("persisted-usage"); err != nil {
		t.Fatalf("Retrieve() failed: %v", err)
	}

	reloaded, err := New(cfg, "testproj")
	if err != nil {
		t.Fatalf("New() reload failed: %v", err)
	}

	mem, err := reloaded.Get("persisted-usage")
	if err != nil {
		t.Fatalf("Get() failed after reload: %v", err)
	}

	if mem.Usage != 1 {
		t.Fatalf("expected persisted usage 1, got %d", mem.Usage)
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

	m, err := New(cfg, "testproj")
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

func TestListMemoriesFiltersByDecisionType(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}

	m, err := New(cfg, "testproj")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if err := m.StoreWithType("dec-1", "Use PostgreSQL", []string{"db"}, "architecture", "", nil); err != nil {
		t.Fatalf("StoreWithType() failed: %v", err)
	}
	if err := m.StoreWithType("dec-2", "Fix retry logic", []string{"bug"}, "fix", "", nil); err != nil {
		t.Fatalf("StoreWithType() failed: %v", err)
	}

	results, err := m.ListMemories(0, "architecture")
	if err != nil {
		t.Fatalf("ListMemories() failed: %v", err)
	}

	if len(results) != 1 || results[0].Key != "dec-1" {
		t.Fatalf("unexpected filtered memories: %+v", results)
	}
}

func TestDeleteRemovesMemoryAndIndexes(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}

	m, err := New(cfg, "testproj")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	if err := m.Store("delete-key", "delete me", []string{"cleanup"}); err != nil {
		t.Fatalf("Store() failed: %v", err)
	}

	if err := m.Delete("delete-key"); err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	if _, err := m.Retrieve("delete-key"); err == nil {
		t.Fatal("expected deleted memory to be unavailable")
	}

	if len(m.tagIndex) != 0 || len(m.wordIndex) != 0 {
		t.Fatalf("expected indexes to be empty after delete, got tags=%v words=%v", m.tagIndex, m.wordIndex)
	}
}

func TestInferProject(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"empty -> default", "", "default"},
		{"windows path", `C:\MCPs\clone\mcp-go-context`, "mcp-go-context"},
		{"windows trailing slash", `C:\MCPs\clone\mcp-go-context\`, "mcp-go-context"},
		{"unix path", "/home/user/my-app", "my-app"},
		{"path with spaces", "/home/user/my project", "my-project"},
		{"path with accents", "/home/user/café", "cafe"},
		{"path with ñ", "/home/user/españa", "espana"},
		{"emoji only -> default", "/tmp/💥", "default"},
		{"drive letter only -> default", `C:\`, "default"},
		{"uppercase normalized", "/srv/MyApp", "myapp"},
		{"mixed punctuation", "/srv/my.cool_app!", "my-cool-app"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := InferProject(tc.in); got != tc.want {
				t.Errorf("InferProject(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestMemoryProjectIsolation(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}
	m, err := New(cfg, "alpha")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Store one in alpha.
	if err := m.Store("k-alpha", "alpha content about widgets", []string{"alpha"}); err != nil {
		t.Fatalf("Store alpha failed: %v", err)
	}

	// Inject a memory that pretends to belong to project "beta" by manipulating activeProject.
	m.activeProject = "beta"
	if err := m.Store("k-beta", "beta content about widgets", []string{"beta"}); err != nil {
		t.Fatalf("Store beta failed: %v", err)
	}

	// Search active (beta) should only return beta.
	results, err := m.Search("widgets", nil)
	if err != nil {
		t.Fatalf("Search() failed: %v", err)
	}
	if len(results) != 1 || results[0].Key != "k-beta" {
		t.Fatalf("active-project search should return only k-beta, got %v", keysOf(results))
	}

	// Explicit alpha filter.
	results, err = m.SearchWithProject("widgets", nil, "alpha")
	if err != nil {
		t.Fatalf("SearchWithProject(alpha) failed: %v", err)
	}
	if len(results) != 1 || results[0].Key != "k-alpha" {
		t.Fatalf("alpha filter should return only k-alpha, got %v", keysOf(results))
	}

	// Wildcard returns both.
	results, err = m.SearchWithProject("widgets", nil, WildcardProject)
	if err != nil {
		t.Fatalf("SearchWithProject(*) failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("wildcard should return 2 memories, got %d: %v", len(results), keysOf(results))
	}
}

func TestMemoryUnassignedLegacy(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}
	m, err := New(cfg, "current")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Simulate a legacy memory by direct injection (empty Project).
	sess := m.getCurrentSession()
	sess.Memories["legacy"] = Memory{
		Key:       "legacy",
		Content:   "old data about widgets",
		Tags:      []string{"old"},
		Timestamp: time.Now(),
	}
	m.addToIndexes("legacy", sess.Memories["legacy"])

	// Active project search must NOT include legacy.
	results, _ := m.Search("widgets", nil)
	for _, r := range results {
		if r.Key == "legacy" {
			t.Fatalf("legacy memory leaked into active-project search")
		}
	}

	// Explicit unassigned filter must include legacy.
	results, _ = m.SearchWithProject("widgets", nil, UnassignedProject)
	found := false
	for _, r := range results {
		if r.Key == "legacy" {
			found = true
		}
	}
	if !found {
		t.Fatalf("unassigned filter should surface legacy memory, got %v", keysOf(results))
	}
}

func TestListProjects(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}
	m, err := New(cfg, "active")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	m.Store("a", "x", nil)
	m.Store("b", "y", nil)
	m.activeProject = "other"
	m.Store("c", "z", nil)
	m.activeProject = "active"

	// Inject a legacy entry.
	sess := m.getCurrentSession()
	sess.Memories["legacy"] = Memory{Key: "legacy", Content: "old", Timestamp: time.Now()}

	stats := m.ListProjects()
	counts := map[string]int{}
	var activeStat ProjectStat
	for _, s := range stats {
		counts[s.Project] = s.Count
		if s.Active {
			activeStat = s
		}
	}
	if counts["active"] != 2 {
		t.Errorf("expected 2 in active, got %d (stats=%v)", counts["active"], stats)
	}
	if counts["other"] != 1 {
		t.Errorf("expected 1 in other, got %d", counts["other"])
	}
	if counts[UnassignedProject] != 1 {
		t.Errorf("expected 1 in unassigned, got %d", counts[UnassignedProject])
	}
	if activeStat.Project != "active" {
		t.Errorf("expected active flag on 'active', got %q", activeStat.Project)
	}
}

func keysOf(mems []*Memory) []string {
	out := make([]string, 0, len(mems))
	for _, m := range mems {
		out = append(out, m.Key)
	}
	return out
}

// Fase 1 tests for intelligent promotion suggestions

func TestMemorySuggestForPromotion(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     20,
		SessionTTLDays: 30,
	}

	m, err := New(cfg, "testproj")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	now := time.Now()

	// High-value candidate: structured decision with reason
	m.StoreWithType("arch-decision", "Use PostgreSQL for primary DB", []string{"db", "architecture"},
		"architecture", "Better ACID compliance and ecosystem", []string{"MySQL", "SQLite"})

	// Another strong candidate: recent fix with usage
	m.StoreWithType("fix-leak", "Fixed memory leak in worker pool", []string{"bug", "performance"},
		"fix", "Valgrind + pprof confirmed leak in goroutine", nil)

	// Simulate usage on the fix
	for i := 0; i < 6; i++ {
		m.Retrieve("fix-leak")
	}

	// Weak candidate: plain memory, no structure, old
	oldTime := now.Add(-30 * 24 * time.Hour)
	sess := m.getCurrentSession()
	sess.Memories["weak-note"] = Memory{
		Key:       "weak-note",
		Content:   "Some random note about nothing important",
		Timestamp: oldTime,
		Usage:     0,
	}

	// Promoted memory — must be excluded from suggestions
	m.StoreWithType("already-good", "We decided on this long ago", []string{"decision"},
		"architecture", "It was the right call", nil)
	m.Promote("already-good", "high")

	// Call suggestion
	suggestions, err := m.SuggestForPromotion(5)
	if err != nil {
		t.Fatalf("SuggestForPromotion() failed: %v", err)
	}

	if len(suggestions) == 0 {
		t.Fatal("expected at least one promotion suggestion")
	}

	// The two strong ones should be present
	keys := keysOf(suggestions)
	if !contains(keys, "arch-decision") && !contains(keys, "fix-leak") {
		t.Errorf("expected strong decisions in suggestions, got %v", keys)
	}

	// Promoted item must NOT appear
	for _, s := range suggestions {
		if s.Key == "already-good" {
			t.Error("promoted memory should never be suggested for promotion")
		}
		if s.Key == "weak-note" {
			t.Error("weak unstructured old memory should not be suggested")
		}
	}

	// Check ordering preference: structured decision should usually rank high
	// (fix-leak has high usage + type, arch-decision has full structure)
}

func TestMemorySuggestForPromotionLimit(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     20,
		SessionTTLDays: 30,
	}

	m, err := New(cfg, "testproj")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Create 8 decent candidates
	for i := 0; i < 8; i++ {
		key := fmt.Sprintf("good-decision-%d", i)
		m.StoreWithType(key, fmt.Sprintf("Important choice %d", i),
			[]string{"decision"}, "technical", "It made sense", nil)
	}

	suggestions, err := m.SuggestForPromotion(3)
	if err != nil {
		t.Fatalf("SuggestForPromotion failed: %v", err)
	}

	if len(suggestions) > 3 {
		t.Errorf("expected at most 3 suggestions with limit=3, got %d", len(suggestions))
	}
}

func TestMemorySuggestForPromotionEmpty(t *testing.T) {
	cfg := config.MemoryConfig{
		Enabled:        true,
		StoragePath:    t.TempDir(),
		MaxEntries:     100,
		MaxResults:     10,
		SessionTTLDays: 30,
	}

	m, err := New(cfg, "testproj")
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	// Only promoted memories
	m.StoreWithType("p1", "Old decision", nil, "architecture", "reason", nil)
	m.Promote("p1", "high")

	suggestions, err := m.SuggestForPromotion(5)
	if err != nil {
		t.Fatalf("SuggestForPromotion failed: %v", err)
	}

	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions when everything is promoted, got %d", len(suggestions))
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
