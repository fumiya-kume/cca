package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRunner provides comprehensive test execution and reporting
type TestRunner struct {
	config    *TestConfig
	reporter  *TestReporter
	framework *TestFramework
	outputDir string
	ctx       context.Context
	cancel    context.CancelFunc
}

// TestConfig configures test execution
type TestConfig struct {
	ProjectRoot    string
	OutputDir      string
	CoverageTarget float64
	Timeout        time.Duration
	Parallel       bool
	Verbose        bool
	PackageFilters []string
	Tags           []string
	RunBenchmarks  bool
	RunE2E         bool
	RunIntegration bool
	RunPerformance bool
	GenerateReport bool
	FailFast       bool
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() *TestConfig {
	return &TestConfig{
		ProjectRoot:    ".",
		OutputDir:      "./test-results",
		CoverageTarget: 80.0,
		Timeout:        30 * time.Minute,
		Parallel:       true,
		Verbose:        false,
		PackageFilters: []string{"./..."},
		Tags:           []string{},
		RunBenchmarks:  false,
		RunE2E:         false,
		RunIntegration: true,
		RunPerformance: false,
		GenerateReport: true,
		FailFast:       false,
	}
}

// NewTestRunner creates a new test runner
func NewTestRunner(config *TestConfig) *TestRunner {
	if config == nil {
		config = DefaultTestConfig()
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0750); err != nil {
		panic(fmt.Sprintf("Failed to create output directory: %v", err))
	}

	reportFile, err := os.Create(filepath.Join(config.OutputDir, "test-report.txt"))
	if err != nil {
		panic(fmt.Sprintf("Failed to create report file: %v", err))
	}

	return &TestRunner{
		config:    config,
		reporter:  NewTestReporter(reportFile),
		framework: NewTestFramework(),
		outputDir: config.OutputDir,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// RunAllTests runs all test suites
func (tr *TestRunner) RunAllTests() error {
	fmt.Printf("🚀 Starting comprehensive test execution...\n")
	fmt.Printf("📁 Project: %s\n", tr.config.ProjectRoot)
	fmt.Printf("📊 Output: %s\n", tr.config.OutputDir)
	fmt.Printf("🎯 Coverage Target: %.1f%%\n", tr.config.CoverageTarget)
	fmt.Printf("⏱️  Timeout: %v\n", tr.config.Timeout)
	fmt.Println()

	startTime := time.Now()

	// Run unit tests
	if err := tr.runUnitTests(); err != nil {
		return fmt.Errorf("unit tests failed: %w", err)
	}

	// Run integration tests
	if tr.config.RunIntegration {
		if err := tr.runIntegrationTests(); err != nil {
			return fmt.Errorf("integration tests failed: %w", err)
		}
	}

	// Run benchmarks
	if tr.config.RunBenchmarks {
		if err := tr.runBenchmarks(); err != nil {
			return fmt.Errorf("benchmarks failed: %w", err)
		}
	}

	// Run performance tests
	if tr.config.RunPerformance {
		if err := tr.runPerformanceTests(); err != nil {
			return fmt.Errorf("performance tests failed: %w", err)
		}
	}

	// Run E2E tests
	if tr.config.RunE2E {
		if err := tr.runE2ETests(); err != nil {
			return fmt.Errorf("E2E tests failed: %w", err)
		}
	}

	duration := time.Since(startTime)

	// Generate coverage report
	if err := tr.generateCoverageReport(); err != nil {
		return fmt.Errorf("coverage report generation failed: %w", err)
	}

	// Generate final report
	if tr.config.GenerateReport {
		if err := tr.generateFinalReport(duration); err != nil {
			return fmt.Errorf("report generation failed: %w", err)
		}
	}

	fmt.Printf("\n✅ All tests completed successfully in %v\n", duration)
	return nil
}

// runUnitTests runs unit tests with coverage
func (tr *TestRunner) runUnitTests() error {
	fmt.Println("🧪 Running unit tests...")

	args := []string{"test"}

	// Add coverage flags
	coverageFile := filepath.Join(tr.outputDir, "coverage.out")
	args = append(args, "-coverprofile="+coverageFile)
	args = append(args, "-covermode=atomic")

	// Add verbosity
	if tr.config.Verbose {
		args = append(args, "-v")
	}

	// Add parallel execution
	if tr.config.Parallel {
		args = append(args, "-parallel", "4")
	}

	// Add timeout
	args = append(args, "-timeout", tr.config.Timeout.String())

	// Add fail fast
	if tr.config.FailFast {
		args = append(args, "-failfast")
	}

	// Add package filters
	args = append(args, tr.config.PackageFilters...)

	// Add build tags
	if len(tr.config.Tags) > 0 {
		args = append(args, "-tags", strings.Join(tr.config.Tags, ","))
	}

	cmd := exec.CommandContext(tr.ctx, "go", args...)
	cmd.Dir = tr.config.ProjectRoot

	output, err := cmd.CombinedOutput()

	// Save output
	outputFile := filepath.Join(tr.outputDir, "unit-tests.log")
	if writeErr := os.WriteFile(outputFile, output, 0600); writeErr != nil {
		fmt.Printf("Warning: Failed to save unit test output: %v\n", writeErr)
	}

	if err != nil {
		fmt.Printf("❌ Unit tests failed:\n%s\n", string(output))
		return err
	}

	fmt.Printf("✅ Unit tests passed\n")
	return nil
}

// runIntegrationTests runs integration tests
func (tr *TestRunner) runIntegrationTests() error {
	fmt.Println("🔗 Running integration tests...")
	
	// Look for integration test files
	integrationPattern := filepath.Join(tr.config.ProjectRoot, "test", "*integration*")
	integrationFiles, err := filepath.Glob(integrationPattern)
	if err != nil {
		return fmt.Errorf("failed to find integration tests: %w", err)
	}
	
	// Also check for *_integration_test.go files
	codeIntegrationPattern := filepath.Join(tr.config.ProjectRoot, "**", "*integration_test.go")
	codeIntegrationFiles, _ := filepath.Glob(codeIntegrationPattern) //nolint:errcheck // Additional patterns are optional
	integrationFiles = append(integrationFiles, codeIntegrationFiles...)
	
	if len(integrationFiles) == 0 {
		fmt.Println("ℹ️  No integration tests found, skipping...")
		return nil
	}
	
	// Build test command
	args := []string{"test", "-v"}
	
	// Add timeout
	args = append(args, "-timeout", tr.config.Timeout.String())
	
	// Add tags for integration tests
	args = append(args, "-tags", "integration")
	
	// Add parallel execution if enabled
	if tr.config.Parallel {
		args = append(args, "-parallel", "4")
	}
	
	// Add coverage if needed
	if tr.config.CoverageTarget > 0 {
		coverageFile := filepath.Join(tr.config.OutputDir, "integration-coverage.out")
		args = append(args, "-coverprofile", coverageFile)
	}
	
	// Run integration test packages or files
	if len(tr.config.PackageFilters) > 0 {
		args = append(args, tr.config.PackageFilters...)
	} else {
		args = append(args, "./test/...")
	}
	
	// Create and run command
	cmd := exec.Command("go", args...)
	cmd.Dir = tr.config.ProjectRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	fmt.Printf("🏃 Running: go %s\n", strings.Join(args, " "))
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("integration tests failed: %w", err)
	}
	
	fmt.Println("✅ Integration tests completed successfully")
	return nil
}

// runBenchmarks runs performance benchmarks
func (tr *TestRunner) runBenchmarks() error {
	fmt.Println("📊 Running performance benchmarks...")

	args := []string{"test", "-bench=.", "-benchmem", "-run=^$"}

	// Add timeout
	args = append(args, "-timeout", tr.config.Timeout.String())

	// Add package filters
	args = append(args, tr.config.PackageFilters...)

	cmd := exec.CommandContext(tr.ctx, "go", args...)
	cmd.Dir = tr.config.ProjectRoot

	output, err := cmd.CombinedOutput()

	// Save benchmark results
	benchmarkFile := filepath.Join(tr.outputDir, "benchmarks.log")
	if writeErr := os.WriteFile(benchmarkFile, output, 0600); writeErr != nil {
		fmt.Printf("Warning: Failed to save benchmark output: %v\n", writeErr)
	}

	if err != nil {
		fmt.Printf("❌ Benchmarks failed:\n%s\n", string(output))
		return err
	}

	fmt.Printf("✅ Benchmarks completed\n")
	return nil
}

// runPerformanceTests runs comprehensive performance tests
func (tr *TestRunner) runPerformanceTests() error {
	fmt.Println("⚡ Running performance tests...")
	
	// Look for performance test files
	perfPattern := filepath.Join(tr.config.ProjectRoot, "**", "*performance_test.go")
	perfFiles, _ := filepath.Glob(perfPattern) //nolint:errcheck // Performance test patterns are optional
	
	// Also check for dedicated performance test directories
	perfDirPattern := filepath.Join(tr.config.ProjectRoot, "test", "performance", "*_test.go")
	perfDirFiles, _ := filepath.Glob(perfDirPattern) //nolint:errcheck // Performance test patterns are optional
	perfFiles = append(perfFiles, perfDirFiles...)
	
	if len(perfFiles) == 0 {
		fmt.Println("ℹ️  No performance tests found, running benchmarks instead...")
		return tr.runBenchmarks()
	}
	
	// Build performance test command
	args := []string{"test", "-v"}
	
	// Add timeout
	args = append(args, "-timeout", tr.config.Timeout.String())
	
	// Add tags for performance tests
	args = append(args, "-tags", "performance")
	
	// Add benchmarking flags for performance measurement
	args = append(args, "-bench=.", "-benchmem")
	
	// Run specific performance test pattern
	args = append(args, "-run", "Performance")
	
	// Add CPU profiling for performance analysis
	cpuProfileFile := filepath.Join(tr.config.OutputDir, "performance-cpu.prof")
	args = append(args, "-cpuprofile", cpuProfileFile)
	
	// Add memory profiling
	memProfileFile := filepath.Join(tr.config.OutputDir, "performance-mem.prof")
	args = append(args, "-memprofile", memProfileFile)
	
	// Run test packages
	if len(tr.config.PackageFilters) > 0 {
		args = append(args, tr.config.PackageFilters...)
	} else {
		args = append(args, "./...")
	}
	
	// Create and run command
	cmd := exec.Command("go", args...)
	cmd.Dir = tr.config.ProjectRoot
	
	// Capture output for analysis
	output, err := cmd.CombinedOutput()
	
	// Save performance test results
	perfResultFile := filepath.Join(tr.config.OutputDir, "performance-results.log")
	if writeErr := os.WriteFile(perfResultFile, output, 0600); writeErr != nil {
		fmt.Printf("Warning: Failed to save performance results: %v\n", writeErr)
	}
	
	// Print results
	fmt.Printf("🏃 Running: go %s\n", strings.Join(args, " "))
	fmt.Printf("📊 Performance test output:\n%s\n", string(output))
	
	if err != nil {
		return fmt.Errorf("performance tests failed: %w", err)
	}
	
	// Analyze performance results
	if err := tr.analyzePerformanceResults(string(output)); err != nil {
		fmt.Printf("Warning: Performance analysis failed: %v\n", err)
	}
	
	fmt.Println("✅ Performance tests completed successfully")
	return nil
}

// runE2ETests runs end-to-end tests
func (tr *TestRunner) runE2ETests() error {
	fmt.Println("🎭 Running end-to-end tests...")
	
	// Look for E2E test files
	e2ePattern := filepath.Join(tr.config.ProjectRoot, "**", "*e2e_test.go")
	e2eFiles, _ := filepath.Glob(e2ePattern) //nolint:errcheck // E2E test patterns are optional
	
	// Also check for dedicated E2E test directories
	e2eDirPattern := filepath.Join(tr.config.ProjectRoot, "test", "e2e", "*_test.go")
	e2eDirFiles, _ := filepath.Glob(e2eDirPattern) //nolint:errcheck // E2E test patterns are optional
	e2eFiles = append(e2eFiles, e2eDirFiles...)
	
	// Check for integration examples that might be E2E tests
	integrationPattern := filepath.Join(tr.config.ProjectRoot, "**", "*integration*test.go")
	integrationFiles, _ := filepath.Glob(integrationPattern) //nolint:errcheck // Integration test patterns are optional
	e2eFiles = append(e2eFiles, integrationFiles...)
	
	if len(e2eFiles) == 0 {
		fmt.Println("ℹ️  No end-to-end tests found, skipping...")
		return nil
	}
	
	// Build E2E test command
	args := []string{"test", "-v"}
	
	// Add longer timeout for E2E tests
	e2eTimeout := tr.config.Timeout
	if e2eTimeout < 30*time.Minute {
		e2eTimeout = 30 * time.Minute // E2E tests typically need more time
	}
	args = append(args, "-timeout", e2eTimeout.String())
	
	// Add tags for E2E tests
	args = append(args, "-tags", "e2e")
	
	// Run E2E tests sequentially (no parallel)
	args = append(args, "-parallel", "1")
	
	// Add coverage if needed
	if tr.config.CoverageTarget > 0 {
		coverageFile := filepath.Join(tr.config.OutputDir, "e2e-coverage.out")
		args = append(args, "-coverprofile", coverageFile)
	}
	
	// Run E2E test packages
	if len(tr.config.PackageFilters) > 0 {
		args = append(args, tr.config.PackageFilters...)
	} else {
		args = append(args, "./test/e2e/...")
	}
	
	// Create and run command
	cmd := exec.Command("go", args...)
	cmd.Dir = tr.config.ProjectRoot
	
	// Set environment variables for E2E tests
	cmd.Env = append(os.Environ(),
		"E2E_TEST=true",
		"TEST_MODE=e2e",
	)
	
	// Capture output
	output, err := cmd.CombinedOutput()
	
	// Save E2E test results
	e2eResultFile := filepath.Join(tr.config.OutputDir, "e2e-results.log")
	if writeErr := os.WriteFile(e2eResultFile, output, 0600); writeErr != nil {
		fmt.Printf("Warning: Failed to save E2E results: %v\n", writeErr)
	}
	
	// Print results
	fmt.Printf("🏃 Running: go %s\n", strings.Join(args, " "))
	fmt.Printf("📊 E2E test output:\n%s\n", string(output))
	
	if err != nil {
		return fmt.Errorf("E2E tests failed: %w", err)
	}
	
	fmt.Println("✅ End-to-end tests completed successfully")
	return nil
}

// analyzePerformanceResults analyzes performance test output for issues
func (tr *TestRunner) analyzePerformanceResults(output string) error {
	lines := strings.Split(output, "\n")
	
	var warnings []string
	var benchmarks []string
	
	for _, line := range lines {
		// Look for benchmark results
		if strings.Contains(line, "Benchmark") && strings.Contains(line, "ns/op") {
			benchmarks = append(benchmarks, line)
			
			// Check for performance regressions (basic heuristics)
			if strings.Contains(line, "FAIL") {
				warnings = append(warnings, "❌ Failed benchmark: "+line)
			}
			
			// Check for very slow operations (> 1ms)
			if strings.Contains(line, "ms/op") {
				warnings = append(warnings, "⚠️  Slow operation detected: "+line)
			}
		}
		
		// Look for memory allocation issues
		if strings.Contains(line, "allocs/op") {
			// Basic check for high allocation count
			if strings.Contains(line, " B/op") && 
			   (strings.Contains(line, " MB/op") || strings.Contains(line, " KB/op")) {
				warnings = append(warnings, "🧠 High memory usage: "+line)
			}
		}
	}
	
	// Generate performance summary
	if len(benchmarks) > 0 {
		fmt.Printf("\n📊 Performance Summary:\n")
		for _, benchmark := range benchmarks {
			fmt.Printf("  %s\n", benchmark)
		}
	}
	
	// Report warnings
	if len(warnings) > 0 {
		fmt.Printf("\n⚠️  Performance Warnings:\n")
		for _, warning := range warnings {
			fmt.Printf("  %s\n", warning)
		}
	}
	
	return nil
}

// generateCoverageReport generates code coverage reports
func (tr *TestRunner) generateCoverageReport() error {
	fmt.Println("📈 Generating coverage report...")

	coverageFile := filepath.Join(tr.outputDir, "coverage.out")
	if _, err := os.Stat(coverageFile); os.IsNotExist(err) {
		fmt.Println("⚠️  No coverage file found, skipping coverage report")
		return nil
	}

	// Generate HTML coverage report
	htmlFile := filepath.Join(tr.outputDir, "coverage.html")
	// #nosec G204 - using go tool with controlled file paths
	cmd := exec.CommandContext(tr.ctx, "go", "tool", "cover", "-html="+coverageFile, "-o", htmlFile)
	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  Failed to generate HTML coverage report: %v\n", err)
	}

	// Generate coverage summary
	// #nosec G204 - using go tool with controlled file paths
	cmd = exec.CommandContext(tr.ctx, "go", "tool", "cover", "-func="+coverageFile)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to generate coverage summary: %w", err)
	}

	// Save coverage summary
	summaryFile := filepath.Join(tr.outputDir, "coverage-summary.txt")
	if err := os.WriteFile(summaryFile, output, 0600); err != nil {
		return fmt.Errorf("failed to save coverage summary: %w", err)
	}

	// Extract overall coverage percentage
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "total:") {
			fmt.Printf("📊 %s\n", strings.TrimSpace(line))

			// Check if coverage meets target
			if strings.Contains(line, "%") {
				coverageStr := strings.Split(line, "%")[0]
				coverageStr = strings.Split(coverageStr, "\t")[len(strings.Split(coverageStr, "\t"))-1]

				var coverage float64
				if _, err := fmt.Sscanf(coverageStr, "%f", &coverage); err == nil {
					if coverage < tr.config.CoverageTarget {
						return fmt.Errorf("coverage %.1f%% is below target %.1f%%", coverage, tr.config.CoverageTarget)
					}
				}
			}
			break
		}
	}

	fmt.Printf("✅ Coverage report generated\n")
	return nil
}

// generateFinalReport generates a comprehensive final report
func (tr *TestRunner) generateFinalReport(duration time.Duration) error {
	fmt.Println("📋 Generating final test report...")

	reportFile := filepath.Join(tr.outputDir, "final-report.md")
	// #nosec G304 - reportFile is from validated config output directory
	file, err := os.Create(reportFile)
	if err != nil {
		return fmt.Errorf("failed to create final report: %w", err)
	}
	defer func() { _ = file.Close() }() //nolint:errcheck // File close in defer is best effort

	// Write report header
	_, _ = fmt.Fprintf(file, "# ccAgents Test Report\n\n") //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(file, "**Generated**: %s\n", time.Now().Format("2006-01-02 15:04:05")) //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(file, "**Duration**: %v\n", duration) //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(file, "**Project**: %s\n\n", tr.config.ProjectRoot) //nolint:errcheck // Report output errors are not critical

	// Test execution summary
	_, _ = fmt.Fprintf(file, "## Test Execution Summary\n\n") //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(file, "| Test Type | Status | Duration |\n") //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(file, "|-----------|--------|----------|\n") //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(file, "| Unit Tests | ✅ | - |\n") //nolint:errcheck // Report output errors are not critical

	if tr.config.RunIntegration {
		_, _ = fmt.Fprintf(file, "| Integration Tests | ✅ | - |\n") //nolint:errcheck // Report output errors are not critical
	}

	if tr.config.RunBenchmarks {
		_, _ = fmt.Fprintf(file, "| Benchmarks | ✅ | - |\n") //nolint:errcheck // Report output errors are not critical
	}

	if tr.config.RunPerformance {
		_, _ = fmt.Fprintf(file, "| Performance Tests | ✅ | - |\n") //nolint:errcheck // Report output errors are not critical
	}

	if tr.config.RunE2E {
		_, _ = fmt.Fprintf(file, "| E2E Tests | ✅ | - |\n") //nolint:errcheck // Report output errors are not critical
	}

	_, _ = fmt.Fprintf(file, "\n") //nolint:errcheck // Report output errors are not critical

	// Coverage information
	_, _ = fmt.Fprintf(file, "## Code Coverage\n\n") //nolint:errcheck // Report output errors are not critical
	if coverageData, err := os.ReadFile(filepath.Join(tr.outputDir, "coverage-summary.txt")); err == nil {
		_, _ = fmt.Fprintf(file, "```\n%s```\n\n", string(coverageData)) //nolint:errcheck // Report output errors are not critical
	}

	// Performance results
	if tr.config.RunPerformance {
		_, _ = fmt.Fprintf(file, "## Performance Results\n\n") //nolint:errcheck // Report output errors are not critical
		if perfData, err := os.ReadFile(filepath.Join(tr.outputDir, "performance-results.json")); err == nil {
			_, _ = fmt.Fprintf(file, "```json\n%s\n```\n\n", string(perfData)) //nolint:errcheck // Report output errors are not critical
		}
	}

	// E2E results
	if tr.config.RunE2E {
		_, _ = fmt.Fprintf(file, "## End-to-End Test Results\n\n") //nolint:errcheck // Report output errors are not critical
		if e2eData, err := os.ReadFile(filepath.Join(tr.outputDir, "e2e-results.json")); err == nil {
			_, _ = fmt.Fprintf(file, "```json\n%s\n```\n\n", string(e2eData)) //nolint:errcheck // Report output errors are not critical
		}
	}

	// Test artifacts
	_, _ = fmt.Fprintf(file, "## Test Artifacts\n\n") //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(file, "- [Unit Test Output](unit-tests.log)\n") //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(file, "- [Coverage Report](coverage.html)\n") //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(file, "- [Coverage Summary](coverage-summary.txt)\n") //nolint:errcheck // Report output errors are not critical

	if tr.config.RunBenchmarks {
		_, _ = fmt.Fprintf(file, "- [Benchmark Results](benchmarks.log)\n") //nolint:errcheck // Report output errors are not critical
	}

	if tr.config.RunPerformance {
		_, _ = fmt.Fprintf(file, "- [Performance Results](performance-results.json)\n") //nolint:errcheck // Report output errors are not critical
	}

	if tr.config.RunE2E {
		_, _ = fmt.Fprintf(file, "- [E2E Results](e2e-results.json)\n") //nolint:errcheck // Report output errors are not critical
	}

	_, _ = fmt.Fprintf(file, "\n") //nolint:errcheck // Report output errors are not critical

	// Configuration
	_, _ = fmt.Fprintf(file, "## Test Configuration\n\n") //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(file, "```json\n") //nolint:errcheck // Report output errors are not critical
	configData, err := json.MarshalIndent(tr.config, "", "  ")
	if err != nil {
		// If config marshaling fails, use default placeholder
		configData = []byte("Error: could not serialize configuration")
	}
	_, _ = fmt.Fprintf(file, "%s\n", string(configData)) //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(file, "```\n") //nolint:errcheck // Report output errors are not critical

	fmt.Printf("✅ Final report generated: %s\n", reportFile)
	return nil
}

// Cleanup cleans up test runner resources
func (tr *TestRunner) Cleanup() {
	tr.cancel()
	tr.framework.Cleanup(&testing.T{})
}

// TestSuiteRunner provides a high-level interface for running test suites
type TestSuiteRunner struct {
	runner *TestRunner
}

// NewTestSuiteRunner creates a new test suite runner
func NewTestSuiteRunner(config *TestConfig) *TestSuiteRunner {
	return &TestSuiteRunner{
		runner: NewTestRunner(config),
	}
}

// RunUnitTestsOnly runs only unit tests
func (tsr *TestSuiteRunner) RunUnitTestsOnly() error {
	tsr.runner.config.RunIntegration = false
	tsr.runner.config.RunBenchmarks = false
	tsr.runner.config.RunPerformance = false
	tsr.runner.config.RunE2E = false

	return tsr.runner.RunAllTests()
}

// RunIntegrationTestsOnly runs only integration tests
func (tsr *TestSuiteRunner) RunIntegrationTestsOnly() error {
	tsr.runner.config.RunIntegration = true
	tsr.runner.config.RunBenchmarks = false
	tsr.runner.config.RunPerformance = false
	tsr.runner.config.RunE2E = false

	return tsr.runner.runIntegrationTests()
}

// RunFullTestSuite runs the complete test suite
func (tsr *TestSuiteRunner) RunFullTestSuite() error {
	tsr.runner.config.RunIntegration = true
	tsr.runner.config.RunBenchmarks = true
	tsr.runner.config.RunPerformance = true
	tsr.runner.config.RunE2E = true

	return tsr.runner.RunAllTests()
}

// GetCoverageReport returns the coverage report path
func (tsr *TestSuiteRunner) GetCoverageReport() string {
	return filepath.Join(tsr.runner.outputDir, "coverage.html")
}

// GetFinalReport returns the final report path
func (tsr *TestSuiteRunner) GetFinalReport() string {
	return filepath.Join(tsr.runner.outputDir, "final-report.md")
}

// Cleanup cleans up test suite runner resources
func (tsr *TestSuiteRunner) Cleanup() {
	tsr.runner.Cleanup()
}
