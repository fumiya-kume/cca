// Package analysis provides comprehensive codebase analysis capabilities for ccAgents
package analysis

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/clock"
)

// AnalysisCache provides caching for analysis results
type AnalysisCache struct {
	clock            clock.Clock
	expiration       time.Duration
	cacheDir         string
	memoryCache      map[string]*CacheEntry
	maxMemoryEntries int
	mutex            sync.RWMutex
}

// CacheEntry represents a cached analysis result
type CacheEntry struct {
	Key        string          `json:"key"`
	Data       *AnalysisResult `json:"data"`
	Timestamp  time.Time       `json:"timestamp"`
	Expiration time.Time       `json:"expiration"`
	Checksum   string          `json:"checksum"`
	Metadata   CacheMetadata   `json:"metadata"`
}

// CacheMetadata contains metadata about the cached entry
type CacheMetadata struct {
	ProjectPath   string            `json:"project_path"`
	ConfigHash    string            `json:"config_hash"`
	FileCount     int               `json:"file_count"`
	LastModified  time.Time         `json:"last_modified"`
	CacheVersion  string            `json:"cache_version"`
	Dependencies  map[string]string `json:"dependencies"`
	FileChecksums map[string]string `json:"file_checksums"`
}

// NewAnalysisCache creates a new analysis cache
func NewAnalysisCache(expiration time.Duration) *AnalysisCache {
	return NewAnalysisCacheWithClock(expiration, clock.NewRealClock())
}

// NewAnalysisCacheWithClock creates a new analysis cache with a custom clock
func NewAnalysisCacheWithClock(expiration time.Duration, clk clock.Clock) *AnalysisCache {
	cacheDir := getCacheDirectory()

	cache := &AnalysisCache{
		clock:            clk,
		expiration:       expiration,
		cacheDir:         cacheDir,
		memoryCache:      make(map[string]*CacheEntry),
		maxMemoryEntries: 50, // Limit memory cache size
		mutex:            sync.RWMutex{},
	}

	// Ensure cache directory exists with retry mechanism for race conditions
	if err := cache.ensureCacheDirectoryExists(); err != nil {
		// Log error but continue without file caching
		fmt.Printf("Warning: Could not create cache directory: %v\n", err)
	}

	// Load existing cache entries from disk
	cache.loadFromDisk()

	return cache
}

// Get retrieves a cached analysis result
func (ac *AnalysisCache) Get(projectRoot string) *AnalysisResult {
	key := ac.generateKey(projectRoot)

	// First, try to get from memory cache with read lock
	ac.mutex.RLock()
	entry, exists := ac.memoryCache[key]
	if exists && ac.isValid(entry, projectRoot) {
		ac.mutex.RUnlock()
		return entry.Data
	}
	ac.mutex.RUnlock()

	// If we need to modify the cache, acquire write lock
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	// Double-check memory cache (another goroutine might have updated it)
	if entry, exists := ac.memoryCache[key]; exists {
		if ac.isValid(entry, projectRoot) {
			return entry.Data
		}
		// Invalid entry, remove it
		delete(ac.memoryCache, key)
	}

	// Check disk cache
	if entry := ac.loadFromDiskByKey(key); entry != nil {
		if ac.isValid(entry, projectRoot) {
			// Add to memory cache
			ac.memoryCache[key] = entry
			ac.evictOldMemoryEntries()
			return entry.Data
		}
		// Invalid entry, remove from disk
		ac.removeFromDisk(key)
	}

	return nil
}

// Set stores an analysis result in the cache
func (ac *AnalysisCache) Set(projectRoot string, result *AnalysisResult) {
	key := ac.generateKey(projectRoot)

	metadata, err := ac.generateMetadata(projectRoot, result)
	if err != nil {
		// Log error but continue
		fmt.Printf("Warning: Could not generate cache metadata: %v\n", err)
		return
	}

	entry := &CacheEntry{
		Key:        key,
		Data:       result,
		Timestamp:  ac.clock.Now(),
		Expiration: ac.clock.Now().Add(ac.expiration),
		Checksum:   ac.generateChecksum(result),
		Metadata:   *metadata,
	}

	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	// Store in memory cache
	ac.memoryCache[key] = entry
	ac.evictOldMemoryEntries()

	// Store on disk
	ac.saveToDisk(entry)
}

// Clear removes all cached entries
func (ac *AnalysisCache) Clear() {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	// Clear memory cache
	ac.memoryCache = make(map[string]*CacheEntry)

	// Clear disk cache
	if err := os.RemoveAll(ac.cacheDir); err != nil {
		fmt.Printf("Warning: Could not clear disk cache: %v\n", err)
	}

	// Recreate cache directory with proper error handling
	if err := ac.ensureCacheDirectoryExists(); err != nil {
		fmt.Printf("Warning: Could not recreate cache directory: %v\n", err)
	}
}

// Invalidate removes a specific cache entry
func (ac *AnalysisCache) Invalidate(projectRoot string) {
	key := ac.generateKey(projectRoot)

	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	// Remove from memory cache
	delete(ac.memoryCache, key)

	// Remove from disk cache
	ac.removeFromDisk(key)
}

// GetCacheStats returns cache statistics
func (ac *AnalysisCache) GetCacheStats() CacheStats {
	ac.mutex.RLock()
	defer ac.mutex.RUnlock()

	stats := CacheStats{
		MemoryEntries: len(ac.memoryCache),
		DiskEntries:   ac.countDiskEntries(),
		HitRate:       0.0, // Would need to track hits/misses
		TotalSize:     ac.calculateTotalSize(),
		OldestEntry:   ac.clock.Now(),
		NewestEntry:   time.Time{},
	}

	// Find oldest and newest entries
	for _, entry := range ac.memoryCache {
		if entry.Timestamp.Before(stats.OldestEntry) {
			stats.OldestEntry = entry.Timestamp
		}
		if entry.Timestamp.After(stats.NewestEntry) {
			stats.NewestEntry = entry.Timestamp
		}
	}

	return stats
}

// CleanupExpired removes expired cache entries
func (ac *AnalysisCache) CleanupExpired() {
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	now := ac.clock.Now()

	// Clean memory cache
	for key, entry := range ac.memoryCache {
		if now.After(entry.Expiration) {
			delete(ac.memoryCache, key)
		}
	}

	// Clean disk cache
	ac.cleanupExpiredDiskEntries()
}

// Private methods

func (ac *AnalysisCache) generateKey(projectRoot string) string {
	// Generate a unique key based on project path
	hasher := sha256.New()
	hasher.Write([]byte(projectRoot))
	return hex.EncodeToString(hasher.Sum(nil))[:16] // Use first 16 chars
}

func (ac *AnalysisCache) generateChecksum(result *AnalysisResult) string {
	// Generate checksum of the analysis result
	data, err := json.Marshal(result)
	if err != nil {
		return ""
	}

	hasher := sha256.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

func (ac *AnalysisCache) generateMetadata(projectRoot string, result *AnalysisResult) (*CacheMetadata, error) {
	metadata := &CacheMetadata{
		ProjectPath:   projectRoot,
		CacheVersion:  "1.0.0",
		Dependencies:  make(map[string]string),
		FileChecksums: make(map[string]string),
	}

	// Get project statistics
	if stat, err := os.Stat(projectRoot); err == nil {
		metadata.LastModified = stat.ModTime()
	}

	// Count files
	if result.FileStructure != nil {
		metadata.FileCount = result.FileStructure.TotalFiles
	}

	// Generate config hash
	if result.ProjectInfo != nil && len(result.ProjectInfo.ConfigFiles) > 0 {
		configData := strings.Join(result.ProjectInfo.ConfigFiles, ",")
		hasher := sha256.New()
		hasher.Write([]byte(configData))
		metadata.ConfigHash = hex.EncodeToString(hasher.Sum(nil))[:8]
	}

	// Store dependency versions for cache invalidation
	if result.Dependencies != nil {
		for _, dep := range result.Dependencies.Direct {
			metadata.Dependencies[dep.Name] = dep.Version
		}
	}

	// Generate checksums for important files (limited to avoid performance issues)
	ac.generateFileChecksums(projectRoot, result, metadata)

	return metadata, nil
}

func (ac *AnalysisCache) generateFileChecksums(projectRoot string, result *AnalysisResult, metadata *CacheMetadata) {
	// Generate checksums for important files to detect changes
	importantFiles := []string{}

	// Add config files
	if result.ProjectInfo != nil {
		importantFiles = append(importantFiles, result.ProjectInfo.ConfigFiles...)

		// Add entry points
		importantFiles = append(importantFiles, result.ProjectInfo.EntryPoints...)
	}

	// Limit to first 10 files to avoid performance issues
	if len(importantFiles) > 10 {
		importantFiles = importantFiles[:10]
	}

	for _, file := range importantFiles {
		filePath := filepath.Join(projectRoot, file)
		if checksum := ac.calculateFileChecksum(filePath); checksum != "" {
			metadata.FileChecksums[file] = checksum
		}
	}
}

func (ac *AnalysisCache) calculateFileChecksum(filePath string) string {
	// #nosec G304 - filePath is constructed from validated cache directory
	data, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	hasher := sha256.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))[:8] // Use first 8 chars
}

func (ac *AnalysisCache) isValid(entry *CacheEntry, projectRoot string) bool {
	now := ac.clock.Now()

	// Check expiration
	if now.After(entry.Expiration) {
		return false
	}

	// Check if project files have been modified
	if ac.projectHasChanged(entry, projectRoot) {
		return false
	}

	// Check cache version compatibility
	if entry.Metadata.CacheVersion != "1.0.0" {
		return false
	}

	return true
}

func (ac *AnalysisCache) projectHasChanged(entry *CacheEntry, projectRoot string) bool {
	// Check if important files have changed
	for file, expectedChecksum := range entry.Metadata.FileChecksums {
		filePath := filepath.Join(projectRoot, file)
		currentChecksum := ac.calculateFileChecksum(filePath)

		if currentChecksum != expectedChecksum {
			return true // File has changed
		}
	}

	// Check if project directory modification time has changed significantly
	if stat, err := os.Stat(projectRoot); err == nil {
		// Allow for some time difference due to filesystem precision
		timeDiff := absDuration(stat.ModTime().Sub(entry.Metadata.LastModified))
		if timeDiff > time.Minute {
			return true
		}
	}

	return false
}

func (ac *AnalysisCache) evictOldMemoryEntries() {
	if len(ac.memoryCache) <= ac.maxMemoryEntries {
		return
	}

	// Find oldest entries and remove them
	type entryAge struct {
		key       string
		timestamp time.Time
	}

	var entries []entryAge
	for key, entry := range ac.memoryCache {
		entries = append(entries, entryAge{key: key, timestamp: entry.Timestamp})
	}

	// Sort by timestamp (oldest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].timestamp.Before(entries[j].timestamp)
	})

	// Remove oldest entries until we're under the limit
	entriesToRemove := len(ac.memoryCache) - ac.maxMemoryEntries + 5 // Remove a few extra
	for i := 0; i < entriesToRemove && i < len(entries); i++ {
		delete(ac.memoryCache, entries[i].key)
	}
}

func (ac *AnalysisCache) saveToDisk(entry *CacheEntry) {
	filePath := filepath.Join(ac.cacheDir, entry.Key+".json")

	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		fmt.Printf("Warning: Could not marshal cache entry: %v\n", err)
		return
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		fmt.Printf("Warning: Could not write cache file: %v\n", err)
	}
}

func (ac *AnalysisCache) loadFromDisk() {
	files, err := filepath.Glob(filepath.Join(ac.cacheDir, "*.json"))
	if err != nil {
		return
	}

	for _, file := range files {
		if entry := ac.loadCacheFile(file); entry != nil {
			// Only load valid entries
			if ac.clock.Now().Before(entry.Expiration) {
				ac.memoryCache[entry.Key] = entry
			} else {
				// Remove expired file
				if err := os.Remove(file); err != nil {
					// Log error but continue cleanup - file may already be deleted
					// This is best-effort cleanup
					continue
				}
			}
		}
	}

	ac.evictOldMemoryEntries()
}

func (ac *AnalysisCache) loadFromDiskByKey(key string) *CacheEntry {
	filePath := filepath.Join(ac.cacheDir, key+".json")
	return ac.loadCacheFile(filePath)
}

func (ac *AnalysisCache) loadCacheFile(filePath string) *CacheEntry {
	// #nosec G304 - filePath is constructed from validated cache directory
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Remove corrupted file
		_ = os.Remove(filePath) //nolint:errcheck // Ignore error - file may already be deleted
		return nil
	}

	return &entry
}

func (ac *AnalysisCache) removeFromDisk(key string) {
	filePath := filepath.Join(ac.cacheDir, key+".json")
	_ = os.Remove(filePath) //nolint:errcheck // Ignore error - file may already be deleted
}

func (ac *AnalysisCache) countDiskEntries() int {
	files, err := filepath.Glob(filepath.Join(ac.cacheDir, "*.json"))
	if err != nil {
		return 0
	}
	return len(files)
}

func (ac *AnalysisCache) calculateTotalSize() int64 {
	var totalSize int64

	// Calculate memory cache size (approximate)
	for _, entry := range ac.memoryCache {
		if data, err := json.Marshal(entry); err == nil {
			totalSize += int64(len(data))
		}
	}

	// Calculate disk cache size
	files, err := filepath.Glob(filepath.Join(ac.cacheDir, "*.json"))
	if err != nil {
		return totalSize
	}

	for _, file := range files {
		if stat, err := os.Stat(file); err == nil {
			totalSize += stat.Size()
		}
	}

	return totalSize
}

func (ac *AnalysisCache) cleanupExpiredDiskEntries() {
	files, err := filepath.Glob(filepath.Join(ac.cacheDir, "*.json"))
	if err != nil {
		return
	}

	now := ac.clock.Now()
	for _, file := range files {
		if entry := ac.loadCacheFile(file); entry != nil {
			if now.After(entry.Expiration) {
				_ = os.Remove(file) //nolint:errcheck // Ignore error - cleanup operation
			}
		}
	}
}

// ensureCacheDirectoryExists creates the cache directory with proper error handling
// for race conditions that can occur in parallel execution environments
func (ac *AnalysisCache) ensureCacheDirectoryExists() error {
	// Try to create the directory with secure permissions
	err := os.MkdirAll(ac.cacheDir, 0750)

	// If creation failed, check if it's because the directory already exists
	if err != nil {
		// Check if the directory now exists (another process might have created it)
		if stat, statErr := os.Stat(ac.cacheDir); statErr == nil && stat.IsDir() {
			// Directory exists and is accessible, this is fine
			return nil
		}
		// Return the original error if directory doesn't exist or isn't accessible
		return fmt.Errorf("failed to create cache directory %s: %w", ac.cacheDir, err)
	}

	return nil
}

func getCacheDirectory() string {
	// Create unique cache directory to avoid conflicts in parallel execution
	baseDir := ""

	// Get base cache directory based on OS
	if homeDir, err := os.UserHomeDir(); err == nil {
		baseDir = filepath.Join(homeDir, ".ccagents", "cache", "analysis")
	} else {
		// Fallback to temp directory
		baseDir = filepath.Join(os.TempDir(), "ccagents-cache")
	}

	// Add process ID to make directory unique per process
	// This prevents conflicts when multiple processes/tests run in parallel
	pid := os.Getpid()
	return filepath.Join(baseDir, "proc-"+strconv.Itoa(pid))
}

// Helper functions

func absDuration(t time.Duration) time.Duration {
	if t < 0 {
		return -t
	}
	return t
}

// CacheStats represents cache statistics
type CacheStats struct {
	MemoryEntries int       `json:"memory_entries"`
	DiskEntries   int       `json:"disk_entries"`
	HitRate       float64   `json:"hit_rate"`
	TotalSize     int64     `json:"total_size"`
	OldestEntry   time.Time `json:"oldest_entry"`
	NewestEntry   time.Time `json:"newest_entry"`
}
