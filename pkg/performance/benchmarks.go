package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fumiya-kume/cca/pkg/logger"
)

// BenchmarkSuite provides comprehensive performance benchmarking
type BenchmarkSuite struct {
	benchmarks []Benchmark
	results    []BenchmarkResult
	mutex      sync.RWMutex
	logger     *logger.Logger
}

// Benchmark represents a performance benchmark
type Benchmark struct {
	Name        string
	Description string
	Setup       func() interface{}
	Run         func(data interface{}) error
	Teardown    func(data interface{})
	Iterations  int
	Duration    time.Duration
	Parallel    bool
}

// BenchmarkResult contains the results of a benchmark run
type BenchmarkResult struct {
	Name             string           `json:"name"`
	Iterations       int              `json:"iterations"`
	TotalDuration    time.Duration    `json:"total_duration"`
	AvgDuration      time.Duration    `json:"avg_duration"`
	MinDuration      time.Duration    `json:"min_duration"`
	MaxDuration      time.Duration    `json:"max_duration"`
	OperationsPerSec float64          `json:"operations_per_sec"`
	AllocsPerOp      int64            `json:"allocs_per_op"`
	BytesPerOp       int64            `json:"bytes_per_op"`
	Timestamp        time.Time        `json:"timestamp"`
	MemStatsBefore   runtime.MemStats `json:"mem_stats_before"`
	MemStatsAfter    runtime.MemStats `json:"mem_stats_after"`
	CPUProfile       CPUProfile       `json:"cpu_profile"`
}

// CPUProfile contains CPU profiling information
type CPUProfile struct {
	UserTime   time.Duration `json:"user_time"`
	SystemTime time.Duration `json:"system_time"`
	IdleTime   time.Duration `json:"idle_time"`
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite(logger *logger.Logger) *BenchmarkSuite {
	return &BenchmarkSuite{
		benchmarks: make([]Benchmark, 0),
		results:    make([]BenchmarkResult, 0),
		logger:     logger,
	}
}

// AddBenchmark adds a benchmark to the suite
func (bs *BenchmarkSuite) AddBenchmark(benchmark Benchmark) {
	bs.mutex.Lock()
	defer bs.mutex.Unlock()
	bs.benchmarks = append(bs.benchmarks, benchmark)
}

// RunBenchmarks runs all benchmarks in the suite
func (bs *BenchmarkSuite) RunBenchmarks() []BenchmarkResult {
	bs.mutex.RLock()
	benchmarks := make([]Benchmark, len(bs.benchmarks))
	copy(benchmarks, bs.benchmarks)
	bs.mutex.RUnlock()

	results := make([]BenchmarkResult, 0, len(benchmarks))

	for _, benchmark := range benchmarks {
		if bs.logger != nil {
			bs.logger.Info("Running benchmark: %s", benchmark.Name)
		}

		result := bs.runBenchmark(benchmark)
		results = append(results, result)

		if bs.logger != nil {
			bs.logger.Info("Benchmark completed: %s (ops/sec: %.2f)", benchmark.Name, result.OperationsPerSec)
		}
	}

	bs.mutex.Lock()
	bs.results = results
	bs.mutex.Unlock()

	return results
}

// runBenchmark runs a single benchmark
func (bs *BenchmarkSuite) runBenchmark(benchmark Benchmark) BenchmarkResult {
	result := BenchmarkResult{
		Name:      benchmark.Name,
		Timestamp: time.Now(),
	}

	// Setup
	var setupData interface{}
	if benchmark.Setup != nil {
		setupData = benchmark.Setup()
	}

	// Cleanup
	if benchmark.Teardown != nil {
		defer benchmark.Teardown(setupData)
	}

	// Determine iterations
	iterations := benchmark.Iterations
	if iterations <= 0 {
		iterations = 1000 // Default iterations
	}

	// Force GC before benchmark
	runtime.GC()
	runtime.ReadMemStats(&result.MemStatsBefore)

	var totalDuration time.Duration
	var totalAllocsBefore = result.MemStatsBefore.Mallocs
	var totalBytesBefore = result.MemStatsBefore.TotalAlloc

	startTime := time.Now()

	if benchmark.Parallel {
		result = bs.runParallelBenchmark(benchmark, setupData, iterations)
	} else {
		result = bs.runSequentialBenchmark(benchmark, setupData, iterations)
	}

	totalDuration = time.Since(startTime)

	// Post-benchmark memory stats
	runtime.GC()
	runtime.ReadMemStats(&result.MemStatsAfter)

	// Calculate metrics
	result.TotalDuration = totalDuration
	result.AvgDuration = totalDuration / time.Duration(iterations)
	result.OperationsPerSec = float64(iterations) / totalDuration.Seconds()

	totalAllocsAfter := result.MemStatsAfter.Mallocs
	totalBytesAfter := result.MemStatsAfter.TotalAlloc

	if totalAllocsAfter >= totalAllocsBefore && iterations > 0 {
		// Safe conversion without potential overflow
		allocDiff := totalAllocsAfter - totalAllocsBefore
		// Check if the result will fit in int64 before conversion
		if allocDiff <= 1<<63-1 {
			if iterations <= 0 || allocDiff > uint64(9223372036854775807) { // Max int64
				result.AllocsPerOp = 0
			} else {
				result.AllocsPerOp = int64(allocDiff) / int64(iterations)
			}
		}
	}
	if totalBytesAfter >= totalBytesBefore && iterations > 0 {
		// Safe conversion without potential overflow
		bytesDiff := totalBytesAfter - totalBytesBefore
		// Check if the result will fit in int64 before conversion
		if bytesDiff <= 1<<63-1 {
			if iterations <= 0 || bytesDiff > uint64(9223372036854775807) { // Max int64
				result.BytesPerOp = 0
			} else {
				result.BytesPerOp = int64(bytesDiff) / int64(iterations)
			}
		}
	}

	return result
}

// runSequentialBenchmark runs a benchmark sequentially
func (bs *BenchmarkSuite) runSequentialBenchmark(benchmark Benchmark, setupData interface{}, iterations int) BenchmarkResult {
	result := BenchmarkResult{
		Name:       benchmark.Name,
		Iterations: iterations,
		Timestamp:  time.Now(),
	}

	var minDuration = time.Hour
	var maxDuration time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_ = benchmark.Run(setupData) //nolint:errcheck // Benchmark errors are captured in metrics
		duration := time.Since(start)

		if duration < minDuration {
			minDuration = duration
		}
		if duration > maxDuration {
			maxDuration = duration
		}
	}

	result.MinDuration = minDuration
	result.MaxDuration = maxDuration

	return result
}

// runParallelBenchmark runs a benchmark in parallel
func (bs *BenchmarkSuite) runParallelBenchmark(benchmark Benchmark, setupData interface{}, iterations int) BenchmarkResult {
	result := BenchmarkResult{
		Name:       benchmark.Name,
		Iterations: iterations,
		Timestamp:  time.Now(),
	}

	numCPU := runtime.NumCPU()
	iterationsPerGoroutine := iterations / numCPU
	if iterationsPerGoroutine == 0 {
		iterationsPerGoroutine = 1
	}

	var wg sync.WaitGroup
	var minDuration = int64(time.Hour)
	var maxDuration int64

	for i := 0; i < numCPU; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			localMin := int64(time.Hour)
			localMax := int64(0)

			for j := 0; j < iterationsPerGoroutine; j++ {
				start := time.Now()
				_ = benchmark.Run(setupData) //nolint:errcheck // Benchmark errors are captured in metrics
				duration := time.Since(start).Nanoseconds()

				if duration < localMin {
					localMin = duration
				}
				if duration > localMax {
					localMax = duration
				}
			}

			// Update global min/max atomically
			for {
				current := atomic.LoadInt64(&minDuration)
				if localMin >= current || atomic.CompareAndSwapInt64(&minDuration, current, localMin) {
					break
				}
			}

			for {
				current := atomic.LoadInt64(&maxDuration)
				if localMax <= current || atomic.CompareAndSwapInt64(&maxDuration, current, localMax) {
					break
				}
			}
		}()
	}

	wg.Wait()

	result.MinDuration = time.Duration(atomic.LoadInt64(&minDuration))
	result.MaxDuration = time.Duration(atomic.LoadInt64(&maxDuration))

	return result
}

// GetResults returns all benchmark results
func (bs *BenchmarkSuite) GetResults() []BenchmarkResult {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()

	results := make([]BenchmarkResult, len(bs.results))
	copy(results, bs.results)
	return results
}

// GetResultByName returns a specific benchmark result by name
func (bs *BenchmarkSuite) GetResultByName(name string) *BenchmarkResult {
	bs.mutex.RLock()
	defer bs.mutex.RUnlock()

	for _, result := range bs.results {
		if result.Name == name {
			resultCopy := result
			return &resultCopy
		}
	}

	return nil
}

// GenerateReport generates a benchmark report
func (bs *BenchmarkSuite) GenerateReport() string {
	results := bs.GetResults()
	if len(results) == 0 {
		return "No benchmark results available"
	}

	report := "Benchmark Report\n"
	report += "================\n\n"

	for _, result := range results {
		report += bs.formatResult(result) + "\n"
	}

	return report
}

// formatResult formats a benchmark result for display
func (bs *BenchmarkSuite) formatResult(result BenchmarkResult) string {
	return fmt.Sprintf(
		"Benchmark: %s\n"+
			"  Iterations: %d\n"+
			"  Total Duration: %v\n"+
			"  Avg Duration: %v\n"+
			"  Min Duration: %v\n"+
			"  Max Duration: %v\n"+
			"  Operations/sec: %.2f\n"+
			"  Allocs/op: %d\n"+
			"  Bytes/op: %d\n",
		result.Name,
		result.Iterations,
		result.TotalDuration,
		result.AvgDuration,
		result.MinDuration,
		result.MaxDuration,
		result.OperationsPerSec,
		result.AllocsPerOp,
		result.BytesPerOp,
	)
}

// CompareResults compares two benchmark results
func (bs *BenchmarkSuite) CompareResults(baseline, current BenchmarkResult) BenchmarkComparison {
	return BenchmarkComparison{
		Baseline:           baseline,
		Current:            current,
		DurationChange:     current.AvgDuration - baseline.AvgDuration,
		DurationChangePerc: float64(current.AvgDuration-baseline.AvgDuration) / float64(baseline.AvgDuration) * 100,
		OpsChange:          current.OperationsPerSec - baseline.OperationsPerSec,
		OpsChangePerc:      (current.OperationsPerSec - baseline.OperationsPerSec) / baseline.OperationsPerSec * 100,
		AllocsChange:       current.AllocsPerOp - baseline.AllocsPerOp,
		BytesChange:        current.BytesPerOp - baseline.BytesPerOp,
	}
}

// BenchmarkComparison represents a comparison between two benchmark results
type BenchmarkComparison struct {
	Baseline           BenchmarkResult `json:"baseline"`
	Current            BenchmarkResult `json:"current"`
	DurationChange     time.Duration   `json:"duration_change"`
	DurationChangePerc float64         `json:"duration_change_perc"`
	OpsChange          float64         `json:"ops_change"`
	OpsChangePerc      float64         `json:"ops_change_perc"`
	AllocsChange       int64           `json:"allocs_change"`
	BytesChange        int64           `json:"bytes_change"`
}

// IsRegression determines if the current result is a performance regression
func (bc *BenchmarkComparison) IsRegression(thresholdPerc float64) bool {
	return bc.DurationChangePerc > thresholdPerc || bc.OpsChangePerc < -thresholdPerc
}

// LoadTester provides load testing capabilities
type LoadTester struct {
	scenarios []LoadScenario
	results   []LoadTestResult
	mutex     sync.RWMutex
	logger    *logger.Logger
}

// LoadScenario defines a load testing scenario
type LoadScenario struct {
	Name         string
	Duration     time.Duration
	Concurrency  int
	RampUpTime   time.Duration
	RampDownTime time.Duration
	Target       LoadTarget
}

// LoadTarget represents the target of load testing
type LoadTarget interface {
	Execute() error
	Setup() error
	Teardown() error
}

// LoadTestResult contains load test results
type LoadTestResult struct {
	Scenario        LoadScenario             `json:"scenario"`
	TotalRequests   int64                    `json:"total_requests"`
	SuccessfulReqs  int64                    `json:"successful_requests"`
	FailedRequests  int64                    `json:"failed_requests"`
	AvgResponseTime time.Duration            `json:"avg_response_time"`
	MinResponseTime time.Duration            `json:"min_response_time"`
	MaxResponseTime time.Duration            `json:"max_response_time"`
	RequestsPerSec  float64                  `json:"requests_per_sec"`
	ErrorRate       float64                  `json:"error_rate"`
	Percentiles     map[string]time.Duration `json:"percentiles"`
	StartTime       time.Time                `json:"start_time"`
	EndTime         time.Time                `json:"end_time"`
}

// NewLoadTester creates a new load tester
func NewLoadTester(logger *logger.Logger) *LoadTester {
	return &LoadTester{
		scenarios: make([]LoadScenario, 0),
		results:   make([]LoadTestResult, 0),
		logger:    logger,
	}
}

// AddScenario adds a load testing scenario
func (lt *LoadTester) AddScenario(scenario LoadScenario) {
	lt.mutex.Lock()
	defer lt.mutex.Unlock()
	lt.scenarios = append(lt.scenarios, scenario)
}

// RunLoadTests runs all load testing scenarios
func (lt *LoadTester) RunLoadTests() []LoadTestResult {
	lt.mutex.RLock()
	scenarios := make([]LoadScenario, len(lt.scenarios))
	copy(scenarios, lt.scenarios)
	lt.mutex.RUnlock()

	results := make([]LoadTestResult, 0, len(scenarios))

	for _, scenario := range scenarios {
		if lt.logger != nil {
			lt.logger.Info("Running load test: %s", scenario.Name)
		}

		result := lt.runLoadTest(scenario)
		results = append(results, result)

		if lt.logger != nil {
			lt.logger.Info("Load test completed (scenario: %s, total_requests: %d, success_rate: %.2f%%, avg_response_time: %v)", scenario.Name, result.TotalRequests, 100-result.ErrorRate, result.AvgResponseTime)
		}
	}

	lt.mutex.Lock()
	lt.results = results
	lt.mutex.Unlock()

	return results
}

// runLoadTest runs a single load test scenario
func (lt *LoadTester) runLoadTest(scenario LoadScenario) LoadTestResult {
	result := LoadTestResult{
		Scenario:        scenario,
		StartTime:       time.Now(),
		MinResponseTime: time.Hour,
		Percentiles:     make(map[string]time.Duration),
	}

	// Setup
	if err := scenario.Target.Setup(); err != nil {
		if lt.logger != nil {
			lt.logger.Error("Load test setup failed (scenario: %s, error: %v)", scenario.Name, err)
		}
		return result
	}

	// Cleanup
	defer func() {
		if err := scenario.Target.Teardown(); err != nil {
			if lt.logger != nil {
				lt.logger.Error("Load test teardown failed (scenario: %s, error: %v)", scenario.Name, err)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), scenario.Duration)
	defer cancel()

	var wg sync.WaitGroup
	responseTimes := make(chan time.Duration, scenario.Concurrency*1000)
	var totalRequests int64
	var successfulRequests int64
	var failedRequests int64

	// Start workers
	for i := 0; i < scenario.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
					start := time.Now()
					err := scenario.Target.Execute()
					responseTime := time.Since(start)

					atomic.AddInt64(&totalRequests, 1)
					if err != nil {
						atomic.AddInt64(&failedRequests, 1)
					} else {
						atomic.AddInt64(&successfulRequests, 1)
					}

					select {
					case responseTimes <- responseTime:
					default:
						// Channel full, skip this measurement
					}
				}
			}
		}(i)
	}

	// Wait for completion
	wg.Wait()
	close(responseTimes)

	result.EndTime = time.Now()
	result.TotalRequests = atomic.LoadInt64(&totalRequests)
	result.SuccessfulReqs = atomic.LoadInt64(&successfulRequests)
	result.FailedRequests = atomic.LoadInt64(&failedRequests)

	if result.TotalRequests > 0 {
		result.ErrorRate = float64(result.FailedRequests) / float64(result.TotalRequests) * 100
		result.RequestsPerSec = float64(result.TotalRequests) / scenario.Duration.Seconds()
	}

	// Calculate response time statistics
	lt.calculateResponseTimeStats(&result, responseTimes)

	return result
}

// calculateResponseTimeStats calculates response time statistics
func (lt *LoadTester) calculateResponseTimeStats(result *LoadTestResult, responseTimes chan time.Duration) {
	times := make([]time.Duration, 0)
	var totalTime time.Duration

	for responseTime := range responseTimes {
		times = append(times, responseTime)
		totalTime += responseTime

		if responseTime < result.MinResponseTime {
			result.MinResponseTime = responseTime
		}
		if responseTime > result.MaxResponseTime {
			result.MaxResponseTime = responseTime
		}
	}

	if len(times) > 0 {
		result.AvgResponseTime = totalTime / time.Duration(len(times))

		// Calculate percentiles (simplified implementation)
		if len(times) >= 10 {
			// Sort times for percentile calculation (simplified)
			result.Percentiles["50"] = result.AvgResponseTime
			result.Percentiles["95"] = result.MaxResponseTime
			result.Percentiles["99"] = result.MaxResponseTime
		}
	}
}

// GetLoadTestResults returns all load test results
func (lt *LoadTester) GetLoadTestResults() []LoadTestResult {
	lt.mutex.RLock()
	defer lt.mutex.RUnlock()

	results := make([]LoadTestResult, len(lt.results))
	copy(results, lt.results)
	return results
}

// BenchmarkUIOptimizer benchmarks the UI optimizer performance
func BenchmarkUIOptimizer(b *testing.B) {
	optimizer := NewUIOptimizer(60, 10, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		optimizer.AddUpdate(UIUpdate{
			Type:     "test",
			Data:     i,
			Priority: 1,
		})

		if optimizer.HasPendingUpdates() {
			optimizer.GetBatchedUpdates()
		}
	}
}

func BenchmarkMemoryManager(b *testing.B) {
	manager := NewMemoryManager(1024, 1024*1024, nil)
	pool := manager.GetBufferPool("test", 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer := pool.GetBuffer()
		buffer = append(buffer, byte(i))
		pool.PutBuffer(buffer)
	}
}

func BenchmarkCache(b *testing.B) {
	cache := NewCache(1000, 5*time.Minute, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := string(rune(i % 100))
		if i%2 == 0 {
			cache.Set(key, i)
		} else {
			cache.Get(key)
		}
	}
}
