// Package testing provides comprehensive testing utilities and frameworks for ccAgents
package testing

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/fumiya-kume/cca/pkg/errors"
)

// TestFramework provides comprehensive testing utilities
type TestFramework struct {
	tempDirs []string
	cleanup  []func() error
	mutex    sync.Mutex
}

// NewTestFramework creates a new test framework instance
func NewTestFramework() *TestFramework {
	return &TestFramework{
		tempDirs: make([]string, 0),
		cleanup:  make([]func() error, 0),
	}
}

// CreateTempDir creates a temporary directory for testing
func (tf *TestFramework) CreateTempDir(t *testing.T, prefix string) string {
	tf.mutex.Lock()
	defer tf.mutex.Unlock()

	tempDir, err := os.MkdirTemp("", prefix)
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	tf.tempDirs = append(tf.tempDirs, tempDir)
	return tempDir
}

// CreateTempFile creates a temporary file with content
func (tf *TestFramework) CreateTempFile(t *testing.T, dir, filename, content string) string {
	filePath := filepath.Join(dir, filename)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
		t.Fatalf("Failed to create directory for temp file: %v", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	return filePath
}

// AddCleanup adds a cleanup function to be called during cleanup
func (tf *TestFramework) AddCleanup(cleanup func() error) {
	tf.mutex.Lock()
	defer tf.mutex.Unlock()
	tf.cleanup = append(tf.cleanup, cleanup)
}

// Cleanup cleans up all temporary resources
func (tf *TestFramework) Cleanup(t *testing.T) {
	tf.mutex.Lock()
	defer tf.mutex.Unlock()

	// Run custom cleanup functions
	for _, cleanup := range tf.cleanup {
		if err := cleanup(); err != nil {
			t.Logf("Cleanup error: %v", err)
		}
	}

	// Remove temporary directories
	for _, dir := range tf.tempDirs {
		if err := os.RemoveAll(dir); err != nil {
			t.Logf("Failed to remove temp dir %s: %v", dir, err)
		}
	}

	tf.tempDirs = nil
	tf.cleanup = nil
}

// MockGitRepository creates a mock git repository for testing
func (tf *TestFramework) MockGitRepository(t *testing.T, name string) string {
	repoDir := tf.CreateTempDir(t, fmt.Sprintf("git-repo-%s-", name))

	// Create basic git structure
	gitDir := filepath.Join(repoDir, ".git")
	if err := os.MkdirAll(gitDir, 0750); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	// Create basic files
	tf.CreateTempFile(t, repoDir, "README.md", "# Test Repository\n\nThis is a test repository.")
	tf.CreateTempFile(t, repoDir, ".gitignore", "*.log\n*.tmp\n")
	tf.CreateTempFile(t, gitDir, "config", "[core]\n\trepositoryformatversion = 0\n")

	return repoDir
}

// TestCase represents a test case with setup and validation
type TestCase struct {
	Name        string
	Setup       func(t *testing.T, tf *TestFramework) interface{}
	Execute     func(t *testing.T, tf *TestFramework, setupData interface{}) interface{}
	Validate    func(t *testing.T, tf *TestFramework, result interface{}) error
	Cleanup     func(t *testing.T, tf *TestFramework, setupData interface{})
	ShouldFail  bool
	ExpectedErr string
}

// RunTestCases runs a suite of test cases
func (tf *TestFramework) RunTestCases(t *testing.T, testCases []TestCase) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			defer tf.Cleanup(t)

			// Setup
			var setupData interface{}
			if tc.Setup != nil {
				setupData = tc.Setup(t, tf)
			}

			// Custom cleanup
			if tc.Cleanup != nil {
				defer func() {
					tc.Cleanup(t, tf, setupData)
				}()
			}

			// Execute
			var result interface{}
			var executeErr error

			func() {
				defer func() {
					if r := recover(); r != nil {
						executeErr = fmt.Errorf("panic during execution: %v", r)
					}
				}()

				if tc.Execute != nil {
					result = tc.Execute(t, tf, setupData)
				}
			}()

			// Check for expected failures
			if tc.ShouldFail {
				if executeErr == nil {
					t.Errorf("Expected test case to fail, but it succeeded")
					return
				}
				if tc.ExpectedErr != "" && !strings.Contains(executeErr.Error(), tc.ExpectedErr) {
					t.Errorf("Expected error containing '%s', got: %v", tc.ExpectedErr, executeErr)
					return
				}
				return // Test passed (it failed as expected)
			}

			// Check for unexpected failures
			if executeErr != nil {
				t.Errorf("Test case failed unexpectedly: %v", executeErr)
				return
			}

			// Validate results
			if tc.Validate != nil {
				if err := tc.Validate(t, tf, result); err != nil {
					t.Errorf("Validation failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkFramework provides performance testing utilities
type BenchmarkFramework struct {
	framework *TestFramework
}

// NewBenchmarkFramework creates a new benchmark framework
func NewBenchmarkFramework() *BenchmarkFramework {
	return &BenchmarkFramework{
		framework: NewTestFramework(),
	}
}

// BenchmarkFunc represents a function to benchmark
type BenchmarkFunc func(b *testing.B, bf *BenchmarkFramework) error

// RunBenchmark runs a benchmark with setup and cleanup
func (bf *BenchmarkFramework) RunBenchmark(b *testing.B, setup func(*testing.B, *BenchmarkFramework), fn BenchmarkFunc) {
	if setup != nil {
		setup(b, bf)
	}

	defer func() {
		// Convert testing.B to testing.T for cleanup
		t := &testing.T{}
		bf.framework.Cleanup(t)
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if err := fn(b, bf); err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

// CreateTempDir delegates to the underlying framework
func (bf *BenchmarkFramework) CreateTempDir(b *testing.B, prefix string) string {
	// Convert testing.B to testing.T for the framework
	t := &testing.T{}
	return bf.framework.CreateTempDir(t, prefix)
}

// UITestFramework provides UI testing utilities using teatest
type UITestFramework struct {
	models   []tea.Model
	programs []*tea.Program
	mutex    sync.Mutex
}

// NewUITestFramework creates a new UI test framework
func NewUITestFramework() *UITestFramework {
	return &UITestFramework{
		models:   make([]tea.Model, 0),
		programs: make([]*tea.Program, 0),
	}
}

// TestUIModel tests a Bubble Tea model
func (utf *UITestFramework) TestUIModel(t *testing.T, model tea.Model, interactions []UIInteraction) *UITestResult {
	utf.mutex.Lock()
	defer utf.mutex.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(120, 40))
	utf.models = append(utf.models, model)

	result := utf.createUITestResult(model, tm, interactions)
	utf.executeInteractions(ctx, t, tm, result, interactions)
	utf.finalizeTestResult(ctx, result)

	return result
}

// createUITestResult creates a new UI test result
func (utf *UITestFramework) createUITestResult(model tea.Model, tm *teatest.TestModel, interactions []UIInteraction) *UITestResult {
	return &UITestResult{
		Model:        model,
		TestModel:    tm,
		Interactions: interactions,
		StartTime:    time.Now(),
	}
}

// executeInteractions executes all UI interactions with timeout protection
func (utf *UITestFramework) executeInteractions(ctx context.Context, t *testing.T, tm *teatest.TestModel, result *UITestResult, interactions []UIInteraction) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		utf.runInteractionLoop(ctx, t, tm, result, interactions)
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		// Completed normally
	case <-ctx.Done():
		result.Errors = append(result.Errors, fmt.Errorf("test timed out"))
	}
}

// runInteractionLoop runs the main interaction processing loop
func (utf *UITestFramework) runInteractionLoop(ctx context.Context, t *testing.T, tm *teatest.TestModel, result *UITestResult, interactions []UIInteraction) {
	for i, interaction := range interactions {
		select {
		case <-ctx.Done():
			result.Errors = append(result.Errors, fmt.Errorf("test timed out during interaction %d", i))
			return
		default:
		}

		result.CurrentStep = i
		utf.processInteraction(ctx, t, tm, result, interaction, i)
	}
}

// processInteraction processes a single UI interaction
func (utf *UITestFramework) processInteraction(ctx context.Context, t *testing.T, tm *teatest.TestModel, result *UITestResult, interaction UIInteraction, stepIndex int) {
	switch interaction.Type {
	case UIInteractionKey:
		tm.Send(interaction.Key)
	case UIInteractionMessage:
		tm.Send(interaction.Message)
	case UIInteractionWait:
		utf.handleWaitInteraction(ctx, interaction)
	case UIInteractionValidate:
		utf.handleValidationInteraction(ctx, t, tm, result, interaction, stepIndex)
	}
}

// handleWaitInteraction handles wait-type interactions
func (utf *UITestFramework) handleWaitInteraction(ctx context.Context, interaction UIInteraction) {
	timer := time.NewTimer(interaction.Duration)
	defer timer.Stop()
	
	select {
	case <-timer.C:
		// Wait completed
	case <-ctx.Done():
		// Context canceled
	}
}

// handleValidationInteraction handles validation-type interactions
func (utf *UITestFramework) handleValidationInteraction(ctx context.Context, t *testing.T, tm *teatest.TestModel, result *UITestResult, interaction UIInteraction, stepIndex int) {
	if interaction.Validator == nil {
		return
	}

	// Send quit message to ensure model terminates properly
	tm.Send(tea.QuitMsg{})

	// Get final output with timeout
	modelChan := make(chan tea.Model, 1)
	go func() {
		modelChan <- tm.FinalModel(t)
	}()

	select {
	case finalModel := <-modelChan:
		output := []byte(finalModel.View())
		if err := interaction.Validator(output); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("validation failed at step %d: %v", stepIndex, err))
		}
	case <-time.After(1 * time.Second):
		result.Errors = append(result.Errors, fmt.Errorf("timeout waiting for final model at step %d", stepIndex))
	case <-ctx.Done():
		result.Errors = append(result.Errors, fmt.Errorf("context canceled during validation at step %d", stepIndex))
	}
}

// finalizeTestResult finalizes the test result with timing information
func (utf *UITestFramework) finalizeTestResult(ctx context.Context, result *UITestResult) {
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)
}

// Cleanup cleans up UI test resources
func (utf *UITestFramework) Cleanup() {
	utf.mutex.Lock()
	defer utf.mutex.Unlock()

	for _, program := range utf.programs {
		program.Quit()
	}

	utf.models = nil
	utf.programs = nil
}

// UIInteractionType represents different types of UI interactions
type UIInteractionType int

const (
	UIInteractionKey UIInteractionType = iota
	UIInteractionMessage
	UIInteractionWait
	UIInteractionValidate
)

// UIInteraction represents a UI test interaction
type UIInteraction struct {
	Type      UIInteractionType
	Key       tea.KeyMsg
	Message   tea.Msg
	Duration  time.Duration
	Validator func([]byte) error
}

// UITestResult contains the results of a UI test
type UITestResult struct {
	Model        tea.Model
	TestModel    *teatest.TestModel
	Interactions []UIInteraction
	CurrentStep  int
	StartTime    time.Time
	EndTime      time.Time
	Duration     time.Duration
	Errors       []error
}

// HasErrors returns true if the test has any errors
func (utr *UITestResult) HasErrors() bool {
	return len(utr.Errors) > 0
}

// GetErrors returns all errors from the test
func (utr *UITestResult) GetErrors() []error {
	return utr.Errors
}

// IntegrationTestFramework provides integration testing utilities
type IntegrationTestFramework struct {
	testFramework *TestFramework
	processes     []TestProcess
	mutex         sync.Mutex
}

// NewIntegrationTestFramework creates a new integration test framework
func NewIntegrationTestFramework() *IntegrationTestFramework {
	return &IntegrationTestFramework{
		testFramework: NewTestFramework(),
		processes:     make([]TestProcess, 0),
	}
}

// TestProcess represents a test process
type TestProcess struct {
	Name    string
	Command string
	Args    []string
	Env     map[string]string
	WorkDir string
	Started time.Time
	Stopped time.Time
	Output  []byte
	Error   error
}

// StartProcess starts a test process
func (itf *IntegrationTestFramework) StartProcess(t *testing.T, process TestProcess) *TestProcess {
	itf.mutex.Lock()
	defer itf.mutex.Unlock()

	process.Started = time.Now()
	// Implementation would start actual process here

	itf.processes = append(itf.processes, process)
	return &itf.processes[len(itf.processes)-1]
}

// Cleanup cleans up integration test resources
func (itf *IntegrationTestFramework) Cleanup(t *testing.T) {
	itf.mutex.Lock()
	defer itf.mutex.Unlock()

	// Stop all processes
	for i := range itf.processes {
		if itf.processes[i].Stopped.IsZero() {
			itf.processes[i].Stopped = time.Now()
		}
	}

	itf.testFramework.Cleanup(t)
	itf.processes = nil
}

// ChaosTestFramework provides chaos testing utilities
type ChaosTestFramework struct {
	framework *TestFramework
	faults    []ChaosFault
	mutex     sync.Mutex
}

// NewChaosTestFramework creates a new chaos test framework
func NewChaosTestFramework() *ChaosTestFramework {
	return &ChaosTestFramework{
		framework: NewTestFramework(),
		faults:    make([]ChaosFault, 0),
	}
}

// ChaosFaultType represents different types of chaos faults
type ChaosFaultType int

const (
	ChaosFaultNetworkDelay ChaosFaultType = iota
	ChaosFaultNetworkDrop
	ChaosFaultFileSystemError
	ChaosFaultMemoryPressure
	ChaosFaultCPUPressure
	ChaosFaultProcessKill
)

// ChaosFault represents a chaos testing fault
type ChaosFault struct {
	Type        ChaosFaultType
	Target      string
	Duration    time.Duration
	Probability float64
	Parameters  map[string]interface{}
	Applied     bool
	StartTime   time.Time
	EndTime     time.Time
}

// InjectFault injects a chaos fault
func (ctf *ChaosTestFramework) InjectFault(t *testing.T, fault ChaosFault) error {
	ctf.mutex.Lock()
	defer ctf.mutex.Unlock()

	fault.Applied = true
	fault.StartTime = time.Now()

	// Implementation would inject actual fault here
	// For now, just record the fault
	ctf.faults = append(ctf.faults, fault)

	return nil
}

// RemoveFault removes a chaos fault
func (ctf *ChaosTestFramework) RemoveFault(t *testing.T, faultIndex int) error {
	ctf.mutex.Lock()
	defer ctf.mutex.Unlock()

	if faultIndex < 0 || faultIndex >= len(ctf.faults) {
		return errors.ValidationError("invalid fault index")
	}

	ctf.faults[faultIndex].Applied = false
	ctf.faults[faultIndex].EndTime = time.Now()

	return nil
}

// Cleanup cleans up chaos test resources
func (ctf *ChaosTestFramework) Cleanup(t *testing.T) {
	ctf.mutex.Lock()
	defer ctf.mutex.Unlock()

	// Remove all active faults
	for i := range ctf.faults {
		if ctf.faults[i].Applied {
			ctf.faults[i].Applied = false
			ctf.faults[i].EndTime = time.Now()
		}
	}

	ctf.framework.Cleanup(t)
	ctf.faults = nil
}

// TestDataManager manages test data
type TestDataManager struct {
	dataDir string
	files   map[string]string
}

// NewTestDataManager creates a new test data manager
func NewTestDataManager(dataDir string) *TestDataManager {
	return &TestDataManager{
		dataDir: dataDir,
		files:   make(map[string]string),
	}
}

// LoadTestData loads test data from a file
func (tdm *TestDataManager) LoadTestData(filename string) ([]byte, error) {
	filepath := filepath.Join(tdm.dataDir, filename)

	// #nosec G304 - filepath is constructed from validated dataDir and filename
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to load test data").
			WithCause(err).
			WithContext("filename", filename).
			WithContext("filepath", filepath).
			Build()
	}

	tdm.files[filename] = filepath
	return data, nil
}

// SaveTestData saves test data to a file
func (tdm *TestDataManager) SaveTestData(filename string, data []byte) error {
	if err := os.MkdirAll(tdm.dataDir, 0750); err != nil {
		return errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to create test data directory").
			WithCause(err).
			WithContext("dataDir", tdm.dataDir).
			Build()
	}

	filepath := filepath.Join(tdm.dataDir, filename)

	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return errors.NewError(errors.ErrorTypeFileSystem).
			WithMessage("failed to save test data").
			WithCause(err).
			WithContext("filename", filename).
			WithContext("filepath", filepath).
			Build()
	}

	tdm.files[filename] = filepath
	return nil
}

// GetTestDataFiles returns all loaded/saved test data files
func (tdm *TestDataManager) GetTestDataFiles() map[string]string {
	files := make(map[string]string)
	for k, v := range tdm.files {
		files[k] = v
	}
	return files
}

// TestReporter handles test reporting and metrics
type TestReporter struct {
	results []TestResult
	metrics TestMetrics
	writer  io.Writer
}

// NewTestReporter creates a new test reporter
func NewTestReporter(writer io.Writer) *TestReporter {
	return &TestReporter{
		results: make([]TestResult, 0),
		writer:  writer,
	}
}

// TestResult represents the result of a test
type TestResult struct {
	Name      string
	Package   string
	Passed    bool
	Duration  time.Duration
	Coverage  float64
	Errors    []error
	Benchmark *BenchmarkResult
	StartTime time.Time
	EndTime   time.Time
}

// BenchmarkResult represents benchmark results
type BenchmarkResult struct {
	Name        string
	Iterations  int
	NsPerOp     int64
	MBPerSec    float64
	AllocsPerOp int64
	BytesPerOp  int64
}

// TestMetrics contains overall test metrics
type TestMetrics struct {
	TotalTests  int
	PassedTests int
	FailedTests int
	TotalTime   time.Duration
	Coverage    float64
	Benchmarks  []BenchmarkResult
}

// AddResult adds a test result
func (tr *TestReporter) AddResult(result TestResult) {
	tr.results = append(tr.results, result)

	tr.metrics.TotalTests++
	if result.Passed {
		tr.metrics.PassedTests++
	} else {
		tr.metrics.FailedTests++
	}
	tr.metrics.TotalTime += result.Duration
}

// GenerateReport generates a test report
func (tr *TestReporter) GenerateReport() error {
	// Calculate overall coverage
	totalCoverage := 0.0
	testsWithCoverage := 0

	for _, result := range tr.results {
		if result.Coverage > 0 {
			totalCoverage += result.Coverage
			testsWithCoverage++
		}
	}

	if testsWithCoverage > 0 {
		tr.metrics.Coverage = totalCoverage / float64(testsWithCoverage)
	}

	// Generate report
	_, _ = fmt.Fprintf(tr.writer, "\n=== Test Report ===\n")                              //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(tr.writer, "Total Tests: %d\n", tr.metrics.TotalTests)              //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(tr.writer, "Passed: %d\n", tr.metrics.PassedTests)                  //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(tr.writer, "Failed: %d\n", tr.metrics.FailedTests)                  //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(tr.writer, "Total Time: %v\n", tr.metrics.TotalTime)                //nolint:errcheck // Report output errors are not critical
	_, _ = fmt.Fprintf(tr.writer, "Coverage: %.2f%%\n", tr.metrics.Coverage)               //nolint:errcheck // Report output errors are not critical

	if len(tr.results) > 0 {
		_, _ = fmt.Fprintf(tr.writer, "\n=== Test Results ===\n") //nolint:errcheck // Report output errors are not critical
		for _, result := range tr.results {
			status := "PASS"
			if !result.Passed {
				status = "FAIL"
			}
			_, _ = fmt.Fprintf(tr.writer, "[%s] %s (%v)\n", status, result.Name, result.Duration) //nolint:errcheck // Report output errors are not critical

			if len(result.Errors) > 0 {
				for _, err := range result.Errors {
					_, _ = fmt.Fprintf(tr.writer, "  Error: %v\n", err) //nolint:errcheck // Report output errors are not critical
				}
			}
		}
	}

	return nil
}

// GetMetrics returns the current test metrics
func (tr *TestReporter) GetMetrics() TestMetrics {
	return tr.metrics
}
