package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRepositoryManager(t *testing.T) {
	// Test with custom path
	customPath := "/tmp/test-ccagents"
	rm := NewRepositoryManager(customPath)

	if rm.basePath != customPath {
		t.Errorf("expected basePath to be %s, got %s", customPath, rm.basePath)
	}

	// Test with empty path (should use default)
	rm2 := NewRepositoryManager("")

	if rm2.basePath == "" {
		t.Error("expected default basePath to be set")
	}

	// Should contain .ccagents/repos
	if !filepath.IsAbs(rm2.basePath) {
		t.Error("expected absolute path for default basePath")
	}
}

func TestRepositoryManager_GetRepositoryPath(t *testing.T) {
	rm := NewRepositoryManager("/tmp/test")

	path := rm.GetRepositoryPath("owner", "repo")
	expected := "/tmp/test/owner/repo"

	if path != expected {
		t.Errorf("expected path %s, got %s", expected, path)
	}
}

func TestRepositoryManager_ListRepositories(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "ccagents-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	rm := NewRepositoryManager(tmpDir)

	// Should return empty list for non-existent directory
	repos, err := rm.ListRepositories()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(repos) != 0 {
		t.Errorf("expected empty list, got %v", repos)
	}

	// Create a mock git repository structure
	repoPath := filepath.Join(tmpDir, "owner", "repo")
	gitPath := filepath.Join(repoPath, ".git")

	err = os.MkdirAll(gitPath, 0750)
	if err != nil {
		t.Fatalf("failed to create mock repo: %v", err)
	}

	// Now should find the repository
	repos, err = rm.ListRepositories()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(repos) != 1 {
		t.Errorf("expected 1 repository, got %d", len(repos))
	}

	if len(repos) > 0 && repos[0] != "owner/repo" {
		t.Errorf("expected 'owner/repo', got %s", repos[0])
	}
}

func TestRepositoryManager_CleanupRepository(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "ccagents-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	rm := NewRepositoryManager(tmpDir)

	// Create a mock repository
	repoPath := filepath.Join(tmpDir, "owner", "repo")
	err = os.MkdirAll(repoPath, 0750)
	if err != nil {
		t.Fatalf("failed to create mock repo: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Cleanup should remove the repository
	err = rm.CleanupRepository("owner", "repo")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Repository should no longer exist
	if _, err := os.Stat(repoPath); !os.IsNotExist(err) {
		t.Error("expected repository to be removed")
	}

	// Cleanup non-existent repository should not error
	err = rm.CleanupRepository("nonexistent", "repo")
	if err != nil {
		t.Errorf("expected no error for non-existent repo, got %v", err)
	}
}

func TestLocalRepository_GetRepositoryInfo(t *testing.T) {
	lr := &LocalRepository{
		Owner:         "testowner",
		Repo:          "testrepo",
		Path:          "/tmp/test/path",
		DefaultBranch: "main",
	}

	info := lr.GetRepositoryInfo()

	expectedKeys := []string{"owner", "repo", "path", "default_branch", "full_name"}
	for _, key := range expectedKeys {
		if _, exists := info[key]; !exists {
			t.Errorf("expected key %s in repository info", key)
		}
	}

	if info["owner"] != "testowner" {
		t.Errorf("expected owner 'testowner', got %v", info["owner"])
	}

	if info["full_name"] != "testowner/testrepo" {
		t.Errorf("expected full_name 'testowner/testrepo', got %v", info["full_name"])
	}
}

// Integration tests that require actual git operations would go here
// These would be marked with build tags to run only when appropriate

func TestRepositoryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	// These tests would require network access and would clone real repositories
	// They should be run in CI or integration test environments
	t.Skip("integration tests require network access and GitHub authentication")
}

// Benchmark tests for performance-critical operations
func BenchmarkRepositoryManager_GetRepositoryPath(b *testing.B) {
	rm := NewRepositoryManager("/tmp/benchmark")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rm.GetRepositoryPath("owner", "repo")
	}
}
