package workflow

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fumiya-kume/cca/internal/types"
	"github.com/fumiya-kume/cca/pkg/analysis"
	"github.com/fumiya-kume/cca/pkg/claude"
	"github.com/fumiya-kume/cca/pkg/git"
	"github.com/fumiya-kume/cca/pkg/testing"
)

// IssueWorkflowService orchestrates the complete issue-to-PR workflow
type IssueWorkflowService struct {
	// Core services
	engine          *Engine
	claudeService   *claude.Service
	ghExecutor      *GitHubCLIExecutor
	analyzer        *analysis.Analyzer
	worktreeManager *git.WorktreeManager

	// Configuration
	config IssueWorkflowConfig

	// Runtime state
	activeWorkflows map[string]*IssueWorkflowInstance
	mutex           sync.RWMutex

	// Progress tracking
	progressCallbacks map[string][]ProgressCallback
	callbackMutex     sync.RWMutex
}

// IssueWorkflowConfig configures the issue workflow service
type IssueWorkflowConfig struct {
	// Timeouts
	OverallTimeout time.Duration `json:"overall_timeout"`
	StageTimeout   time.Duration `json:"stage_timeout"`
	ClaudeTimeout  time.Duration `json:"claude_timeout"`

	// Retry configuration
	MaxRetries int           `json:"max_retries"`
	RetryDelay time.Duration `json:"retry_delay"`

	// Worktree management
	CleanupOnComplete bool `json:"cleanup_on_complete"`
	CleanupOnError    bool `json:"cleanup_on_error"`

	// Claude configuration
	ClaudeModel       string  `json:"claude_model"`
	ClaudeMaxTokens   int     `json:"claude_max_tokens"`
	ClaudeTemperature float64 `json:"claude_temperature"`

	// GitHub configuration
	DefaultLabels []string `json:"default_labels"`
	AutoMerge     bool     `json:"auto_merge"`
	CreateDraft   bool     `json:"create_draft"`

	// Feature flags
	EnableAnalysis       bool `json:"enable_analysis"`
	EnableCodeGeneration bool `json:"enable_code_generation"`
	EnableTesting        bool `json:"enable_testing"`
	EnableReview         bool `json:"enable_review"`
}

// IssueWorkflowOptions provides options for workflow execution
type IssueWorkflowOptions struct {
	// PR configuration
	Draft     bool     `json:"draft"`
	Labels    []string `json:"labels"`
	Assignees []string `json:"assignees"`
	Base      string   `json:"base"`
	AutoMerge bool     `json:"auto_merge"`

	// Workflow control
	SkipReview   bool   `json:"skip_review"`
	SkipTesting  bool   `json:"skip_testing"`
	CustomBranch string `json:"custom_branch"`

	// Timeouts
	Timeout time.Duration `json:"timeout"`

	// Advanced options
	AnalysisOnly bool `json:"analysis_only"`
	DryRun       bool `json:"dry_run"`
}

// IssueWorkflowInstance represents a running issue workflow
type IssueWorkflowInstance struct {
	ID               string                `json:"id"`
	IssueRef         *types.IssueReference `json:"issue_ref"`
	Options          *IssueWorkflowOptions `json:"options"`
	WorkflowInstance *WorkflowInstance     `json:"workflow_instance"`

	// Runtime data
	IssueData        *types.IssueData             `json:"issue_data"`
	CodeContext      *types.CodeContext           `json:"code_context"`
	Analysis         *claude.IssueAnalysis        `json:"analysis"`
	GeneratedCode    *claude.CodeGenerationResult `json:"generated_code"`
	Worktree         *git.Worktree                `json:"worktree"`
	PullRequestURL   string                       `json:"pull_request_url"`

	// Progress tracking
	CurrentStage string              `json:"current_stage"`
	Progress     float64             `json:"progress"`
	StartTime    time.Time           `json:"start_time"`
	EndTime      time.Time           `json:"end_time"`
	Status       IssueWorkflowStatus `json:"status"`
	Error        error               `json:"error,omitempty"`

	// Synchronization
	mutex sync.RWMutex //nolint:unused
}

// IssueWorkflowStatus represents the status of an issue workflow
type IssueWorkflowStatus int

const (
	IssueWorkflowStatusInitializing IssueWorkflowStatus = iota
	IssueWorkflowStatusFetchingIssue
	IssueWorkflowStatusAnalyzing
	IssueWorkflowStatusBuildingContext
	IssueWorkflowStatusGeneratingCode
	IssueWorkflowStatusTesting
	IssueWorkflowStatusReviewing
	IssueWorkflowStatusCreatingPR
	IssueWorkflowStatusCompleted
	IssueWorkflowStatusFailed
	IssueWorkflowStatusCancelled
)

// ProgressCallback is called when workflow progress changes
type ProgressCallback func(instance *IssueWorkflowInstance)

// NewIssueWorkflowService creates a new issue workflow service
func NewIssueWorkflowService(config IssueWorkflowConfig) (*IssueWorkflowService, error) {
	// Set defaults
	if config.OverallTimeout == 0 {
		config.OverallTimeout = 15 * time.Minute
	}
	if config.StageTimeout == 0 {
		config.StageTimeout = 5 * time.Minute
	}
	if config.ClaudeTimeout == 0 {
		config.ClaudeTimeout = 3 * time.Minute
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 2 * time.Second
	}

	// Create workflow engine
	engineConfig := EngineConfig{
		MaxConcurrentWorkflows: 5,
		MaxConcurrentStages:    3,
		DefaultTimeout:         config.StageTimeout,
		RetryAttempts:          config.MaxRetries,
		RetryDelay:             config.RetryDelay,
		PersistenceEnabled:     true,
		MetricsEnabled:         true,
		EventBufferSize:        100,
	}

	engine, err := NewEngine(engineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow engine: %w", err)
	}

	// Create Claude service with proper client config
	clientConfig := claude.DefaultClientConfig()
	claudeConfig := claude.ServiceConfig{
		ClientConfig:   clientConfig,
		DefaultTimeout: config.ClaudeTimeout,
		RetryAttempts:  config.MaxRetries,
		RetryDelay:     config.RetryDelay,
	}

	claudeService, err := claude.NewService(claudeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Claude service: %w", err)
	}

	// Create GitHub CLI executor
	ghExecutor := NewGitHubCLIExecutor(config.StageTimeout)

	// Create analyzer
	analyzer, err := analysis.NewAnalyzer(analysis.AnalyzerConfig{
		CacheEnabled:    true,
		CacheExpiration: 30 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create analyzer: %w", err)
	}

	// Create worktree manager - use project-local worktrees
	projectRoot, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}
	worktreeManager := git.NewWorktreeManager(projectRoot)

	service := &IssueWorkflowService{
		engine:            engine,
		claudeService:     claudeService,
		ghExecutor:        ghExecutor,
		analyzer:          analyzer,
		worktreeManager:   worktreeManager,
		config:            config,
		activeWorkflows:   make(map[string]*IssueWorkflowInstance),
		progressCallbacks: make(map[string][]ProgressCallback),
	}

	return service, nil
}

// ProcessIssue processes a GitHub issue and creates a pull request
func (s *IssueWorkflowService) ProcessIssue(ctx context.Context, issueRef *types.IssueReference, options *IssueWorkflowOptions) (*IssueWorkflowInstance, error) {
	fmt.Println("🔍 Debug: ProcessIssue starting...")

	// Validate inputs
	if issueRef == nil {
		return nil, fmt.Errorf("issue reference is required")
	}

	if options == nil {
		options = &IssueWorkflowOptions{}
	}

	fmt.Printf("🔍 Debug: Processing issue %s/%s#%d\n", issueRef.Owner, issueRef.Repo, issueRef.Number)

	// Set timeout
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, options.Timeout)
		defer cancel()
	} else {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.OverallTimeout)
		defer cancel()
	}

	// Check authentication first
	fmt.Println("🔍 Debug: Checking GitHub authentication...")
	if err := s.ghExecutor.CheckAuthentication(ctx); err != nil {
		fmt.Printf("❌ GitHub authentication failed: %v\n", err)
		return nil, fmt.Errorf("authentication check failed: %w", err)
	}
	fmt.Println("✅ GitHub authentication OK")

	// Create workflow instance
	instance := &IssueWorkflowInstance{
		ID:        fmt.Sprintf("issue_%s_%s_%d_%d", issueRef.Owner, issueRef.Repo, issueRef.Number, time.Now().Unix()),
		IssueRef:  issueRef,
		Options:   options,
		StartTime: time.Now(),
		Status:    IssueWorkflowStatusInitializing,
	}

	// Register instance
	s.mutex.Lock()
	s.activeWorkflows[instance.ID] = instance
	s.mutex.Unlock()

	// Start workflow
	fmt.Println("🔍 Debug: Starting executeWorkflow...")
	if err := s.executeWorkflow(ctx, instance); err != nil {
		fmt.Printf("❌ executeWorkflow failed: %v\n", err)
		instance.Error = err
		instance.Status = IssueWorkflowStatusFailed
		instance.EndTime = time.Now()
		s.notifyProgress(instance)
		return instance, err
	}
	fmt.Println("✅ executeWorkflow completed")

	instance.Status = IssueWorkflowStatusCompleted
	instance.EndTime = time.Now()
	instance.Progress = 1.0
	s.notifyProgress(instance)

	// Cleanup if configured
	if s.config.CleanupOnComplete && instance.Worktree != nil {
		s.cleanupWorktree(instance)
	}

	return instance, nil
}

// executeWorkflow executes the complete workflow
func (s *IssueWorkflowService) executeWorkflow(ctx context.Context, instance *IssueWorkflowInstance) error {
	fmt.Println("🔍 Debug: executeWorkflow starting...")

	// Stage 1: Fetch Issue Data
	fmt.Println("📋 Stage 1/5: Fetching issue data...")
	if err := s.fetchIssueData(ctx, instance); err != nil {
		fmt.Printf("❌ Stage 1 failed: %v\n", err)
		return fmt.Errorf("failed to fetch issue data: %w", err)
	}
	fmt.Println("✅ Stage 1 completed")

	// Early exit for analysis-only mode
	if instance.Options.AnalysisOnly {
		return s.analyzeIssue(ctx, instance)
	}

	// Stage 2: Clone Repository and Build Context
	fmt.Println("📂 Stage 2/5: Building code context...")
	if err := s.buildCodeContext(ctx, instance); err != nil {
		fmt.Printf("❌ Stage 2 failed: %v\n", err)
		return fmt.Errorf("failed to build code context: %w", err)
	}
	fmt.Println("✅ Stage 2 completed")

	// Stage 3: Analyze Issue with Claude
	fmt.Println("🤖 Stage 3/5: Analyzing issue with Claude...")
	if err := s.analyzeIssue(ctx, instance); err != nil {
		fmt.Printf("❌ Stage 3 failed: %v\n", err)
		return fmt.Errorf("failed to analyze issue: %w", err)
	}
	fmt.Println("✅ Stage 3 completed")

	// Stage 4: Generate Code
	fmt.Println("⚡ Stage 4/5: Generating code...")
	if err := s.generateCode(ctx, instance); err != nil {
		fmt.Printf("❌ Stage 4 failed: %v\n", err)
		return fmt.Errorf("failed to generate code: %w", err)
	}
	fmt.Println("✅ Stage 4 completed")

	// Early exit for dry run
	if instance.Options.DryRun {
		fmt.Println("🚀 Dry run mode - stopping before changes")
		return nil
	}

	// Stage 5: Create Branch and Commit Changes
	fmt.Println("📝 Stage 5/5: Creating branch and committing changes...")
	if err := s.commitChanges(ctx, instance); err != nil {
		fmt.Printf("❌ Stage 5 failed: %v\n", err)
		return fmt.Errorf("failed to commit changes: %w", err)
	}
	fmt.Println("✅ Stage 5 completed")

	// Stage 6: Run Tests (if enabled)
	if !instance.Options.SkipTesting && s.config.EnableTesting {
		fmt.Println("🧪 Running tests...")
		if err := s.runTests(ctx, instance); err != nil {
			fmt.Printf("❌ Tests failed: %v\n", err)
			return fmt.Errorf("tests failed: %w", err)
		}
		fmt.Println("✅ Tests passed")
	} else {
		fmt.Println("⏭️  Skipping tests")
	}

	// Stage 7: Code Review (if enabled)
	if !instance.Options.SkipReview && s.config.EnableReview {
		fmt.Println("🔍 Running code review...")
		if err := s.reviewCode(ctx, instance); err != nil {
			fmt.Printf("❌ Code review failed: %v\n", err)
			return fmt.Errorf("code review failed: %w", err)
		}
		fmt.Println("✅ Code review passed")
	} else {
		fmt.Println("⏭️  Skipping code review")
	}

	// Stage 8: Create Pull Request
	fmt.Println("🚀 Creating pull request...")
	if err := s.createPullRequest(ctx, instance); err != nil {
		fmt.Printf("❌ PR creation failed: %v\n", err)
		return fmt.Errorf("failed to create pull request: %w", err)
	}
	fmt.Println("✅ Pull request created")

	return nil
}

// fetchIssueData fetches GitHub issue data
func (s *IssueWorkflowService) fetchIssueData(ctx context.Context, instance *IssueWorkflowInstance) error {
	instance.Status = IssueWorkflowStatusFetchingIssue
	instance.CurrentStage = "Fetching Issue Data"
	instance.Progress = 0.1
	s.notifyProgress(instance)

	issueData, err := s.ghExecutor.FetchIssueData(ctx, instance.IssueRef)
	if err != nil {
		return err
	}

	instance.IssueData = issueData
	instance.Progress = 0.15
	s.notifyProgress(instance)

	return nil
}

// buildCodeContext creates a worktree and builds code context
func (s *IssueWorkflowService) buildCodeContext(ctx context.Context, instance *IssueWorkflowInstance) error {
	instance.Status = IssueWorkflowStatusBuildingContext
	instance.CurrentStage = "Creating Issue Worktree"
	instance.Progress = 0.2
	s.notifyProgress(instance)

	// Create worktree for this issue
	taskName := fmt.Sprintf("issue-%d", instance.IssueRef.Number)
	worktree, err := s.worktreeManager.CreateTaskWorktree(taskName)
	if err != nil {
		return fmt.Errorf("failed to create worktree for issue: %w", err)
	}

	instance.Worktree = worktree
	instance.Progress = 0.3

	fmt.Printf("✅ Created worktree: %s\n", worktree.Path)
	fmt.Printf("✅ Branch: %s\n", worktree.BranchName)
	s.notifyProgress(instance)

	// Analyze codebase in the worktree
	instance.CurrentStage = "Analyzing Codebase"
	codeContext, err := s.analyzer.AnalyzeProject(ctx, worktree.Path)
	if err != nil {
		return fmt.Errorf("failed to analyze codebase: %w", err)
	}

	instance.CodeContext = codeContext.CodeContext
	instance.Progress = 0.4
	s.notifyProgress(instance)

	return nil
}

// analyzeIssue analyzes the issue using Claude
func (s *IssueWorkflowService) analyzeIssue(ctx context.Context, instance *IssueWorkflowInstance) error {
	instance.Status = IssueWorkflowStatusAnalyzing
	instance.CurrentStage = "Analyzing Issue with Claude"
	instance.Progress = 0.45
	s.notifyProgress(instance)

	// Set working directory for Claude to the worktree path
	if instance.IssueData != nil {
		instance.IssueData.WorkingDirectory = instance.Worktree.Path
	}

	fmt.Println("🔍 Debug: Calling Claude service AnalyzeIssue...")
	analysis, err := s.claudeService.AnalyzeIssue(ctx, instance.IssueData, instance.CodeContext)
	if err != nil {
		fmt.Printf("❌ Claude AnalyzeIssue failed: %v\n", err)
		return err
	}
	fmt.Println("✅ Claude AnalyzeIssue completed")

	instance.Analysis = analysis
	instance.Progress = 0.55
	s.notifyProgress(instance)

	return nil
}

// generateCode generates code using Claude
func (s *IssueWorkflowService) generateCode(ctx context.Context, instance *IssueWorkflowInstance) error {
	instance.Status = IssueWorkflowStatusGeneratingCode
	instance.CurrentStage = "Generating Code"
	instance.Progress = 0.6
	s.notifyProgress(instance)

	request := &claude.CodeGenerationRequest{
		Repository:       instance.IssueData.Repository,
		IssueNumber:      instance.IssueData.Number,
		WorkingDirectory: instance.Worktree.Path,
		Context:          instance.CodeContext,
		Requirements:     instance.Analysis.Requirements,
		Language:         "go", // TODO: detect from codebase
		Framework:        "",   // TODO: detect from codebase
		Constraints:      []string{},
	}

	result, err := s.claudeService.GenerateCode(ctx, request)
	if err != nil {
		return err
	}

	instance.GeneratedCode = result
	instance.Progress = 0.75
	s.notifyProgress(instance)

	return nil
}

// commitChanges creates branch and commits changes
func (s *IssueWorkflowService) commitChanges(ctx context.Context, instance *IssueWorkflowInstance) error {
	instance.CurrentStage = "Committing Changes"
	instance.Progress = 0.8
	s.notifyProgress(instance)

	// Branch is already created by the worktree
	fmt.Printf("✅ Using worktree branch: %s\n", instance.Worktree.BranchName)

	// Apply generated changes
	if instance.GeneratedCode != nil {
		// Apply main files to worktree
		for _, file := range instance.GeneratedCode.Files {
			filePath := filepath.Join(instance.Worktree.Path, file.Path)

			// Create directory if needed
			if err := os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", file.Path, err)
			}

			// Write file
			if err := os.WriteFile(filePath, []byte(file.Content), 0600); err != nil {
				return fmt.Errorf("failed to write file %s: %w", file.Path, err)
			}
		}

		// Apply test files to worktree
		for _, file := range instance.GeneratedCode.Tests {
			filePath := filepath.Join(instance.Worktree.Path, file.Path)

			// Create directory if needed
			if err := os.MkdirAll(filepath.Dir(filePath), 0750); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", file.Path, err)
			}

			// Write file
			if err := os.WriteFile(filePath, []byte(file.Content), 0600); err != nil {
				return fmt.Errorf("failed to write file %s: %w", file.Path, err)
			}
		}
	}

	// Commit changes in worktree
	commitMessage := fmt.Sprintf("feat: %s\n\nFixes #%d\n\n%s",
		instance.IssueData.Title,
		instance.IssueData.Number,
		instance.Analysis.Summary)

	if err := s.ghExecutor.CommitChanges(ctx, instance.Worktree.Path, commitMessage); err != nil {
		return err
	}

	// Push branch from worktree
	if err := s.ghExecutor.PushBranch(ctx, instance.Worktree.Path, instance.Worktree.BranchName); err != nil {
		return err
	}

	instance.Progress = 0.85
	s.notifyProgress(instance)
	return nil
}

// runTests runs the test suite
func (s *IssueWorkflowService) runTests(ctx context.Context, instance *IssueWorkflowInstance) error {
	instance.Status = IssueWorkflowStatusTesting
	instance.CurrentStage = "Running Tests"
	instance.Progress = 0.9
	s.notifyProgress(instance)

	// Skip tests if requested
	if instance.Options.SkipTesting {
		fmt.Println("⏭️  Skipping tests as requested")
		return nil
	}

	// Use worktree path as project root for testing
	projectRoot := instance.Worktree.Path

	// Create test configuration
	testConfig := &testing.TestConfig{
		ProjectRoot:    projectRoot,
		OutputDir:      filepath.Join(projectRoot, ".ccagents", "test-results"),
		CoverageTarget: 80.0, // 80% coverage target
		Timeout:        10 * time.Minute,
		RunIntegration: false, // Skip integration tests in workflow by default
		RunBenchmarks:  false, // Skip benchmarks in workflow by default
		RunPerformance: false, // Skip performance tests in workflow by default
		RunE2E:         false, // Skip E2E tests in workflow by default
		GenerateReport: true,
		Verbose:        true,
	}

	// Create test runner
	testRunner := testing.NewTestRunner(testConfig)

	// Run tests
	fmt.Println("🧪 Running test suite...")
	if err := testRunner.RunAllTests(); err != nil {
		fmt.Printf("❌ Tests failed: %v\n", err)
		return fmt.Errorf("test execution failed: %w", err)
	}

	fmt.Println("✅ All tests passed successfully")
	return nil
}

// reviewCode performs automated code review
func (s *IssueWorkflowService) reviewCode(ctx context.Context, instance *IssueWorkflowInstance) error {
	instance.Status = IssueWorkflowStatusReviewing
	instance.CurrentStage = "Reviewing Code"
	instance.Progress = 0.92
	s.notifyProgress(instance)

	// Convert generated files to file paths for review
	var filePaths []string
	if instance.GeneratedCode != nil {
		for _, file := range instance.GeneratedCode.Files {
			filePaths = append(filePaths, file.Path)
		}
		for _, file := range instance.GeneratedCode.Tests {
			filePaths = append(filePaths, file.Path)
		}
	}

	request := &claude.CodeReviewRequest{
		Repository:       instance.IssueData.Repository,
		Branch:           instance.Worktree.BranchName,
		WorkingDirectory: instance.Worktree.Path,
		Files:            filePaths,
		Context:          instance.CodeContext,
		ReviewType:       claude.ReviewFull,
	}

	_, err := s.claudeService.ReviewCode(ctx, request)
	if err != nil {
		return err
	}

	return nil
}

// createPullRequest creates the pull request
func (s *IssueWorkflowService) createPullRequest(ctx context.Context, instance *IssueWorkflowInstance) error {
	instance.Status = IssueWorkflowStatusCreatingPR
	instance.CurrentStage = "Creating Pull Request"
	instance.Progress = 0.95
	s.notifyProgress(instance)

	// Build PR body
	prBody := fmt.Sprintf("Fixes #%d\n\n## Summary\n%s\n\n## Technical Details\n%s",
		instance.IssueData.Number,
		instance.Analysis.Summary,
		strings.Join(instance.Analysis.TechnicalDetails, "\n- "))

	if instance.GeneratedCode != nil && (len(instance.GeneratedCode.Files) > 0 || len(instance.GeneratedCode.Tests) > 0) {
		prBody += "\n\n## Files Changed\n"
		for _, file := range instance.GeneratedCode.Files {
			prBody += fmt.Sprintf("- %s (%s)\n", file.Path, file.Description)
		}
		for _, file := range instance.GeneratedCode.Tests {
			prBody += fmt.Sprintf("- %s (test)\n", file.Path)
		}
	}

	prBody += "\n\n🤖 Generated with ccAgents"

	// Merge labels
	labels := append(s.config.DefaultLabels, instance.Options.Labels...)
	labels = append(labels, "automated")

	prRequest := &PullRequestRequest{
		Title:     fmt.Sprintf("Fix: %s", instance.IssueData.Title),
		Body:      prBody,
		Base:      instance.Options.Base,
		Labels:    labels,
		Assignees: instance.Options.Assignees,
		Draft:     instance.Options.Draft || s.config.CreateDraft,
	}

	result, err := s.ghExecutor.CreatePullRequest(ctx, instance.Worktree.Path, prRequest)
	if err != nil {
		return err
	}

	instance.PullRequestURL = result.URL
	instance.Progress = 1.0
	s.notifyProgress(instance)

	return nil
}

// AddProgressCallback adds a progress callback
func (s *IssueWorkflowService) AddProgressCallback(workflowID string, callback ProgressCallback) {
	s.callbackMutex.Lock()
	defer s.callbackMutex.Unlock()

	s.progressCallbacks[workflowID] = append(s.progressCallbacks[workflowID], callback)
}

// notifyProgress notifies all progress callbacks
func (s *IssueWorkflowService) notifyProgress(instance *IssueWorkflowInstance) {
	s.callbackMutex.RLock()
	callbacks := s.progressCallbacks[instance.ID]
	s.callbackMutex.RUnlock()

	for _, callback := range callbacks {
		go callback(instance)
	}
}

// GetWorkflowStatus returns the status of a workflow
func (s *IssueWorkflowService) GetWorkflowStatus(workflowID string) (*IssueWorkflowInstance, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	instance, exists := s.activeWorkflows[workflowID]
	if !exists {
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}

	return instance, nil
}

// CancelWorkflow cancels a running workflow
func (s *IssueWorkflowService) CancelWorkflow(workflowID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	instance, exists := s.activeWorkflows[workflowID]
	if !exists {
		return fmt.Errorf("workflow %s not found", workflowID)
	}

	if instance.WorkflowInstance != nil && instance.WorkflowInstance.Cancel != nil {
		instance.WorkflowInstance.Cancel()
	}

	instance.Status = IssueWorkflowStatusCancelled
	instance.EndTime = time.Now()

	// Cleanup worktree on error if configured
	if s.config.CleanupOnError && instance.Worktree != nil {
		s.cleanupWorktree(instance)
	}

	return nil
}

// cleanupWorktree cleans up the worktree for an instance
func (s *IssueWorkflowService) cleanupWorktree(instance *IssueWorkflowInstance) {
	if instance.Worktree == nil {
		return
	}
	
	fmt.Printf("🧹 Cleaning up worktree: %s\n", instance.Worktree.Path)
	if err := s.worktreeManager.CleanupWorktree(instance.Worktree); err != nil {
		fmt.Printf("⚠️  Warning: Failed to cleanup worktree %s: %v\n", instance.Worktree.Path, err)
	} else {
		fmt.Printf("✅ Worktree cleaned up successfully\n")
	}
}

// CleanupCompletedWorkflows cleans up worktrees for completed workflows
func (s *IssueWorkflowService) CleanupCompletedWorkflows() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	for id, instance := range s.activeWorkflows {
		if instance.Status == IssueWorkflowStatusCompleted || instance.Status == IssueWorkflowStatusFailed {
			s.cleanupWorktree(instance)
			delete(s.activeWorkflows, id)
		}
	}
	
	return nil
}

// Close shuts down the workflow service
func (s *IssueWorkflowService) Close() error {
	// Cancel all active workflows and cleanup worktrees
	s.mutex.Lock()
	for _, instance := range s.activeWorkflows {
		if instance.WorkflowInstance != nil && instance.WorkflowInstance.Cancel != nil {
			instance.WorkflowInstance.Cancel()
		}
		// Cleanup worktree
		s.cleanupWorktree(instance)
	}
	s.mutex.Unlock()

	// Shutdown engine
	if s.engine != nil {
		return s.engine.Shutdown(context.Background())
	}

	return nil
}

// DefaultIssueWorkflowConfig returns default configuration
func DefaultIssueWorkflowConfig() IssueWorkflowConfig {
	return IssueWorkflowConfig{
		OverallTimeout:       15 * time.Minute,
		StageTimeout:         5 * time.Minute,
		ClaudeTimeout:        3 * time.Minute,
		MaxRetries:           3,
		RetryDelay:           2 * time.Second,
		CleanupOnComplete:    true,
		CleanupOnError:       true,
		DefaultLabels:        []string{"automated"},
		EnableAnalysis:       true,
		EnableCodeGeneration: true,
		EnableTesting:        true,
		EnableReview:         true,
	}
}
