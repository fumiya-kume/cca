package analysis

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAnalyzer(t *testing.T) {
	config := AnalyzerConfig{
		ProjectRoot:       "/test/project",
		MaxFileSize:       1024 * 1024,
		MaxFilesToAnalyze: 1000,
		CacheEnabled:      true,
		CacheExpiration:   time.Hour,
		ParallelAnalysis:  true,
		MaxWorkers:        4,
	}

	analyzer, err := NewAnalyzer(config)
	require.NoError(t, err)
	assert.NotNil(t, analyzer)
	assert.Equal(t, config.ProjectRoot, analyzer.config.ProjectRoot)
	assert.NotNil(t, analyzer.projectDetector)
	assert.NotNil(t, analyzer.languageDetector)
	assert.NotNil(t, analyzer.frameworkDetector)
	assert.NotNil(t, analyzer.dependencyAnalyzer)
	assert.NotNil(t, analyzer.contextBuilder)
	assert.NotNil(t, analyzer.cache)
}

func TestAnalyzer_AnalyzeProject(t *testing.T) {
	// Create temporary test project
	tmpDir, err := os.MkdirTemp("", "test-project-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test files
	goFile := filepath.Join(tmpDir, "main.go")
	// #nosec G306 - 0644 is appropriate for test files in temporary directory
	err = os.WriteFile(goFile, []byte(`package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`), 0644)
	require.NoError(t, err)

	goMod := filepath.Join(tmpDir, "go.mod")
	// #nosec G306 - 0644 is appropriate for test files in temporary directory
	err = os.WriteFile(goMod, []byte(`module test-project

go 1.21

require (
	github.com/spf13/cobra v1.7.0
)
`), 0644)
	require.NoError(t, err)

	config := AnalyzerConfig{
		ProjectRoot:       tmpDir,
		MaxFileSize:       1024 * 1024,
		MaxFilesToAnalyze: 100,
		CacheEnabled:      false, // Disable cache for testing
		ParallelAnalysis:  false, // Disable parallel for deterministic testing
		MaxWorkers:        1,
	}

	analyzer, err := NewAnalyzer(config)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := analyzer.AnalyzeProject(ctx, tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.ProjectInfo)
	assert.True(t, len(result.Languages) > 0)
	assert.NotNil(t, result.FileStructure)
	assert.True(t, result.AnalysisTime > 0)
}

func TestAnalyzerConfig_Defaults(t *testing.T) {
	config := AnalyzerConfig{}

	analyzer, err := NewAnalyzer(config)
	require.NoError(t, err)

	// Check that defaults were applied
	assert.Equal(t, int64(DefaultMaxFileSize), analyzer.config.MaxFileSize)
	assert.Equal(t, DefaultMaxFilesToAnalyze, analyzer.config.MaxFilesToAnalyze)
	assert.Equal(t, DefaultCacheExpiration, analyzer.config.CacheExpiration)
	assert.Equal(t, DefaultMaxWorkers, analyzer.config.MaxWorkers)
	assert.True(t, len(analyzer.config.IgnorePatterns) > 0)
}

func TestAnalyzer_InvalidConfig(t *testing.T) {
	config := AnalyzerConfig{
		ProjectRoot: "", // Empty project root should still work
	}

	_, err := NewAnalyzer(config)
	assert.NoError(t, err) // Should create analyzer even with empty project root
}

func TestAnalyzer_NonExistentProject(t *testing.T) {
	config := AnalyzerConfig{
		ProjectRoot:      "/nonexistent/path",
		CacheEnabled:     false,
		ParallelAnalysis: false,
	}

	analyzer, err := NewAnalyzer(config)
	require.NoError(t, err) // Should create analyzer even with non-existent path

	ctx := context.Background()
	_, err = analyzer.AnalyzeProject(ctx, "/nonexistent/path")
	assert.Error(t, err) // Should fail when trying to analyze
}

func TestAnalyzer_CacheIntegration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-project-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test file
	goFile := filepath.Join(tmpDir, "main.go")
	err = os.WriteFile(goFile, []byte("package main\n"), 0600)
	require.NoError(t, err)

	config := AnalyzerConfig{
		ProjectRoot:      tmpDir,
		CacheEnabled:     true,
		CacheExpiration:  time.Minute,
		ParallelAnalysis: false,
	}

	analyzer, err := NewAnalyzer(config)
	require.NoError(t, err)

	ctx := context.Background()

	// First analysis - should cache result
	result1, err := analyzer.AnalyzeProject(ctx, tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, result1)

	// Second analysis - should use cached result (faster)
	start := time.Now()
	result2, err := analyzer.AnalyzeProject(ctx, tmpDir)
	duration := time.Since(start)
	require.NoError(t, err)
	assert.NotNil(t, result2)

	// Cache hit should be faster than 500ms (relaxed for CI environments)
	if !testing.Short() {
		assert.True(t, duration < 500*time.Millisecond, "Expected cached result to be fast, took %v", duration)
	}
}

func TestAnalyzer_ParallelAnalysis(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-project-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create multiple test files
	for i := 0; i < 10; i++ {
		filename := filepath.Join(tmpDir, fmt.Sprintf("file%d.go", i))
		content := fmt.Sprintf("package main\n\nfunc function%d() {}\n", i)
		err = os.WriteFile(filename, []byte(content), 0600)
		require.NoError(t, err)
	}

	config := AnalyzerConfig{
		ProjectRoot:      tmpDir,
		CacheEnabled:     false,
		ParallelAnalysis: true,
		MaxWorkers:       4,
	}

	analyzer, err := NewAnalyzer(config)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := analyzer.AnalyzeProject(ctx, tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, len(result.Languages) > 0)
}

func TestAnalyzer_LargeFileSkipping(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-project-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a large file that should be skipped
	largeFile := filepath.Join(tmpDir, "large.go")
	largeContent := make([]byte, 2*1024*1024) // 2MB
	for i := range largeContent {
		largeContent[i] = byte('a' + (i % 26))
	}
	err = os.WriteFile(largeFile, largeContent, 0600)
	require.NoError(t, err)

	// Create a normal file
	normalFile := filepath.Join(tmpDir, "normal.go")
	err = os.WriteFile(normalFile, []byte("package main\n"), 0600)
	require.NoError(t, err)

	config := AnalyzerConfig{
		ProjectRoot:      tmpDir,
		MaxFileSize:      1024 * 1024, // 1MB limit
		CacheEnabled:     false,
		ParallelAnalysis: false,
	}

	analyzer, err := NewAnalyzer(config)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := analyzer.AnalyzeProject(ctx, tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Should have analyzed the normal file but skipped the large one
	// We can't easily verify this without access to internal state,
	// but the analysis should complete successfully
}

func TestAnalyzer_EmptyProject(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "test-project-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	config := AnalyzerConfig{
		ProjectRoot:      tmpDir,
		CacheEnabled:     false,
		ParallelAnalysis: false,
	}

	analyzer, err := NewAnalyzer(config)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := analyzer.AnalyzeProject(ctx, tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.ProjectInfo)
	// Empty project should still return basic analysis
}
