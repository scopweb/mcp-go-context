package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/scopweb/mcp-go-context/internal/analyzer"
	"github.com/scopweb/mcp-go-context/internal/memory"
)

// ServerInterface defines methods needed from the server
type ServerInterface interface {
	GetAnalyzer() AnalyzerInterface
	GetMemory() MemoryInterface
	GetConfig() ConfigInterface
}

type AnalyzerInterface interface {
	AnalyzeProject(string, int) (*analyzer.ProjectStructure, error)
	GetRelevantContext(string, []string, int) (string, error)
	AnalyzeDependencies(bool) ([]analyzer.Dependency, error)
	GetRecentlyChangedFiles(string, int) ([]analyzer.ChangedFile, error)
	EnsureLightIndex(int, time.Duration) // Fase 0: bounded cold-start indexing
	IsLightIndexed() bool
	LightIndexStats() map[string]interface{}
}

type MemoryInterface interface {
	Store(string, string, []string) error
	StoreWithType(string, string, []string, string, string, []string) error
	Retrieve(string) (*memory.Memory, error)
	Search(string, []string) ([]*memory.Memory, error)
	SearchWithProject(string, []string, string) ([]*memory.Memory, error)
	ActiveProject() string
	SearchDecisions(string, string, int) ([]*memory.Memory, error)
	GetDecisionTypes() ([]string, error)
	GetPromotedMemories(int) ([]*memory.Memory, error)
	Promote(string, string) error
	Demote(string) error
	SuggestForPromotion(int) ([]*memory.Memory, error) // Fase 1: intelligent promotion suggestions
}

type ConfigInterface interface {
	GetProjectPaths() []string
}

func textResponse(text string) []map[string]interface{} {
	return []map[string]interface{}{{
		"type": "text",
		"text": text,
	}}
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
	result.WriteString("## Project Statistics\n")
	result.WriteString(fmt.Sprintf("- **Total Files**: %d\n", structure.Stats.TotalFiles))
	result.WriteString(fmt.Sprintf("- **Total Size**: %.2f MB\n", float64(structure.Stats.TotalSize)/(1024*1024)))

	if len(structure.Stats.Languages) > 0 {
		totalFiles := structure.Stats.TotalFiles
		if totalFiles == 0 {
			totalFiles = 1
		}
		result.WriteString("\n### Languages Distribution\n")
		for lang, count := range structure.Stats.Languages {
			percentage := float64(count) / float64(totalFiles) * 100
			result.WriteString(fmt.Sprintf("- **%s**: %d files (%.1f%%)\n", lang, count, percentage))
		}
	}

	// Directory structure
	if len(structure.Structure) > 0 {
		result.WriteString("\n## Directory Structure\n")
		for dir, files := range structure.Structure {
			if len(files) > 0 {
				result.WriteString(fmt.Sprintf("- `%s/` (%d files)\n", dir, len(files)))
			}
		}
	}

	// Dependencies
	if len(structure.Dependencies) > 0 {
		result.WriteString("\n## Dependencies\n")
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
	if len(structure.Files) > 0 {
		result.WriteString("\n## Key Files\n")
		keyFiles := findKeyFiles(structure.Files)
		for _, file := range keyFiles {
			relPath, _ := filepath.Rel(structure.RootPath, file.Path)
			result.WriteString(fmt.Sprintf("- `%s` (%s, %.2f KB)\n",
				relPath, file.Language, float64(file.Size)/1024))
		}
	}

	return textResponse(result.String()), nil
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
			context.WriteString("## Relevant Memory\n\n")
			for i := 0; i < min(3, len(memories)); i++ {
				mem := memories[i]
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
		context.WriteString("\n## Query Analysis\n")
		context.WriteString(analysis)
	}

	return textResponse(context.String()), nil
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
		return textResponse(docs), nil
	}

	// Fallback to local documentation search
	localDocs := searchLocalDocs(params.Library, params.Topic)
	if localDocs != "" {
		return textResponse(fmt.Sprintf("# Local Documentation for %s\n\n%s", params.Library, localDocs)), nil
	}

	// Generate basic library info
	basicInfo := generateLibraryInfo(params.Library, params.Version)

	return textResponse(basicInfo), nil
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

	return textResponse(fmt.Sprintf("Stored memory '%s' with tags: %v", params.Key, params.Tags)), nil
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

	analyzerSvc := srv.GetAnalyzer()
	if analyzerSvc == nil {
		return createErrorResponse("Analyzer not available")
	}

	deps, err := analyzerSvc.AnalyzeDependencies(params.IncludeTransitive && !params.OnlyDirect)
	if err != nil {
		return createErrorResponse(fmt.Sprintf("Dependency analysis failed: %v", err))
	}

	var result strings.Builder
	result.WriteString("# Dependency Analysis\n\n")

	// Categorize dependencies
	var directDeps []analyzer.Dependency
	var indirectDeps []analyzer.Dependency

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
				result.WriteString(fmt.Sprintf(" - [Docs](%s)", docSuggestion))
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
	result.WriteString("\n## Recommendations\n\n")
	recommendations := generateDepRecommendations(directDeps)
	for _, rec := range recommendations {
		result.WriteString(fmt.Sprintf("- %s\n", rec))
	}

	return textResponse(result.String()), nil
}

// Helper functions

func createErrorResponse(message string) ([]map[string]interface{}, error) {
	return textResponse(fmt.Sprintf("Error: %s", message)), nil
}

func findKeyFiles(files []*analyzer.FileInfo) []*analyzer.FileInfo {
	keyFiles := []*analyzer.FileInfo{}

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
		"error|bug|fix|debug":          "Debugging context: look for error handling, logs, and related functions.",
		"test|testing|unit":            "Testing context: focus on test files and testing utilities.",
		"api|endpoint|route|handler":   "API context: examine route handlers and API definitions.",
		"database|db|sql|query":        "Database context: check database models and queries.",
		"config|configuration|setting": "Configuration context: review config files and environment setup.",
		"deploy|deployment|docker":     "Deployment context: focus on deployment and infrastructure files.",
		"security|auth|permission":     "Security context: examine authentication and authorization code.",
		"performance|optimize|slow":    "Performance context: look for bottlenecks and optimization opportunities.",
	}

	for pattern, description := range patterns {
		if matched, _ := regexp.MatchString(pattern, query); matched {
			return description + "\n"
		}
	}

	return ""
}

func fetchFromContext7(library, version, topic string, tokens int) (string, error) {
	// FASE 2 FIX: Context7 uses libraryId-based flow, not direct library names
	// Step 1: Resolve libraryId from library name + query
	resolveURL := "https://context7.com/api/v1/library/resolve"
	resolveBody := fmt.Sprintf(`{"library": "%s", "query": "%s"}`, library, topic)
	req, err := http.NewRequest("POST", resolveURL, strings.NewReader(resolveBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Context7-Source", "mcp-server-go")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("library resolve returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse libraryId from response
	var resolveResp struct {
		LibraryID string `json:"libraryId"`
	}
	if err := json.Unmarshal(body, &resolveResp); err != nil {
		return "", fmt.Errorf("failed to parse library response: %w", err)
	}

	if resolveResp.LibraryID == "" {
		return "", fmt.Errorf("no libraryId resolved for %s", library)
	}

	// Step 2: Fetch docs using libraryId
	docsURL := fmt.Sprintf("https://context7.com/api/v1/library/%s/docs", resolveResp.LibraryID)
	docsReq, err := http.NewRequest("GET", docsURL, nil)
	if err != nil {
		return "", err
	}

	q := docsReq.URL.Query()
	if version != "" {
		q.Add("version", version)
	}
	q.Add("tokens", fmt.Sprintf("%d", tokens))
	q.Add("type", "txt")
	docsReq.URL.RawQuery = q.Encode()

	docsReq.Header.Set("X-Context7-Source", "mcp-server-go")

	docsResp, err := client.Do(docsReq)
	if err != nil {
		return "", err
	}
	defer docsResp.Body.Close()

	if docsResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("docs fetch returned status %d", docsResp.StatusCode)
	}

	docsBody, err := io.ReadAll(docsResp.Body)
	if err != nil {
		return "", err
	}

	content := string(docsBody)
	if content == "No content available" || content == "No context data available" {
		return "", fmt.Errorf("no documentation available")
	}

	return "[Context7]\n" + content, nil
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
		"error|bug|issue|problem": "bug",
		"test|testing|spec":       "testing",
		"config|configuration":    "config",
		"api|endpoint|route":      "api",
		"database|db|sql":         "database",
		"deploy|deployment":       "deployment",
		"security|auth":           "security",
		"performance|optimize":    "performance",
		"feature|functionality":   "feature",
		"documentation|docs":      "docs",
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
		"gin-gonic/gin":       "https://gin-gonic.com/docs/",
		"gorilla/mux":         "https://pkg.go.dev/github.com/gorilla/mux",
		"lib/pq":              "https://pkg.go.dev/github.com/lib/pq",
		"go-sql-driver/mysql": "https://pkg.go.dev/github.com/go-sql-driver/mysql",
		"go-redis/redis":      "https://redis.uptrace.dev/",
		"sirupsen/logrus":     "https://pkg.go.dev/github.com/sirupsen/logrus",
		"stretchr/testify":    "https://pkg.go.dev/github.com/stretchr/testify",
	}

	for key, url := range docMap {
		if strings.Contains(depName, key) {
			return url
		}
	}

	// Default to pkg.go.dev
	return fmt.Sprintf("https://pkg.go.dev/%s", depName)
}

func generateDepRecommendations(deps []analyzer.Dependency) []string {
	recommendations := []string{}

	// Check for common security recommendations
	for _, dep := range deps {
		if strings.Contains(dep.Name, "crypto") || strings.Contains(dep.Name, "security") {
			recommendations = append(recommendations,
				fmt.Sprintf("Review security implementation for %s", dep.Name))
		}

		if strings.Contains(dep.Name, "test") {
			recommendations = append(recommendations,
				"Ensure adequate test coverage with testing libraries")
		}

		if strings.Contains(dep.Name, "http") || strings.Contains(dep.Name, "gin") {
			recommendations = append(recommendations,
				"Implement proper rate limiting and security headers for web services")
		}
	}

	// General recommendations
	recommendations = append(recommendations,
		"Regularly update dependencies to latest stable versions",
		"Use `go mod tidy` to clean up unused dependencies",
		"Consider using `go mod audit` for security vulnerability checks")

	return recommendations
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ChangedFilesContextHandler - Gets context from recently changed files
func ChangedFilesContextHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Path        string `json:"path"`
		CommitCount int    `json:"commitCount"`
		MaxFiles    int    `json:"maxFiles"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Defaults
	if params.Path == "" {
		params.Path = "."
	}
	if params.CommitCount == 0 {
		params.CommitCount = 5
	}
	if params.MaxFiles == 0 {
		params.MaxFiles = 10
	}

	srv, ok := server.(ServerInterface)
	if !ok {
		return createErrorResponse("Server interface error")
	}

	analyzerSvc := srv.GetAnalyzer()
	if analyzerSvc == nil {
		return createErrorResponse("Analyzer not available")
	}

	// Get recently changed files
	changedFiles, err := analyzerSvc.GetRecentlyChangedFiles(params.Path, params.CommitCount)
	if err != nil {
		return createErrorResponse(fmt.Sprintf("Failed to get changed files: %v", err))
	}

	if len(changedFiles) == 0 {
		return textResponse("No recent changes found in git history"), nil
	}

	var result strings.Builder
	result.WriteString("# Recently Changed Files Context\n\n")
	result.WriteString(fmt.Sprintf("Analyzing last %d commits\n\n", params.CommitCount))

	// Limit files
	if len(changedFiles) > params.MaxFiles {
		changedFiles = changedFiles[:params.MaxFiles]
	}

	result.WriteString(fmt.Sprintf("## Changed Files (%d)\n\n", len(changedFiles)))
	for _, f := range changedFiles {
		path := f.Path
		status := f.Status
		result.WriteString(fmt.Sprintf("- `%s` (%s)\n", path, status))
	}

	// Add content summary for each file
	result.WriteString("\n## File Summaries\n\n")
	for _, f := range changedFiles {
		path := f.Path
		fullPath := filepath.Join(params.Path, path)
		if content, err := os.ReadFile(fullPath); err == nil {
			lines := strings.Split(string(content), "\n")
			previewLines := min(10, len(lines))
			result.WriteString(fmt.Sprintf("### %s\n\n", path))
			result.WriteString("```" + toolDetectLanguage(path) + "\n")
			result.WriteString(strings.Join(lines[:previewLines], "\n"))
			if len(lines) > previewLines {
				result.WriteString("\n...")
			}
			result.WriteString("\n```\n\n")
		}
	}

	return textResponse(result.String()), nil
}

// toolDetectLanguage detects language from file extension
func toolDetectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	langMap := map[string]string{
		".go":    "go",
		".js":    "javascript",
		".ts":    "typescript",
		".py":    "python",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".rs":    "rust",
		".rb":    "ruby",
		".php":   "php",
		".cs":    "csharp",
		".swift": "swift",
		".kt":    "kotlin",
		".md":    "markdown",
		".json":  "json",
		".yaml":  "yaml",
		".yml":   "yaml",
		".xml":   "xml",
		".html":  "html",
		".css":   "css",
		".sql":   "sql",
		".sh":    "bash",
	}
	if lang, exists := langMap[ext]; exists {
		return lang
	}
	return "text"
}

// SearchMemoryHandler - Advanced memory search with ranking
func SearchMemoryHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Query   string   `json:"query"`
		Tags    []string `json:"tags"`
		Project string   `json:"project"`
		Limit   int      `json:"limit"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 10
	}

	srv, ok := server.(ServerInterface)
	if !ok {
		return createErrorResponse("Server interface error")
	}

	memory := srv.GetMemory()
	if memory == nil {
		return createErrorResponse("Memory manager not available")
	}

	results, err := memory.SearchWithProject(params.Query, params.Tags, params.Project)
	if err != nil {
		return createErrorResponse(fmt.Sprintf("Search failed: %v", err))
	}

	if len(results) == 0 {
		return textResponse("No memories found matching your query"), nil
	}

	items := results
	if len(items) > params.Limit {
		items = items[:params.Limit]
	}

	return textResponse(formatSearchResults(items)), nil
}

// SaveDecisionHandler - Stores a technical decision with structured metadata
func SaveDecisionHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Key          string   `json:"key"`
		Content      string   `json:"content"`
		DecisionType string   `json:"decisionType"` // architecture, fix, approach, tech-debt, etc.
		Reason       string   `json:"reason"`       // why this decision was made
		Alternatives []string `json:"alternatives"` // what else was considered
		Tags         []string `json:"tags"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Key == "" || params.Content == "" {
		return createErrorResponse("key and content are required")
	}

	// Default decision type
	if params.DecisionType == "" {
		params.DecisionType = "technical"
	}

	// Auto-generate tags if not provided
	if len(params.Tags) == 0 {
		params.Tags = []string{"decision", params.DecisionType}
	}

	srv, ok := server.(ServerInterface)
	if !ok {
		return createErrorResponse("Server interface error")
	}

	memory := srv.GetMemory()
	if memory == nil {
		return createErrorResponse("Memory manager not available")
	}

	err := memory.StoreWithType(params.Key, params.Content, params.Tags, params.DecisionType, params.Reason, params.Alternatives)
	if err != nil {
		return createErrorResponse(fmt.Sprintf("Failed to save decision: %v", err))
	}

	return textResponse(fmt.Sprintf("Decision saved successfully:\n- Key: %s\n- Type: %s\n- Tags: %v", params.Key, params.DecisionType, params.Tags)), nil
}

// GetDecisionsHandler - Retrieves decisions with optional filtering
func GetDecisionsHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		DecisionType string `json:"decisionType"`
		Keyword      string `json:"keyword"`
		Limit        int    `json:"limit"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 10
	}

	srv, ok := server.(ServerInterface)
	if !ok {
		return createErrorResponse("Server interface error")
	}

	memory := srv.GetMemory()
	if memory == nil {
		return createErrorResponse("Memory manager not available")
	}

	results, err := memory.SearchDecisions(params.DecisionType, params.Keyword, params.Limit)
	if err != nil {
		return createErrorResponse(fmt.Sprintf("Failed to get decisions: %v", err))
	}

	if len(results) == 0 {
		return textResponse("No decisions found matching your criteria"), nil
	}

	var result strings.Builder
	result.WriteString("# Technical Decisions\n\n")

	for _, mem := range results {
		result.WriteString(fmt.Sprintf("## %s\n", mem.Key))
		result.WriteString(fmt.Sprintf("**Type**: %s\n", mem.DecisionType))
		result.WriteString(fmt.Sprintf("**Created**: %s\n\n", mem.Timestamp.Format("2006-01-02 15:04")))
		result.WriteString(mem.Content + "\n\n")
		if mem.Reason != "" {
			result.WriteString(fmt.Sprintf("**Reason**: %s\n\n", mem.Reason))
		}
		if len(mem.Alternatives) > 0 {
			result.WriteString("**Alternatives considered**:\n")
			for _, alt := range mem.Alternatives {
				result.WriteString(fmt.Sprintf("- %s\n", alt))
			}
			result.WriteString("\n")
		}
		result.WriteString("---\n\n")
	}

	return textResponse(result.String()), nil
}

// PromoteMemoryHandler - Marks a memory as promoted (high-value, from session convergence)
func PromoteMemoryHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Key       string `json:"key"`       // memory key to promote
		Confidence string `json:"confidence"` // "low", "medium", "high"
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Key == "" {
		return createErrorResponse("key is required")
	}

	// Default confidence if not provided
	if params.Confidence == "" {
		params.Confidence = "high"
	}

	srv, ok := server.(ServerInterface)
	if !ok {
		return createErrorResponse("Server interface error")
	}

	mem := srv.GetMemory()
	if mem == nil {
		return createErrorResponse("Memory manager not available")
	}

	// Verify memory exists
	existing, err := mem.Retrieve(params.Key)
	if err != nil {
		return createErrorResponse(fmt.Sprintf("Memory not found: %s", params.Key))
	}

	// Promote it
	if err := mem.Promote(params.Key, params.Confidence); err != nil {
		return createErrorResponse(fmt.Sprintf("Failed to promote memory: %v", err))
	}

	return textResponse(fmt.Sprintf(
		"Memory '%s' promoted to persistent storage.\n- Confidence: %s\n- Tags: %v\n- Content preview: %s...",
		params.Key,
		params.Confidence,
		existing.Tags,
		truncate(existing.Content, 100),
	)), nil
}

// GetPromotedMemoriesHandler - Returns memories that have been promoted from session
func GetPromotedMemoriesHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Limit int `json:"limit"` // max results (default 10)
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	if params.Limit == 0 {
		params.Limit = 10
	}

	srv, ok := server.(ServerInterface)
	if !ok {
		return createErrorResponse("Server interface error")
	}

	mem := srv.GetMemory()
	if mem == nil {
		return createErrorResponse("Memory manager not available")
	}

	promoted, err := mem.GetPromotedMemories(params.Limit)
	if err != nil {
		return createErrorResponse(fmt.Sprintf("Failed to get promoted memories: %v", err))
	}

	if len(promoted) == 0 {
		return textResponse("No promoted memories found.\n\nUse promote-memory to mark high-value memories for persistence."), nil
	}

	var result strings.Builder
	result.WriteString("# Promoted Memories (High-Value)\n\n")
	result.WriteString(fmt.Sprintf("Found %d promoted memories:\n\n", len(promoted)))

	for _, m := range promoted {
		result.WriteString(fmt.Sprintf("## %s\n", m.Key))
		result.WriteString(fmt.Sprintf("**Confidence**: %s | **Type**: %s\n", m.Confidence, m.DecisionType))
		result.WriteString(fmt.Sprintf("**Tags**: %v\n", m.Tags))
		result.WriteString(fmt.Sprintf("**Promoted**: %s\n\n", m.Timestamp.Format("2006-01-02 15:04")))
		result.WriteString(m.Content + "\n\n")
		if m.Reason != "" {
			result.WriteString(fmt.Sprintf("**Reason**: %s\n\n", m.Reason))
		}
		result.WriteString("---\n\n")
	}

	return textResponse(result.String()), nil
}

// SuggestPromotionsHandler - Suggests high-value memories worth promoting (Fase 1)
func SuggestPromotionsHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Limit int `json:"limit"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	if params.Limit == 0 {
		params.Limit = 5
	}

	srv, ok := server.(ServerInterface)
	if !ok {
		return createErrorResponse("Server interface error")
	}

	mem := srv.GetMemory()
	if mem == nil {
		return createErrorResponse("Memory manager not available")
	}

	suggestions, err := mem.SuggestForPromotion(params.Limit)
	if err != nil {
		return createErrorResponse(fmt.Sprintf("Failed to generate suggestions: %v", err))
	}

	if len(suggestions) == 0 {
		return textResponse("No strong promotion candidates found at the moment.\n\nConsider using save-decision for important technical choices so they can be suggested later."), nil
	}

	var result strings.Builder
	result.WriteString("# Suggested Memories for Promotion\n\n")
	result.WriteString(fmt.Sprintf("Found %d high-value candidates (use `promote-memory` with the key):\n\n", len(suggestions)))

	for i, m := range suggestions {
		result.WriteString(fmt.Sprintf("## %d. %s\n", i+1, m.Key))
		result.WriteString(fmt.Sprintf("**Type**: %s | **Usage**: %d | **Age**: %s\n",
			m.DecisionType, m.Usage, m.Timestamp.Format("2006-01-02")))
		if m.Confidence != "" {
			result.WriteString(fmt.Sprintf("**Confidence**: %s\n", m.Confidence))
		}
		result.WriteString(fmt.Sprintf("**Tags**: %v\n\n", m.Tags))
		result.WriteString(m.Content + "\n\n")
		if m.Reason != "" {
			result.WriteString(fmt.Sprintf("**Reason**: %s\n\n", m.Reason))
		}
		result.WriteString(fmt.Sprintf("**Suggested action**:\n`promote-memory` with key = \"%s\"\n\n", m.Key))
		result.WriteString("---\n\n")
	}

	result.WriteString("\nAfter promoting, these memories will be returned by `get-promoted-memories` and will have higher priority in future context retrieval.\n")

	return textResponse(result.String()), nil
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// formatSearchResults formats memory search results for display
func formatSearchResults(items []*memory.Memory) string {
	if len(items) == 0 {
		return "No results found"
	}

	var result strings.Builder
	result.WriteString("# Memory Search Results\n\n")
	result.WriteString(fmt.Sprintf("Found %d results:\n\n", len(items)))

	for _, item := range items {
		result.WriteString(fmt.Sprintf("## %s\n", item.Key))
		result.WriteString(fmt.Sprintf("**Tags**: %v\n", item.Tags))
		result.WriteString(fmt.Sprintf("**Usage**: %d times\n", item.Usage))
		result.WriteString(fmt.Sprintf("**Content**:\n%s\n\n", item.Content))
	}

	return result.String()
}
