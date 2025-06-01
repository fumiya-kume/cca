package testing

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestTestingFramework tests the testing framework itself
func TestTestingFramework(t *testing.T) {
	framework := NewTestFramework()
	defer framework.Cleanup(t)

	// Test temp directory creation
	tempDir := framework.CreateTempDir(t, "framework-test-")
	if tempDir == "" {
		t.Fatal("Failed to create temp directory")
	}

	// Test temp file creation
	filename := framework.CreateTempFile(t, tempDir, "test.txt", "Hello, World!")
	if filename == "" {
		t.Fatal("Failed to create temp file")
	}

	// Test mock git repository
	repoDir := framework.MockGitRepository(t, "test-repo")
	if repoDir == "" {
		t.Fatal("Failed to create mock git repository")
	}
}

// TestTestCases tests the test case framework
func TestTestCases(t *testing.T) {
	framework := NewTestFramework()
	defer framework.Cleanup(t)

	testCases := []TestCase{
		{
			Name: "SuccessfulTest",
			Setup: func(t *testing.T, tf *TestFramework) interface{} {
				return "test data"
			},
			Execute: func(t *testing.T, tf *TestFramework, setupData interface{}) interface{} {
				data := setupData.(string)
				return data + " processed"
			},
			Validate: func(t *testing.T, tf *TestFramework, result interface{}) error {
				expected := "test data processed"
				if result.(string) != expected {
					t.Errorf("Expected '%s', got '%s'", expected, result.(string))
				}
				return nil
			},
		},
		{
			Name:        "FailingTest",
			ShouldFail:  true,
			ExpectedErr: "intentional failure",
			Execute: func(t *testing.T, tf *TestFramework, setupData interface{}) interface{} {
				panic("intentional failure")
			},
		},
	}

	framework.RunTestCases(t, testCases)
}

// TestBenchmarkFramework tests the benchmark framework
func TestBenchmarkFramework(t *testing.T) {
	bf := NewBenchmarkFramework()

	// Create a simple benchmark test
	b := &testing.B{N: 100}

	bf.RunBenchmark(b, nil, func(b *testing.B, bf *BenchmarkFramework) error {
		// Simulate some work without sleep for faster tests
		_ = make([]byte, 100) // Simulate memory allocation instead
		return nil
	})
}

// TestUITestFramework tests the UI testing framework
func TestUITestFramework(t *testing.T) {
	utf := NewUITestFramework()
	defer utf.Cleanup()

	// Create a simple model for testing
	model := &testModel{value: "initial"}

	interactions := []UIInteraction{
		{
			Type:    UIInteractionMessage,
			Message: testMsg{value: "update"},
		},
		{
			Type: UIInteractionValidate,
			Validator: func(output []byte) error {
				// Simple validation
				return nil
			},
		},
	}

	result := utf.TestUIModel(t, model, interactions)
	if result.HasErrors() {
		t.Errorf("UI test failed with errors: %v", result.GetErrors())
	}
}

// TestIntegrationTestSuite tests the integration testing framework
func TestIntegrationTestSuite(t *testing.T) {
	t.Skip("Integration test suite removed - skipping test")
}

// TestPerformanceTestSuite tests the performance testing framework
func TestPerformanceTestSuite(t *testing.T) {
	t.Skip("Performance test suite removed - skipping test")
}

// TestE2ETestSuite tests the end-to-end testing framework
func TestE2ETestSuite(t *testing.T) {
	t.Skip("E2E test suite removed - skipping test")
}

// TestChaosTestFramework tests the chaos testing framework
func TestChaosTestFramework(t *testing.T) {
	t.Skip("Chaos test framework removed - skipping test")
}

// TestTestDataManager tests the test data management
func TestTestDataManager(t *testing.T) {
	framework := NewTestFramework()
	defer framework.Cleanup(t)

	tempDir := framework.CreateTempDir(t, "test-data-")

	tdm := NewTestDataManager(tempDir)

	// Test saving test data
	testData := []byte("test data content")
	if err := tdm.SaveTestData("test.json", testData); err != nil {
		t.Errorf("Failed to save test data: %v", err)
	}

	// Test loading test data
	loadedData, err := tdm.LoadTestData("test.json")
	if err != nil {
		t.Errorf("Failed to load test data: %v", err)
	}

	if string(loadedData) != string(testData) {
		t.Errorf("Loaded data doesn't match saved data")
	}

	// Test file listing
	files := tdm.GetTestDataFiles()
	if len(files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(files))
	}
}

// TestTestReporter tests the test reporting functionality
func TestTestReporter(t *testing.T) {
	reporter := NewTestReporter(&testWriter{})

	// Add test results
	reporter.AddResult(TestResult{
		Name:     "TestExample",
		Package:  "example",
		Passed:   true,
		Duration: time.Millisecond * 100,
		Coverage: 85.5,
	})

	reporter.AddResult(TestResult{
		Name:     "TestFailing",
		Package:  "example",
		Passed:   false,
		Duration: time.Millisecond * 50,
		Errors:   []error{fmt.Errorf("test error")},
	})

	// Generate report
	if err := reporter.GenerateReport(); err != nil {
		t.Errorf("Failed to generate report: %v", err)
	}

	// Check metrics
	metrics := reporter.GetMetrics()
	if metrics.TotalTests != 2 {
		t.Errorf("Expected 2 total tests, got %d", metrics.TotalTests)
	}
	if metrics.PassedTests != 1 {
		t.Errorf("Expected 1 passed test, got %d", metrics.PassedTests)
	}
	if metrics.FailedTests != 1 {
		t.Errorf("Expected 1 failed test, got %d", metrics.FailedTests)
	}
}

// BenchmarkTestingFramework benchmarks the testing framework performance
func BenchmarkTestingFramework(b *testing.B) {
	bf := NewBenchmarkFramework()

	bf.RunBenchmark(b, nil, func(b *testing.B, bf *BenchmarkFramework) error {
		// Benchmark framework operations
		framework := NewTestFramework()
		defer framework.Cleanup(&testing.T{})

		tempDir := framework.CreateTempDir(&testing.T{}, "benchmark-")
		framework.CreateTempFile(&testing.T{}, tempDir, "test.txt", "benchmark content")

		return nil
	})
}

// Helper types for testing

type testModel struct {
	value string
}

func (m testModel) Init() tea.Cmd {
	return nil
}

func (m testModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case testMsg:
		m.value = msg.value
	}
	return m, nil
}

func (m testModel) View() string {
	return fmt.Sprintf("Value: %s", m.value)
}

type testMsg struct {
	value string
}

type testWriter struct {
	content []byte
}

func (tw *testWriter) Write(p []byte) (n int, err error) {
	tw.content = append(tw.content, p...)
	return len(p), nil
}

// ExampleTestRunner demonstrates how to use the test runner
func ExampleTestRunner() {
	config := DefaultTestConfig()
	config.ProjectRoot = "."
	config.OutputDir = "./test-results"
	config.RunBenchmarks = true
	config.RunIntegration = true

	runner := NewTestRunner(config)

	if err := runner.RunAllTests(); err != nil {
		panic(err)
	}
}
