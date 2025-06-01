package performance

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/pkg/logger"
)

// ConnectionPool manages HTTP client connections for API calls
type ConnectionPool struct {
	clients    map[string]*http.Client
	maxClients int
	timeout    time.Duration
	mutex      sync.RWMutex
	logger     *logger.Logger
	stats      ConnectionStats
}

// ConnectionStats tracks connection pool statistics
type ConnectionStats struct {
	TotalConnections  int64
	ActiveConnections int64
	ReuseCount        int64
	CreateCount       int64
	mutex             sync.RWMutex
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxClients int, timeout time.Duration, logger *logger.Logger) *ConnectionPool {
	if maxClients <= 0 {
		maxClients = 10
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &ConnectionPool{
		clients:    make(map[string]*http.Client),
		maxClients: maxClients,
		timeout:    timeout,
		logger:     logger,
	}
}

// GetClient returns an HTTP client for the specified host
func (cp *ConnectionPool) GetClient(host string) *http.Client {
	cp.mutex.RLock()
	client, exists := cp.clients[host]
	cp.mutex.RUnlock()

	if exists {
		cp.stats.mutex.Lock()
		cp.stats.ReuseCount++
		cp.stats.mutex.Unlock()
		return client
	}

	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	// Double-check after acquiring write lock
	if client, exists := cp.clients[host]; exists {
		cp.stats.ReuseCount++
		return client
	}

	// Create new client if under limit
	if len(cp.clients) < cp.maxClients {
		client = &http.Client{
			Timeout: cp.timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		}

		cp.clients[host] = client
		cp.stats.TotalConnections++
		cp.stats.ActiveConnections++
		cp.stats.CreateCount++

		if cp.logger != nil {
			cp.logger.Debug("Created new HTTP client (host: %s, total: %d)", host, len(cp.clients))
		}

		return client
	}

	// Return a default client if pool is full
	return &http.Client{Timeout: cp.timeout}
}

// RemoveClient removes a client from the pool
func (cp *ConnectionPool) RemoveClient(host string) {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()

	if _, exists := cp.clients[host]; exists {
		delete(cp.clients, host)
		cp.stats.ActiveConnections--

		if cp.logger != nil {
			cp.logger.Debug("Removed HTTP client (host: %s, remaining: %d)", host, len(cp.clients))
		}
	}
}

// GetStats returns connection pool statistics
func (cp *ConnectionPool) GetStats() map[string]interface{} {
	cp.stats.mutex.RLock()
	defer cp.stats.mutex.RUnlock()

	cp.mutex.RLock()
	poolSize := len(cp.clients)
	cp.mutex.RUnlock()

	return map[string]interface{}{
		"pool_size":          poolSize,
		"max_clients":        cp.maxClients,
		"total_connections":  cp.stats.TotalConnections,
		"active_connections": cp.stats.ActiveConnections,
		"reuse_count":        cp.stats.ReuseCount,
		"create_count":       cp.stats.CreateCount,
	}
}

// LazyLoader provides lazy loading for UI components
type LazyLoader struct {
	components map[string]*Component
	loader     ComponentLoader
	mutex      sync.RWMutex
	logger     *logger.Logger
}

// Component represents a lazy-loaded component
type Component struct {
	Name       string
	Data       interface{}
	LoadTime   time.Time
	AccessTime time.Time
	IsLoaded   bool
	Loading    bool
	LoadChan   chan bool
}

// ComponentLoader is a function that loads component data
type ComponentLoader func(name string) (interface{}, error)

// NewLazyLoader creates a new lazy loader
func NewLazyLoader(loader ComponentLoader, logger *logger.Logger) *LazyLoader {
	return &LazyLoader{
		components: make(map[string]*Component),
		loader:     loader,
		logger:     logger,
	}
}

// LoadComponent loads a component lazily
func (ll *LazyLoader) LoadComponent(name string) (interface{}, error) {
	ll.mutex.RLock()
	component, exists := ll.components[name]
	ll.mutex.RUnlock()

	if exists {
		component.AccessTime = time.Now()

		if component.IsLoaded {
			return component.Data, nil
		}

		if component.Loading {
			// Wait for ongoing load
			<-component.LoadChan
			return component.Data, nil
		}
	}

	ll.mutex.Lock()

	// Double-check after acquiring write lock
	if component, exists := ll.components[name]; exists {
		component.AccessTime = time.Now()

		if component.IsLoaded {
			ll.mutex.Unlock()
			return component.Data, nil
		}

		if component.Loading {
			// Release mutex before waiting on channel to avoid deadlock
			loadChan := component.LoadChan
			ll.mutex.Unlock()
			<-loadChan
			return component.Data, nil
		}
	}

	// Create new component entry
	component = &Component{
		Name:     name,
		Loading:  true,
		LoadChan: make(chan bool, 1),
	}
	ll.components[name] = component

	// Release mutex before starting goroutine to avoid deadlock
	ll.mutex.Unlock()

	// Load component data
	go func() {
		defer func() {
			ll.mutex.Lock()
			component.Loading = false
			ll.mutex.Unlock()
			close(component.LoadChan)
		}()

		if ll.logger != nil {
			ll.logger.Debug("Loading component: %s", name)
		}

		data, err := ll.loader(name)
		if err != nil {
			if ll.logger != nil {
				ll.logger.Error("Failed to load component: %s (error: %v)", name, err)
			}
			return
		}

		ll.mutex.Lock()
		component.Data = data
		component.IsLoaded = true
		component.LoadTime = time.Now()
		ll.mutex.Unlock()

		if ll.logger != nil {
			ll.logger.Debug("Component loaded successfully: %s", name)
		}
	}()

	// Wait for load to complete
	<-component.LoadChan

	return component.Data, nil
}

// UnloadComponent unloads a component to free memory
func (ll *LazyLoader) UnloadComponent(name string) {
	ll.mutex.Lock()
	defer ll.mutex.Unlock()

	if component, exists := ll.components[name]; exists {
		component.Data = nil
		component.IsLoaded = false

		if ll.logger != nil {
			ll.logger.Debug("Unloaded component: %s", name)
		}
	}
}

// GetLoadedComponents returns information about loaded components
func (ll *LazyLoader) GetLoadedComponents() map[string]interface{} {
	ll.mutex.RLock()
	defer ll.mutex.RUnlock()

	result := make(map[string]interface{})
	for name, component := range ll.components {
		result[name] = map[string]interface{}{
			"is_loaded":   component.IsLoaded,
			"load_time":   component.LoadTime,
			"access_time": component.AccessTime,
		}
	}

	return result
}

// BatchProcessor handles batch processing optimizations
type BatchProcessor struct {
	batchSize    int
	flushTimeout time.Duration
	batches      map[string]*Batch
	mutex        sync.RWMutex
	processor    BatchProcessorFunc
	logger       *logger.Logger
}

// Batch represents a batch of items to process
type Batch struct {
	Items     []interface{}
	CreatedAt time.Time
	Timer     *time.Timer
	mutex     sync.Mutex
}

// BatchProcessorFunc processes a batch of items
type BatchProcessorFunc func(batchType string, items []interface{}) error

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize int, flushTimeout time.Duration, processor BatchProcessorFunc, logger *logger.Logger) *BatchProcessor {
	if batchSize <= 0 {
		batchSize = 100
	}
	if flushTimeout <= 0 {
		flushTimeout = 1 * time.Second
	}

	return &BatchProcessor{
		batchSize:    batchSize,
		flushTimeout: flushTimeout,
		batches:      make(map[string]*Batch),
		processor:    processor,
		logger:       logger,
	}
}

// AddItem adds an item to a batch
func (bp *BatchProcessor) AddItem(batchType string, item interface{}) error {
	bp.mutex.Lock()
	batch, exists := bp.batches[batchType]
	if !exists {
		batch = &Batch{
			Items:     make([]interface{}, 0, bp.batchSize),
			CreatedAt: time.Now(),
		}
		bp.batches[batchType] = batch

		// Set up flush timer
		batch.Timer = time.AfterFunc(bp.flushTimeout, func() {
			_ = bp.flushBatch(batchType) //nolint:errcheck // Batch flush errors are logged internally
		})
	}
	bp.mutex.Unlock()

	batch.mutex.Lock()
	batch.Items = append(batch.Items, item)
	shouldFlush := len(batch.Items) >= bp.batchSize
	batch.mutex.Unlock()

	if shouldFlush {
		return bp.flushBatch(batchType)
	}

	return nil
}

// flushBatch processes and clears a batch
func (bp *BatchProcessor) flushBatch(batchType string) error {
	bp.mutex.Lock()
	batch, exists := bp.batches[batchType]
	if !exists {
		bp.mutex.Unlock()
		return nil
	}

	// Remove batch from map
	delete(bp.batches, batchType)
	bp.mutex.Unlock()

	// Stop timer if still running
	if batch.Timer != nil {
		batch.Timer.Stop()
	}

	batch.mutex.Lock()
	items := make([]interface{}, len(batch.Items))
	copy(items, batch.Items)
	batch.mutex.Unlock()

	if len(items) == 0 {
		return nil
	}

	if bp.logger != nil {
		bp.logger.Debug("Processing batch (type: %s, size: %d, age: %v)", batchType, len(items), time.Since(batch.CreatedAt))
	}

	return bp.processor(batchType, items)
}

// FlushAll flushes all pending batches
func (bp *BatchProcessor) FlushAll() error {
	bp.mutex.RLock()
	batchTypes := make([]string, 0, len(bp.batches))
	for batchType := range bp.batches {
		batchTypes = append(batchTypes, batchType)
	}
	bp.mutex.RUnlock()

	var lastError error
	for _, batchType := range batchTypes {
		if err := bp.flushBatch(batchType); err != nil {
			lastError = err
			if bp.logger != nil {
				bp.logger.Error("Failed to flush batch (type: %s, error: %v)", batchType, err)
			}
		}
	}

	return lastError
}

// GetBatchStats returns batch processing statistics
func (bp *BatchProcessor) GetBatchStats() map[string]interface{} {
	bp.mutex.RLock()
	defer bp.mutex.RUnlock()

	stats := map[string]interface{}{
		"active_batches": len(bp.batches),
		"batch_size":     bp.batchSize,
		"flush_timeout":  bp.flushTimeout,
	}

	batchDetails := make(map[string]interface{})
	for batchType, batch := range bp.batches {
		batch.mutex.Lock()
		batchDetails[batchType] = map[string]interface{}{
			"item_count": len(batch.Items),
			"age":        time.Since(batch.CreatedAt),
		}
		batch.mutex.Unlock()
	}
	stats["batches"] = batchDetails

	return stats
}

// ResourceCleanup handles automatic resource cleanup
type ResourceCleanup struct {
	cleaners     []CleanupFunc
	interval     time.Duration
	ctx          context.Context
	cancel       context.CancelFunc
	running      bool
	mutex        sync.RWMutex
	logger       *logger.Logger
	lastCleanup  time.Time
	cleanupCount int64
}

// CleanupFunc is a function that performs cleanup
type CleanupFunc func() error

// NewResourceCleanup creates a new resource cleanup manager
func NewResourceCleanup(interval time.Duration, logger *logger.Logger) *ResourceCleanup {
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &ResourceCleanup{
		cleaners: make([]CleanupFunc, 0),
		interval: interval,
		ctx:      ctx,
		cancel:   cancel,
		logger:   logger,
	}
}

// AddCleanupFunc adds a cleanup function
func (rc *ResourceCleanup) AddCleanupFunc(cleanup CleanupFunc) {
	rc.mutex.Lock()
	defer rc.mutex.Unlock()
	rc.cleaners = append(rc.cleaners, cleanup)
}

// Start starts the cleanup routine
func (rc *ResourceCleanup) Start() {
	rc.mutex.Lock()
	if rc.running {
		rc.mutex.Unlock()
		return
	}
	rc.running = true
	rc.mutex.Unlock()

	go rc.cleanupLoop()
}

// Stop stops the cleanup routine
func (rc *ResourceCleanup) Stop() {
	rc.cancel()

	rc.mutex.Lock()
	rc.running = false
	rc.mutex.Unlock()
}

// RunCleanup manually runs all cleanup functions
func (rc *ResourceCleanup) RunCleanup() error {
	rc.mutex.RLock()
	cleaners := make([]CleanupFunc, len(rc.cleaners))
	copy(cleaners, rc.cleaners)
	rc.mutex.RUnlock()

	var lastError error
	for i, cleanup := range cleaners {
		if err := cleanup(); err != nil {
			lastError = err
			if rc.logger != nil {
				rc.logger.Error("Cleanup function failed (index: %d, error: %v)", i, err)
			}
		}
	}

	rc.mutex.Lock()
	rc.lastCleanup = time.Now()
	rc.cleanupCount++
	rc.mutex.Unlock()

	if rc.logger != nil {
		rc.logger.Debug("Cleanup completed (functions: %d, errors: %t)", len(cleaners), lastError != nil)
	}

	return lastError
}

// GetCleanupStats returns cleanup statistics
func (rc *ResourceCleanup) GetCleanupStats() map[string]interface{} {
	rc.mutex.RLock()
	defer rc.mutex.RUnlock()

	return map[string]interface{}{
		"cleanup_functions": len(rc.cleaners),
		"last_cleanup":      rc.lastCleanup,
		"cleanup_count":     rc.cleanupCount,
		"interval":          rc.interval,
		"running":           rc.running,
	}
}

// cleanupLoop runs the cleanup loop
func (rc *ResourceCleanup) cleanupLoop() {
	ticker := time.NewTicker(rc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = rc.RunCleanup() //nolint:errcheck // Cleanup errors are not critical
		case <-rc.ctx.Done():
			return
		}
	}
}
