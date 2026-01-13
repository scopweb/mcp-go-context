package analyzer

import (
	"bufio"
	"path/filepath"
	"strings"
)

// GitignorePattern represents a single gitignore pattern
type GitignorePattern struct {
	pattern    string
	isNegation bool
	isDir      bool
}

// GitignoreParser parses and evaluates gitignore patterns
type GitignoreParser struct {
	patterns []*GitignorePattern
	basePath string
}

// NewGitignoreParser creates a new gitignore parser
func NewGitignoreParser(basePath string) *GitignoreParser {
	return &GitignoreParser{
		patterns: make([]*GitignorePattern, 0),
		basePath: basePath,
	}
}

// Parse reads and compiles gitignore patterns from content
func (gp *GitignoreParser) Parse(content string) error {
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for negation
		isNegation := false
		if strings.HasPrefix(line, "!") {
			isNegation = true
			line = strings.TrimPrefix(line, "!")
			line = strings.TrimSpace(line)
		}

		// Check if pattern ends with / (directory-only)
		isDir := strings.HasSuffix(line, "/")
		if isDir {
			line = strings.TrimSuffix(line, "/")
		}

		gp.patterns = append(gp.patterns, &GitignorePattern{
			pattern:    line,
			isNegation: isNegation,
			isDir:      isDir,
		})
	}

	return scanner.Err()
}

// IsIgnored checks if a path should be ignored
func (gp *GitignoreParser) IsIgnored(path string) bool {
	// Make path relative to basePath
	relPath, err := filepath.Rel(gp.basePath, path)
	if err != nil {
		return false
	}

	// Normalize path separators
	relPath = filepath.ToSlash(relPath)

	// Start with not ignored, then apply patterns in order
	ignored := false

	for _, pattern := range gp.patterns {
		// Check if pattern matches
		if gp.patternMatches(relPath, pattern.pattern, pattern.isDir) {
			if pattern.isNegation {
				ignored = false
			} else {
				ignored = true
			}
		}
	}

	return ignored
}

// patternMatches checks if a path matches a gitignore pattern using simple wildcard matching
func (gp *GitignoreParser) patternMatches(path, pattern string, isDir bool) bool {
	// Handle absolute paths (starting with /)
	if strings.HasPrefix(pattern, "/") {
		pattern = strings.TrimPrefix(pattern, "/")
		return gp.matchSimple(path, pattern, isDir)
	}

	// Pattern can match at any level
	parts := strings.Split(path, "/")
	for i := range parts {
		subpath := strings.Join(parts[i:], "/")
		if gp.matchSimple(subpath, pattern, isDir) {
			return true
		}
	}

	return false
}

// matchSimple performs simple wildcard matching (* and ?)
func (gp *GitignoreParser) matchSimple(text, pattern string, isDir bool) bool {
	if isDir {
		// For directory patterns, match against path component
		text = strings.Split(text, "/")[0]
	}

	return gp.glob(text, pattern)
}

// glob implements a simple glob pattern matcher
func (gp *GitignoreParser) glob(text, pattern string) bool {
	return gp.globHelper(text, pattern, 0, 0)
}

// globHelper is a recursive helper for glob matching
func (gp *GitignoreParser) globHelper(text, pattern string, ti, pi int) bool {
	// Both exhausted - match
	if ti == len(text) && pi == len(pattern) {
		return true
	}

	// Pattern exhausted but text remaining - no match
	if pi == len(pattern) {
		return false
	}

	// Handle * wildcard
	if pattern[pi] == '*' {
		// * matches zero or more characters (except /)
		// Try matching zero characters
		if gp.globHelper(text, pattern, ti, pi+1) {
			return true
		}
		// Try matching one or more characters
		if ti < len(text) && text[ti] != '/' {
			return gp.globHelper(text, pattern, ti+1, pi)
		}
		return false
	}

	// Handle ? wildcard - matches single character except /
	if pattern[pi] == '?' {
		if ti >= len(text) || text[ti] == '/' {
			return false
		}
		return gp.globHelper(text, pattern, ti+1, pi+1)
	}

	// Handle character ranges [abc] or [a-z]
	if pattern[pi] == '[' {
		// Find closing bracket
		closeBracket := strings.IndexByte(pattern[pi+1:], ']')
		if closeBracket == -1 {
			// No closing bracket - treat [ as literal
			if ti >= len(text) || text[ti] != '[' {
				return false
			}
			return gp.globHelper(text, pattern, ti+1, pi+1)
		}

		if ti >= len(text) {
			return false
		}

		charSet := pattern[pi+1 : pi+1+closeBracket]
		char := text[ti]

		// Check for negation
		if strings.HasPrefix(charSet, "!") {
			charSet = charSet[1:]
			// Invert matching
			if !gp.charInSet(char, charSet) {
				return gp.globHelper(text, pattern, ti+1, pi+2+closeBracket)
			}
			return false
		}

		if gp.charInSet(char, charSet) {
			return gp.globHelper(text, pattern, ti+1, pi+2+closeBracket)
		}
		return false
	}

	// Regular character - must match exactly
	if ti >= len(text) || text[ti] != pattern[pi] {
		return false
	}

	return gp.globHelper(text, pattern, ti+1, pi+1)
}

// charInSet checks if a character is in a character set (handles ranges like a-z)
func (gp *GitignoreParser) charInSet(char byte, charSet string) bool {
	i := 0
	for i < len(charSet) {
		// Check for range (e.g., a-z)
		if i+2 < len(charSet) && charSet[i+1] == '-' {
			if char >= charSet[i] && char <= charSet[i+2] {
				return true
			}
			i += 3
		} else {
			// Single character
			if char == charSet[i] {
				return true
			}
			i++
		}
	}
	return false
}

// GetCommonIgnorePatterns returns common patterns to exclude
func GetCommonIgnorePatterns() string {
	return `# Dependencies
node_modules/
.venv/
venv/
env/
.env
*.pyc
__pycache__/
.pytest_cache/
Cargo.lock
vendor/

# Build and dist
build/
dist/
target/
out/
bin/
*.o
*.a
*.so
*.dylib
*.exe
*.dll

# IDE and editors
.vscode/
.idea/
*.swp
*.swo
*~
.DS_Store
.project
.classpath
.c9/
*.launch
.settings/
*.sublime-workspace

# Git and VCS
.git/
.gitignore
.hg/
.svn/
CVS/

# OS files
Thumbs.db
.DS_Store
.AppleDouble
.LSOverride

# Logs
*.log
logs/
npm-debug.log*

# Temporary files
.tmp/
tmp/
temp/
*.tmp
*.bak
*.swp

# Coverage
coverage/
.coverage
*.lcov
`
}
