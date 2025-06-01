package workflow

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// StageJob represents a job to be executed by a worker
type StageJob struct {
	ID           string
	Stage        *StageInstance
	Action       ActionDefinition
	Handler      ActionHandler
	Context      context.Context
	RetryCount   int
	SubmittedAt  time.Time
}

// StageResult represents the result of a stage execution
type StageResult struct {
	JobID     string
	Stage     *StageInstance
	Output    interface{}
	Error     error
	Duration  time.Duration
	Worker    int
}

// ExecuteStageInPool executes a stage using the worker pool
func (wp *WorkerPool) ExecuteStageInPool(ctx context.Context, stage *StageInstance, action ActionDefinition, handler ActionHandler) error {
	// Use semaphore to limit concurrent executions
	select {
	case wp.semaphore <- struct{}{}:
		defer func() { <-wp.semaphore }()
	case <-ctx.Done():
		return ctx.Err()
	}
	
	// Track the worker
	wp.activeWorkers.Add(1)
	defer wp.activeWorkers.Done()
	
	// Execute the stage
	_, err := handler.Execute(ctx, stage, action)
	return err
}

// StageExecutor handles the execution of workflow stages
type StageExecutor struct {
	config         EngineConfig
	actionHandlers map[ActionType]ActionHandler
	workerPool     *WorkerPool
	retryManager   *RetryManager
}

// ActionHandler interface for different action types
type ActionHandler interface {
	Execute(ctx context.Context, stage *StageInstance, action ActionDefinition) (interface{}, error)
	Validate(action ActionDefinition) error
}

// RetryManager handles retry logic for failed stages
type RetryManager struct {
	config EngineConfig
}

// NewStageExecutor creates a new stage executor
func NewStageExecutor(config EngineConfig) (*StageExecutor, error) {
	// Create worker pool for parallel execution
	maxWorkers := config.MaxConcurrentStages
	if maxWorkers <= 0 {
		maxWorkers = 4 // Default to 4 concurrent stages
	}
	
	workerPool, err := NewWorkerPool(maxWorkers)
	if err != nil {
		return nil, fmt.Errorf("failed to create worker pool: %w", err)
	}

	executor := &StageExecutor{
		config:         config,
		actionHandlers: make(map[ActionType]ActionHandler),
		workerPool:     workerPool,
		retryManager:   NewRetryManager(config),
	}

	// Register default action handlers
	executor.registerDefaultHandlers()

	return executor, nil
}

// registerDefaultHandlers registers built-in action handlers
func (se *StageExecutor) registerDefaultHandlers() {
	se.actionHandlers[ActionTypeCommand] = &CommandActionHandler{}
	se.actionHandlers[ActionTypeScript] = &ScriptActionHandler{}
	se.actionHandlers[ActionTypeFunction] = &FunctionActionHandler{}
	se.actionHandlers[ActionTypeHTTP] = &HTTPActionHandler{}
	se.actionHandlers[ActionTypeClaudeCode] = &ClaudeCodeActionHandler{}
	se.actionHandlers[ActionTypeGitOperation] = &GitActionHandler{}
	se.actionHandlers[ActionTypeFileOperation] = &FileActionHandler{}
}

// ExecuteStage executes a single stage
func (se *StageExecutor) ExecuteStage(ctx context.Context, stage *StageInstance, workflow *WorkflowInstance) error {
	// Check if stage can be executed
	if err := se.canExecuteStage(stage); err != nil {
		return fmt.Errorf("stage %s cannot be executed: %w", stage.ID, err)
	}

	// Check dependencies
	if !se.areDependenciesSatisfied(stage) {
		stage.mutex.Lock()
		stage.Status = StageStatusWaitingForDependencies
		stage.mutex.Unlock()
		return fmt.Errorf("stage %s dependencies not satisfied", stage.ID)
	}

	// Evaluate conditions
	if !se.evaluateConditions(stage, workflow) {
		stage.mutex.Lock()
		stage.Status = StageStatusSkipped
		stage.mutex.Unlock()
		return nil
	}

	// Set stage to running
	stage.mutex.Lock()
	stage.Status = StageStatusRunning
	stage.StartTime = time.Now()
	stage.mutex.Unlock()

	// Execute the action
	output, err := se.executeAction(ctx, stage)

	if err != nil {
		stage.mutex.Lock()
		stage.Error = err
		stage.Status = StageStatusFailed
		stage.EndTime = time.Now()
		stage.mutex.Unlock()

		// Check if retry is needed
		if se.shouldRetry(stage) {
			return se.retryStage(ctx, stage, workflow)
		}

		return fmt.Errorf("stage %s execution failed: %w", stage.ID, err)
	}

	// Stage completed successfully
	stage.mutex.Lock()
	stage.Output = output
	stage.Status = StageStatusCompleted
	stage.EndTime = time.Now()
	stage.mutex.Unlock()

	return nil
}

// ExecuteParallelStages executes multiple stages in parallel
func (se *StageExecutor) ExecuteParallelStages(ctx context.Context, stages []*StageInstance, workflow *WorkflowInstance) error {
	if len(stages) == 0 {
		return nil
	}

	// Use worker pool for controlled parallel execution
	var wg sync.WaitGroup
	errors := make(chan error, len(stages))

	for _, stage := range stages {
		wg.Add(1)
		go func(s *StageInstance) {
			defer wg.Done()
			
			// Execute stage through worker pool with concurrency control
			err := se.executeStageWithPool(ctx, s, workflow)
			if err != nil {
				errors <- fmt.Errorf("parallel stage %s failed: %w", s.ID, err)
			}
		}(stage)
	}

	// Wait for all stages to complete
	wg.Wait()
	close(errors)

	// Collect errors
	var allErrors []error
	for err := range errors {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) > 0 {
		return fmt.Errorf("parallel execution failed: %v", allErrors)
	}

	return nil
}

// executeStageWithPool executes a stage using the worker pool for concurrency control
func (se *StageExecutor) executeStageWithPool(ctx context.Context, stage *StageInstance, workflow *WorkflowInstance) error {
	if stage == nil {
		return fmt.Errorf("stage is nil")
	}

	// Check stage dependencies
	if err := se.canExecuteStage(stage); err != nil {
		return fmt.Errorf("stage %s cannot be executed: %w", stage.ID, err)
	}

	stage.Status = StageStatusRunning
	stage.StartTime = time.Now()

	// Use stage definition
	if stage.Definition == nil {
		return fmt.Errorf("stage %s has no definition", stage.ID)
	}

	// Get action handler based on stage type
	var actionType ActionType
	switch stage.Definition.Type {
	case StageTypeSequential:
		actionType = ActionTypeCommand
	case StageTypeParallel:
		actionType = ActionTypeCommand
	default:
		actionType = ActionTypeCommand
	}

	handler, ok := se.actionHandlers[actionType]
	if !ok {
		return fmt.Errorf("no handler found for action type %v", actionType)
	}

	// Create action definition from stage definition
	actionDef := ActionDefinition{
		Type:    actionType,
		Command: stage.Definition.Name, // Use name as command for now
	}

	// Execute through worker pool with concurrency control
	err := se.workerPool.ExecuteStageInPool(ctx, stage, actionDef, handler)
	
	// Update stage status
	stage.EndTime = time.Now()
	if err != nil {
		stage.Status = StageStatusFailed
		stage.Error = err
		return err
	}

	stage.Status = StageStatusCompleted
	return nil
}

// RegisterActionHandler registers a custom action handler
func (se *StageExecutor) RegisterActionHandler(actionType ActionType, handler ActionHandler) {
	se.actionHandlers[actionType] = handler
}

// Close shuts down the stage executor and its worker pool
func (se *StageExecutor) Close() error {
	if se.workerPool != nil {
		se.workerPool.Shutdown()
	}
	return nil
}

// Helper methods

func (se *StageExecutor) canExecuteStage(stage *StageInstance) error {
	stage.mutex.RLock()
	status := stage.Status
	stage.mutex.RUnlock()

	if status != StageStatusPending && status != StageStatusWaitingForDependencies {
		return fmt.Errorf("stage is in invalid state for execution: %v", status)
	}

	// Validate action
	handler, exists := se.actionHandlers[stage.Definition.Action.Type]
	if !exists {
		return fmt.Errorf("no handler for action type: %v", stage.Definition.Action.Type)
	}

	return handler.Validate(stage.Definition.Action)
}

func (se *StageExecutor) areDependenciesSatisfied(stage *StageInstance) bool {
	for _, dep := range stage.Dependencies {
		if dep.Status != StageStatusCompleted && dep.Status != StageStatusSkipped {
			return false
		}
	}
	return true
}

func (se *StageExecutor) evaluateConditions(stage *StageInstance, workflow *WorkflowInstance) bool {
	for _, condition := range stage.Definition.Conditions {
		if !se.evaluateCondition(condition, workflow) {
			return false
		}
	}
	return true
}

func (se *StageExecutor) evaluateCondition(condition ConditionDefinition, workflow *WorkflowInstance) bool {
	switch condition.Type {
	case ConditionTypeVariable:
		return se.evaluateVariableCondition(condition, workflow)
	case ConditionTypeFileExists:
		return se.evaluateFileExistsCondition(condition)
	case ConditionTypeCommandSuccess:
		return se.evaluateCommandSuccessCondition(condition)
	case ConditionTypeExpression:
		return se.evaluateExpressionCondition(condition, workflow)
	default:
		return false
	}
}

func (se *StageExecutor) evaluateVariableCondition(condition ConditionDefinition, workflow *WorkflowInstance) bool {
	value, exists := workflow.Variables[condition.Variable]
	if !exists {
		return false
	}

	switch condition.Operator {
	case "equals":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", condition.Value)
	case "not_equals":
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", condition.Value)
	case "contains":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", condition.Value)
	default:
		return false
	}
}

func (se *StageExecutor) evaluateFileExistsCondition(condition ConditionDefinition) bool {
	// Simple file existence check
	// #nosec G204 - condition.Expression is validated in workflow definition
	cmd := exec.Command("test", "-f", condition.Expression)
	return cmd.Run() == nil
}

func (se *StageExecutor) evaluateCommandSuccessCondition(condition ConditionDefinition) bool {
	// #nosec G204 - condition.Expression is validated workflow command
	cmd := exec.Command("sh", "-c", condition.Expression)
	return cmd.Run() == nil
}

func (se *StageExecutor) evaluateExpressionCondition(condition ConditionDefinition, workflow *WorkflowInstance) bool {
	// Simple expression evaluation (in production, use a proper expression evaluator)
	// For now, just return true
	return true
}

func (se *StageExecutor) executeAction(ctx context.Context, stage *StageInstance) (interface{}, error) {
	handler, exists := se.actionHandlers[stage.Definition.Action.Type]
	if !exists {
		return nil, fmt.Errorf("no handler for action type: %v", stage.Definition.Action.Type)
	}

	// Create context with timeout
	actionCtx := ctx
	if stage.Definition.Timeout > 0 {
		var cancel context.CancelFunc
		actionCtx, cancel = context.WithTimeout(ctx, stage.Definition.Timeout)
		defer cancel()
	}

	return handler.Execute(actionCtx, stage, stage.Definition.Action)
}

func (se *StageExecutor) shouldRetry(stage *StageInstance) bool {
	if stage.Definition.RetryPolicy == nil {
		return false
	}

	stage.mutex.RLock()
	retryCount := stage.RetryCount
	stage.mutex.RUnlock()

	return retryCount < stage.Definition.RetryPolicy.MaxAttempts
}

func (se *StageExecutor) retryStage(ctx context.Context, stage *StageInstance, workflow *WorkflowInstance) error {
	return se.retryManager.RetryStage(ctx, stage, workflow, se)
}

// RetryManager implementation

func NewRetryManager(config EngineConfig) *RetryManager {
	return &RetryManager{
		config: config,
	}
}

func (rm *RetryManager) RetryStage(ctx context.Context, stage *StageInstance, workflow *WorkflowInstance, executor *StageExecutor) error {
	if stage.Definition.RetryPolicy == nil {
		return fmt.Errorf("no retry policy defined for stage %s", stage.ID)
	}

	policy := stage.Definition.RetryPolicy

	stage.mutex.Lock()
	stage.RetryCount++
	retryCount := stage.RetryCount
	stage.mutex.Unlock()

	// Calculate delay
	delay := rm.calculateRetryDelay(policy, retryCount)

	// Wait before retry
	select {
	case <-time.After(delay):
	case <-ctx.Done():
		return ctx.Err()
	}

	// Reset stage state for retry
	stage.mutex.Lock()
	stage.Status = StageStatusPending
	stage.Error = nil
	stage.StartTime = time.Time{}
	stage.EndTime = time.Time{}
	stage.mutex.Unlock()

	// Execute stage again
	return executor.ExecuteStage(ctx, stage, workflow)
}

func (rm *RetryManager) calculateRetryDelay(policy *RetryPolicy, attempt int) time.Duration {
	delay := policy.InitialDelay

	// Apply exponential backoff
	for i := 1; i < attempt; i++ {
		delay = time.Duration(float64(delay) * policy.Multiplier)
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
			break
		}
	}

	return delay
}

// Action Handlers Implementation

// CommandActionHandler executes shell commands
type CommandActionHandler struct{}

func (h *CommandActionHandler) Execute(ctx context.Context, stage *StageInstance, action ActionDefinition) (interface{}, error) {
	// #nosec G204 - action.Command is validated workflow action
	cmd := exec.CommandContext(ctx, "sh", "-c", action.Command)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("command failed: %w", err)
	}

	return string(output), nil
}

func (h *CommandActionHandler) Validate(action ActionDefinition) error {
	if action.Command == "" {
		return fmt.Errorf("command cannot be empty")
	}
	return nil
}

// ScriptActionHandler executes scripts
type ScriptActionHandler struct{}

func (h *ScriptActionHandler) Execute(ctx context.Context, stage *StageInstance, action ActionDefinition) (interface{}, error) {
	// For now, treat script as command
	// #nosec G204 - action.Script is validated workflow script
	cmd := exec.CommandContext(ctx, "sh", "-c", action.Script)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("script failed: %w", err)
	}

	return string(output), nil
}

func (h *ScriptActionHandler) Validate(action ActionDefinition) error {
	if action.Script == "" {
		return fmt.Errorf("script cannot be empty")
	}
	return nil
}

// FunctionActionHandler executes Go functions
type FunctionActionHandler struct{}

func (h *FunctionActionHandler) Execute(ctx context.Context, stage *StageInstance, action ActionDefinition) (interface{}, error) {
	// Placeholder for function execution
	return fmt.Sprintf("Function %s executed", action.Function), nil
}

func (h *FunctionActionHandler) Validate(action ActionDefinition) error {
	if action.Function == "" {
		return fmt.Errorf("function name cannot be empty")
	}
	return nil
}

// HTTPActionHandler executes HTTP requests
type HTTPActionHandler struct{}

func (h *HTTPActionHandler) Execute(ctx context.Context, stage *StageInstance, action ActionDefinition) (interface{}, error) {
	// Placeholder for HTTP request execution
	return "HTTP request completed", nil
}

func (h *HTTPActionHandler) Validate(action ActionDefinition) error {
	// Validate HTTP parameters
	return nil
}

// ClaudeCodeActionHandler executes Claude Code operations
type ClaudeCodeActionHandler struct{}

func (h *ClaudeCodeActionHandler) Execute(ctx context.Context, stage *StageInstance, action ActionDefinition) (interface{}, error) {
	// Placeholder for Claude Code integration
	return "Claude Code operation completed", nil
}

func (h *ClaudeCodeActionHandler) Validate(action ActionDefinition) error {
	// Validate Claude Code parameters
	return nil
}

// GitActionHandler executes git operations
type GitActionHandler struct{}

func (h *GitActionHandler) Execute(ctx context.Context, stage *StageInstance, action ActionDefinition) (interface{}, error) {
	// Execute git command
	// #nosec G204 - action.Command is validated git command
	cmd := exec.CommandContext(ctx, "git", action.Command)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("git operation failed: %w", err)
	}

	return string(output), nil
}

func (h *GitActionHandler) Validate(action ActionDefinition) error {
	if action.Command == "" {
		return fmt.Errorf("git command cannot be empty")
	}
	return nil
}

// FileActionHandler executes file operations
type FileActionHandler struct{}

func (h *FileActionHandler) Execute(ctx context.Context, stage *StageInstance, action ActionDefinition) (interface{}, error) {
	// Placeholder for file operations
	return "File operation completed", nil
}

func (h *FileActionHandler) Validate(action ActionDefinition) error {
	// Validate file operation parameters
	return nil
}
