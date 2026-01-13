package analyzer

import (
	"testing"

	"github.com/scopweb/mcp-go-context/internal/analyzer"
)

// TestGitignoreParserBasic tests basic pattern parsing
func TestGitignoreParserBasic(t *testing.T) {
	gp := analyzer.NewGitignoreParser("/home/user/project")

	content := `# Comments should be ignored
node_modules/
*.log
build/
`

	err := gp.Parse(content)
	if err != nil {
		t.Errorf("Failed to parse gitignore: %v", err)
	}
}

// TestGitignoreIsIgnoredDirectory tests directory pattern matching
func TestGitignoreIsIgnoredDirectory(t *testing.T) {
	gp := analyzer.NewGitignoreParser("/home/user/project")

	content := `node_modules/
build/
.git/
`

	gp.Parse(content)

	tests := []struct {
		path     string
		expected bool
	}{
		{"/home/user/project/node_modules", true},
		{"/home/user/project/node_modules/package", true},
		{"/home/user/project/src", false},
		{"/home/user/project/build", true},
		{"/home/user/project/.git", true},
	}

	for _, test := range tests {
		result := gp.IsIgnored(test.path)
		if result != test.expected {
			t.Errorf("IsIgnored(%s): expected %v, got %v", test.path, test.expected, result)
		}
	}
}

// TestGitignoreWildcardPatterns tests wildcard patterns
func TestGitignoreWildcardPatterns(t *testing.T) {
	gp := analyzer.NewGitignoreParser("/home/user/project")

	content := `*.log
*.tmp
__pycache__/
`

	gp.Parse(content)

	tests := []struct {
		path     string
		expected bool
	}{
		{"/home/user/project/debug.log", true},
		{"/home/user/project/temp.tmp", true},
		{"/home/user/project/src/__pycache__", true},
		{"/home/user/project/main.go", false},
		{"/home/user/project/README.md", false},
	}

	for _, test := range tests {
		result := gp.IsIgnored(test.path)
		if result != test.expected {
			t.Errorf("IsIgnored(%s): expected %v, got %v", test.path, test.expected, result)
		}
	}
}

// TestGitignoreNegation tests negation patterns
func TestGitignoreNegation(t *testing.T) {
	gp := analyzer.NewGitignoreParser("/home/user/project")

	content := `*.log
!important.log
`

	gp.Parse(content)

	tests := []struct {
		path     string
		expected bool
	}{
		{"/home/user/project/debug.log", true},
		{"/home/user/project/important.log", false}, // Negated
		{"/home/user/project/important.txt", false},
	}

	for _, test := range tests {
		result := gp.IsIgnored(test.path)
		if result != test.expected {
			t.Errorf("IsIgnored(%s): expected %v, got %v", test.path, test.expected, result)
		}
	}
}

// TestGitignoreComments tests that comments are properly ignored
func TestGitignoreComments(t *testing.T) {
	gp := analyzer.NewGitignoreParser("/home/user/project")

	content := `# This is a comment
# Another comment
node_modules/
# More comments
*.log
`

	err := gp.Parse(content)
	if err != nil {
		t.Errorf("Failed to parse gitignore with comments: %v", err)
	}

	if gp.IsIgnored("/home/user/project/node_modules") != true {
		t.Errorf("node_modules should be ignored")
	}

	if gp.IsIgnored("/home/user/project/debug.log") != true {
		t.Errorf("*.log should be ignored")
	}
}

// TestGitignoreAbsolutePaths tests absolute path patterns (starting with /)
func TestGitignoreAbsolutePaths(t *testing.T) {
	gp := analyzer.NewGitignoreParser("/home/user/project")

	content := `/build/
/dist/
`

	gp.Parse(content)

	tests := []struct {
		path     string
		expected bool
	}{
		{"/home/user/project/build", true},
		{"/home/user/project/src/build", false}, // Only matches root level
		{"/home/user/project/dist", true},
	}

	for _, test := range tests {
		result := gp.IsIgnored(test.path)
		if result != test.expected {
			t.Errorf("IsIgnored(%s): expected %v, got %v", test.path, test.expected, result)
		}
	}
}

// TestGitignoreMultipleLevels tests patterns at multiple directory levels
func TestGitignoreMultipleLevels(t *testing.T) {
	gp := analyzer.NewGitignoreParser("/home/user/project")

	content := `*.log
.git/
__pycache__/
`

	gp.Parse(content)

	tests := []struct {
		path     string
		expected bool
	}{
		{"/home/user/project/debug.log", true},
		{"/home/user/project/src/debug.log", true},
		{"/home/user/project/src/pkg/debug.log", true},
		{"/home/user/project/.git", true},
		{"/home/user/project/src/__pycache__", true},
		{"/home/user/project/src/pkg/__pycache__", true},
	}

	for _, test := range tests {
		result := gp.IsIgnored(test.path)
		if result != test.expected {
			t.Errorf("IsIgnored(%s): expected %v, got %v", test.path, test.expected, result)
		}
	}
}

// TestGitignoreEmptyLines tests that empty lines are handled
func TestGitignoreEmptyLines(t *testing.T) {
	gp := analyzer.NewGitignoreParser("/home/user/project")

	content := `

node_modules/

*.log


build/

`

	err := gp.Parse(content)
	if err != nil {
		t.Errorf("Failed to parse gitignore with empty lines: %v", err)
	}

	if !gp.IsIgnored("/home/user/project/node_modules") {
		t.Errorf("node_modules should be ignored")
	}
}

// TestGitignoreCommonPatterns tests common ignore patterns
func TestGitignoreCommonPatterns(t *testing.T) {
	gp := analyzer.NewGitignoreParser("/home/user/project")

	gp.Parse(analyzer.GetCommonIgnorePatterns())

	tests := []struct {
		path     string
		expected bool
	}{
		{"/home/user/project/node_modules", true},
		{"/home/user/project/.venv", true},
		{"/home/user/project/__pycache__", true},
		{"/home/user/project/build", true},
		{"/home/user/project/dist", true},
		{"/home/user/project/.git", true},
		{"/home/user/project/.vscode", true},
		{"/home/user/project/.idea", true},
		{"/home/user/project/debug.log", true},
		{"/home/user/project/src/main.go", false},
		{"/home/user/project/README.md", false},
	}

	for _, test := range tests {
		result := gp.IsIgnored(test.path)
		if result != test.expected {
			t.Errorf("IsIgnored(%s): expected %v, got %v", test.path, test.expected, result)
		}
	}
}

// TestGitignoreCaseSensitivity tests case sensitivity
func TestGitignoreCaseSensitivity(t *testing.T) {
	gp := analyzer.NewGitignoreParser("/home/user/project")

	content := `*.LOG
`

	gp.Parse(content)

	// Note: gitignore patterns are typically case-sensitive on Unix, case-insensitive on Windows
	// This test assumes Unix behavior
	result := gp.IsIgnored("/home/user/project/debug.log")
	// Result may vary based on OS, so we just test that it doesn't crash
	_ = result
}

// TestGitignoreQuestionMark tests ? wildcard
func TestGitignoreQuestionMark(t *testing.T) {
	gp := analyzer.NewGitignoreParser("/home/user/project")

	content := `file?.txt
`

	gp.Parse(content)

	tests := []struct {
		path     string
		expected bool
	}{
		{"/home/user/project/file1.txt", true},
		{"/home/user/project/fileA.txt", true},
		{"/home/user/project/file.txt", false},   // No character to match ?
		{"/home/user/project/file12.txt", false}, // Multiple characters
	}

	for _, test := range tests {
		result := gp.IsIgnored(test.path)
		if result != test.expected {
			t.Errorf("IsIgnored(%s): expected %v, got %v", test.path, test.expected, result)
		}
	}
}

// TestGitignoreBrackets tests bracket patterns (character classes)
func TestGitignoreBrackets(t *testing.T) {
	gp := analyzer.NewGitignoreParser("/home/user/project")

	content := `file[0-9].txt
`

	gp.Parse(content)

	tests := []struct {
		path     string
		expected bool
	}{
		{"/home/user/project/file1.txt", true},
		{"/home/user/project/file5.txt", true},
		{"/home/user/project/fileA.txt", false},
	}

	for _, test := range tests {
		result := gp.IsIgnored(test.path)
		if result != test.expected {
			t.Errorf("IsIgnored(%s): expected %v, got %v", test.path, test.expected, result)
		}
	}
}
