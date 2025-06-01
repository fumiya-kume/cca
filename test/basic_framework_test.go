package test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	testframework "github.com/fumiya-kume/cca/pkg/testing"
)

// TestBasicFramework tests basic framework functionality
func TestBasicFramework(t *testing.T) {
	framework := testframework.NewTestFramework()
	defer framework.Cleanup(t)

	// Test temp directory creation
	tempDir := framework.CreateTempDir(t, "basic-test-")
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
	// #nosec G304 - filename is from test fixture, safe
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(content) != "Hello, World!" {
		t.Fatalf("File content mismatch. Expected 'Hello, World!', got '%s'", string(content))
	}

	t.Logf("✅ Basic framework test passed")
}

// TestTestReporter tests the test reporting functionality
func TestBasicTestReporter(t *testing.T) {
	writer := &simpleWriter{}
	reporter := testframework.NewTestReporter(writer)

	// Add test results
	reporter.AddResult(testframework.TestResult{
		Name:     "TestExample",
		Package:  "example",
		Passed:   true,
		Duration: time.Millisecond * 100,
		Coverage: 85.5,
	})

	reporter.AddResult(testframework.TestResult{
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

	t.Logf("✅ Test reporter test passed")
}

// TestMockGitRepository tests mock git repository creation
func TestMockGitRepository(t *testing.T) {
	framework := testframework.NewTestFramework()
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

	t.Logf("✅ Mock git repository test passed")
}

// TestTestCases tests the test case framework
func TestBasicTestCases(t *testing.T) {
	framework := testframework.NewTestFramework()
	defer framework.Cleanup(t)

	testCases := []testframework.TestCase{
		{
			Name: "SuccessfulTest",
			Setup: func(t *testing.T, tf *testframework.TestFramework) interface{} {
				return "test data"
			},
			Execute: func(t *testing.T, tf *testframework.TestFramework, setupData interface{}) interface{} {
				data := setupData.(string)
				return data + " processed"
			},
			Validate: func(t *testing.T, tf *testframework.TestFramework, result interface{}) error {
				expected := "test data processed"
				if result.(string) != expected {
					t.Errorf("Expected '%s', got '%s'", expected, result.(string))
				}
				return nil
			},
		},
	}

	framework.RunTestCases(t, testCases)
	t.Logf("✅ Test cases framework test passed")
}

// BenchmarkBasicFramework benchmarks the basic framework operations
func BenchmarkBasicFramework(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		framework := testframework.NewTestFramework()
		// Create a testing.T from testing.B for the framework
		t := testing.T{}
		tempDir := framework.CreateTempDir(&t, "benchmark-")
		framework.CreateTempFile(&t, tempDir, "test.txt", "benchmark content")
		framework.Cleanup(&t)
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
