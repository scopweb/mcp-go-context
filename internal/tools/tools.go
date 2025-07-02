package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"path/filepath"
	"os"
	"time"
	"net/http"
	"io"
	"regexp"
	"sort"
)

// ServerInterface defines methods needed from the server
type ServerInterface interface {
	GetAnalyzer() AnalyzerInterface
	GetMemory() MemoryInterface
	GetConfig() ConfigInterface
}

type AnalyzerInterface interface {
	AnalyzeProject(string, int) (*ProjectStructure, error)
	GetRelevantContext(string, []string, int) (string, error)
	AnalyzeDependencies(bool) ([]Dependency, error)
}

type MemoryInterface interface {
	Store(string, string, []string) error
	Retrieve(string) (*Memory, error)
	Search(string, []string) ([]*Memory, error)
}

type ConfigInterface interface {
	GetProjectPaths() []string
}

// Structs imported from memory and analyzer packages
type Memory struct {
	Key       string
	Content   string
	Tags      []string
	Timestamp time.Time
	Usage     int
}
type ProjectStructure struct {
	RootPath     string
	Files        []*FileInfo
	Dependencies []Dependency
	Structure    map[string][]string
	Stats        ProjectStats
}

type FileInfo struct {
	Path         string
	Size         int64
	Language     string
	Imports      []string
	Functions    []string
	Types        []string
	LastModified int64
}

type ProjectStats struct {
	TotalFiles   int
	TotalLines   int
	Languages    map[string]int
	TotalSize    int64
	GoModules    []string
	MainPackages []string
}

type Dependency struct {
	Name    string
	Version string
	Type    string
	Path    string
}

type MemoryEntry struct {
	Key     string
	Content string
	Tags    []string
	Created time.Time
}

// Convert memory.Memory to MemoryEntry for compatibility
func convertMemoryToEntry(m *Memory) MemoryEntry {
	return MemoryEntry{
		Key:     m.Key,
		Content: m.Content,
		Tags:    m.Tags,
		Created: m.Timestamp,
	}
}

// Tool handler implementations

// AnalyzeProjectHandler - Complete implementation
func AnalyzeProjectHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Path  string `json:"path"`
		Depth int    `json:"depth"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Defaults
	if params.Path == "" {
		params.Path = "."
	}
	if params.Depth == 0 {
		params.Depth = 3
	}

	// Get server interface
	srv, ok := server.(ServerInterface)
	if !ok {
		return createErrorResponse("Server interface error")
	}

	// Perform analysis
	analyzer := srv.GetAnalyzer()
	if analyzer == nil {
		return createErrorResponse("Analyzer not available")
	}

	structure, err := analyzer.AnalyzeProject(params.Path, params.Depth)
	if err != nil {
		return createErrorResponse(fmt.Sprintf("Analysis failed: %v", err))
	}

	// Format comprehensive response
	var result strings.Builder
	result.WriteString(fmt.Sprintf("# Project Analysis: %s\n\n", structure.RootPath))
	
	// Stats summary
	result.WriteString("## 📊 Project Statistics\n")
	result.WriteString(fmt.Sprintf("- **Total Files**: %d\n", structure.Stats.TotalFiles))
	result.WriteString(fmt.Sprintf("- **Total Size**: %.2f MB\n", float64(structure.Stats.TotalSize)/(1024*1024)))
	
	// Languages breakdown
	result.WriteString("\n### Languages Distribution\n")
	for lang, count := range structure.Stats.Languages {
		percentage := float64(count) / float64(structure.Stats.TotalFiles) * 100
		result.WriteString(fmt.Sprintf("- **%s**: %d files (%.1f%%)\n", lang, count, percentage))
	}

	// Directory structure
	result.WriteString("\n## 📁 Directory Structure\n")
	for dir, files := range structure.Structure {
		if len(files) > 0 {
			result.WriteString(fmt.Sprintf("- `%s/` (%d files)\n", dir, len(files)))
		}
	}

	// Dependencies
	if len(structure.Dependencies) > 0 {
		result.WriteString("\n## 📦 Dependencies\n")
		directDeps := 0
		indirectDeps := 0
		for _, dep := range structure.Dependencies {
			if dep.Type == "direct" {
				directDeps++
			} else {
				indirectDeps++
			}
		}
		result.WriteString(fmt.Sprintf("- **Direct**: %d dependencies\n", directDeps))
		result.WriteString(fmt.Sprintf("- **Indirect**: %d dependencies\n", indirectDeps))
		
		// Show top dependencies
		result.WriteString("\n### Key Dependencies\n")
		count := 0
		for _, dep := range structure.Dependencies {
			if dep.Type == "direct" && count < 10 {
				result.WriteString(fmt.Sprintf("- `%s` %s\n", dep.Name, dep.Version))
				count++
			}
		}
	}

	// Important files
	result.WriteString("\n## 🔍 Key Files\n")
	keyFiles := findKeyFiles(structure.Files)
	for _, file := range keyFiles {
		relPath, _ := filepath.Rel(structure.RootPath, file.Path)
		result.WriteString(fmt.Sprintf("- `%s` (%s, %.2f KB)\n", 
			relPath, file.Language, float64(file.Size)/1024))
	}

	return []map[string]interface{}{
		{
			"type": "text",
			"text": result.String(),
		},
	}, nil
}

// GetContextHandler - Complete implementation with smart context retrieval
func GetContextHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Query     string   `json:"query"`
		Files     []string `json:"files"`
		MaxTokens int      `json:"maxTokens"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.MaxTokens == 0 {
		params.MaxTokens = 10000
	}

	srv, ok := server.(ServerInterface)
	if !ok {
		return createErrorResponse("Server interface error")
	}

	analyzer := srv.GetAnalyzer()
	memory := srv.GetMemory()

	var context strings.Builder
	context.WriteString(fmt.Sprintf("# Context for: %s\n\n", params.Query))

	// Add relevant memory
	if memory != nil {
		memories, err := memory.Search(params.Query, []string{})
		if err == nil && len(memories) > 0 {
			context.WriteString("## 💭 Relevant Memory\n\n")
			for i, mem := range memories {
				if i >= 3 {
					break
				}
				context.WriteString(fmt.Sprintf("**%s**: %s\n\n", mem.Key, mem.Content))
			}
		}
	}

	// Get file context
	if analyzer != nil {
		fileContext, err := analyzer.GetRelevantContext(params.Query, params.Files, params.MaxTokens-len(context.String()))
		if err == nil {
			context.WriteString(fileContext)
		}
	}

	// Add query-specific analysis
	analysis := analyzeQuery(params.Query)
	if analysis != "" {
		context.WriteString("\n## 🔍 Query Analysis\n")
		context.WriteString(analysis)
	}

	return []map[string]interface{}{
		{
			"type": "text",
			"text": context.String(),
		},
	}, nil
}
// FetchDocsHandler - Context7-like API integration
func FetchDocsHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Library string `json:"library"`
		Version string `json:"version"`
		Topic   string `json:"topic"`
		Tokens  int    `json:"tokens"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Tokens == 0 {
		params.Tokens = 5000
	}

	// Try Context7 API first
	docs, err := fetchFromContext7(params.Library, params.Version, params.Topic, params.Tokens)
	if err == nil && docs != "" {
		return []map[string]interface{}{
			{
				"type": "text",
				"text": docs,
			},
		}, nil
	}

	// Fallback to local documentation search
	localDocs := searchLocalDocs(params.Library, params.Topic)
	if localDocs != "" {
		return []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("# Local Documentation for %s\n\n%s", params.Library, localDocs),
			},
		}, nil
	}

	// Generate basic library info
	basicInfo := generateLibraryInfo(params.Library, params.Version)
	
	return []map[string]interface{}{
		{
			"type": "text",
			"text": basicInfo,
		},
	}, nil
}

// RememberConversationHandler - Enhanced memory storage
func RememberConversationHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Key     string   `json:"key"`
		Content string   `json:"content"`
		Tags    []string `json:"tags"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	srv, ok := server.(ServerInterface)
	if !ok {
		return createErrorResponse("Server interface error")
	}

	memory := srv.GetMemory()
	if memory == nil {
		return createErrorResponse("Memory manager not available")
	}

	// Auto-generate tags if none provided
	if len(params.Tags) == 0 {
		params.Tags = generateTags(params.Content)
	}

	err := memory.Store(params.Key, params.Content, params.Tags)
	if err != nil {
		return createErrorResponse(fmt.Sprintf("Failed to store memory: %v", err))
	}

	return []map[string]interface{}{
		{
			"type": "text",
			"text": fmt.Sprintf("✅ Successfully stored memory '%s' with tags: %v", 
				params.Key, params.Tags),
		},
	}, nil
}

// DependencyAnalysisHandler - Complete dependency analysis
func DependencyAnalysisHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		IncludeTransitive bool `json:"includeTransitive"`
		OnlyDirect        bool `json:"onlyDirect"`
		SuggestDocs       bool `json:"suggestDocs"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	srv, ok := server.(ServerInterface)
	if !ok {
		return createErrorResponse("Server interface error")
	}

	analyzer := srv.GetAnalyzer()
	if analyzer == nil {
		return createErrorResponse("Analyzer not available")
	}

	deps, err := analyzer.AnalyzeDependencies(params.IncludeTransitive && !params.OnlyDirect)
	if err != nil {
		return createErrorResponse(fmt.Sprintf("Dependency analysis failed: %v", err))
	}

	var result strings.Builder
	result.WriteString("# 📦 Dependency Analysis\n\n")

	// Categorize dependencies
	directDeps := []Dependency{}
	indirectDeps := []Dependency{}
	
	for _, dep := range deps {
		if dep.Type == "direct" {
			directDeps = append(directDeps, dep)
		} else {
			indirectDeps = append(indirectDeps, dep)
		}
	}

	// Direct dependencies
	result.WriteString(fmt.Sprintf("## Direct Dependencies (%d)\n\n", len(directDeps)))
	for _, dep := range directDeps {
		result.WriteString(fmt.Sprintf("- **%s** `%s`", dep.Name, dep.Version))
		if params.SuggestDocs {
			docSuggestion := suggestDocumentation(dep.Name)
			if docSuggestion != "" {
				result.WriteString(fmt.Sprintf(" - [📚 Docs](%s)", docSuggestion))
			}
		}
		result.WriteString("\n")
	}

	// Indirect dependencies if requested
	if params.IncludeTransitive && len(indirectDeps) > 0 {
		result.WriteString(fmt.Sprintf("\n## Indirect Dependencies (%d)\n\n", len(indirectDeps)))
		// Show only first 20 to avoid clutter
		displayCount := min(20, len(indirectDeps))
		for i, dep := range indirectDeps[:displayCount] {
			result.WriteString(fmt.Sprintf("%d. %s `%s`\n", i+1, dep.Name, dep.Version))
		}
		if len(indirectDeps) > 20 {
			result.WriteString(fmt.Sprintf("\n... and %d more indirect dependencies\n", len(indirectDeps)-20))
		}
	}

	// Security and update recommendations
	result.WriteString("\n## 🔍 Recommendations\n\n")
	recommendations := generateDepRecommendations(directDeps)
	for _, rec := range recommendations {
		result.WriteString(fmt.Sprintf("- %s\n", rec))
	}

	return []map[string]interface{}{
		{
			"type": "text",
			"text": result.String(),
		},
	}, nil
}
// Helper functions

func createErrorResponse(message string) ([]map[string]interface{}, error) {
	return []map[string]interface{}{
		{
			"type": "text",
			"text": fmt.Sprintf("❌ Error: %s", message),
		},
	}, nil
}

func findKeyFiles(files []*FileInfo) []*FileInfo {
	keyFiles := []*FileInfo{}
	
	for _, file := range files {
		fileName := filepath.Base(file.Path)
		
		// Key file patterns
		if fileName == "main.go" || fileName == "README.md" || 
		   fileName == "go.mod" || fileName == "Dockerfile" ||
		   strings.Contains(fileName, "config") ||
		   strings.Contains(fileName, "server") {
			keyFiles = append(keyFiles, file)
		}
	}
	
	// Sort by relevance (size, type, etc.)
	sort.Slice(keyFiles, func(i, j int) bool {
		return keyFiles[i].Size > keyFiles[j].Size
	})
	
	if len(keyFiles) > 10 {
		return keyFiles[:10]
	}
	
	return keyFiles
}

func analyzeQuery(query string) string {
	query = strings.ToLower(query)
	
	// Pattern matching for different query types
	patterns := map[string]string{
		"error|bug|fix|debug":           "🐛 Debugging context - Look for error handling, logs, and related functions",
		"test|testing|unit":             "🧪 Testing context - Focus on test files and testing utilities",
		"api|endpoint|route|handler":    "🌐 API context - Examine route handlers and API definitions", 
		"database|db|sql|query":         "💾 Database context - Check database models and queries",
		"config|configuration|setting":  "⚙️ Configuration context - Look at config files and environment setup",
		"deploy|deployment|docker":      "🚀 Deployment context - Focus on deployment and infrastructure files",
		"security|auth|permission":      "🔒 Security context - Examine authentication and authorization code",
		"performance|optimize|slow":     "⚡ Performance context - Look for bottlenecks and optimization opportunities",
	}
	
	for pattern, description := range patterns {
		if matched, _ := regexp.MatchString(pattern, query); matched {
			return description + "\n"
		}
	}
	
	return ""
}

func fetchFromContext7(library, version, topic string, tokens int) (string, error) {
	// Context7 API integration
	baseURL := "https://context7.com/api/v1"
	var url string
	
	if library != "" {
		url = fmt.Sprintf("%s/%s", baseURL, library)
		if version != "" {
			url = fmt.Sprintf("%s/%s", url, version)
		}
	} else {
		return "", fmt.Errorf("library name required")
	}
	
	// Add query parameters
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	
	q := req.URL.Query()
	if topic != "" {
		q.Add("topic", topic)
	}
	q.Add("tokens", fmt.Sprintf("%d", tokens))
	q.Add("type", "txt")
	req.URL.RawQuery = q.Encode()
	
	req.Header.Set("X-Context7-Source", "mcp-server-go")
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	content := string(body)
	if content == "No content available" || content == "No context data available" {
		return "", fmt.Errorf("no documentation available")
	}
	
	return content, nil
}

func searchLocalDocs(library, topic string) string {
	// Search for local documentation
	searchPaths := []string{
		"./docs",
		"./doc", 
		"./README.md",
		"./readme.md",
		"./documentation",
	}
	
	for _, path := range searchPaths {
		if content := searchInPath(path, library, topic); content != "" {
			return content
		}
	}
	
	return ""
}

func searchInPath(path, library, topic string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ""
	}
	
	// Simple file content search
	if strings.HasSuffix(path, ".md") {
		content, err := os.ReadFile(path)
		if err == nil {
			contentStr := string(content)
			if strings.Contains(strings.ToLower(contentStr), strings.ToLower(library)) {
				return fmt.Sprintf("Found in %s:\n\n%s", path, contentStr)
			}
		}
	}
	
	return ""
}

func generateLibraryInfo(library, version string) string {
	// Generate basic library information
	var info strings.Builder
	
	info.WriteString(fmt.Sprintf("# Library Information: %s\n\n", library))
	
	if version != "" {
		info.WriteString(fmt.Sprintf("**Version**: %s\n\n", version))
	}
	
	// Try to determine library type
	if strings.Contains(library, "gin") {
		info.WriteString("**Type**: Go Web Framework\n")
		info.WriteString("**Description**: Fast HTTP web framework for Go\n")
		info.WriteString("**Common Usage**: REST APIs, web services\n\n")
	} else if strings.Contains(library, "postgres") || strings.Contains(library, "mysql") {
		info.WriteString("**Type**: Database Driver\n")
		info.WriteString("**Description**: Database connection and query library\n")
	} else if strings.Contains(library, "redis") {
		info.WriteString("**Type**: Caching/Storage\n")
		info.WriteString("**Description**: Redis client library\n")
	} else {
		info.WriteString("**Type**: Library/Package\n")
		info.WriteString("**Description**: External dependency\n")
	}
	
	info.WriteString("\n**Note**: For detailed documentation, consider using official sources or package documentation.\n")
	
	return info.String()
}

func generateTags(content string) []string {
	content = strings.ToLower(content)
	tags := []string{}
	
	// Common tag patterns
	tagPatterns := map[string]string{
		"error|bug|issue|problem":    "bug",
		"test|testing|spec":          "testing", 
		"config|configuration":       "config",
		"api|endpoint|route":         "api",
		"database|db|sql":            "database",
		"deploy|deployment":          "deployment",
		"security|auth":              "security",
		"performance|optimize":       "performance",
		"feature|functionality":      "feature",
		"documentation|docs":         "docs",
	}
	
	for pattern, tag := range tagPatterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			tags = append(tags, tag)
		}
	}
	
	if len(tags) == 0 {
		tags = append(tags, "general")
	}
	
	return tags
}

func suggestDocumentation(depName string) string {
	// Common Go library documentation URLs
	docMap := map[string]string{
		"gin-gonic/gin":     "https://gin-gonic.com/docs/",
		"gorilla/mux":       "https://pkg.go.dev/github.com/gorilla/mux",
		"lib/pq":            "https://pkg.go.dev/github.com/lib/pq",
		"go-sql-driver/mysql": "https://pkg.go.dev/github.com/go-sql-driver/mysql",
		"go-redis/redis":    "https://redis.uptrace.dev/",
		"sirupsen/logrus":   "https://pkg.go.dev/github.com/sirupsen/logrus",
		"stretchr/testify":  "https://pkg.go.dev/github.com/stretchr/testify",
	}
	
	for key, url := range docMap {
		if strings.Contains(depName, key) {
			return url
		}
	}
	
	// Default to pkg.go.dev
	return fmt.Sprintf("https://pkg.go.dev/%s", depName)
}

func generateDepRecommendations(deps []Dependency) []string {
	recommendations := []string{}
	
	// Check for common security recommendations
	for _, dep := range deps {
		if strings.Contains(dep.Name, "crypto") || strings.Contains(dep.Name, "security") {
			recommendations = append(recommendations, 
				fmt.Sprintf("🔒 Review security implementation for %s", dep.Name))
		}
		
		if strings.Contains(dep.Name, "test") {
			recommendations = append(recommendations, 
				"🧪 Ensure adequate test coverage with testing libraries")
		}
		
		if strings.Contains(dep.Name, "http") || strings.Contains(dep.Name, "gin") {
			recommendations = append(recommendations, 
				"🌐 Implement proper rate limiting and security headers for web services")
		}
	}
	
	// General recommendations
	recommendations = append(recommendations, 
		"📊 Regularly update dependencies to latest stable versions",
		"🔍 Use `go mod tidy` to clean up unused dependencies",
		"📋 Consider using `go mod audit` for security vulnerability checks")
	
	return recommendations
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}