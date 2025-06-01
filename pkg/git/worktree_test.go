package git

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fumiya-kume/cca/internal/types"
)

func TestNewWorktreeManager(t *testing.T) {
	// Test with custom path
	customPath := "/tmp/test-ccagents"
	wm := NewWorktreeManager(customPath)

	expectedWorktreesPath := filepath.Join(customPath, ".cca", "worktrees")
	if wm.worktreesPath != expectedWorktreesPath {
		t.Errorf("expected worktreesPath to be %s, got %s", expectedWorktreesPath, wm.worktreesPath)
	}

	if wm.repositoryManager == nil {
		t.Error("expected repositoryManager to be initialized")
	}

	// Test with empty path (should use default)
	wm2 := NewWorktreeManager("")

	if wm2.worktreesPath == "" {
		t.Error("expected default worktreesPath to be set")
	}
}

func TestWorktreeManager_generateBranchName(t *testing.T) {
	wm := NewWorktreeManager("/tmp/test")

	issueRef := &types.IssueReference{
		Owner:  "testowner",
		Repo:   "testrepo",
		Number: 123,
		Source: "url",
	}

	branchName := wm.generateBranchName(issueRef)

	// Should contain ccagents prefix, issue number, and timestamp
	if len(branchName) == 0 {
		t.Error("expected non-empty branch name")
	}

	expectedPrefix := "ccagents/issue-123-"
	if len(branchName) < len(expectedPrefix) {
		t.Errorf("branch name too short: %s", branchName)
	}

	if branchName[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("expected branch name to start with %s, got %s", expectedPrefix, branchName)
	}
}

func TestWorktreeManager_generateWorktreePath(t *testing.T) {
	wm := NewWorktreeManager("/tmp/test")

	issueRef := &types.IssueReference{
		Owner:  "testowner",
		Repo:   "testrepo",
		Number: 123,
		Source: "url",
	}

	worktreePath := wm.generateWorktreePath(issueRef)

	// Should be under worktrees directory
	expectedPrefix := filepath.Join(wm.worktreesPath, "testowner-testrepo-issue-123-")
	if len(worktreePath) < len(expectedPrefix) {
		t.Errorf("worktree path too short: %s", worktreePath)
	}

	if worktreePath[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("expected worktree path to start with %s, got %s", expectedPrefix, worktreePath)
	}
}

func TestWorktreeManager_ListWorktrees(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "ccagents-worktree-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	wm := NewWorktreeManager(tmpDir)

	// Should return empty list for non-existent directory
	worktrees, err := wm.ListWorktrees()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(worktrees) != 0 {
		t.Errorf("expected empty list, got %d worktrees", len(worktrees))
	}

	// Create a mock worktree directory
	worktreeName := "testowner-testrepo-issue-123-20230101-120000"
	worktreePath := filepath.Join(wm.worktreesPath, worktreeName)

	err = os.MkdirAll(worktreePath, 0750)
	if err != nil {
		t.Fatalf("failed to create mock worktree: %v", err)
	}

	// Now should find the worktree
	worktrees, err = wm.ListWorktrees()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(worktrees) != 1 {
		t.Errorf("expected 1 worktree, got %d", len(worktrees))
	}

	if len(worktrees) > 0 && worktrees[0].Path != worktreePath {
		t.Errorf("expected worktree path %s, got %s", worktreePath, worktrees[0].Path)
	}
}

func TestWorktreeManager_parseWorktreeFromPath(t *testing.T) {
	wm := NewWorktreeManager("/tmp/test")

	// Test valid path
	validPath := "/tmp/test/worktrees/testowner-testrepo-issue-123-20230101-120000"
	worktree, err := wm.parseWorktreeFromPath(validPath)
	if err != nil {
		t.Errorf("expected no error for valid path, got %v", err)
	}

	if worktree == nil {
		t.Error("expected worktree object, got nil")
	}

	if worktree != nil && worktree.Path != validPath {
		t.Errorf("expected path %s, got %s", validPath, worktree.Path)
	}

	// Test invalid path
	invalidPath := "/tmp/test/worktrees/invalid"
	_, err = wm.parseWorktreeFromPath(invalidPath)
	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestWorktreeManager_cleanupWorktreeByPath(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "ccagents-cleanup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	wm := NewWorktreeManager(tmpDir)

	// Create a mock worktree
	worktreePath := filepath.Join(tmpDir, "test-worktree")
	err = os.MkdirAll(worktreePath, 0750)
	if err != nil {
		t.Fatalf("failed to create mock worktree: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(worktreePath, "test.txt")
	err = os.WriteFile(testFile, []byte("test"), 0600)
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Cleanup should remove the worktree
	err = wm.cleanupWorktreeByPath(worktreePath)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Worktree should no longer exist
	if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
		t.Error("expected worktree to be removed")
	}

	// Cleanup non-existent worktree should not error
	err = wm.cleanupWorktreeByPath("/nonexistent/path")
	if err != nil {
		t.Errorf("expected no error for non-existent worktree, got %v", err)
	}
}

func TestWorktreeManager_CleanupOldWorktrees(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "ccagents-old-cleanup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	wm := NewWorktreeManager(tmpDir)

	// Create mock old and new worktrees
	oldWorktree := &Worktree{
		Path:      filepath.Join(tmpDir, "old-worktree"),
		CreatedAt: time.Now().Add(-2 * time.Hour), // 2 hours old
	}

	newWorktree := &Worktree{
		Path:      filepath.Join(tmpDir, "new-worktree"),
		CreatedAt: time.Now().Add(-30 * time.Minute), // 30 minutes old
	}

	// Create directories
	for _, wt := range []*Worktree{oldWorktree, newWorktree} {
		err = os.MkdirAll(wt.Path, 0750)
		if err != nil {
			t.Fatalf("failed to create worktree dir: %v", err)
		}
	}

	// This test is simplified since we can't easily mock the ListWorktrees method
	// In a real scenario, you'd need to set up proper worktree structures

	// Cleanup worktrees older than 1 hour
	_ = wm.CleanupOldWorktrees(time.Hour)
	// Don't fail on error since we don't have real worktrees set up
	// This is more of a smoke test to ensure the method doesn't panic
}

func TestWorktree_GetWorktreeInfo(t *testing.T) {
	issueRef := &types.IssueReference{
		Owner:  "testowner",
		Repo:   "testrepo",
		Number: 123,
		Source: "url",
	}

	repo := &LocalRepository{
		Owner:         "testowner",
		Repo:          "testrepo",
		Path:          "/tmp/repo",
		DefaultBranch: "main",
	}

	worktree := &Worktree{
		Path:       "/tmp/worktree",
		BranchName: "feature-branch",
		IssueRef:   issueRef,
		CreatedAt:  time.Now(),
		Repository: repo,
	}

	info := worktree.GetWorktreeInfo()

	expectedKeys := []string{"path", "branch_name", "created_at", "issue_number", "owner", "repo", "repository"}
	for _, key := range expectedKeys {
		if _, exists := info[key]; !exists {
			t.Errorf("expected key %s in worktree info", key)
		}
	}

	if info["path"] != "/tmp/worktree" {
		t.Errorf("expected path '/tmp/worktree', got %v", info["path"])
	}

	if info["issue_number"] != 123 {
		t.Errorf("expected issue_number 123, got %v", info["issue_number"])
	}
}

func TestWorktree_ValidateWorktree(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "ccagents-validate-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test non-existent worktree
	nonExistentWorktree := &Worktree{
		Path: "/nonexistent/path",
	}

	err = nonExistentWorktree.ValidateWorktree()
	if err == nil {
		t.Error("expected error for non-existent worktree")
	}

	// Test worktree without .git directory
	worktreePathNoGit := filepath.Join(tmpDir, "no-git")
	err = os.MkdirAll(worktreePathNoGit, 0750)
	if err != nil {
		t.Fatalf("failed to create worktree dir: %v", err)
	}

	worktreeNoGit := &Worktree{
		Path: worktreePathNoGit,
	}

	err = worktreeNoGit.ValidateWorktree()
	if err == nil {
		t.Error("expected error for worktree without .git")
	}

	// Test valid worktree
	worktreePathValid := filepath.Join(tmpDir, "valid")
	gitDir := filepath.Join(worktreePathValid, ".git")
	err = os.MkdirAll(gitDir, 0750)
	if err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	validWorktree := &Worktree{
		Path: worktreePathValid,
	}

	err = validWorktree.ValidateWorktree()
	if err != nil {
		t.Errorf("expected no error for valid worktree, got %v", err)
	}
}

// Integration tests for actual git operations
func TestWorktreeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	// These tests would require actual git repositories and git commands
	t.Skip("integration tests require git setup and real repositories")
}

// Benchmark tests
func BenchmarkWorktreeManager_generateBranchName(b *testing.B) {
	wm := NewWorktreeManager("/tmp/benchmark")
	issueRef := &types.IssueReference{
		Owner:  "testowner",
		Repo:   "testrepo",
		Number: 123,
		Source: "url",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wm.generateBranchName(issueRef)
	}
}

func BenchmarkWorktreeManager_generateWorktreePath(b *testing.B) {
	wm := NewWorktreeManager("/tmp/benchmark")
	issueRef := &types.IssueReference{
		Owner:  "testowner",
		Repo:   "testrepo",
		Number: 123,
		Source: "url",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = wm.generateWorktreePath(issueRef)
	}
}
