package memory

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/scopweb/mcp-go-context/internal/config"
)

func TestOpenManagerSeesExternalWrite(t *testing.T) {
	dir := t.TempDir()
	cfg := config.MemoryConfig{Enabled: true, StoragePath: dir, MaxEntries: 100, MaxResults: 10, SessionTTLDays: 30}
	first, err := New(cfg, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	second, err := New(cfg, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if err := second.Store("from-peer", "visible without restart", []string{"test"}); err != nil {
		t.Fatal(err)
	}
	got, err := first.Retrieve("from-peer")
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "visible without restart" {
		t.Fatalf("got %q", got.Content)
	}
}

func TestCrossProcessMemory(t *testing.T) {
	if os.Getenv("MCP_MEM_CHILD") == "1" {
		cfg := config.MemoryConfig{Enabled: true, StoragePath: os.Getenv("MCP_MEM_DIR"), MaxEntries: 100, SessionTTLDays: 30}
		child, err := New(cfg, "alpha")
		if err != nil {
			t.Fatal(err)
		}
		if err := child.Store("from-child", "cross-process", nil); err != nil {
			t.Fatal(err)
		}
		return
	}
	dir := t.TempDir()
	cfg := config.MemoryConfig{Enabled: true, StoragePath: dir, MaxEntries: 100, SessionTTLDays: 30}
	parent, err := New(cfg, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestCrossProcessMemory", "-test.count=1")
	cmd.Env = append(os.Environ(), "MCP_MEM_CHILD=1", "MCP_MEM_DIR="+dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("child failed: %v\n%s", err, out)
	}
	got, err := parent.Retrieve("from-child")
	if err != nil {
		t.Fatal(err)
	}
	if got.Content != "cross-process" {
		t.Fatalf("got %q", got.Content)
	}
}

func TestPromotedMemorySurvivesCleanup(t *testing.T) {
	dir := t.TempDir()
	cfg := config.MemoryConfig{Enabled: true, StoragePath: dir, MaxEntries: 100, SessionTTLDays: 1}
	m, err := New(cfg, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Store("keep-me", "durable decision", nil); err != nil {
		t.Fatal(err)
	}
	if err := m.Promote("keep-me", "high"); err != nil {
		t.Fatal(err)
	}
	if err := m.Store("drop-me", "temporary note", nil); err != nil {
		t.Fatal(err)
	}
	sessionPath := filepath.Join(dir, "current.json")
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	old := time.Now().AddDate(0, 0, -10).Format(time.RFC3339Nano)
	// LastUsed is written by encoding/json using RFC3339.
	updated := replaceJSONTime(t, text, old)
	if err := os.WriteFile(sessionPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}
	m.cleanup()
	if _, err := m.Retrieve("keep-me"); err != nil {
		t.Fatalf("promoted memory was removed: %v", err)
	}
	if _, err := m.Retrieve("drop-me"); err == nil {
		t.Fatal("expired unpromoted memory should be removed")
	}
}

func TestSearchRequiresRelevance(t *testing.T) {
	dir := t.TempDir()
	cfg := config.MemoryConfig{Enabled: true, StoragePath: dir, MaxEntries: 20, MaxResults: 10, SessionTTLDays: 30}
	m, err := New(cfg, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Store("bananas", "unrelated fruit note", nil); err != nil {
		t.Fatal(err)
	}
	got, err := m.Search("postgresql migrations", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("irrelevant memory returned: %+v", got[0])
	}
}

func TestUpdatePreservesPromotion(t *testing.T) {
	dir := t.TempDir()
	cfg := config.MemoryConfig{Enabled: true, StoragePath: dir, MaxEntries: 20, SessionTTLDays: 30}
	m, err := New(cfg, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.StoreWithType("choice", "use files", nil, "architecture", "simple", []string{"database"}); err != nil {
		t.Fatal(err)
	}
	if err := m.Promote("choice", "high"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Retrieve("choice"); err != nil {
		t.Fatal(err)
	}
	if err := m.Store("choice", "use files with locking", nil); err != nil {
		t.Fatal(err)
	}
	got, err := m.Get("choice")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Promoted || got.Usage == 0 || got.Reason != "simple" {
		t.Fatalf("metadata lost: %+v", got)
	}
}

func TestCapacityProtectsPromoted(t *testing.T) {
	dir := t.TempDir()
	cfg := config.MemoryConfig{Enabled: true, StoragePath: dir, MaxEntries: 1, SessionTTLDays: 30}
	m, err := New(cfg, "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Store("only", "promoted", nil); err != nil {
		t.Fatal(err)
	}
	if err := m.Promote("only", "high"); err != nil {
		t.Fatal(err)
	}
	err = m.Store("another", "should fail", nil)
	if err != ErrCapacity {
		t.Fatalf("expected capacity error, got %v", err)
	}
}

func replaceJSONTime(t *testing.T, text, replacement string) string {
	t.Helper()
	const marker = `"lastUsed": "`
	start := indexOf(text, marker)
	if start < 0 {
		t.Fatalf("lastUsed not found in %s", text)
	}
	start += len(marker)
	end := indexOf(text[start:], `"`)
	if end < 0 {
		t.Fatal("unterminated lastUsed")
	}
	return text[:start] + replacement + text[start+end:]
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
