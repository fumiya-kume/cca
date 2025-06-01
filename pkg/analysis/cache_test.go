package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants
const (
	testProjectPath = "/test/project"
)

func TestNewAnalysisCache(t *testing.T) {
	expiration := time.Hour
	cache := NewAnalysisCache(expiration)

	assert.NotNil(t, cache)
	assert.Equal(t, expiration, cache.expiration)
	assert.NotEmpty(t, cache.cacheDir)
	assert.NotNil(t, cache.memoryCache)
	assert.Equal(t, 50, cache.maxMemoryEntries)

	// Verify cache directory exists
	_, err := os.Stat(cache.cacheDir)
	assert.NoError(t, err)

	// Cleanup
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()
}

func TestAnalysisCache_SetAndGet(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	projectRoot := testProjectPath
	result := &AnalysisResult{
		ProjectInfo: &ProjectInfo{
			Name:        "test-project",
			ConfigFiles: []string{"go.mod", "go.sum"},
			EntryPoints: []string{"main.go"},
		},
		FileStructure: &FileStructure{
			TotalFiles: 10,
		},
		Dependencies: &DependencyInfo{
			Direct: []Dependency{
				{Name: "github.com/stretchr/testify", Version: "v1.8.0"},
			},
		},
	}

	// Test Set
	cache.Set(projectRoot, result)

	// Test Get
	retrieved := cache.Get(projectRoot)
	require.NotNil(t, retrieved)
	assert.Equal(t, result.ProjectInfo.Name, retrieved.ProjectInfo.Name)
	assert.Equal(t, result.FileStructure.TotalFiles, retrieved.FileStructure.TotalFiles)
}

func TestAnalysisCache_GetNonExistent(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	projectRoot := "/nonexistent/project"
	result := cache.Get(projectRoot)
	assert.Nil(t, result)
}

func TestAnalysisCache_Clear(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	projectRoot := testProjectPath
	result := &AnalysisResult{
		ProjectInfo: &ProjectInfo{Name: "test-project"},
	}

	// Add entry
	cache.Set(projectRoot, result)
	assert.NotNil(t, cache.Get(projectRoot))

	// Clear cache
	cache.Clear()
	assert.Nil(t, cache.Get(projectRoot))
	assert.Equal(t, 0, len(cache.memoryCache))
}

func TestAnalysisCache_Invalidate(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	projectRoot := testProjectPath
	result := &AnalysisResult{
		ProjectInfo: &ProjectInfo{Name: "test-project"},
	}

	// Add entry
	cache.Set(projectRoot, result)
	assert.NotNil(t, cache.Get(projectRoot))

	// Invalidate specific entry
	cache.Invalidate(projectRoot)
	assert.Nil(t, cache.Get(projectRoot))
}

func TestAnalysisCache_GetCacheStats(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	// Initially empty
	stats := cache.GetCacheStats()
	assert.Equal(t, 0, stats.MemoryEntries)
	assert.Equal(t, 0, stats.DiskEntries)

	// Add some entries
	result := &AnalysisResult{
		ProjectInfo: &ProjectInfo{Name: "test-project"},
	}
	cache.Set("/project1", result)
	cache.Set("/project2", result)

	stats = cache.GetCacheStats()
	assert.Equal(t, 2, stats.MemoryEntries)
	assert.True(t, stats.TotalSize > 0)
}

func TestAnalysisCache_CleanupExpired(t *testing.T) {
	// Use very short expiration for testing
	cache := NewAnalysisCache(time.Millisecond * 10)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	projectRoot := testProjectPath
	result := &AnalysisResult{
		ProjectInfo: &ProjectInfo{Name: "test-project"},
	}

	// Add entry
	cache.Set(projectRoot, result)
	assert.NotNil(t, cache.Get(projectRoot))

	// Wait for expiration with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Poll for expiration without sleep
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if cache.Get(projectRoot) == nil {
				goto expired
			}
		case <-ctx.Done():
			goto expired
		}
	}

expired:

	// Cleanup expired entries
	cache.CleanupExpired()

	// Entry should be gone
	assert.Nil(t, cache.Get(projectRoot))
}

func TestAnalysisCache_generateKey(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	projectRoot1 := "/path/to/project1"
	projectRoot2 := "/path/to/project2"

	key1 := cache.generateKey(projectRoot1)
	key2 := cache.generateKey(projectRoot2)

	assert.NotEmpty(t, key1)
	assert.NotEmpty(t, key2)
	assert.NotEqual(t, key1, key2)
	assert.Len(t, key1, 16) // Should be 16 characters
	assert.Len(t, key2, 16)

	// Same project should generate same key
	key1Again := cache.generateKey(projectRoot1)
	assert.Equal(t, key1, key1Again)
}

func TestAnalysisCache_generateChecksum(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	result1 := &AnalysisResult{
		ProjectInfo: &ProjectInfo{Name: "project1"},
	}
	result2 := &AnalysisResult{
		ProjectInfo: &ProjectInfo{Name: "project2"},
	}

	checksum1 := cache.generateChecksum(result1)
	checksum2 := cache.generateChecksum(result2)

	assert.NotEmpty(t, checksum1)
	assert.NotEmpty(t, checksum2)
	assert.NotEqual(t, checksum1, checksum2)

	// Same result should generate same checksum
	checksum1Again := cache.generateChecksum(result1)
	assert.Equal(t, checksum1, checksum1Again)
}

func TestAnalysisCache_generateMetadata(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	// Create a temporary project directory
	tempDir, err := os.MkdirTemp("", "test-project")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	result := &AnalysisResult{
		ProjectInfo: &ProjectInfo{
			ConfigFiles: []string{"go.mod", "package.json"},
			EntryPoints: []string{"main.go"},
		},
		FileStructure: &FileStructure{
			TotalFiles: 5,
		},
		Dependencies: &DependencyInfo{
			Direct: []Dependency{
				{Name: "testify", Version: "v1.8.0"},
			},
		},
	}

	metadata, err := cache.generateMetadata(tempDir, result)
	require.NoError(t, err)
	assert.NotNil(t, metadata)

	assert.Equal(t, tempDir, metadata.ProjectPath)
	assert.Equal(t, "1.0.0", metadata.CacheVersion)
	assert.Equal(t, 5, metadata.FileCount)
	assert.NotNil(t, metadata.Dependencies)
	assert.NotNil(t, metadata.FileChecksums)
	assert.Equal(t, "v1.8.0", metadata.Dependencies["testify"])
}

func TestAnalysisCache_calculateFileChecksum(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	// Create a temporary file
	tempFile, err := os.CreateTemp("", "test-file")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()

	content := "test file content"
	_, err = tempFile.WriteString(content)
	require.NoError(t, err)
	_ = tempFile.Close()

	checksum := cache.calculateFileChecksum(tempFile.Name())
	assert.NotEmpty(t, checksum)
	assert.Len(t, checksum, 8) // Should be 8 characters

	// Same file should generate same checksum
	checksum2 := cache.calculateFileChecksum(tempFile.Name())
	assert.Equal(t, checksum, checksum2)

	// Non-existent file should return empty checksum
	emptyChecksum := cache.calculateFileChecksum("/nonexistent/file")
	assert.Empty(t, emptyChecksum)
}

func TestAnalysisCache_isValid(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	// Create a temporary project directory
	tempDir, err := os.MkdirTemp("", "test-project")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	now := time.Now()

	// Valid entry
	validEntry := &CacheEntry{
		Timestamp:  now,
		Expiration: now.Add(time.Hour),
		Metadata: CacheMetadata{
			CacheVersion:  "1.0.0",
			LastModified:  now,
			FileChecksums: make(map[string]string),
		},
	}

	assert.True(t, cache.isValid(validEntry, tempDir))

	// Expired entry
	expiredEntry := &CacheEntry{
		Timestamp:  now.Add(-2 * time.Hour),
		Expiration: now.Add(-time.Hour),
		Metadata: CacheMetadata{
			CacheVersion:  "1.0.0",
			LastModified:  now,
			FileChecksums: make(map[string]string),
		},
	}

	assert.False(t, cache.isValid(expiredEntry, tempDir))

	// Wrong cache version
	wrongVersionEntry := &CacheEntry{
		Timestamp:  now,
		Expiration: now.Add(time.Hour),
		Metadata: CacheMetadata{
			CacheVersion:  "0.9.0",
			LastModified:  now,
			FileChecksums: make(map[string]string),
		},
	}

	assert.False(t, cache.isValid(wrongVersionEntry, tempDir))
}

func TestAnalysisCache_evictOldMemoryEntries(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	cache.maxMemoryEntries = 3 // Set low limit for testing
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	// Add entries to exceed the limit
	for i := 0; i < 5; i++ {
		entry := &CacheEntry{
			Key:       cache.generateKey(string(rune(i))),
			Timestamp: time.Now().Add(time.Duration(-i) * time.Minute), // Different timestamps
		}
		cache.memoryCache[entry.Key] = entry
	}

	assert.Equal(t, 5, len(cache.memoryCache))

	// Trigger eviction
	cache.evictOldMemoryEntries()

	// Should be at or below the limit
	assert.True(t, len(cache.memoryCache) <= cache.maxMemoryEntries)
}

func TestAnalysisCache_saveToDisk_loadFromDisk(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	entry := &CacheEntry{
		Key:        "test-key",
		Timestamp:  time.Now(),
		Expiration: time.Now().Add(time.Hour),
		Data: &AnalysisResult{
			ProjectInfo: &ProjectInfo{Name: "test-project"},
		},
		Metadata: CacheMetadata{
			CacheVersion: "1.0.0",
		},
	}

	// Save to disk
	cache.saveToDisk(entry)

	// Verify file exists
	filePath := filepath.Join(cache.cacheDir, entry.Key+".json")
	_, err := os.Stat(filePath)
	assert.NoError(t, err)

	// Load from disk
	loadedEntry := cache.loadFromDiskByKey(entry.Key)
	require.NotNil(t, loadedEntry)
	assert.Equal(t, entry.Key, loadedEntry.Key)
	assert.Equal(t, entry.Data.ProjectInfo.Name, loadedEntry.Data.ProjectInfo.Name)
}

func TestAnalysisCache_loadCacheFile_corruptedFile(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	// Create corrupted cache file
	corruptedFile := filepath.Join(cache.cacheDir, "corrupted.json")
	err := os.WriteFile(corruptedFile, []byte("invalid json"), 0600)
	require.NoError(t, err)

	// Should return nil and remove corrupted file
	entry := cache.loadCacheFile(corruptedFile)
	assert.Nil(t, entry)

	// File should be removed
	_, err = os.Stat(corruptedFile)
	assert.True(t, os.IsNotExist(err))
}

func TestCacheMetadata_JSON(t *testing.T) {
	metadata := CacheMetadata{
		ProjectPath:   testProjectPath,
		ConfigHash:    "abc123",
		FileCount:     10,
		LastModified:  time.Now(),
		CacheVersion:  "1.0.0",
		Dependencies:  map[string]string{"testify": "v1.8.0"},
		FileChecksums: map[string]string{"main.go": "def456"},
	}

	// Test JSON serialization
	data, err := json.Marshal(metadata)
	require.NoError(t, err)

	var unmarshaled CacheMetadata
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, metadata.ProjectPath, unmarshaled.ProjectPath)
	assert.Equal(t, metadata.ConfigHash, unmarshaled.ConfigHash)
	assert.Equal(t, metadata.FileCount, unmarshaled.FileCount)
	assert.Equal(t, metadata.CacheVersion, unmarshaled.CacheVersion)
	assert.Equal(t, metadata.Dependencies, unmarshaled.Dependencies)
	assert.Equal(t, metadata.FileChecksums, unmarshaled.FileChecksums)
}

func TestCacheEntry_JSON(t *testing.T) {
	now := time.Now()
	entry := CacheEntry{
		Key:        "test-key",
		Timestamp:  now,
		Expiration: now.Add(time.Hour),
		Checksum:   "abc123",
		Data: &AnalysisResult{
			ProjectInfo: &ProjectInfo{Name: "test-project"},
		},
		Metadata: CacheMetadata{
			CacheVersion: "1.0.0",
		},
	}

	// Test JSON serialization
	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var unmarshaled CacheEntry
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, entry.Key, unmarshaled.Key)
	assert.Equal(t, entry.Checksum, unmarshaled.Checksum)
	assert.Equal(t, entry.Data.ProjectInfo.Name, unmarshaled.Data.ProjectInfo.Name)
	assert.Equal(t, entry.Metadata.CacheVersion, unmarshaled.Metadata.CacheVersion)
}

func TestCacheStats(t *testing.T) {
	now := time.Now()
	stats := CacheStats{
		MemoryEntries: 5,
		DiskEntries:   3,
		HitRate:       0.85,
		TotalSize:     1024,
		OldestEntry:   now.Add(-time.Hour),
		NewestEntry:   now,
	}

	// Test JSON serialization
	data, err := json.Marshal(stats)
	require.NoError(t, err)

	var unmarshaled CacheStats
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, stats.MemoryEntries, unmarshaled.MemoryEntries)
	assert.Equal(t, stats.DiskEntries, unmarshaled.DiskEntries)
	assert.Equal(t, stats.HitRate, unmarshaled.HitRate)
	assert.Equal(t, stats.TotalSize, unmarshaled.TotalSize)
}

func TestGetCacheDirectory(t *testing.T) {
	cacheDir := getCacheDirectory()
	assert.NotEmpty(t, cacheDir)

	// Should contain ccagents and cache
	assert.Contains(t, cacheDir, "ccagents")
	assert.Contains(t, cacheDir, "cache")
}

func TestAbsDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected time.Duration
	}{
		{"positive duration", time.Hour, time.Hour},
		{"negative duration", -time.Hour, time.Hour},
		{"zero duration", 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := absDuration(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnalysisCache_MemoryAndDiskIntegration(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	projectRoot := testProjectPath
	result := &AnalysisResult{
		ProjectInfo: &ProjectInfo{Name: "integration-test"},
	}

	// Set entry (should be in both memory and disk)
	cache.Set(projectRoot, result)

	// Clear memory cache but keep disk
	cache.memoryCache = make(map[string]*CacheEntry)

	// Get should load from disk into memory
	retrieved := cache.Get(projectRoot)
	require.NotNil(t, retrieved)
	assert.Equal(t, result.ProjectInfo.Name, retrieved.ProjectInfo.Name)

	// Verify it's now in memory cache
	key := cache.generateKey(projectRoot)
	_, exists := cache.memoryCache[key]
	assert.True(t, exists)
}

func TestAnalysisCache_ProjectHasChanged(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	// Create temporary file
	tempFile, err := os.CreateTemp("", "test-file")
	require.NoError(t, err)
	defer func() { _ = os.Remove(tempFile.Name()) }()

	content := "original content"
	_, err = tempFile.WriteString(content)
	require.NoError(t, err)
	_ = tempFile.Close()

	// Create cache entry with file checksum
	originalChecksum := cache.calculateFileChecksum(tempFile.Name())
	tempDir := filepath.Dir(tempFile.Name())
	fileName := filepath.Base(tempFile.Name())

	entry := &CacheEntry{
		Metadata: CacheMetadata{
			FileChecksums: map[string]string{fileName: originalChecksum},
			LastModified:  time.Now(),
		},
	}

	hasChanged := cache.projectHasChanged(entry, tempDir)
	assert.False(t, hasChanged)

	// Modify file
	err = os.WriteFile(tempFile.Name(), []byte("modified content"), 0600)
	require.NoError(t, err)

	// Project should have changed
	hasChanged = cache.projectHasChanged(entry, tempDir)
	assert.True(t, hasChanged)
}

func TestAnalysisCache_ConcurrentDirectoryCreation(t *testing.T) {
	// Test concurrent cache creation to ensure no race conditions
	const numGoroutines = 10
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)
	caches := make([]*AnalysisCache, numGoroutines)

	// Create multiple caches concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			cache := NewAnalysisCache(time.Hour)
			if cache == nil {
				errors <- fmt.Errorf("cache %d is nil", index)
				return
			}

			caches[index] = cache

			// Test basic cache operations
			projectRoot := fmt.Sprintf("/test/project%d", index)
			result := &AnalysisResult{
				ProjectInfo: &ProjectInfo{Name: fmt.Sprintf("test-project-%d", index)},
			}

			cache.Set(projectRoot, result)
			retrieved := cache.Get(projectRoot)
			if retrieved == nil {
				errors <- fmt.Errorf("cache %d failed to retrieve data", index)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent cache creation error: %v", err)
	}

	// Cleanup all caches
	for _, cache := range caches {
		if cache != nil {
			cache.Clear()
			_ = os.RemoveAll(cache.cacheDir)
		}
	}
}

func TestGetCacheDirectory_UniquePerProcess(t *testing.T) {
	// Test that cache directory includes process ID for uniqueness
	dir1 := getCacheDirectory()
	dir2 := getCacheDirectory()

	// Should be the same for the same process
	assert.Equal(t, dir1, dir2)

	// Should contain process ID
	pid := os.Getpid()
	expectedSuffix := "proc-" + strconv.Itoa(pid)
	assert.Contains(t, dir1, expectedSuffix)

	// Should be a valid path
	assert.NotEmpty(t, dir1)
	assert.True(t, filepath.IsAbs(dir1) || strings.Contains(dir1, string(filepath.Separator)))
}

func TestAnalysisCache_EnsureCacheDirectoryExists(t *testing.T) {
	cache := NewAnalysisCache(time.Hour)
	defer func() { _ = os.RemoveAll(cache.cacheDir) }()

	// Directory should already exist from constructor
	_, err := os.Stat(cache.cacheDir)
	assert.NoError(t, err)

	// Remove directory and test recreation
	err = os.RemoveAll(cache.cacheDir)
	require.NoError(t, err)

	// Ensure directory creation works
	err = cache.ensureCacheDirectoryExists()
	assert.NoError(t, err)

	// Verify directory exists
	stat, err := os.Stat(cache.cacheDir)
	assert.NoError(t, err)
	assert.True(t, stat.IsDir())
}
