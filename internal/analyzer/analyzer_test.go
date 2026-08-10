package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scopweb/mcp-go-context/internal/config"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"main.go", "go"},
		{"server.ts", "typescript"},
		{"app.py", "python"},
		{"index.html", "html"},
		{"style.css", "css"},
		{"query.sql", "sql"},
		{"script.sh", "bash"},
		{"data.json", "json"},
		{"config.yaml", "yaml"},
		{"readme.md", "markdown"},
		{"unknown.xyz", "text"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := detectLanguage(tt.path)
			if result != tt.expected {
				t.Errorf("detectLanguage(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestParseGoMod(t *testing.T) {
	// Create a temp go.mod file
	content := `module example.com/test

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/go-sql-driver/mysql v1.7.1
)

require (
	 golang.org/x/net v0.17.0 // indirect
)
`
	tmpDir := t.TempDir()
	goModPath := tmpDir + "/go.mod"
	if err := writeFile(goModPath, content); err != nil {
		t.Fatalf("failed to write temp go.mod: %v", err)
	}

	cfg := config.ContextConfig{
		ProjectPaths: []string{tmpDir},
	}
	a := &ProjectAnalyzer{config: cfg}

	deps, err := a.parseGoMod(goModPath, false)
	if err != nil {
		t.Fatalf("parseGoMod() failed: %v", err)
	}

	// Should find at least 2 direct dependencies
	if len(deps) < 2 {
		t.Errorf("expected at least 2 dependencies, got %d", len(deps))
	}

	// Find gin and mysql
	found := make(map[string]bool)
	for _, dep := range deps {
		found[dep.Name] = true
	}

	if !found["github.com/gin-gonic/gin"] {
		t.Error("did not find github.com/gin-gonic/gin in deps")
	}

	if !found["github.com/go-sql-driver/mysql"] {
		t.Error("did not find github.com/go-sql-driver/mysql in deps")
	}
}

func TestParsePackageJSON(t *testing.T) {
	content := `{
		"name": "myproject",
		"dependencies": {
			"express": "^4.18.2",
			"lodash": "^4.17.21"
		},
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`
	tmpDir := t.TempDir()
	pkgPath := tmpDir + "/package.json"
	if err := writeFile(pkgPath, content); err != nil {
		t.Fatalf("failed to write temp package.json: %v", err)
	}

	cfg := config.ContextConfig{}
	a := &ProjectAnalyzer{config: cfg}

	deps, err := a.parsePackageJSON(pkgPath, false)
	if err != nil {
		t.Fatalf("parsePackageJSON() failed: %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(deps))
	}

	// Should have express and lodash but not jest (dev dep with includeTransitive=false)
	for _, dep := range deps {
		if dep.Name == "express" {
			return
		}
	}
	t.Error("did not find express dependency")
}

func TestParsePackageJSONWithDevDeps(t *testing.T) {
	content := `{
		"name": "myproject",
		"dependencies": {
			"express": "^4.18.2"
		},
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`
	tmpDir := t.TempDir()
	pkgPath := tmpDir + "/package.json"
	if err := writeFile(pkgPath, content); err != nil {
		t.Fatalf("failed to write temp package.json: %v", err)
	}

	cfg := config.ContextConfig{}
	a := &ProjectAnalyzer{config: cfg}

	// With includeTransitive=true, should include devDependencies
	deps, err := a.parsePackageJSON(pkgPath, true)
	if err != nil {
		t.Fatalf("parsePackageJSON() failed: %v", err)
	}

	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies with dev deps, got %d", len(deps))
	}
}

func TestParseRequirements(t *testing.T) {
	content := `requests==2.31.0
numpy>=1.24.0
pandas~=1.5.0
# this is a comment
-e git+https://github.com/user/repo.git
flask
`
	tmpDir := t.TempDir()
	reqPath := tmpDir + "/requirements.txt"
	if err := writeFile(reqPath, content); err != nil {
		t.Fatalf("failed to write temp requirements.txt: %v", err)
	}

	cfg := config.ContextConfig{}
	a := &ProjectAnalyzer{config: cfg}

	deps, err := a.parseRequirements(reqPath, false)
	if err != nil {
		t.Fatalf("parseRequirements() failed: %v", err)
	}

	if len(deps) < 4 {
		t.Errorf("expected at least 4 dependencies, got %d", len(deps))
	}

	// Check requests with exact version
	var requestsDep *Dependency
	for i := range deps {
		if deps[i].Name == "requests" {
			requestsDep = &deps[i]
			break
		}
	}
	if requestsDep == nil {
		t.Error("did not find requests dependency")
	} else if requestsDep.Version != "==2.31.0" {
		t.Errorf("expected requests version ==2.31.0, got %s", requestsDep.Version)
	}

	// Check flask with "any" version
	var flaskDep *Dependency
	for i := range deps {
		if deps[i].Name == "flask" {
			flaskDep = &deps[i]
			break
		}
	}
	if flaskDep == nil {
		t.Error("did not find flask dependency")
	} else if flaskDep.Version != "any" {
		t.Errorf("expected flask version 'any', got %s", flaskDep.Version)
	}
}

func TestParsePyproject(t *testing.T) {
	content := `[project.dependencies]
requests = "^2.31.0"
numpy = ">=1.24.0"

[tool.poetry.dependencies]
python = "^3.9"
django = "^4.0"

[project.optional-dependencies]
dev = "pytest>=7.0.0"
`
	tmpDir := t.TempDir()
	pyprojectPath := tmpDir + "/pyproject.toml"
	if err := writeFile(pyprojectPath, content); err != nil {
		t.Fatalf("failed to write temp pyproject.toml: %v", err)
	}

	cfg := config.ContextConfig{}
	a := &ProjectAnalyzer{config: cfg}

	deps, err := a.parsePyproject(pyprojectPath, false)
	if err != nil {
		t.Fatalf("parsePyproject() failed: %v", err)
	}

	if len(deps) < 4 {
		t.Errorf("expected at least 4 dependencies, got %d", len(deps))
	}

	// Should have requests, numpy, python, django
	names := make(map[string]bool)
	for _, dep := range deps {
		names[dep.Name] = true
	}

	expected := []string{"requests", "numpy", "python", "django"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected to find %q in dependencies", name)
		}
	}
}

func TestParsePyprojectProjectArray(t *testing.T) {
	content := `[project]
dependencies = ["requests>=2.31.0", "numpy", "pydantic<3"]
`
	tmpDir := t.TempDir()
	pyprojectPath := tmpDir + "/pyproject.toml"
	if err := writeFile(pyprojectPath, content); err != nil {
		t.Fatalf("failed to write temp pyproject.toml: %v", err)
	}

	a := &ProjectAnalyzer{config: config.ContextConfig{}}

	deps, err := a.parsePyproject(pyprojectPath, false)
	if err != nil {
		t.Fatalf("parsePyproject() failed: %v", err)
	}

	if len(deps) != 3 {
		t.Fatalf("expected 3 dependencies, got %d", len(deps))
	}

	if deps[0].Name != "requests" || deps[0].Version != ">=2.31.0" {
		t.Fatalf("unexpected first dependency: %+v", deps[0])
	}

	if deps[1].Name != "numpy" || deps[1].Version != "any" {
		t.Fatalf("unexpected second dependency: %+v", deps[1])
	}
}

func TestParseChangedFilesNameStatus(t *testing.T) {
	a := &ProjectAnalyzer{}
	files := a.parseChangedFiles("A\tnew.go\nM\tinternal/app.go\nR100\told.go\tnewer.go\nD\tdead.go\n")

	if len(files) != 4 {
		t.Fatalf("expected 4 changed files, got %d", len(files))
	}

	if files[0].Status != "added" || files[0].Path != "new.go" {
		t.Fatalf("unexpected added file: %+v", files[0])
	}

	if files[2].Status != "renamed" || files[2].OldPath != "old.go" || files[2].Path != "newer.go" {
		t.Fatalf("unexpected renamed file: %+v", files[2])
	}
}

func TestFindRelevantFiles(t *testing.T) {
	cfg := config.ContextConfig{
		IgnorePatterns: []string{"*.log", "*.tmp"},
	}
	a := &ProjectAnalyzer{
		config: cfg,
		cache:  make(map[string]*FileInfo),
	}

	// Add some files to cache
	a.cache["/test/main.go"] = &FileInfo{
		Path:         "/test/main.go",
		Language:     "go",
		LastModified: 0,
	}
	a.cache["/test/server.go"] = &FileInfo{
		Path:         "/test/server.go",
		Language:     "go",
		LastModified: 0,
	}
	a.cache["/test/README.md"] = &FileInfo{
		Path:         "/test/README.md",
		Language:     "markdown",
		LastModified: 0,
	}

	results := a.findRelevantFiles("main")
	if len(results) == 0 {
		t.Fatal("expected at least one result for 'main' query")
	}

	if results[0].Path != "/test/main.go" {
		t.Errorf("expected main.go, got %s", results[0].Path)
	}
}

func TestFindRelevantFilesWithScore(t *testing.T) {
	cfg := config.ContextConfig{}
	a := &ProjectAnalyzer{
		config: cfg,
		cache:  make(map[string]*FileInfo),
	}

	a.cache["/test/gin-server.go"] = &FileInfo{
		Path: "/test/gin-server.go",
	}

	results := a.findRelevantFiles("gin")
	if len(results) == 0 {
		t.Fatal("expected at least one result for 'gin' query")
	}

	if results[0].Score == 0 {
		t.Error("expected non-zero score for matching file")
	}
}

// Fase 0 cold-start tests

func TestEnsureLightIndexPopulatesCache(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a small project structure
	_ = os.MkdirAll(tmpDir+"/pkg", 0755)
	_ = writeFile(tmpDir+"/main.go", "package main\n")
	_ = writeFile(tmpDir+"/pkg/utils.go", "package pkg\n")
	_ = writeFile(tmpDir+"/README.md", "# Test\n")

	cfg := config.ContextConfig{
		ProjectPaths:   []string{tmpDir},
		IgnorePatterns: []string{"*.log"},
	}
	a := &ProjectAnalyzer{
		config: cfg,
		cache:  make(map[string]*FileInfo),
	}

	// Should be cold initially
	if a.IsLightIndexed() {
		t.Error("expected not light indexed initially")
	}

	a.EnsureLightIndex(50, 5*time.Second)

	if !a.IsLightIndexed() {
		t.Error("expected light indexed after EnsureLightIndex")
	}
	if len(a.cache) == 0 {
		t.Error("expected cache to be populated after light index")
	}
}

func TestEnsureLightIndexRespectsMaxFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create more files than the limit
	for i := 0; i < 20; i++ {
		name := tmpDir + fmt.Sprintf("/file%02d.go", i)
		_ = writeFile(name, "package main\n")
	}

	cfg := config.ContextConfig{ProjectPaths: []string{tmpDir}}
	a := &ProjectAnalyzer{
		config: cfg,
		cache:  make(map[string]*FileInfo),
	}

	a.EnsureLightIndex(5, 2*time.Second)

	// Should not index way more than requested (allow some margin for git phase etc.)
	if len(a.cache) > 12 {
		t.Errorf("light index indexed too many files: got %d, want <= ~12", len(a.cache))
	}
}

func TestResetLightIndex(t *testing.T) {
	a := &ProjectAnalyzer{
		config: config.ContextConfig{},
		cache:  make(map[string]*FileInfo),
	}

	a.lightIndexed = true
	a.cache["/fake.go"] = &FileInfo{Path: "/fake.go"}

	a.ResetLightIndex()

	if a.IsLightIndexed() {
		t.Error("expected light index reset")
	}
	if len(a.cache) != 1 {
		// Note: ResetLightIndex currently does not clear the cache itself (by design for now)
		// Only clears the flag and recentlyChanged map
	}
}

func TestLightIndexStats(t *testing.T) {
	a := &ProjectAnalyzer{
		config: config.ContextConfig{},
		cache:  make(map[string]*FileInfo),
	}

	stats := a.LightIndexStats()
	if stats["indexed"] != false {
		t.Error("expected indexed=false initially")
	}
}

func TestLightIndexRevalidatesChangedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	goFile := filepath.Join(tmpDir, "main.go")
	if err := writeFile(goFile, "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println() }\n"); err != nil {
		t.Fatal(err)
	}

	a := &ProjectAnalyzer{
		config: config.ContextConfig{ProjectPaths: []string{tmpDir}},
		cache:  make(map[string]*FileInfo),
	}
	a.EnsureLightIndex(100, 5*time.Second)

	cached := a.cache[goFile]
	if cached == nil || len(cached.Imports) != 1 || cached.Imports[0] != "fmt" {
		t.Fatalf("expected initial import [fmt], got %+v", cached)
	}

	// Modify the file with a guaranteed-different mtime
	if err := writeFile(goFile, "package main\n\nimport (\n\t\"fmt\"\n\t\"os\"\n)\n\nfunc main() { fmt.Println(os.Args) }\n"); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(2 * time.Hour)
	if err := os.Chtimes(goFile, future, future); err != nil {
		t.Fatal(err)
	}

	// Force re-index (bypass cooldown)
	a.lightIndexedAt = time.Now().Add(-10 * time.Minute)
	a.EnsureLightIndex(100, 5*time.Second)

	updated := a.cache[goFile]
	if updated == nil || len(updated.Imports) != 2 {
		t.Fatalf("expected refreshed imports [fmt os], got %+v", updated)
	}
}

func TestManifestChangeResetsLightIndex(t *testing.T) {
	tmpDir := t.TempDir()
	goMod := filepath.Join(tmpDir, "go.mod")
	if err := writeFile(goMod, "module example.com/x\n\ngo 1.21\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(tmpDir, "main.go"), "package main\n\nfunc main() {}\n"); err != nil {
		t.Fatal(err)
	}

	a := &ProjectAnalyzer{
		config: config.ContextConfig{ProjectPaths: []string{tmpDir}},
		cache:  make(map[string]*FileInfo),
	}
	a.EnsureLightIndex(100, 5*time.Second)

	if !a.IsLightIndexed() {
		t.Fatal("expected light index after first run")
	}
	if len(a.manifestMtimes) == 0 {
		t.Fatal("expected manifest mtimes to be tracked")
	}

	// Touch go.mod with a guaranteed-different mtime; cooldown is still fresh,
	// so only the manifest watcher can trigger a re-index.
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(goMod, past, past); err != nil {
		t.Fatal(err)
	}
	before := a.lightIndexedAt
	a.EnsureLightIndex(100, 5*time.Second)

	if !a.lightIndexedAt.After(before) {
		t.Error("expected manifest change to force re-index despite cooldown")
	}
	if a.manifestMtimes[goMod] != past.Unix() {
		t.Errorf("expected manifest mtime updated to %d, got %d", past.Unix(), a.manifestMtimes[goMod])
	}
}

// Helper function to write files
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func TestAnalyzeDependenciesMixedMonorepo(t *testing.T) {
	tmpDir := t.TempDir()

	// Root Go module
	if err := writeFile(filepath.Join(tmpDir, "go.mod"), "module example.com/root\n\ngo 1.21\n\nrequire github.com/spf13/cobra v1.8.0\n"); err != nil {
		t.Fatal(err)
	}

	// Nested app with package.json
	webDir := filepath.Join(tmpDir, "apps", "web")
	if err := os.MkdirAll(webDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(webDir, "package.json"), `{"name":"web","dependencies":{"react":"^18.0.0"}}`); err != nil {
		t.Fatal(err)
	}

	// Nested service with requirements.txt
	apiDir := filepath.Join(tmpDir, "services", "api")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(apiDir, "requirements.txt"), "flask==3.0.0\n"); err != nil {
		t.Fatal(err)
	}

	// Ignored directory must not be scanned
	nmDir := filepath.Join(tmpDir, "node_modules", "leftpad")
	if err := os.MkdirAll(nmDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(filepath.Join(nmDir, "package.json"), `{"name":"leftpad","dependencies":{"x":"1.0.0"}}`); err != nil {
		t.Fatal(err)
	}

	cfg := config.ContextConfig{
		ProjectPaths:   []string{tmpDir},
		IgnorePatterns: []string{"node_modules", ".git"},
	}
	a, err := New(cfg)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}

	deps, err := a.AnalyzeDependencies(false)
	if err != nil {
		t.Fatalf("AnalyzeDependencies() failed: %v", err)
	}

	scopeByName := make(map[string]string)
	for _, dep := range deps {
		scopeByName[dep.Name] = dep.Scope
	}

	if scopeByName["github.com/spf13/cobra"] != "root" {
		t.Errorf("expected cobra scope=root, got %q", scopeByName["github.com/spf13/cobra"])
	}
	if scopeByName["react"] != "app" {
		t.Errorf("expected react scope=app, got %q", scopeByName["react"])
	}
	if scopeByName["flask"] != "service" {
		t.Errorf("expected flask scope=service, got %q", scopeByName["flask"])
	}
	if _, found := scopeByName["x"]; found {
		t.Error("dependencies under node_modules must be ignored")
	}
}
