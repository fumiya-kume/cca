package performance

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestUIOptimizer(t *testing.T) {
	optimizer := NewUIOptimizer(60, 5, nil)

	// Test frame rate limiting
	if !optimizer.ShouldRender() {
		t.Error("First render should be allowed")
	}

	// Should not render immediately after
	if optimizer.ShouldRender() {
		t.Error("Second render should be rate limited")
	}

	// Test update batching
	for i := 0; i < 10; i++ {
		optimizer.AddUpdate(UIUpdate{
			Type:     "test",
			Data:     i,
			Priority: i,
		})
	}

	if !optimizer.HasPendingUpdates() {
		t.Error("Should have pending updates")
	}

	updates := optimizer.GetBatchedUpdates()
	if len(updates) != 5 { // batch size is 5
		t.Errorf("Expected 5 updates in batch, got %d", len(updates))
	}

	// Updates should be sorted by priority (higher first)
	if updates[0].Priority <= updates[1].Priority {
		t.Error("Updates should be sorted by priority")
	}
}

func TestMemoryManager(t *testing.T) {
	manager := NewMemoryManager(1024, 1024*1024, nil)

	// Test buffer pool creation
	pool := manager.GetBufferPool("test", 512)
	if pool == nil {
		t.Fatal("Failed to create buffer pool")
	}

	// Test buffer allocation and reuse
	buffer1 := pool.GetBuffer()
	if cap(buffer1) != 512 {
		t.Errorf("Expected buffer capacity 512, got %d", cap(buffer1))
	}

	buffer1 = append(buffer1, []byte("test data")...)
	pool.PutBuffer(buffer1)

	buffer2 := pool.GetBuffer()
	if len(buffer2) != 0 {
		t.Errorf("Expected clean buffer, got length %d", len(buffer2))
	}

	// Test stats
	allocated, reused := pool.GetStats()
	if allocated != 2 {
		t.Errorf("Expected 2 allocations, got %d", allocated)
	}
	if reused != 1 {
		t.Errorf("Expected 1 reuse, got %d", reused)
	}
}

func TestProcessPool(t *testing.T) {
	pool := NewProcessPool(2, nil)
	defer func() { _ = pool.Shutdown(5 * time.Second) }()

	// Test job submission
	jobResult := make(chan error, 1)
	job := Job{
		ID: "test-job",
		Function: func() error {
			// Simulate work with CPU-bound operation
			for i := 0; i < 1000; i++ {
				_ = i * i * i
			}
			return nil
		},
		Result: jobResult,
	}

	err := pool.Submit(job)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Wait for job completion
	select {
	case err := <-jobResult:
		if err != nil {
			t.Errorf("Job failed: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Job timed out")
	}

	// Test stats
	stats := pool.GetStats()
	if stats["total_jobs"].(int64) != 1 {
		t.Errorf("Expected 1 total job, got %v", stats["total_jobs"])
	}
}

func TestCache(t *testing.T) {
	cache := NewCache(3, 100*time.Millisecond, nil)

	// Test basic operations
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	value, exists := cache.Get("key1")
	if !exists || value != "value1" {
		t.Errorf("Expected value1, got %v, exists: %v", value, exists)
	}

	// Test cache size limit
	cache.Set("key3", "value3")
	cache.Set("key4", "value4") // Should evict oldest

	_, _ = cache.Get("key1") // May be evicted
	cache.Get("key2")             // Access to update usage

	// Test TTL - skip in short mode as it's time-dependent
	if !testing.Short() {
		// Wait for cache expiration with ticker polling
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()

		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if _, exists := cache.Get("key2"); !exists {
					return // Key expired as expected
				}
			case <-ctx.Done():
				t.Error("Test timed out waiting for key expiration")
				return
			}
		}
	}

	// Test stats
	stats := cache.GetStats()
	if stats["size"].(int) > 3 {
		t.Errorf("Cache size should not exceed 3, got %v", stats["size"])
	}
}

func TestConnectionPool(t *testing.T) {
	pool := NewConnectionPool(2, 30*time.Second, nil)

	// Test client creation
	client1 := pool.GetClient("example.com")
	if client1 == nil {
		t.Fatal("Failed to get client")
	}

	client2 := pool.GetClient("example.com")
	if client1 != client2 {
		t.Error("Should reuse existing client")
	}

	// Test different host
	client3 := pool.GetClient("test.com")
	if client3 == client1 {
		t.Error("Different hosts should get different clients")
	}

	// Test stats
	stats := pool.GetStats()
	if stats["pool_size"].(int) != 2 {
		t.Errorf("Expected pool size 2, got %v", stats["pool_size"])
	}
}

func TestLazyLoader(t *testing.T) {
	loader := NewLazyLoader(func(name string) (interface{}, error) {
		return "data-" + name, nil
	}, nil)

	// Test lazy loading
	data, err := loader.LoadComponent("test")
	if err != nil {
		t.Fatalf("Failed to load component: %v", err)
	}

	if data != "data-test" {
		t.Errorf("Expected 'data-test', got %v", data)
	}

	// Test caching
	data2, err := loader.LoadComponent("test")
	if err != nil {
		t.Fatalf("Failed to load cached component: %v", err)
	}

	if data2 != data {
		t.Error("Should return cached data")
	}

	// Test unloading
	loader.UnloadComponent("test")

	components := loader.GetLoadedComponents()
	if comp, exists := components["test"]; !exists || comp.(map[string]interface{})["is_loaded"].(bool) {
		t.Error("Component should be unloaded")
	}
}

func TestBatchProcessor(t *testing.T) {
	processed := make(chan []interface{}, 1)

	processor := NewBatchProcessor(3, 100*time.Millisecond, func(batchType string, items []interface{}) error {
		processed <- items
		return nil
	}, nil)

	// Test batch size trigger
	_ = processor.AddItem("test", 1)
	_ = processor.AddItem("test", 2)
	_ = processor.AddItem("test", 3) // Should trigger flush

	select {
	case items := <-processed:
		if len(items) != 3 {
			t.Errorf("Expected 3 items, got %d", len(items))
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("Batch should have been processed")
	}

	// Test timeout trigger
	_ = processor.AddItem("test", 4)
	_ = processor.AddItem("test", 5)

	select {
	case items := <-processed:
		if len(items) != 2 {
			t.Errorf("Expected 2 items, got %d", len(items))
		}
	case <-time.After(150 * time.Millisecond):
		t.Error("Batch should have been flushed by timeout")
	}
}

func TestResourceCleanup(t *testing.T) {
	cleanupCalled := make(chan bool, 1)

	cleanup := NewResourceCleanup(50*time.Millisecond, nil)
	cleanup.AddCleanupFunc(func() error {
		cleanupCalled <- true
		return nil
	})

	cleanup.Start()
	defer cleanup.Stop()

	// Wait for cleanup to be called
	select {
	case <-cleanupCalled:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Error("Cleanup function should have been called")
	}

	// Test manual cleanup
	err := cleanup.RunCleanup()
	if err != nil {
		t.Errorf("Manual cleanup failed: %v", err)
	}

	stats := cleanup.GetCleanupStats()
	if stats["cleanup_count"].(int64) < 1 {
		t.Error("Cleanup should have been called at least once")
	}
}

func TestPerformanceMonitor(t *testing.T) {
	monitor := NewPerformanceMonitor(50*time.Millisecond, nil)
	monitor.Start()
	defer monitor.Stop()

	// Test metric recording
	monitor.RecordMetric("test.metric", 42.0, MetricTypeGauge, "units", map[string]string{"tag": "value"})

	metric, err := monitor.GetMetric("test.metric")
	if err != nil {
		t.Fatalf("Failed to get metric: %v", err)
	}

	if metric.Value != 42.0 {
		t.Errorf("Expected value 42.0, got %v", metric.Value)
	}

	if metric.Unit != "units" {
		t.Errorf("Expected unit 'units', got %v", metric.Unit)
	}

	// Test timer metric
	timer := monitor.StartTimer("test.timer", nil)
	if !testing.Short() {
		// Simulate work with actual sleep to ensure measurable time
		time.Sleep(2 * time.Millisecond)
	}
	duration := timer.Stop()

	if !testing.Short() && duration < time.Millisecond {
		t.Errorf("Timer should measure at least 1ms, got %v", duration)
	}

	// Test counter metric
	counter := monitor.GetCounter("test.counter", nil)
	counter.Increment()
	counter.Add(5)

	if counter.Value() != 6 {
		t.Errorf("Expected counter value 6, got %d", counter.Value())
	}

	// Test gauge metric
	gauge := monitor.GetGauge("test.gauge", nil)
	gauge.Set(100.0)
	gauge.Add(50.0)

	if gauge.Value() != 150.0 {
		t.Errorf("Expected gauge value 150.0, got %v", gauge.Value())
	}
}

func TestBenchmarkSuite(t *testing.T) {
	suite := NewBenchmarkSuite(nil)

	// Add a simple benchmark
	suite.AddBenchmark(Benchmark{
		Name:        "TestBenchmark",
		Description: "A test benchmark",
		Setup: func() interface{} {
			return 0
		},
		Run: func(data interface{}) error {
			// Simulate some work with CPU-bound operation
			for i := 0; i < 10; i++ {
				_ = i * i
			}
			return nil
		},
		Iterations: 100,
	})

	results := suite.RunBenchmarks()
	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.Name != "TestBenchmark" {
		t.Errorf("Expected name 'TestBenchmark', got %v", result.Name)
	}

	if result.Iterations != 100 {
		t.Errorf("Expected 100 iterations, got %d", result.Iterations)
	}

	if result.OperationsPerSec <= 0 {
		t.Errorf("Expected positive ops/sec, got %v", result.OperationsPerSec)
	}

	// Test report generation
	report := suite.GenerateReport()
	if report == "" {
		t.Error("Report should not be empty")
	}
}

func TestPerformanceManager(t *testing.T) {
	config := DefaultPerformanceConfig()
	manager := NewPerformanceManager(config, nil)
	defer func() { _ = manager.Shutdown(5 * time.Second) }()

	// Test component access
	if manager.GetUIOptimizer() == nil {
		t.Error("UI optimizer should not be nil")
	}

	if manager.GetMemoryManager() == nil {
		t.Error("Memory manager should not be nil")
	}

	if manager.GetProcessPool() == nil {
		t.Error("Process pool should not be nil")
	}

	if manager.GetCache() == nil {
		t.Error("Cache should not be nil")
	}

	// Test overall stats
	stats := manager.GetOverallStats()
	if stats["memory"] == nil {
		t.Error("Memory stats should be present")
	}

	if stats["process_pool"] == nil {
		t.Error("Process pool stats should be present")
	}

	if stats["cache"] == nil {
		t.Error("Cache stats should be present")
	}
}

// Benchmark tests using Go's testing package
func BenchmarkUIOptimizerUpdates(b *testing.B) {
	optimizer := NewUIOptimizer(60, 10, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.AddUpdate(UIUpdate{
			Type:     "test",
			Data:     i,
			Priority: 1,
		})
	}
}

func BenchmarkBufferPoolReuse(b *testing.B) {
	manager := NewMemoryManager(1024, 1024*1024, nil)
	pool := manager.GetBufferPool("test", 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer := pool.GetBuffer()
		buffer = append(buffer, byte(i%256))
		pool.PutBuffer(buffer)
	}
}

func BenchmarkCacheOperations(b *testing.B) {
	cache := NewCache(1000, 5*time.Minute, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("key%d", i%100)
		if i%2 == 0 {
			cache.Set(key, i)
		} else {
			cache.Get(key)
		}
	}
}

func BenchmarkProcessPoolSubmission(b *testing.B) {
	pool := NewProcessPool(4, nil)
	defer func() { _ = pool.Shutdown(5 * time.Second) }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := make(chan error, 1)
		job := Job{
			ID:       fmt.Sprintf("job%d", i),
			Function: func() error { return nil },
			Result:   result,
		}

		_ = pool.Submit(job)
		<-result
	}
}
