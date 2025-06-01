// Package performance provides performance optimization utilities for ccAgents
package performance

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/errors"
	"github.com/fumiya-kume/cca/pkg/logger"
)

// UIOptimizer handles UI rendering optimizations
type UIOptimizer struct {
	frameRate      int
	lastRender     time.Time
	renderMutex    sync.Mutex
	batchSize      int
	pendingUpdates []UIUpdate
	updateMutex    sync.Mutex
	logger         *logger.Logger
}

// UIUpdate represents a UI update operation
type UIUpdate struct {
	Type      string
	Data      interface{}
	Timestamp time.Time
	Priority  int
}

// NewUIOptimizer creates a new UI optimizer
func NewUIOptimizer(frameRate int, batchSize int, logger *logger.Logger) *UIOptimizer {
	if frameRate <= 0 {
		frameRate = 60 // Default 60 FPS
	}
	if batchSize <= 0 {
		batchSize = 10 // Default batch size
	}

	return &UIOptimizer{
		frameRate:      frameRate,
		batchSize:      batchSize,
		pendingUpdates: make([]UIUpdate, 0, batchSize),
		logger:         logger,
	}
}

// ShouldRender determines if a render should occur based on frame rate limiting
func (uo *UIOptimizer) ShouldRender() bool {
	uo.renderMutex.Lock()
	defer uo.renderMutex.Unlock()

	minInterval := time.Second / time.Duration(uo.frameRate)
	now := time.Now()

	if now.Sub(uo.lastRender) >= minInterval {
		uo.lastRender = now
		return true
	}

	return false
}

// AddUpdate adds a UI update to the batch
func (uo *UIOptimizer) AddUpdate(update UIUpdate) {
	uo.updateMutex.Lock()
	defer uo.updateMutex.Unlock()

	update.Timestamp = time.Now()
	uo.pendingUpdates = append(uo.pendingUpdates, update)
}

// GetBatchedUpdates returns batched updates and clears the pending list
func (uo *UIOptimizer) GetBatchedUpdates() []UIUpdate {
	uo.updateMutex.Lock()
	defer uo.updateMutex.Unlock()

	if len(uo.pendingUpdates) == 0 {
		return nil
	}

	// Sort by priority (higher priority first)
	updates := make([]UIUpdate, len(uo.pendingUpdates))
	copy(updates, uo.pendingUpdates)

	// Simple priority sort
	for i := 0; i < len(updates)-1; i++ {
		for j := i + 1; j < len(updates); j++ {
			if updates[i].Priority < updates[j].Priority {
				updates[i], updates[j] = updates[j], updates[i]
			}
		}
	}

	// Clear pending updates
	uo.pendingUpdates = uo.pendingUpdates[:0]

	// Return up to batch size
	if len(updates) > uo.batchSize {
		return updates[:uo.batchSize]
	}

	return updates
}

// HasPendingUpdates returns true if there are pending UI updates
func (uo *UIOptimizer) HasPendingUpdates() bool {
	uo.updateMutex.Lock()
	defer uo.updateMutex.Unlock()
	return len(uo.pendingUpdates) > 0
}

// MemoryManager handles memory optimization for output buffers
type MemoryManager struct {
	bufferPools   map[string]*BufferPool
	maxBufferSize int
	gcThreshold   int64
	lastGC        time.Time
	mutex         sync.RWMutex
	logger        *logger.Logger
}

// BufferPool manages reusable buffers
type BufferPool struct {
	pool      sync.Pool
	size      int
	allocated int64
	reused    int64
	mutex     sync.RWMutex
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager(maxBufferSize int, gcThreshold int64, logger *logger.Logger) *MemoryManager {
	if maxBufferSize <= 0 {
		maxBufferSize = 64 * 1024 // 64KB default
	}
	if gcThreshold <= 0 {
		gcThreshold = 100 * 1024 * 1024 // 100MB default
	}

	return &MemoryManager{
		bufferPools:   make(map[string]*BufferPool),
		maxBufferSize: maxBufferSize,
		gcThreshold:   gcThreshold,
		logger:        logger,
	}
}

// GetBufferPool returns or creates a buffer pool for the given name
func (mm *MemoryManager) GetBufferPool(name string, size int) *BufferPool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	if pool, exists := mm.bufferPools[name]; exists {
		return pool
	}

	pool := &BufferPool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				buffer := make([]byte, 0, size)
				return &buffer
			},
		},
	}

	mm.bufferPools[name] = pool
	return pool
}

// GetBuffer gets a buffer from the pool
func (bp *BufferPool) GetBuffer() []byte {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	buffer := *bp.pool.Get().(*[]byte) //nolint:errcheck // Type assertion is safe as pool is controlled internally
	buffer = buffer[:0] // Reset length but keep capacity
	bp.allocated++
	return buffer
}

// PutBuffer returns a buffer to the pool
func (bp *BufferPool) PutBuffer(buffer []byte) {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	if cap(buffer) <= bp.size {
		bp.pool.Put(&buffer)
		bp.reused++
	}
}

// GetStats returns buffer pool statistics
func (bp *BufferPool) GetStats() (allocated, reused int64) {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()
	return bp.allocated, bp.reused
}

// TriggerGC triggers garbage collection if threshold is exceeded
func (mm *MemoryManager) TriggerGC() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Safely convert uint64 to int64 for comparison
	const maxInt64 = int64(^uint64(0) >> 1)
	memAlloc := int64(m.Alloc)
	if m.Alloc > uint64(maxInt64) {
		memAlloc = maxInt64
	}

	if memAlloc > mm.gcThreshold && time.Since(mm.lastGC) > time.Minute {
		if mm.logger != nil {
			mm.logger.Debug("Triggering garbage collection (allocated: %d, threshold: %d)", m.Alloc, mm.gcThreshold)
		}

		runtime.GC()
		mm.lastGC = time.Now()
	}
}

// GetMemoryStats returns current memory statistics
func (mm *MemoryManager) GetMemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := map[string]interface{}{
		"allocated":   m.Alloc,
		"total_alloc": m.TotalAlloc,
		"sys":         m.Sys,
		"num_gc":      m.NumGC,
		"gc_pause_ns": m.PauseNs[(m.NumGC+255)%256],
		"goroutines":  runtime.NumGoroutine(),
	}

	// Add buffer pool stats
	mm.mutex.RLock()
	poolStats := make(map[string]map[string]int64)
	for name, pool := range mm.bufferPools {
		allocated, reused := pool.GetStats()
		poolStats[name] = map[string]int64{
			"allocated": allocated,
			"reused":    reused,
		}
	}
	mm.mutex.RUnlock()

	stats["buffer_pools"] = poolStats
	return stats
}

// ProcessPool manages a pool of reusable processes/goroutines
type ProcessPool struct {
	workers    chan *Worker
	maxWorkers int
	activeJobs int64
	totalJobs  int64
	mutex      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	logger     *logger.Logger
}

// Worker represents a worker in the process pool
type Worker struct {
	ID       int
	jobChan  chan Job
	quitChan chan bool
	pool     *ProcessPool
}

// Job represents work to be executed
type Job struct {
	ID       string
	Function func() error
	Result   chan error
	Priority int
}

// NewProcessPool creates a new process pool
func NewProcessPool(maxWorkers int, logger *logger.Logger) *ProcessPool {
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &ProcessPool{
		workers:    make(chan *Worker, maxWorkers),
		maxWorkers: maxWorkers,
		ctx:        ctx,
		cancel:     cancel,
		logger:     logger,
	}

	// Start workers
	for i := 0; i < maxWorkers; i++ {
		worker := &Worker{
			ID:       i,
			jobChan:  make(chan Job),
			quitChan: make(chan bool),
			pool:     pool,
		}

		go worker.start()
		pool.workers <- worker
	}

	return pool
}

// Submit submits a job to the process pool
func (pp *ProcessPool) Submit(job Job) error {
	select {
	case worker := <-pp.workers:
		pp.mutex.Lock()
		pp.activeJobs++
		pp.totalJobs++
		pp.mutex.Unlock()

		select {
		case worker.jobChan <- job:
			return nil
		case <-pp.ctx.Done():
			pp.workers <- worker
			return errors.NewError(errors.ErrorTypeSystem).
				WithMessage("process pool is shutting down").
				Build()
		}
	case <-pp.ctx.Done():
		return errors.NewError(errors.ErrorTypeSystem).
			WithMessage("process pool is shutting down").
			Build()
	}
}

// GetStats returns process pool statistics
func (pp *ProcessPool) GetStats() map[string]interface{} {
	pp.mutex.RLock()
	defer pp.mutex.RUnlock()

	return map[string]interface{}{
		"max_workers":       pp.maxWorkers,
		"active_jobs":       pp.activeJobs,
		"total_jobs":        pp.totalJobs,
		"available_workers": len(pp.workers),
	}
}

// Shutdown gracefully shuts down the process pool
func (pp *ProcessPool) Shutdown(timeout time.Duration) error {
	pp.cancel()

	// Wait for all workers to finish with timeout
	done := make(chan bool)
	go func() {
		// Wait for all workers to return
		for i := 0; i < pp.maxWorkers; i++ {
			select {
			case worker := <-pp.workers:
				close(worker.quitChan)
			case <-time.After(timeout):
				return
			}
		}
		done <- true
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return errors.NewError(errors.ErrorTypeSystem).
			WithMessage("timeout waiting for process pool shutdown").
			Build()
	}
}

// start starts the worker loop
func (w *Worker) start() {
	for {
		select {
		case job := <-w.jobChan:
			err := job.Function()

			w.pool.mutex.Lock()
			w.pool.activeJobs--
			w.pool.mutex.Unlock()

			job.Result <- err
			w.pool.workers <- w

		case <-w.quitChan:
			return
		case <-w.pool.ctx.Done():
			return
		}
	}
}

// Cache provides a generic caching interface
type Cache struct {
	data      map[string]*CacheEntry
	maxSize   int
	ttl       time.Duration
	mutex     sync.RWMutex
	evictions int64
	hits      int64
	misses    int64
	logger    *logger.Logger
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Value       interface{}
	Timestamp   time.Time
	AccessCount int64
}

// NewCache creates a new cache
func NewCache(maxSize int, ttl time.Duration, logger *logger.Logger) *Cache {
	if maxSize <= 0 {
		maxSize = 1000
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}

	cache := &Cache{
		data:    make(map[string]*CacheEntry),
		maxSize: maxSize,
		ttl:     ttl,
		logger:  logger,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a value from the cache
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	entry, exists := c.data[key]
	c.mutex.RUnlock()

	if !exists {
		c.mutex.Lock()
		c.misses++
		c.mutex.Unlock()
		return nil, false
	}

	// Check TTL
	if time.Since(entry.Timestamp) > c.ttl {
		c.mutex.Lock()
		delete(c.data, key)
		c.misses++
		c.mutex.Unlock()
		return nil, false
	}

	c.mutex.Lock()
	entry.AccessCount++
	c.hits++
	c.mutex.Unlock()

	return entry.Value, true
}

// Set stores a value in the cache
func (c *Cache) Set(key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if we need to evict
	if len(c.data) >= c.maxSize {
		c.evictLRU()
	}

	c.data[key] = &CacheEntry{
		Value:       value,
		Timestamp:   time.Now(),
		AccessCount: 0,
	}
}

// Delete removes a value from the cache
func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.data, key)
}

// GetStats returns cache statistics
func (c *Cache) GetStats() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return map[string]interface{}{
		"size":      len(c.data),
		"max_size":  c.maxSize,
		"hits":      c.hits,
		"misses":    c.misses,
		"evictions": c.evictions,
		"hit_rate":  float64(c.hits) / float64(c.hits+c.misses),
	}
}

// evictLRU evicts the least recently used item
func (c *Cache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time
	var lowestAccess int64 = -1

	for key, entry := range c.data {
		if lowestAccess == -1 || entry.AccessCount < lowestAccess ||
			(entry.AccessCount == lowestAccess && entry.Timestamp.Before(oldestTime)) {
			oldestKey = key
			oldestTime = entry.Timestamp
			lowestAccess = entry.AccessCount
		}
	}

	if oldestKey != "" {
		delete(c.data, oldestKey)
		c.evictions++
	}
}

// cleanup removes expired entries
func (c *Cache) cleanup() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		c.mutex.Lock()
		now := time.Now()
		for key, entry := range c.data {
			if now.Sub(entry.Timestamp) > c.ttl {
				delete(c.data, key)
			}
		}
		c.mutex.Unlock()
	}
}

// PerformanceManager coordinates all performance optimizations
type PerformanceManager struct {
	uiOptimizer   *UIOptimizer
	memoryManager *MemoryManager
	processPool   *ProcessPool
	cache         *Cache
	logger        *logger.Logger
}

// NewPerformanceManager creates a new performance manager
func NewPerformanceManager(config PerformanceConfig, logger *logger.Logger) *PerformanceManager {
	return &PerformanceManager{
		uiOptimizer:   NewUIOptimizer(config.UIFrameRate, config.UIBatchSize, logger),
		memoryManager: NewMemoryManager(config.MaxBufferSize, config.GCThreshold, logger),
		processPool:   NewProcessPool(config.MaxWorkers, logger),
		cache:         NewCache(config.CacheSize, config.CacheTTL, logger),
		logger:        logger,
	}
}

// PerformanceConfig holds performance configuration
type PerformanceConfig struct {
	UIFrameRate   int
	UIBatchSize   int
	MaxBufferSize int
	GCThreshold   int64
	MaxWorkers    int
	CacheSize     int
	CacheTTL      time.Duration
}

// DefaultPerformanceConfig returns default performance configuration
func DefaultPerformanceConfig() PerformanceConfig {
	return PerformanceConfig{
		UIFrameRate:   60,
		UIBatchSize:   10,
		MaxBufferSize: 64 * 1024,
		GCThreshold:   100 * 1024 * 1024,
		MaxWorkers:    runtime.NumCPU(),
		CacheSize:     1000,
		CacheTTL:      5 * time.Minute,
	}
}

// GetUIOptimizer returns the UI optimizer
func (pm *PerformanceManager) GetUIOptimizer() *UIOptimizer {
	return pm.uiOptimizer
}

// GetMemoryManager returns the memory manager
func (pm *PerformanceManager) GetMemoryManager() *MemoryManager {
	return pm.memoryManager
}

// GetProcessPool returns the process pool
func (pm *PerformanceManager) GetProcessPool() *ProcessPool {
	return pm.processPool
}

// GetCache returns the cache
func (pm *PerformanceManager) GetCache() *Cache {
	return pm.cache
}

// GetOverallStats returns overall performance statistics
func (pm *PerformanceManager) GetOverallStats() map[string]interface{} {
	stats := map[string]interface{}{
		"memory":       pm.memoryManager.GetMemoryStats(),
		"process_pool": pm.processPool.GetStats(),
		"cache":        pm.cache.GetStats(),
		"ui_optimizer": map[string]interface{}{
			"has_pending_updates": pm.uiOptimizer.HasPendingUpdates(),
		},
	}

	return stats
}

// Shutdown gracefully shuts down all performance components
func (pm *PerformanceManager) Shutdown(timeout time.Duration) error {
	return pm.processPool.Shutdown(timeout)
}
