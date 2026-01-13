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

	// Import the actual types we need
	"github.com/scopweb/mcp-go-context/internal/analyzer"
	"github.com/scopweb/mcp-go-context/internal/cache"
	"github.com/scopweb/mcp-go-context/internal/memory"
)

// Pre-compiled regexes for performance
var (
	validLibraryName = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,64}$`)

	// Query analysis patterns
	patternDebug       = regexp.MustCompile(`error|bug|fix|debug`)
	patternTest        = regexp.MustCompile(`test|testing|unit`)
	patternAPI         = regexp.MustCompile(`api|endpoint|route|handler`)
	patternDatabase    = regexp.MustCompile(`database|db|sql|query`)
	patternConfig      = regexp.MustCompile(`config|configuration|setting`)
	patternDeploy      = regexp.MustCompile(`deploy|deployment|docker`)
	patternSecurity    = regexp.MustCompile(`security|auth|permission`)
	patternPerformance = regexp.MustCompile(`performance|optimize|slow`)

	// Tag generation patterns
	tagBug         = regexp.MustCompile(`error|bug|issue|problem`)
	tagTest        = regexp.MustCompile(`test|testing|spec`)
	tagConfig      = regexp.MustCompile(`config|configuration`)
	tagAPI         = regexp.MustCompile(`api|endpoint|route`)
	tagDatabase    = regexp.MustCompile(`database|db|sql`)
	tagDeploy      = regexp.MustCompile(`deploy|deployment`)
	tagSecurity    = regexp.MustCompile(`security|auth`)
	tagPerformance = regexp.MustCompile(`performance|optimize`)
	tagFeature     = regexp.MustCompile(`feature|functionality`)
	tagDocs        = regexp.MustCompile(`documentation|docs`)
)

// Type aliases to match the actual implementation types
type Dependency = analyzer.Dependency
type Memory = memory.Memory
type ProjectStructure = analyzer.ProjectStructure
type FileInfo = analyzer.FileInfo
type ProjectStats = analyzer.ProjectStats

// ServerInterface defines methods needed from the server
type ServerInterface interface {
	GetAnalyzer() AnalyzerInterface
	GetMemory() MemoryInterface
	GetConfig() ConfigInterface
	GetContextCache() *cache.ContextCache
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
	GetRecentMemories(int) ([]*Memory, error)
	Clear() error
}

type ConfigInterface interface {
	GetProjectPaths() []string
}

// Keep existing structs that are tools-specific (now using type aliases)
// All these types are now aliases to the original analyzer types

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
	// Validar ruta: solo relativa, sin .., sin caracteres peligrosos
	if params.Path != "" {
		clean := filepath.Clean(params.Path)
		if strings.Contains(clean, "..") || strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "\\") {
			return createErrorResponse("Invalid path: must be relative and safe")
		}
		params.Path = clean
	} else {
		params.Path = "."
	}
	if params.Depth < 0 || params.Depth > 10 {
		return createErrorResponse("Invalid depth: must be 0-10")
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
	fmt.Fprintf(&result, "# Project Analysis: %s\n\n", structure.RootPath)

	// Stats summary
	result.WriteString("## 📊 Project Statistics\n")
	fmt.Fprintf(&result, "- **Total Files**: %d\n", structure.Stats.TotalFiles)
	fmt.Fprintf(&result, "- **Total Size**: %.2f MB\n", float64(structure.Stats.TotalSize)/(1024*1024))

	// Languages breakdown
	result.WriteString("\n### Languages Distribution\n")
	for lang, count := range structure.Stats.Languages {
		percentage := float64(count) / float64(structure.Stats.TotalFiles) * 100
		fmt.Fprintf(&result, "- **%s**: %d files (%.1f%%)\n", lang, count, percentage)
	}

	// Directory structure
	result.WriteString("\n## 📁 Directory Structure\n")
	for dir, files := range structure.Structure {
		if len(files) > 0 {
			fmt.Fprintf(&result, "- `%s/` (%d files)\n", dir, len(files))
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
		fmt.Fprintf(&result, "- **Direct**: %d dependencies\n", directDeps)
		fmt.Fprintf(&result, "- **Indirect**: %d dependencies\n", indirectDeps)

		// Show top dependencies
		result.WriteString("\n### Key Dependencies\n")
		count := 0
		for _, dep := range structure.Dependencies {
			if dep.Type == "direct" && count < 10 {
				fmt.Fprintf(&result, "- `%s` %s\n", dep.Name, dep.Version)
				count++
			}
		}
	}

	// Important files
	result.WriteString("\n## 🔍 Key Files\n")
	keyFiles := findKeyFiles(structure.Files)
	for _, file := range keyFiles {
		relPath, _ := filepath.Rel(structure.RootPath, file.Path)
		fmt.Fprintf(&result, "- `%s` (%s, %.2f KB)\n",
			relPath, file.Language, float64(file.Size)/1024)
	}

	return []map[string]interface{}{
		{
			"type": "text",
			"text": result.String(),
		},
	}, nil
}

// GetContextHandler - Complete implementation with smart context retrieval and caching
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

	// Generate cache key from query and files
	cacheKey := cache.GenerateCacheKey(params.Query, "balanced", params.Files, ".")

	// Try to get from cache first
	contextCache := srv.GetContextCache()
	if contextCache != nil {
		if cachedValue, found := contextCache.Get(cacheKey); found {
			if cachedText, ok := cachedValue.(string); ok {
				return []map[string]interface{}{
					{
						"type": "text",
						"text": cachedText + "\n\n*[Retrieved from cache - context valid for 30 minutes]*",
					},
				}, nil
			}
		}
	}

	analyzer := srv.GetAnalyzer()
	memory := srv.GetMemory()

	var context strings.Builder
	fmt.Fprintf(&context, "# Context for: %s\n\n", params.Query)

	// Add relevant memory
	if memory != nil {
		memories, err := memory.Search(params.Query, []string{})
		if err == nil && len(memories) > 0 {
			context.WriteString("## 💭 Relevant Memory\n\n")
			for i, mem := range memories {
				if i >= 3 {
					break
				}
				fmt.Fprintf(&context, "**%s**: %s\n\n", mem.Key, mem.Content)
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

	// Cache the result
	contextText := context.String()
	if contextCache != nil {
		contextCache.Set(cacheKey, contextText, 0) // Use default TTL (30 minutes)
	}

	return []map[string]interface{}{
		{
			"type": "text",
			"text": contextText,
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
	// Validar nombre de librería: solo letras, números, guiones, puntos y guiones bajos
	if !validLibraryName.MatchString(params.Library) {
		return createErrorResponse("Invalid library name")
	}
	if len(params.Version) > 32 {
		return createErrorResponse("Version string too long")
	}
	if len(params.Topic) > 64 {
		return createErrorResponse("Topic string too long")
	}
	if params.Tokens <= 0 || params.Tokens > 20000 {
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
	fmt.Fprintf(&result, "## Direct Dependencies (%d)\n\n", len(directDeps))
	for _, dep := range directDeps {
		fmt.Fprintf(&result, "- **%s** `%s`", dep.Name, dep.Version)
		if params.SuggestDocs {
			docSuggestion := suggestDocumentation(dep.Name)
			if docSuggestion != "" {
				fmt.Fprintf(&result, " - [📚 Docs](%s)", docSuggestion)
			}
		}
		result.WriteString("\n")
	}

	// Indirect dependencies if requested
	if params.IncludeTransitive && len(indirectDeps) > 0 {
		fmt.Fprintf(&result, "\n## Indirect Dependencies (%d)\n\n", len(indirectDeps))
		// Show only first 20 to avoid clutter
		displayCount := min(20, len(indirectDeps))
		for i, dep := range indirectDeps[:displayCount] {
			fmt.Fprintf(&result, "%d. %s `%s`\n", i+1, dep.Name, dep.Version)
		}
		if len(indirectDeps) > 20 {
			fmt.Fprintf(&result, "\n... and %d more indirect dependencies\n", len(indirectDeps)-20)
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

	// Pattern matching for different query types using pre-compiled regexes
	if patternDebug.MatchString(query) {
		return "🐛 Debugging context - Look for error handling, logs, and related functions\n"
	}
	if patternTest.MatchString(query) {
		return "🧪 Testing context - Focus on test files and testing utilities\n"
	}
	if patternAPI.MatchString(query) {
		return "🌐 API context - Examine route handlers and API definitions\n"
	}
	if patternDatabase.MatchString(query) {
		return "💾 Database context - Check database models and queries\n"
	}
	if patternConfig.MatchString(query) {
		return "⚙️ Configuration context - Look at config files and environment setup\n"
	}
	if patternDeploy.MatchString(query) {
		return "🚀 Deployment context - Focus on deployment and infrastructure files\n"
	}
	if patternSecurity.MatchString(query) {
		return "🔒 Security context - Examine authentication and authorization code\n"
	}
	if patternPerformance.MatchString(query) {
		return "⚡ Performance context - Look for bottlenecks and optimization opportunities\n"
	}

	return ""
}

// New tools: memory/get
func MemoryGetHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	if params.Key == "" {
		return createErrorResponse("Key is required")
	}
	srv, ok := server.(ServerInterface)
	if !ok {
		return createErrorResponse("Server interface error")
	}
	mem := srv.GetMemory()
	if mem == nil {
		return createErrorResponse("Memory manager not available")
	}
	item, err := mem.Retrieve(params.Key)
	if err != nil {
		return createErrorResponse(fmt.Sprintf("Memory not found: %v", err))
	}
	return []map[string]interface{}{{
		"type": "text",
		"text": fmt.Sprintf("%s", item.Content),
	}}, nil
}

// memory/search
func MemorySearchHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Query string   `json:"query"`
		Tags  []string `json:"tags"`
		Limit int      `json:"limit"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	if params.Limit <= 0 || params.Limit > 50 {
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
	found, err := mem.Search(params.Query, params.Tags)
	if err != nil {
		return createErrorResponse(fmt.Sprintf("Search failed: %v", err))
	}
	if len(found) > params.Limit {
		found = found[:params.Limit]
	}
	var b strings.Builder
	b.WriteString("# Memory Search Results\n\n")
	for _, m := range found {
		fmt.Fprintf(&b, "- %s: %s\n", m.Key, m.Content)
	}
	return []map[string]interface{}{{
		"type": "text",
		"text": b.String(),
	}}, nil
}

// memory/recent
func MemoryRecentHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Limit int `json:"limit"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	if params.Limit <= 0 || params.Limit > 50 {
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
	list, err := mem.GetRecentMemories(params.Limit)
	if err != nil {
		return createErrorResponse(fmt.Sprintf("Failed to get recent: %v", err))
	}
	var b strings.Builder
	b.WriteString("# Recent Memories\n\n")
	for _, m := range list {
		fmt.Fprintf(&b, "- %s: %s\n", m.Key, m.Content)
	}
	return []map[string]interface{}{{
		"type": "text",
		"text": b.String(),
	}}, nil
}

// memory/clear (destructive; require explicit confirm)
func MemoryClearHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	var params struct {
		Confirm string `json:"confirm"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}
	if params.Confirm != "YES_I_UNDERSTAND" {
		return createErrorResponse("Dangerous operation: missing confirmation")
	}
	srv, ok := server.(ServerInterface)
	if !ok {
		return createErrorResponse("Server interface error")
	}
	mem := srv.GetMemory()
	if mem == nil {
		return createErrorResponse("Memory manager not available")
	}
	if err := mem.Clear(); err != nil {
		return createErrorResponse(fmt.Sprintf("Clear failed: %v", err))
	}
	return []map[string]interface{}{{
		"type": "text",
		"text": "✅ Memory cleared",
	}}, nil
}

// config/get-project-paths
func ConfigGetProjectPathsHandler(args json.RawMessage, server interface{}) (interface{}, error) {
	srv, ok := server.(ServerInterface)
	if !ok {
		return createErrorResponse("Server interface error")
	}
	cfg := srv.GetConfig()
	if cfg == nil {
		return createErrorResponse("Config not available")
	}
	paths := cfg.GetProjectPaths()
	var b strings.Builder
	b.WriteString("# Project Paths\n\n")
	for _, p := range paths {
		b.WriteString("- `" + p + "`\n")
	}
	return []map[string]interface{}{{
		"type": "text",
		"text": b.String(),
	}}, nil
}

func fetchFromContext7(library, version, topic string, tokens int) (string, error) {
	// Context7 API integration
	baseURL := "https://context7.com/api/v1"
	var url string

	if !validLibraryName.MatchString(library) {
		return "", fmt.Errorf("invalid library name")
	}
	if library == "" {
		return "", fmt.Errorf("library name required")
	}

	url = fmt.Sprintf("%s/%s", baseURL, library)
	if version != "" {
		url = fmt.Sprintf("%s/%s", url, version)
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

	// Limit response size to 5MB to prevent memory issues
	limitedReader := io.LimitReader(resp.Body, 5*1024*1024)
	body, err := io.ReadAll(limitedReader)
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
	// Solo permitir rutas bajo ./docs, ./doc, ./documentation o archivos .md en raíz
	allowed := false
	allowedDirs := []string{"./docs", "./doc", "./documentation"}
	absPath, _ := filepath.Abs(path)
	for _, dir := range allowedDirs {
		absDir, _ := filepath.Abs(dir)
		if strings.HasPrefix(absPath, absDir) {
			allowed = true
			break
		}
	}
	if !allowed && !(filepath.Ext(path) == ".md" && (filepath.Dir(path) == "." || filepath.Dir(path) == "/")) {
		return ""
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ""
	}
	// Simple file content search con límite de tamaño
	if strings.HasSuffix(path, ".md") {
		file, err := os.Open(path)
		if err != nil {
			return ""
		}
		defer file.Close()
		stat, _ := file.Stat()
		if stat.Size() > 1024*1024 { // 1MB máx
			return ""
		}
		// Use LimitReader to prevent reading excessive data
		limitedReader := io.LimitReader(file, 1024*1024)
		content, err := io.ReadAll(limitedReader)
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

	// Common tag patterns using pre-compiled regexes
	if tagBug.MatchString(content) {
		tags = append(tags, "bug")
	}
	if tagTest.MatchString(content) {
		tags = append(tags, "testing")
	}
	if tagConfig.MatchString(content) {
		tags = append(tags, "config")
	}
	if tagAPI.MatchString(content) {
		tags = append(tags, "api")
	}
	if tagDatabase.MatchString(content) {
		tags = append(tags, "database")
	}
	if tagDeploy.MatchString(content) {
		tags = append(tags, "deployment")
	}
	if tagSecurity.MatchString(content) {
		tags = append(tags, "security")
	}
	if tagPerformance.MatchString(content) {
		tags = append(tags, "performance")
	}
	if tagFeature.MatchString(content) {
		tags = append(tags, "feature")
	}
	if tagDocs.MatchString(content) {
		tags = append(tags, "docs")
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
