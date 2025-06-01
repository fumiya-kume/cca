package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fumiya-kume/cca/internal/types"
)

func TestNewService(t *testing.T) {
	// Test with custom path
	customPath := "/tmp/test-git-service"
	service := NewService(customPath)

	if service == nil {
		t.Fatal("expected service to be created")
	}

	if service.worktreeManager == nil {
		t.Error("expected worktree manager to be initialized")
	}

	if service.repositoryManager == nil {
		t.Error("expected repository manager to be initialized")
	}

	// Test with empty path
	service2 := NewService("")

	if service2 == nil {
		t.Fatal("expected service to be created with empty path")
	}
}

func TestService_PrepareWorkspace_Validation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ccagents-service-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	service := NewService(tmpDir)
	ctx := context.Background()

	// Test with nil issue reference
	_, err = service.PrepareWorkspace(ctx, nil)
	if err == nil {
		t.Error("expected error for nil issue reference")
	}

	// Test with invalid repository
	invalidIssueRef := &types.IssueReference{
		Owner:  "",
		Repo:   "",
		Number: 123,
		Source: "url",
	}

	_, err = service.PrepareWorkspace(ctx, invalidIssueRef)
	if err == nil {
		t.Error("expected error for invalid repository")
	}
}

func TestService_CleanupWorkspace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ccagents-cleanup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	service := NewService(tmpDir)
	ctx := context.Background()

	// Test cleanup with nil context
	err = service.CleanupWorkspace(ctx, nil)
	if err != nil {
		t.Errorf("expected no error for nil context, got %v", err)
	}

	// Test cleanup with mock context
	mockWorktree := &Worktree{
		Path: filepath.Join(tmpDir, "mock-worktree"),
	}

	// Create the directory so cleanup can remove it
	err = os.MkdirAll(mockWorktree.Path, 0750)
	if err != nil {
		t.Fatalf("failed to create mock worktree: %v", err)
	}

	mockContext := &WorkflowGitContext{
		Worktree: mockWorktree,
	}

	err = service.CleanupWorkspace(ctx, mockContext)
	if err != nil {
		t.Errorf("expected no error for cleanup, got %v", err)
	}

	// Directory should be removed
	if _, err := os.Stat(mockWorktree.Path); !os.IsNotExist(err) {
		t.Error("expected worktree directory to be removed")
	}
}

func TestService_ListActiveWorkspaces(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ccagents-list-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	service := NewService(tmpDir)

	// Should return empty list initially
	workspaces, err := service.ListActiveWorkspaces()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(workspaces) != 0 {
		t.Errorf("expected empty list, got %d workspaces", len(workspaces))
	}

	// Create a mock worktree directory
	worktreesPath := filepath.Join(tmpDir, ".cca", "worktrees")
	mockWorktreePath := filepath.Join(worktreesPath, "test-owner-test-repo-issue-123-20230101-120000")

	err = os.MkdirAll(mockWorktreePath, 0750)
	if err != nil {
		t.Fatalf("failed to create mock worktree: %v", err)
	}

	// Now should find the workspace
	workspaces, err = service.ListActiveWorkspaces()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(workspaces) != 1 {
		t.Errorf("expected 1 workspace, got %d", len(workspaces))
	}
}

func TestService_CleanupOldWorkspaces(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ccagents-old-cleanup-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	service := NewService(tmpDir)

	// Test cleanup (this is mostly a smoke test since we don't have real worktrees)
	err = service.CleanupOldWorkspaces(time.Hour)
	if err != nil {
		t.Errorf("expected no error for cleanup, got %v", err)
	}
}

func TestService_ValidateWorkspace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ccagents-validate-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	service := NewService(tmpDir)

	// Test with nil context
	err = service.ValidateWorkspace(nil)
	if err == nil {
		t.Error("expected error for nil context")
	}

	// Test with valid mock context
	worktreePath := filepath.Join(tmpDir, "valid-worktree")
	gitDir := filepath.Join(worktreePath, ".git")

	err = os.MkdirAll(gitDir, 0750)
	if err != nil {
		t.Fatalf("failed to create mock git dir: %v", err)
	}

	mockWorktree := &Worktree{
		Path: worktreePath,
	}

	validContext := &WorkflowGitContext{
		Worktree: mockWorktree,
	}

	err = service.ValidateWorkspace(validContext)
	if err != nil {
		t.Errorf("expected no error for valid workspace, got %v", err)
	}
}

func TestService_GetWorkspaceInfo(t *testing.T) {
	service := NewService("/tmp/test")

	// Test with nil context
	info := service.GetWorkspaceInfo(nil)
	if info == nil {
		t.Error("expected info map, got nil")
	}

	if _, hasError := info["error"]; !hasError {
		t.Error("expected error field for nil context")
	}

	// Test with valid context
	issueRef := &types.IssueReference{
		Owner:  "testowner",
		Repo:   "testrepo",
		Number: 123,
		Source: "url",
	}

	mockRepo := &LocalRepository{
		Owner:         "testowner",
		Repo:          "testrepo",
		Path:          "/tmp/repo",
		DefaultBranch: "main",
	}

	mockWorktree := &Worktree{
		Path:       "/tmp/worktree",
		BranchName: "feature-branch",
		IssueRef:   issueRef,
		CreatedAt:  time.Now(),
		Repository: mockRepo,
	}

	validContext := &WorkflowGitContext{
		Repository:       mockRepo,
		Worktree:         mockWorktree,
		IssueRef:         issueRef,
		OriginalBranch:   "main",
		WorkingDirectory: "/tmp/worktree",
	}

	info = service.GetWorkspaceInfo(validContext)
	if info == nil {
		t.Fatal("expected info map, got nil")
	}

	expectedKeys := []string{"working_directory", "original_branch", "repository", "worktree", "issue"}
	for _, key := range expectedKeys {
		if _, exists := info[key]; !exists {
			t.Errorf("expected key %s in workspace info", key)
		}
	}

	if info["working_directory"] != "/tmp/worktree" {
		t.Errorf("expected working_directory '/tmp/worktree', got %v", info["working_directory"])
	}

	if info["original_branch"] != "main" {
		t.Errorf("expected original_branch 'main', got %v", info["original_branch"])
	}
}

func TestService_SwitchToWorkspace(t *testing.T) {
	service := NewService("/tmp/test")

	// Test with nil context
	_, err := service.SwitchToWorkspace(nil)
	if err == nil {
		t.Error("expected error for nil context")
	}

	// Test with valid context
	validContext := &WorkflowGitContext{
		WorkingDirectory: "/tmp/valid/path",
	}

	cleanup, err := service.SwitchToWorkspace(validContext)
	if err != nil {
		t.Errorf("expected no error for valid context, got %v", err)
	}

	if cleanup == nil {
		t.Error("expected cleanup function, got nil")
	}

	// Test cleanup function
	if cleanup != nil {
		err = cleanup()
		if err != nil {
			t.Errorf("expected no error from cleanup, got %v", err)
		}
	}
}

func TestService_CreateBranchInWorkspace(t *testing.T) {
	service := NewService("/tmp/test")

	// Test with nil context
	err := service.CreateBranchInWorkspace(nil, "test-branch")
	if err == nil {
		t.Error("expected error for nil context")
	}

	// Test with context missing worktree
	invalidContext := &WorkflowGitContext{
		WorkingDirectory: "/tmp/test",
	}

	err = service.CreateBranchInWorkspace(invalidContext, "test-branch")
	if err == nil {
		t.Error("expected error for context without worktree")
	}
}

func TestService_GetWorkspaceStatus(t *testing.T) {
	service := NewService("/tmp/test")

	// Test with nil context
	_, err := service.GetWorkspaceStatus(nil)
	if err == nil {
		t.Error("expected error for nil context")
	}

	// Test with context missing worktree
	invalidContext := &WorkflowGitContext{
		WorkingDirectory: "/tmp/test",
	}

	_, err = service.GetWorkspaceStatus(invalidContext)
	if err == nil {
		t.Error("expected error for context without worktree")
	}

	// Test with valid context (but non-functional worktree)
	mockWorktree := &Worktree{
		Path:       "/tmp/mock-worktree",
		BranchName: "test-branch",
	}

	validContext := &WorkflowGitContext{
		Worktree:         mockWorktree,
		WorkingDirectory: "/tmp/mock-worktree",
	}

	status, err := service.GetWorkspaceStatus(validContext)
	if err != nil {
		t.Errorf("expected no error for valid context, got %v", err)
	}

	if status == nil {
		t.Fatal("expected status map, got nil")
	}

	expectedKeys := []string{"path", "branch", "valid"}
	for _, key := range expectedKeys {
		if _, exists := status[key]; !exists {
			t.Errorf("expected key %s in workspace status", key)
		}
	}

	if status["path"] != "/tmp/mock-worktree" {
		t.Errorf("expected path '/tmp/mock-worktree', got %v", status["path"])
	}

	if status["branch"] != "test-branch" {
		t.Errorf("expected branch 'test-branch', got %v", status["branch"])
	}
}

func TestService_GetManagers(t *testing.T) {
	service := NewService("/tmp/test")

	repoManager := service.GetRepositoryManager()
	if repoManager == nil {
		t.Error("expected repository manager, got nil")
	}

	worktreeManager := service.GetWorktreeManager()
	if worktreeManager == nil {
		t.Error("expected worktree manager, got nil")
	}
}

// Integration tests would go here
func TestServiceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	// These tests would require actual git repositories and commands
	t.Skip("integration tests require git setup and real repositories")
}

// Benchmark tests
func BenchmarkService_GetWorkspaceInfo(b *testing.B) {
	service := NewService("/tmp/benchmark")

	mockContext := &WorkflowGitContext{
		WorkingDirectory: "/tmp/test",
		OriginalBranch:   "main",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.GetWorkspaceInfo(mockContext)
	}
}
