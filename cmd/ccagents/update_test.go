package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const unknownVersion = "unknown"

// Tests for exported functions only - internal functions are implementation details

func TestIsUpdateNeeded(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		latest   string
		expected bool
	}{
		{
			name:     "same version",
			current:  "v1.0.0",
			latest:   "v1.0.0",
			expected: false,
		},
		{
			name:     "newer version available",
			current:  "v1.0.0",
			latest:   "v1.1.0",
			expected: true,
		},
		{
			name:     "current is different from latest",
			current:  "v1.1.0",
			latest:   "v1.0.0",
			expected: true, // Simple string comparison, considers any difference as needing update
		},
		{
			name:     "unknown current version",
			current:  unknownVersion,
			latest:   "v1.0.0",
			expected: false, // unknown versions are excluded
		},
		{
			name:     "dev current version",
			current:  "dev",
			latest:   "v1.0.0",
			expected: false, // dev versions are excluded
		},
		{
			name:     "versions without v prefix",
			current:  "1.0.0",
			latest:   "1.1.0",
			expected: true,
		},
		{
			name:     "mixed prefix versions",
			current:  "v1.0.0",
			latest:   "1.1.0",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsUpdateNeeded(tt.current, tt.latest)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCanWriteToPath(t *testing.T) {
	// Test with a writable temporary directory
	tempDir := t.TempDir()
	// canWriteToPath takes a file path, not directory, so create a test file path
	testFilePath := filepath.Join(tempDir, "test")
	assert.True(t, CanWriteToPath(testFilePath))

	// Test with current directory (should be writable in test environment)
	wd, err := os.Getwd()
	require.NoError(t, err)
	testFilePath = filepath.Join(wd, "test")
	assert.True(t, CanWriteToPath(testFilePath))

	// Test with non-existent directory
	assert.False(t, CanWriteToPath("/non/existent/directory/file"))

	// Test with empty path (will default to current dir)
	assert.True(t, CanWriteToPath(""))
}

func TestFormatReleaseDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid RFC3339 date",
			input:    "2023-12-25T10:30:00Z",
			expected: "December 25, 2023",
		},
		{
			name:     "valid RFC3339 with timezone",
			input:    "2023-01-01T00:00:00-05:00",
			expected: "January 1, 2023",
		},
		{
			name:     "invalid date format",
			input:    "not-a-date",
			expected: "not-a-date",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "partial date",
			input:    "2023-12-25",
			expected: "2023-12-25",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatReleaseDate(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a source file
	srcPath := filepath.Join(tempDir, "source.txt")
	srcContent := "test content for copy"
	err := os.WriteFile(srcPath, []byte(srcContent), 0644)
	require.NoError(t, err)

	// Test successful copy
	dstPath := filepath.Join(tempDir, "destination.txt")
	err = CopyFile(srcPath, dstPath)
	assert.NoError(t, err)

	// Verify content was copied
	dstContent, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, srcContent, string(dstContent))

	// Verify permissions were copied
	srcInfo, err := os.Stat(srcPath)
	require.NoError(t, err)
	dstInfo, err := os.Stat(dstPath)
	require.NoError(t, err)
	assert.Equal(t, srcInfo.Mode(), dstInfo.Mode())

	// Test copy to existing file (should overwrite)
	newContent := "updated content"
	err = os.WriteFile(srcPath, []byte(newContent), 0644)
	require.NoError(t, err)

	err = CopyFile(srcPath, dstPath)
	assert.NoError(t, err)

	dstContent, err = os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, newContent, string(dstContent))

	// Test error cases
	err = CopyFile("/non/existent/file", dstPath)
	assert.Error(t, err)

	err = CopyFile(srcPath, "/invalid/destination/path")
	assert.Error(t, err)
}

func TestExtractTarGz(t *testing.T) {
	// This function currently returns an error suggesting manual installation
	err := extractTarGz("dummy.tar.gz", "/tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "automatic extraction not implemented")
}

func TestExtractZip(t *testing.T) {
	// This function currently returns an error suggesting manual installation
	err := extractZip("dummy.zip", "/tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "automatic extraction not implemented")
}

func TestFetchRelease(t *testing.T) {
	// Create a mock server for testing
	mockRelease := `{
		"tag_name": "v1.0.0",
		"name": "Test Release",
		"body": "Test release body",
		"prerelease": false,
		"published_at": "2023-12-25T10:30:00Z",
		"assets": [
			{
				"name": "ccagents-linux-amd64",
				"browser_download_url": "https://example.com/download",
				"size": 12345
			}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/vnd.github.v3+json", r.Header.Get("Accept"))
		assert.Equal(t, "ccAgents-updater", r.Header.Get("User-Agent"))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(mockRelease))
		assert.NoError(t, err)
	}))
	defer server.Close()

	ctx := context.Background()
	release, err := fetchRelease(ctx, server.URL)

	assert.NoError(t, err)
	assert.NotNil(t, release)
	assert.Equal(t, "v1.0.0", release.TagName)
	assert.Equal(t, "Test Release", release.Name)
	assert.Equal(t, "Test release body", release.Body)
	assert.False(t, release.Prerelease)
	assert.Equal(t, "2023-12-25T10:30:00Z", release.PublishedAt)
	assert.Len(t, release.Assets, 1)
	assert.Equal(t, "ccagents-linux-amd64", release.Assets[0].Name)
}

func TestFetchReleaseError(t *testing.T) {
	ctx := context.Background()

	// Test with invalid URL
	_, err := fetchRelease(ctx, "invalid-url")
	assert.Error(t, err)

	// Test with server error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err = fetchRelease(ctx, server.URL)
	assert.Error(t, err)

	// Test with invalid JSON
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, writeErr := w.Write([]byte("invalid json"))
		assert.NoError(t, writeErr)
	}))
	defer server.Close()

	_, err = fetchRelease(ctx, server.URL)
	assert.Error(t, err)
}

func TestFetchReleaseWithTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := fetchRelease(ctx, server.URL)
	assert.Error(t, err)
	// Check for either timeout error or context canceled
	assert.True(t,
		strings.Contains(err.Error(), "context deadline exceeded") ||
			strings.Contains(err.Error(), "context canceled") ||
			strings.Contains(err.Error(), "timeout"),
		"Expected timeout-related error, got: %v", err)
}
