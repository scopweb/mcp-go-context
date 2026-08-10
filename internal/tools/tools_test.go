package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/scopweb/mcp-go-context/internal/analyzer"
	"github.com/scopweb/mcp-go-context/internal/memory"
)

type fakeServer struct {
	analyzer AnalyzerInterface
	memory   MemoryInterface
}

func (f fakeServer) GetAnalyzer() AnalyzerInterface { return f.analyzer }
func (f fakeServer) GetMemory() MemoryInterface     { return f.memory }
func (f fakeServer) GetConfig() ConfigInterface     { return fakeConfig{} }

type fakeConfig struct{}

func (fakeConfig) GetProjectPaths() []string { return []string{"."} }

type fakeAnalyzer struct {
	project      *analyzer.ProjectStructure
	context      string
	deps         []analyzer.Dependency
	changedFiles []analyzer.ChangedFile
}

func (f fakeAnalyzer) AnalyzeProject(string, int) (*analyzer.ProjectStructure, error) {
	return f.project, nil
}

func (f fakeAnalyzer) GetRelevantContext(string, []string, int) (string, error) {
	return f.context, nil
}

func (f fakeAnalyzer) AnalyzeDependencies(bool) ([]analyzer.Dependency, error) {
	return f.deps, nil
}

func (f fakeAnalyzer) GetRecentlyChangedFiles(string, int) ([]analyzer.ChangedFile, error) {
	return f.changedFiles, nil
}

func (f fakeAnalyzer) EnsureLightIndex(int, time.Duration) {
	// no-op in fake
}

func (f fakeAnalyzer) IsLightIndexed() bool {
	return false
}

func (f fakeAnalyzer) LightIndexStats() map[string]interface{} {
	return map[string]interface{}{"indexed": false}
}

type fakeMemory struct {
	results   []*memory.Memory
	decisions []*memory.Memory
}

func (f fakeMemory) Store(string, string, []string) error { return nil }

func (f fakeMemory) StoreWithType(string, string, []string, string, string, []string) error {
	return nil
}

func (f fakeMemory) Retrieve(string) (*memory.Memory, error) { return nil, nil }

func (f fakeMemory) Search(string, []string) ([]*memory.Memory, error) {
	return f.results, nil
}

func (f fakeMemory) SearchWithProject(string, []string, string) ([]*memory.Memory, error) {
	return f.results, nil
}

func (f fakeMemory) ActiveProject() string { return "test" }

func (f fakeMemory) SearchDecisions(string, string, int) ([]*memory.Memory, error) {
	return f.decisions, nil
}

func (f fakeMemory) GetDecisionTypes() ([]string, error) { return nil, nil }

func (f fakeMemory) GetPromotedMemories(int) ([]*memory.Memory, error) { return nil, nil }

func (f fakeMemory) Promote(string, string) error { return nil }

func (f fakeMemory) Demote(string) error { return nil }

func (f fakeMemory) SuggestForPromotion(int) ([]*memory.Memory, error) { return nil, nil }

func (f fakeMemory) Stats() memory.Stats { return memory.Stats{ActiveProject: "test"} }

func TestAnalyzeProjectHandlerUsesConcreteTypes(t *testing.T) {
	server := fakeServer{
		analyzer: fakeAnalyzer{
			project: &analyzer.ProjectStructure{
				RootPath:     "/repo",
				Files:        []*analyzer.FileInfo{{Path: "/repo/main.go", Size: 2048, Language: "go"}},
				Dependencies: []analyzer.Dependency{{Name: "github.com/google/uuid", Version: "v1.6.0", Type: "direct"}},
				Structure:    map[string][]string{".": []string{"main.go"}},
				Stats:        analyzer.ProjectStats{TotalFiles: 1, TotalSize: 2048, Languages: map[string]int{"go": 1}},
			},
		},
	}

	payload, _ := json.Marshal(map[string]interface{}{"path": ".", "depth": 1})
	result, err := AnalyzeProjectHandler(payload, server)
	if err != nil {
		t.Fatalf("AnalyzeProjectHandler() failed: %v", err)
	}

	text := result.([]map[string]interface{})[0]["text"].(string)
	if !strings.Contains(text, "Project Analysis: /repo") || !strings.Contains(text, "github.com/google/uuid") {
		t.Fatalf("unexpected response: %s", text)
	}
}

func TestGetContextHandlerUsesConcreteMemoryTypes(t *testing.T) {
	server := fakeServer{
		analyzer: fakeAnalyzer{context: "## File: main.go\n"},
		memory:   fakeMemory{results: []*memory.Memory{{Key: "decision-1", Content: "Use Go for the service"}}},
	}

	payload, _ := json.Marshal(map[string]interface{}{"query": "service", "maxTokens": 500})
	result, err := GetContextHandler(payload, server)
	if err != nil {
		t.Fatalf("GetContextHandler() failed: %v", err)
	}

	text := result.([]map[string]interface{})[0]["text"].(string)
	if !strings.Contains(text, "Relevant Memory") || !strings.Contains(text, "Use Go for the service") {
		t.Fatalf("unexpected response: %s", text)
	}
}

func TestChangedFilesContextHandlerUsesConcreteChangedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	server := fakeServer{
		analyzer: fakeAnalyzer{changedFiles: []analyzer.ChangedFile{{Path: "main.go", Status: "modified"}}},
	}

	payload, _ := json.Marshal(map[string]interface{}{"path": tmpDir, "commitCount": 2, "maxFiles": 5})
	result, err := ChangedFilesContextHandler(payload, server)
	if err != nil {
		t.Fatalf("ChangedFilesContextHandler() failed: %v", err)
	}

	text := result.([]map[string]interface{})[0]["text"].(string)
	if !strings.Contains(text, "main.go") || !strings.Contains(text, "package main") {
		t.Fatalf("unexpected response: %s", text)
	}
}

func TestGetDecisionsHandlerUsesConcreteDecisionTypes(t *testing.T) {
	server := fakeServer{
		memory: fakeMemory{decisions: []*memory.Memory{{
			Key:          "arch-1",
			Content:      "Use PostgreSQL",
			DecisionType: "architecture",
			Reason:       "Transactional consistency",
			Timestamp:    time.Date(2026, 4, 12, 10, 30, 0, 0, time.UTC),
			Alternatives: []string{"MySQL"},
		}}},
	}

	payload, _ := json.Marshal(map[string]interface{}{"decisionType": "architecture", "limit": 5})
	result, err := GetDecisionsHandler(payload, server)
	if err != nil {
		t.Fatalf("GetDecisionsHandler() failed: %v", err)
	}

	text := result.([]map[string]interface{})[0]["text"].(string)
	if !strings.Contains(text, "Use PostgreSQL") || !strings.Contains(text, "Transactional consistency") {
		t.Fatalf("unexpected response: %s", text)
	}
}
