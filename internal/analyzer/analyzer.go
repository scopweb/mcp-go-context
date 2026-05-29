package analyzer

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/scopweb/mcp-go-context/internal/config"
)

// Fase 0 cold-start defaults
const (
	DefaultLightIndexMaxFiles    = 400
	DefaultLightIndexMaxDuration = 1800 * time.Millisecond
	LightIndexCooldown           = 5 * time.Minute
)

// ProjectAnalyzer analyzes project structure and content
type ProjectAnalyzer struct {
	config config.ContextConfig
	cache  map[string]*FileInfo

	// Light index state for cold-start optimization (Fase 0)
	lightIndexed     bool
	lightIndexedAt   time.Time
	recentlyChanged  map[string]bool // basenames of files changed in recent git commits
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
	Score        int // Relevance score for queries
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

// ChangedFile represents a file changed in git
type ChangedFile struct {
	Path     string
	Status   string // added, modified, deleted, renamed
	OldPath  string // for renamed files
	Lines    int
	SelfAuthorship bool // committed by the author themselves
}

// GitInfo contains git repository information
type GitInfo struct {
	ChangedFiles []ChangedFile
	CommitCount  int
	LastCommit   string
	Branch       string
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

	ps := &ProjectStructure{
		RootPath:  absPath,
		Files:     []*FileInfo{},
		Structure: make(map[string][]string),
		Stats: ProjectStats{
			Languages: make(map[string]int),
		},
	}

	// Walk project directory
	err = filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
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

		// Analyze file
		info, err := a.analyzeFile(path)
		if err != nil {
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

	// Analyze dependencies if Go project
	if a.config.AutoDetectDeps {
		deps, err := a.AnalyzeDependencies(false)
		if err == nil {
			ps.Dependencies = deps
		}
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
		// Fase 0: Ensure we have a light index on cold start (bounded + git-aware)
		a.EnsureLightIndex(DefaultLightIndexMaxFiles, DefaultLightIndexMaxDuration)

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

// EnsureLightIndex performs a bounded, git-aware light index of the project.
// This makes get-context useful on cold start without requiring analyze-project.
// It prioritizes recently changed files (via git) and limits work by file count and time.
func (a *ProjectAnalyzer) EnsureLightIndex(maxFiles int, maxDuration time.Duration) {
	if maxFiles <= 0 {
		maxFiles = 300
	}
	if maxDuration <= 0 {
		maxDuration = 1500 * time.Millisecond
	}

	// Avoid re-indexing too frequently
	if a.lightIndexed && time.Since(a.lightIndexedAt) < LightIndexCooldown {
		return
	}

	start := time.Now()
	filesIndexed := 0

	// Phase 1: Git-first — index recently changed files immediately (highest value for context)
	for _, projectPath := range a.config.ProjectPaths {
		changed, err := a.GetRecentlyChangedFiles(projectPath, 8)
		if err == nil && len(changed) > 0 {
			for _, cf := range changed {
				if filesIndexed >= maxFiles || time.Since(start) > maxDuration {
					break
				}
				// Build full path
				absRoot, _ := filepath.Abs(projectPath)
				fullPath := filepath.Join(absRoot, cf.Path)
				if _, err := os.Stat(fullPath); err == nil {
					if info, err := a.analyzeFile(fullPath); err == nil {
						a.cache[fullPath] = info
						filesIndexed++
						if a.recentlyChanged == nil {
							a.recentlyChanged = make(map[string]bool)
						}
						a.recentlyChanged[filepath.Base(cf.Path)] = true
					}
				}
			}
		}
	}

	// Phase 2: Bounded walk for the rest of the project (if we still have budget)
	for _, projectPath := range a.config.ProjectPaths {
		if filesIndexed >= maxFiles || time.Since(start) > maxDuration {
			break
		}

		absPath, err := filepath.Abs(projectPath)
		if err != nil {
			continue
		}

		_ = filepath.WalkDir(absPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			if filesIndexed >= maxFiles || time.Since(start) > maxDuration {
				return filepath.SkipDir
			}

			if d.IsDir() {
				if a.shouldIgnore(path) {
					return filepath.SkipDir
				}
				return nil
			}

			if a.shouldIgnore(path) {
				return nil
			}

			// Skip if we already indexed this file (e.g. from git phase)
			if _, exists := a.cache[path]; exists {
				return nil
			}

			info, err := a.analyzeFile(path)
			if err == nil {
				a.cache[path] = info
				filesIndexed++
			}
			return nil
		})
	}

	a.lightIndexed = true
	a.lightIndexedAt = time.Now()
}

// quickDiscovery kept for backward compatibility (delegates to EnsureLightIndex)
func (a *ProjectAnalyzer) quickDiscovery() {
	a.EnsureLightIndex(DefaultLightIndexMaxFiles, DefaultLightIndexMaxDuration)
}

// AnalyzeDependencies analyzes project dependencies
func (a *ProjectAnalyzer) AnalyzeDependencies(includeTransitive bool) ([]Dependency, error) {
	var deps []Dependency

	// FASE 3 FIX: Iterate over all ProjectPaths for multi-ecosystem/monorepo support
	for _, projectPath := range a.config.ProjectPaths {
		absPath, err := filepath.Abs(projectPath)
		if err != nil {
			continue
		}

		// Check for go.mod
		goModPath := filepath.Join(absPath, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			goDeps, err := a.parseGoMod(goModPath, includeTransitive)
			if err == nil {
				deps = append(deps, goDeps...)
			}
		}

		// Check for package.json (Node.js)
		pkgPath := filepath.Join(absPath, "package.json")
		if _, err := os.Stat(pkgPath); err == nil {
			pkgDeps, err := a.parsePackageJSON(pkgPath, includeTransitive)
			if err == nil {
				deps = append(deps, pkgDeps...)
			}
		}

		// Check for pyproject.toml (Python)
		pyprojectPath := filepath.Join(absPath, "pyproject.toml")
		if _, err := os.Stat(pyprojectPath); err == nil {
			pyDeps, err := a.parsePyproject(pyprojectPath, includeTransitive)
			if err == nil {
				deps = append(deps, pyDeps...)
			}
		}

		// Check for requirements.txt (Python)
		requirementsPath := filepath.Join(absPath, "requirements.txt")
		if _, err := os.Stat(requirementsPath); err == nil {
			reqDeps, err := a.parseRequirements(requirementsPath, includeTransitive)
			if err == nil {
				deps = append(deps, reqDeps...)
			}
		}

		// Check for pnpm-workspace.yaml (pnpm monorepo)
		pnpmWorkspacePath := filepath.Join(absPath, "pnpm-workspace.yaml")
		if _, err := os.Stat(pnpmWorkspacePath); err == nil {
			// Find all packages in the monorepo
			monorepoDeps := a.findMonorepoPackages(absPath, includeTransitive)
			deps = append(deps, monorepoDeps...)
		}
	}

	return deps, nil
}

// findMonorepoPackages finds packages in a pnpm monorepo
func (a *ProjectAnalyzer) findMonorepoPackages(rootPath string, includeTransitive bool) []Dependency {
	var deps []Dependency
	packagesDir := filepath.Join(rootPath, "packages")

	info, err := os.Stat(packagesDir)
	if err != nil || !info.IsDir() {
		return deps
	}

	entries, err := os.ReadDir(packagesDir)
	if err != nil {
		return deps
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pkgPath := filepath.Join(packagesDir, entry.Name(), "package.json")
		if _, err := os.Stat(pkgPath); err == nil {
			pkgDeps, err := a.parsePackageJSON(pkgPath, includeTransitive)
			if err == nil {
				for i := range pkgDeps {
					pkgDeps[i].Path = pkgPath // Mark with full path
				}
				deps = append(deps, pkgDeps...)
			}
		}
	}

	return deps
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

// parsePackageJSON parses package.json for npm dependencies
func (a *ProjectAnalyzer) parsePackageJSON(path string, includeTransitive bool) ([]Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Simple JSON parsing without external dependencies
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	var deps []Dependency

	// Add regular dependencies
	for name, version := range pkg.Dependencies {
		deps = append(deps, Dependency{
			Name:    name,
			Version: version,
			Type:    "direct",
			Path:    path,
		})
	}

	// Add dev dependencies if including transitive
	if includeTransitive {
		for name, version := range pkg.DevDependencies {
			deps = append(deps, Dependency{
				Name:    name,
				Version: version,
				Type:    "direct",
				Path:    path,
			})
		}
	}

	return deps, nil
}

// parsePyproject parses pyproject.toml for Python dependencies
func (a *ProjectAnalyzer) parsePyproject(path string, includeTransitive bool) ([]Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	var deps []Dependency

	// Simple TOML parsing for common dependency sections.
	// Supports PEP 621 `[project] dependencies = [...]` and table-based formats.
	lines := strings.Split(content, "\n")

	inProjectSection := false
	inDeps := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if line == "[project]" {
			inProjectSection = true
			inDeps = false
			continue
		}

		// Detect dependency sections
		if strings.HasPrefix(line, "[project.dependencies]") ||
			strings.HasPrefix(line, "[tool.poetry.dependencies]") ||
			strings.HasPrefix(line, "[project.optional-dependencies]") ||
			strings.HasPrefix(line, "[tool pdm.dev-dependencies]") {
			inProjectSection = false
			inDeps = true
			continue
		}

		// End of section
		if strings.HasPrefix(line, "[") {
			inProjectSection = false
			inDeps = false
			continue
		}

		if inProjectSection && strings.HasPrefix(line, "dependencies = [") {
			for _, dep := range parsePyprojectArray(line) {
				deps = append(deps, Dependency{
					Name:    dep.Name,
					Version: dep.Version,
					Type:    "direct",
					Path:    path,
				})
			}
			continue
		}

		if inDeps {
			// Parse dependency line (format: package-name = "version" or package-name>=version)
			if idx := strings.Index(line, "="); idx > 0 {
				name := strings.TrimSpace(line[:idx])
				name = strings.ReplaceAll(name, "\"", "")
				name = strings.ReplaceAll(name, "'", "")

				rest := strings.TrimSpace(line[idx+1:])
				version := strings.TrimSpace(rest)
				version = strings.ReplaceAll(version, "\"", "")
				version = strings.ReplaceAll(version, "'", "")

				// Clean name (remove extras like [extra])
				if idx := strings.Index(name, "["); idx > 0 {
					name = name[:idx]
				}

				deps = append(deps, Dependency{
					Name:    name,
					Version: version,
					Type:    "direct",
					Path:    path,
				})
			}
		}
	}

	return deps, nil
}

func parsePyprojectArray(line string) []Dependency {
	start := strings.Index(line, "[")
	end := strings.LastIndex(line, "]")
	if start < 0 || end <= start {
		return nil
	}

	rawItems := strings.Split(line[start+1:end], ",")
	deps := make([]Dependency, 0, len(rawItems))
	for _, item := range rawItems {
		item = strings.Trim(strings.TrimSpace(item), "\"'")
		if item == "" {
			continue
		}

		name, version := splitRequirement(item)
		deps = append(deps, Dependency{Name: name, Version: version})
	}

	return deps
}

func splitRequirement(req string) (string, string) {
	operators := []string{"==", ">=", "<=", "~=", "!=", ">", "<"}
	for _, op := range operators {
		if idx := strings.Index(req, op); idx >= 0 {
			return strings.TrimSpace(req[:idx]), strings.TrimSpace(req[idx:])
		}
	}
	return strings.TrimSpace(req), "any"
}

// parseRequirements parses requirements.txt for Python dependencies
func (a *ProjectAnalyzer) parseRequirements(path string, includeTransitive bool) ([]Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var deps []Dependency
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Skip options like -r, -e, --index-url, etc.
		if strings.HasPrefix(line, "-") {
			continue
		}

		// Skip git+https:// style deps (we can't resolve them easily)
		if strings.Contains(line, "git+") {
			continue
		}

		// Parse package==version or package>=version style
		var name, version string

		if idx := strings.Index(line, "=="); idx >= 0 {
			parts := strings.SplitN(line, "==", 2)
			name = strings.TrimSpace(parts[0])
			version = "==" + strings.TrimSpace(parts[1])
		} else if idx := strings.Index(line, ">="); idx >= 0 {
			parts := strings.SplitN(line, ">=", 2)
			name = strings.TrimSpace(parts[0])
			version = ">=" + strings.TrimSpace(parts[1])
		} else if idx := strings.Index(line, "<="); idx >= 0 {
			parts := strings.SplitN(line, "<=", 2)
			name = strings.TrimSpace(parts[0])
			version = "<=" + strings.TrimSpace(parts[1])
		} else if idx := strings.Index(line, "~="); idx >= 0 {
			parts := strings.SplitN(line, "~=", 2)
			name = strings.TrimSpace(parts[0])
			version = "~=" + strings.TrimSpace(parts[1])
		} else if idx := strings.Index(line, "!="); idx >= 0 {
			parts := strings.SplitN(line, "!=", 2)
			name = strings.TrimSpace(parts[0])
			version = "!=" + strings.TrimSpace(parts[1])
		} else if idx := strings.Index(line, ">"); idx >= 0 {
			parts := strings.SplitN(line, ">", 2)
			name = strings.TrimSpace(parts[0])
			version = ">" + strings.TrimSpace(parts[1])
		} else if idx := strings.Index(line, "<"); idx >= 0 {
			parts := strings.SplitN(line, "<", 2)
			name = strings.TrimSpace(parts[0])
			version = "<" + strings.TrimSpace(parts[1])
		} else {
			// No version specified
			name = line
			version = "any"
		}

		// Clean package name (remove [extra] notation)
		if idx := strings.Index(name, "["); idx > 0 {
			name = name[:idx]
		}

		if name != "" {
			deps = append(deps, Dependency{
				Name:    name,
				Version: version,
				Type:    "direct",
				Path:    path,
			})
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

		// Check filename match (strong signal)
		base := filepath.Base(file.Path)
		if strings.Contains(strings.ToLower(base), queryLower) {
			score += 12
		}

		// Check imports (for Go files)
		for _, imp := range file.Imports {
			if strings.Contains(strings.ToLower(imp), queryLower) {
				score += 5
			}
		}

		// Boost for recently modified on disk (last 7 days)
		if file.LastModified > 0 {
			if time.Now().Unix()-file.LastModified < 7*24*60*60 {
				score += 4
			}
		}

		// Strong boost for files we know are recently changed in git (populated during EnsureLightIndex)
		if a.recentlyChanged != nil && a.recentlyChanged[base] {
			score += 10
		}

		if score > 0 {
			file.Score = score
			relevant = append(relevant, file)
		}
	}

	// Apply additional git history boost when possible (uses GetGitInfo)
	if len(relevant) > 0 {
		// Use first project path as reference
		if len(a.config.ProjectPaths) > 0 {
			a.boostFromGit(relevant, a.config.ProjectPaths[0])
		}
	}

	// Sort by relevance score (highest first)
	sort.Slice(relevant, func(i, j int) bool {
		return relevant[i].Score > relevant[j].Score
	})

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

// GetGitInfo returns git repository information for a project
func (a *ProjectAnalyzer) GetGitInfo(rootPath string) (*GitInfo, error) {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Check if it's a git repo
	gitDir := filepath.Join(absPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil, nil // Not a git repo, not an error
	}

	info := &GitInfo{
		ChangedFiles: []ChangedFile{},
	}

	// Get current branch
	branch, err := a.runGitCommand(absPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		info.Branch = strings.TrimSpace(branch)
	}

	// Get last commit hash
	commit, err := a.runGitCommand(absPath, "rev-parse", "HEAD")
	if err == nil {
		info.LastCommit = strings.TrimSpace(commit)[:12]
	}

	// Get commit count
	count, err := a.runGitCommand(absPath, "rev-list", "--count", "HEAD")
	if err == nil {
		fmt.Sscanf(strings.TrimSpace(count), "%d", &info.CommitCount)
	}

	// Get changed files from last commit
	output, err := a.runGitCommand(absPath, "diff", "--name-status", "HEAD~1", "HEAD")
	if err == nil {
		info.ChangedFiles = a.parseChangedFiles(output)
	}

	return info, nil
}

// GetRecentlyChangedFiles returns files changed in the last N commits.
// Also feeds the internal recentlyChanged map used for cold-start boosting.
func (a *ProjectAnalyzer) GetRecentlyChangedFiles(rootPath string, commitCount int) ([]ChangedFile, error) {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	gitDir := filepath.Join(absPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return nil, nil
	}

	// Get files changed in last N commits
	commitRange := fmt.Sprintf("HEAD~%d..HEAD", commitCount)
	output, err := a.runGitCommand(absPath, "diff", "--name-status", commitRange)
	if err != nil {
		return nil, nil
	}

	files := a.parseChangedFiles(output)

	// Feed the recentlyChanged map (used by cold-start ranking)
	if a.recentlyChanged == nil {
		a.recentlyChanged = make(map[string]bool)
	}
	for _, f := range files {
		a.recentlyChanged[filepath.Base(f.Path)] = true
	}

	return files, nil
}

// ResetLightIndex clears the light index state (mainly useful for tests)
func (a *ProjectAnalyzer) ResetLightIndex() {
	a.lightIndexed = false
	a.lightIndexedAt = time.Time{}
	a.recentlyChanged = nil
}

// IsLightIndexed returns whether a light index has been performed.
func (a *ProjectAnalyzer) IsLightIndexed() bool {
	return a.lightIndexed
}

// LightIndexStats returns basic information about the current light index state.
func (a *ProjectAnalyzer) LightIndexStats() map[string]interface{} {
	return map[string]interface{}{
		"indexed":        a.lightIndexed,
		"indexedAt":      a.lightIndexedAt,
		"cacheSize":      len(a.cache),
		"recentlyChanged": len(a.recentlyChanged),
	}
}

func (a *ProjectAnalyzer) runGitCommand(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func (a *ProjectAnalyzer) parseChangedFiles(output string) []ChangedFile {
	var files []ChangedFile
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		status := normalizeGitStatus(parts[0])
		file := ChangedFile{Status: status}

		if strings.HasPrefix(parts[0], "R") && len(parts) >= 3 {
			file.OldPath = parts[1]
			file.Path = parts[2]
		} else {
			file.Path = parts[1]
		}

		files = append(files, file)
	}
	return files
}

func normalizeGitStatus(status string) string {
	if status == "" {
		return "modified"
	}

	switch status[0] {
	case 'A':
		return "added"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	default:
		return "modified"
	}
}

// boostFromGit boosts file scores based on git change history
func (a *ProjectAnalyzer) boostFromGit(relevant []*FileInfo, rootPath string) {
	gitInfo, err := a.GetGitInfo(rootPath)
	if err != nil || gitInfo == nil {
		return
	}

	// Create a map of recently changed files
	changedMap := make(map[string]bool)
	for _, cf := range gitInfo.ChangedFiles {
		changedMap[filepath.Base(cf.Path)] = true
	}

	// Boost files that were recently changed
	for _, file := range relevant {
		if changedMap[filepath.Base(file.Path)] {
			file.Score += 5 // Extra boost for recently changed files
		}
	}
}
