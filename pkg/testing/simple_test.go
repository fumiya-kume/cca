package testing

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSimpleFramework tests basic framework functionality
func TestSimpleFramework(t *testing.T) {
	framework := NewTestFramework()
	defer framework.Cleanup(t)

	// Test temp directory creation
	tempDir := framework.CreateTempDir(t, "simple-test-")
	if tempDir == "" {
		t.Fatal("Failed to create temp directory")
	}

	// Verify directory exists
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Fatalf("Temp directory was not created: %s", tempDir)
	}

	// Test temp file creation
	filename := framework.CreateTempFile(t, tempDir, "test.txt", "Hello, World!")
	if filename == "" {
		t.Fatal("Failed to create temp file")
	}

	// Verify file exists and has correct content
	// #nosec G304 - filename is from test temp directory, safe
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(content) != "Hello, World!" {
		t.Fatalf("File content mismatch. Expected 'Hello, World!', got '%s'", string(content))
	}
}

// TestTestReporter tests the test reporting functionality
func TestSimpleTestReporter(t *testing.T) {
	reporter := NewTestReporter(&simpleWriter{})

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
		Errors:   []error{&testError{msg: "test error"}},
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

// TestTestDataManager tests the test data management
func TestSimpleTestDataManager(t *testing.T) {
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

// TestMockGitRepository tests mock git repository creation
func TestMockGitRepository(t *testing.T) {
	framework := NewTestFramework()
	defer framework.Cleanup(t)

	repoDir := framework.MockGitRepository(t, "test-repo")
	if repoDir == "" {
		t.Fatal("Failed to create mock git repository")
	}

	// Verify .git directory exists
	gitDir := filepath.Join(repoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		t.Fatalf(".git directory was not created: %s", gitDir)
	}

	// Verify README.md exists
	readmeFile := filepath.Join(repoDir, "README.md")
	if _, err := os.Stat(readmeFile); os.IsNotExist(err) {
		t.Fatalf("README.md was not created: %s", readmeFile)
	}
}

// TestTestCases tests the test case framework
func TestSimpleTestCases(t *testing.T) {
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

// BenchmarkSimpleFramework benchmarks the basic framework operations
func BenchmarkSimpleFramework(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// For benchmarks, we create temporary directory manually
		tempDir, err := os.MkdirTemp("", "benchmark-")
		if err != nil {
			b.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(tempDir) }()

		testFile := filepath.Join(tempDir, "test.txt")
		err = os.WriteFile(testFile, []byte("benchmark content"), 0600)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper types for testing

type simpleWriter struct {
	content []byte
}

func (sw *simpleWriter) Write(p []byte) (n int, err error) {
	sw.content = append(sw.content, p...)
	return len(p), nil
}

type testError struct {
	msg string
}

func (te *testError) Error() string {
	return te.msg
}
