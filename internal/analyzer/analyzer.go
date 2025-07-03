package analyzer

import (
	"bufio"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/scopweb/mcp-go-context/internal/config"
)

// ProjectAnalyzer analyzes project structure and content
type ProjectAnalyzer struct {
	config config.ContextConfig
	cache  map[string]*FileInfo
}

// FileInfo contains information about a file
type FileInfo struct {
	Path         string
	Size         int64
	Language     string
	Imports      []string
	Functions    []string
	Types        []string
	LastModified int64
}

// ProjectStructure represents the analyzed project
type ProjectStructure struct {
	RootPath     string
	Files        []*FileInfo
	Dependencies []Dependency
	Structure    map[string][]string // directory -> files
	Stats        ProjectStats
}

// ProjectStats contains project statistics
type ProjectStats struct {
	TotalFiles   int
	TotalLines   int
	Languages    map[string]int
	TotalSize    int64
	GoModules    []string
	MainPackages []string
}

// Dependency represents a project dependency
type Dependency struct {
	Name    string
	Version string
	Type    string // direct, indirect
	Path    string
}

// New creates a new project analyzer
func New(cfg config.ContextConfig) (*ProjectAnalyzer, error) {
	return &ProjectAnalyzer{
		config: cfg,
		cache:  make(map[string]*FileInfo),
	}, nil
}

// AnalyzeProject performs a comprehensive project analysis
func (a *ProjectAnalyzer) AnalyzeProject(rootPath string, depth int) (*ProjectStructure, error) {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Check if path exists and is accessible
	if _, err := os.Stat(absPath); err != nil {
		return nil, fmt.Errorf("path not accessible: %w", err)
	}

	ps := &ProjectStructure{
		RootPath:  absPath,
		Files:     []*FileInfo{},
		Structure: make(map[string][]string),
		Stats: ProjectStats{
			Languages: make(map[string]int),
		},
	}

	// Walk project directory with timeout protection
	err = filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip problematic entries instead of failing
		}

		// Check ignore patterns
		if a.shouldIgnore(path) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, _ := filepath.Rel(absPath, path)

		if d.IsDir() {
			ps.Structure[relPath] = []string{}
			return nil
		}

		// Analyze file (with error protection)
		info, err := a.analyzeFile(path)
		if err != nil {
			// Skip files that can't be analyzed instead of failing
			return nil
		}

		ps.Files = append(ps.Files, info)

		// Update structure
		dir := filepath.Dir(relPath)
		ps.Structure[dir] = append(ps.Structure[dir], filepath.Base(path))

		// Update stats
		ps.Stats.TotalFiles++
		ps.Stats.TotalSize += info.Size
		ps.Stats.Languages[info.Language]++

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Analyze dependencies if Go project (with error protection)
	if a.config.AutoDetectDeps {
		deps, err := a.AnalyzeDependencies(false)
		if err == nil {
			ps.Dependencies = deps
		}
		// If dependency analysis fails, continue without dependencies
	}

	return ps, nil
}

// analyzeFile analyzes a single file
func (a *ProjectAnalyzer) analyzeFile(path string) (*FileInfo, error) {
	// Check cache
	if info, exists := a.cache[path]; exists {
		stat, err := os.Stat(path)
		if err == nil && stat.ModTime().Unix() == info.LastModified {
			return info, nil
		}
	}

	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	info := &FileInfo{
		Path:         path,
		Size:         stat.Size(),
		Language:     detectLanguage(path),
		LastModified: stat.ModTime().Unix(),
	}

	// Special handling for Go files
	if strings.HasSuffix(path, ".go") {
		a.analyzeGoFile(path, info)
	}

	// Cache the result
	a.cache[path] = info

	return info, nil
}

// analyzeGoFile performs Go-specific analysis
func (a *ProjectAnalyzer) analyzeGoFile(path string, info *FileInfo) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, content, parser.ImportsOnly)
	if err != nil {
		return err
	}

	// Extract imports
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		info.Imports = append(info.Imports, importPath)
	}

	return nil
}

// GetRelevantContext retrieves context relevant to a query
func (a *ProjectAnalyzer) GetRelevantContext(query string, files []string, maxTokens int) (string, error) {
	var context strings.Builder
	tokenCount := 0

	context.WriteString(fmt.Sprintf("# Context for: %s\n\n", query))

	// If specific files requested
	if len(files) > 0 {
		for _, file := range files {
			content, err := a.getFileContext(file, maxTokens-tokenCount)
			if err != nil {
				continue
			}
			context.WriteString(content)
			tokenCount += len(content) / 4 // Approximate token count

			if tokenCount >= maxTokens {
				break
			}
		}
	} else {
		// Find relevant files based on query
		relevantFiles := a.findRelevantFiles(query)
		for _, file := range relevantFiles {
			content, err := a.getFileContext(file.Path, maxTokens-tokenCount)
			if err != nil {
				continue
			}
			context.WriteString(content)
			tokenCount += len(content) / 4

			if tokenCount >= maxTokens {
				break
			}
		}
	}

	return context.String(), nil
}

// AnalyzeDependencies analyzes project dependencies
func (a *ProjectAnalyzer) AnalyzeDependencies(includeTransitive bool) ([]Dependency, error) {
	var deps []Dependency

	// Check for go.mod
	goModPath := filepath.Join(a.config.ProjectPaths[0], "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		deps, err = a.parseGoMod(goModPath, includeTransitive)
		if err != nil {
			return nil, err
		}
	}

	// TODO: Add support for other dependency files (package.json, requirements.txt, etc.)

	return deps, nil
}

// parseGoMod parses go.mod file for dependencies
func (a *ProjectAnalyzer) parseGoMod(path string, includeTransitive bool) ([]Dependency, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var deps []Dependency
	scanner := bufio.NewScanner(file)
	inRequire := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "require (" {
			inRequire = true
			continue
		}

		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		if inRequire || strings.HasPrefix(line, "require ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[0]
				if name == "require" {
					name = parts[1]
				}
				version := ""
				if len(parts) > 2 {
					version = parts[2]
				} else if len(parts) > 1 && parts[0] != "require" {
					version = parts[1]
				}

				depType := "direct"
				if strings.Contains(line, "// indirect") {
					depType = "indirect"
				}

				if includeTransitive || depType == "direct" {
					deps = append(deps, Dependency{
						Name:    name,
						Version: version,
						Type:    depType,
						Path:    path,
					})
				}
			}
		}
	}

	return deps, scanner.Err()
}

// Helper methods

func (a *ProjectAnalyzer) shouldIgnore(path string) bool {
	for _, pattern := range a.config.IgnorePatterns {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

func (a *ProjectAnalyzer) findRelevantFiles(query string) []*FileInfo {
	var relevant []*FileInfo
	queryLower := strings.ToLower(query)

	for _, file := range a.cache {
		score := 0

		// Check filename
		if strings.Contains(strings.ToLower(filepath.Base(file.Path)), queryLower) {
			score += 10
		}

		// Check imports (for Go files)
		for _, imp := range file.Imports {
			if strings.Contains(strings.ToLower(imp), queryLower) {
				score += 5
			}
		}

		if score > 0 {
			relevant = append(relevant, file)
		}
	}

	// Sort by relevance (implement sorting if needed)

	return relevant
}

func (a *ProjectAnalyzer) getFileContext(path string, maxChars int) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	result := fmt.Sprintf("\n## File: %s\n\n```%s\n", path, detectLanguage(path))

	if len(content) > maxChars {
		result += string(content[:maxChars])
		result += "\n... (truncated)\n"
	} else {
		result += string(content)
	}

	result += "\n```\n\n"

	return result, nil
}

func detectLanguage(path string) string {
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
