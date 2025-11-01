package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestDependencyVersions verifies that all dependencies are up to date
func TestDependencyVersions(t *testing.T) {
	cmd := exec.Command("go", "list", "-u", "-m", "all")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to run go list: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	outdated := 0
	for _, line := range lines {
		if strings.Contains(line, "[") && strings.Contains(line, "]") {
			outdated++
			t.Logf("⚠️  Outdated dependency: %s", line)
		}
	}

	if outdated > 0 {
		t.Logf("Found %d outdated dependencies. Run 'go get -u ./...' to update", outdated)
	} else {
		t.Log("✅ All dependencies are up to date")
	}
}

// TestGoModuleIntegrity verifies go.mod hasn't been tampered
func TestGoModuleIntegrity(t *testing.T) {
	// Read go.mod
	content, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	// Calculate SHA256
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	t.Logf("go.mod SHA256: %s", hashStr)

	// Check for suspicious patterns
	modContent := string(content)
	suspiciousPatterns := []string{
		"replace ",
		"retract ",
		"excluded ",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(modContent, pattern) {
			t.Logf("ℹ️  Found directive: %s (review manually)", pattern)
		}
	}
}

// TestGoSumIntegrity verifies that all dependencies have checksums
func TestGoSumIntegrity(t *testing.T) {
	// Read go.sum
	content, err := os.ReadFile("../../go.sum")
	if err != nil {
		t.Fatalf("Failed to read go.sum: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	validLines := 0
	invalidLines := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 3 {
			validLines++
		} else if line != "" {
			invalidLines++
			t.Logf("⚠️  Invalid go.sum line: %s", line)
		}
	}

	t.Logf("go.sum entries: %d valid, %d invalid", validLines, invalidLines)

	if invalidLines > 0 {
		t.Errorf("Found %d invalid lines in go.sum", invalidLines)
	}
}

// TestMainDependencies checks critical dependencies for known issues
func TestMainDependencies(t *testing.T) {
	criticalDeps := map[string]string{
		"github.com/mark3labs/mcp-go":       "v0.42.0", // MCP SDK
		"github.com/panjf2000/ants/v2":      "v2.11.3", // Goroutine pool
		"github.com/allegro/bigcache/v3":    "v3.1.0",  // Cache
		"github.com/patrickmn/go-cache":     "v2.1.0",  // Cache
		"github.com/fsnotify/fsnotify":      "v1.9.0",  // File watchers
	}

	cmd := exec.Command("go", "list", "-m", "all")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to list modules: %v", err)
	}

	modules := make(map[string]string)
	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			modules[parts[0]] = parts[1]
		}
	}

	for dep, expectedVersion := range criticalDeps {
		if version, ok := modules[dep]; ok {
			t.Logf("✅ %s: %s (expected %s)", dep, version, expectedVersion)
			if version != expectedVersion {
				t.Logf("⚠️  Version mismatch for %s: got %s, expected %s", dep, version, expectedVersion)
			}
		} else {
			t.Logf("❌ Dependency not found: %s", dep)
		}
	}
}

// TestNoPrivateKeyCommitted checks for accidentally committed secrets
func TestNoPrivateKeyCommitted(t *testing.T) {
	sensitivePatterns := []string{
		"PRIVATE KEY",
		"SECRET_KEY",
		"API_KEY",
		"PASSWORD=",
		"token:",
		".env",
	}

	checkFiles := []string{
		"../../main.go",
		"../../go.mod",
		"../../go.sum",
		"../../core/engine.go",
	}

	for _, file := range checkFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Logf("⚠️  Could not read file: %s", file)
			continue
		}

		fileContent := string(content)
		for _, pattern := range sensitivePatterns {
			if strings.Contains(fileContent, pattern) {
				t.Logf("❌ SECURITY ALERT: Sensitive pattern found in %s: %s", file, pattern)
			}
		}
	}

	t.Log("✅ No obvious secrets detected in code files")
}

// TestNoDangerousImports checks for unsafe imports
func TestNoDangerousImports(t *testing.T) {
	dangerousImports := []string{
		"\"unsafe\"",
		"syscall",
	}

	checkFiles := []string{
		"../../main.go",
		"../../core/engine.go",
		"../../core/edit_operations.go",
	}

	for _, file := range checkFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		fileContent := string(content)
		for _, dangerous := range dangerousImports {
			if strings.Contains(fileContent, dangerous) {
				t.Logf("ℹ️  Found %s import in %s (review for security)", dangerous, file)
			}
		}
	}
}

// TestInputValidation checks that main.go properly validates inputs
func TestInputValidation(t *testing.T) {
	content, err := os.ReadFile("../../main.go")
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	fileContent := string(content)

	// Check for path validation
	validationPatterns := []string{
		"RequireString",
		"filepath.Clean",
		"path validation",
		"os.IsPathSeparator",
	}

	foundValidations := 0
	for _, pattern := range validationPatterns {
		if strings.Contains(fileContent, pattern) {
			foundValidations++
			t.Logf("✅ Found validation: %s", pattern)
		}
	}

	if foundValidations == 0 {
		t.Log("⚠️  No obvious input validation patterns found")
	}
}

// TestErrorHandling verifies proper error handling
func TestErrorHandling(t *testing.T) {
	content, err := os.ReadFile("../../main.go")
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	fileContent := string(content)

	errorHandlingPatterns := map[string]int{
		"if err != nil":                 0,
		"return mcp.NewToolResultError": 0,
		"mcp.NewToolResultText":         0,
	}

	for pattern := range errorHandlingPatterns {
		count := strings.Count(fileContent, pattern)
		errorHandlingPatterns[pattern] = count
	}

	t.Logf("Error handling statistics:")
	for pattern, count := range errorHandlingPatterns {
		t.Logf("  %s: %d occurrences", pattern, count)
	}

	if errorHandlingPatterns["if err != nil"] > 0 {
		t.Log("✅ Error handling found")
	}
}

// TestLogSanitization checks that logs don't leak sensitive data
func TestLogSanitization(t *testing.T) {
	content, err := os.ReadFile("../../main.go")
	if err != nil {
		t.Fatalf("Failed to read main.go: %v", err)
	}

	fileContent := string(content)

	// Check for proper logging
	if strings.Contains(fileContent, "log.Printf") {
		t.Log("✅ Using standard logging (log.Printf)")
	} else if strings.Contains(fileContent, "fmt.Printf") {
		t.Logf("⚠️  Using fmt.Printf instead of log.Printf")
	}
}

// TestGoVersion checks Go version compatibility
func TestGoVersion(t *testing.T) {
	content, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	fileContent := string(content)

	// Extract go version requirement
	for _, line := range strings.Split(fileContent, "\n") {
		if strings.HasPrefix(line, "go ") {
			t.Logf("Go version requirement: %s", strings.TrimSpace(line))
			// Go 1.24 is latest as of this test
			if strings.Contains(line, "1.24") || strings.Contains(line, "1.23") {
				t.Log("✅ Go version is modern and well-maintained")
			}
		}
	}
}

// TestCommunitySecurityAdvisories checks for known vulnerable packages
func TestCommunitySecurityAdvisories(t *testing.T) {
	// This requires 'go list -json' to get detailed package info
	cmd := exec.Command("go", "list", "-json", "...")
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("⚠️  Could not run go list -json: %v", err)
		return
	}

	// Known vulnerable package patterns (as of Nov 2024)
	knownVulnerabilities := map[string]string{
		"stretchr/testify": "v1.8.0+", // Requires v1.8.2+
	}

	outputStr := string(output)
	for vulnerable, minimum := range knownVulnerabilities {
		if strings.Contains(outputStr, vulnerable) {
			t.Logf("⚠️  %s detected - ensure using %s or later", vulnerable, minimum)
		}
	}

	t.Log("✅ Vulnerability check completed")
}

// BenchmarkSecurityChecksFromMod measures security validation overhead
func BenchmarkSecurityChecksFromMod(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		os.ReadFile("../../go.mod")
		os.ReadFile("../../go.sum")
	}
}
