package workflow

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStageExecutor(t *testing.T) {
	config := EngineConfig{
		MaxConcurrentStages: 5,
		DefaultTimeout:      time.Minute * 5,
	}

	executor, err := NewStageExecutor(config)
	require.NoError(t, err)
	assert.NotNil(t, executor)
	assert.Equal(t, config, executor.config)
	assert.NotNil(t, executor.actionHandlers)
	assert.NotNil(t, executor.retryManager)
}

func TestStageExecutor_RegisterActionHandler(t *testing.T) {
	config := EngineConfig{}
	executor, err := NewStageExecutor(config)
	require.NoError(t, err)

	// Create a mock action handler
	mockHandler := &mockActionHandler{}

	// Register the handler
	executor.RegisterActionHandler(ActionTypeCommand, mockHandler)

	// Verify handler was registered
	assert.Contains(t, executor.actionHandlers, ActionTypeCommand)
	assert.Equal(t, mockHandler, executor.actionHandlers[ActionTypeCommand])
}

func TestStageExecutor_ExecuteStage(t *testing.T) {
	config := EngineConfig{
		DefaultTimeout: time.Second * 30,
	}
	executor, err := NewStageExecutor(config)
	require.NoError(t, err)

	// Register mock handler
	mockHandler := &mockActionHandler{
		result: "test result",
	}
	executor.RegisterActionHandler(ActionTypeCommand, mockHandler)

	// Create test workflow
	workflow := &WorkflowInstance{
		ID:    "test-workflow",
		State: WorkflowStateRunning,
	}

	// Create test stage
	stage := &StageInstance{
		ID: "test-stage",
		Definition: &StageDefinition{
			Name: "test-stage",
			Action: ActionDefinition{
				Type:       ActionTypeCommand,
				Command:    "echo 'test'",
				Parameters: map[string]interface{}{"key": "value"},
			},
		},
		Status: StageStatusPending,
	}

	ctx := context.Background()
	err = executor.ExecuteStage(ctx, stage, workflow)

	require.NoError(t, err)
	assert.Equal(t, StageStatusCompleted, stage.Status)
	assert.True(t, mockHandler.isExecuteCalled())
	assert.True(t, mockHandler.isValidateCalled())
}

func TestStageExecutor_ExecuteStageWithRetry(t *testing.T) {
	config := EngineConfig{
		DefaultTimeout: time.Second * 30,
		RetryAttempts:  3,
		RetryDelay:     time.Millisecond * 100,
	}
	executor, err := NewStageExecutor(config)
	require.NoError(t, err)

	// Register mock handler that fails first time
	mockHandler := &mockActionHandler{
		failCount: 2, // Fail twice, then succeed
	}
	executor.RegisterActionHandler(ActionTypeCommand, mockHandler)

	// Create test workflow
	workflow := &WorkflowInstance{
		ID:    "test-workflow",
		State: WorkflowStateRunning,
	}

	stage := &StageInstance{
		ID: "retry-stage",
		Definition: &StageDefinition{
			Name: "retry-stage",
			Action: ActionDefinition{
				Type:    ActionTypeCommand,
				Command: "flaky-command",
			},
			RetryPolicy: &RetryPolicy{
				MaxAttempts:  3,
				InitialDelay: time.Millisecond * 100,
				MaxDelay:     time.Second,
				Multiplier:   2.0,
			},
		},
		Status: StageStatusPending,
	}

	ctx := context.Background()
	err = executor.ExecuteStage(ctx, stage, workflow)

	require.NoError(t, err)
	assert.Equal(t, StageStatusCompleted, stage.Status)
	assert.Equal(t, 3, mockHandler.getExecuteCount()) // Called 3 times (2 failures + 1 success)
}

func TestStageExecutor_ExecuteStageFailure(t *testing.T) {
	config := EngineConfig{
		DefaultTimeout: time.Second * 30,
		RetryAttempts:  2,
	}
	executor, err := NewStageExecutor(config)
	require.NoError(t, err)

	// Register mock handler that always fails
	mockHandler := &mockActionHandler{
		alwaysFail: true,
	}
	executor.RegisterActionHandler(ActionTypeCommand, mockHandler)

	// Create test workflow
	workflow := &WorkflowInstance{
		ID:    "test-workflow",
		State: WorkflowStateRunning,
	}

	stage := &StageInstance{
		ID: "failing-stage",
		Definition: &StageDefinition{
			Name: "failing-stage",
			Action: ActionDefinition{
				Type:    ActionTypeCommand,
				Command: "failing-command",
			},
			RetryPolicy: &RetryPolicy{
				MaxAttempts:  2,
				InitialDelay: time.Millisecond * 100,
				MaxDelay:     time.Second,
				Multiplier:   2.0,
			},
		},
		Status: StageStatusPending,
	}

	ctx := context.Background()
	err = executor.ExecuteStage(ctx, stage, workflow)

	require.Error(t, err)
	assert.Equal(t, StageStatusFailed, stage.Status)
	assert.True(t, mockHandler.getExecuteCount() > 1) // Should have retried
}

func TestStageExecutor_ExecuteStageTimeout(t *testing.T) {
	config := EngineConfig{
		DefaultTimeout: time.Millisecond * 10, // Very short timeout
	}
	executor, err := NewStageExecutor(config)
	require.NoError(t, err)

	// Register mock handler that takes too long
	mockHandler := &mockActionHandler{
		delay: time.Millisecond * 100, // Longer than timeout
	}
	executor.RegisterActionHandler(ActionTypeCommand, mockHandler)

	// Create test workflow
	workflow := &WorkflowInstance{
		ID:    "test-workflow",
		State: WorkflowStateRunning,
	}

	stage := &StageInstance{
		ID: "timeout-stage",
		Definition: &StageDefinition{
			Name: "timeout-stage",
			Action: ActionDefinition{
				Type:    ActionTypeCommand,
				Command: "slow-command",
			},
			Timeout: time.Millisecond * 10, // Very short timeout
		},
		Status: StageStatusPending,
	}

	ctx := context.Background()
	err = executor.ExecuteStage(ctx, stage, workflow)

	require.Error(t, err)
	assert.Equal(t, StageStatusFailed, stage.Status)
	assert.Contains(t, err.Error(), "timeout")
}

func TestNewRetryManager(t *testing.T) {
	config := EngineConfig{
		RetryAttempts: 3,
		RetryDelay:    time.Second,
	}
	retryManager := NewRetryManager(config)

	assert.NotNil(t, retryManager)
	assert.Equal(t, config, retryManager.config)
}

func TestActionType_Constants(t *testing.T) {
	// Test that action type constants are properly defined
	actionTypes := []ActionType{
		ActionTypeCommand,
		ActionTypeScript,
		ActionTypeFunction,
		ActionTypeHTTP,
		ActionTypeClaudeCode,
		ActionTypeGitOperation,
		ActionTypeFileOperation,
	}

	// Each action type should have a unique value
	actionTypeMap := make(map[ActionType]bool)
	for _, actionType := range actionTypes {
		assert.False(t, actionTypeMap[actionType], "Duplicate action type value: %v", actionType)
		actionTypeMap[actionType] = true
	}

	// Should have all expected action types
	assert.Len(t, actionTypeMap, len(actionTypes))
}

func TestStageStatus_Constants(t *testing.T) {
	// Test that stage status constants are properly defined
	statuses := []StageStatus{
		StageStatusPending,
		StageStatusRunning,
		StageStatusCompleted,
		StageStatusFailed,
		StageStatusSkipped,
		StageStatusCancelled,
		StageStatusWaitingForDependencies,
		StageStatusWaitingForInput,
	}

	// Each status should have a unique value
	statusMap := make(map[StageStatus]bool)
	for _, status := range statuses {
		assert.False(t, statusMap[status], "Duplicate stage status value: %v", status)
		statusMap[status] = true
	}

	// Should have all expected statuses
	assert.Len(t, statusMap, len(statuses))
}

// Mock action handler for testing
type mockActionHandler struct {
	mu             sync.Mutex
	result         interface{}
	executeCount   int
	validateCount  int
	executeCalled  bool
	validateCalled bool
	failCount      int           // Number of times to fail before succeeding
	alwaysFail     bool          // Always fail
	delay          time.Duration // Delay before completing
}

func (m *mockActionHandler) Execute(ctx context.Context, stage *StageInstance, action ActionDefinition) (interface{}, error) {
	m.mu.Lock()
	m.executeCount++
	m.executeCalled = true

	// Read values we need under lock
	delay := m.delay
	alwaysFail := m.alwaysFail
	shouldFail := m.failCount > 0
	if shouldFail {
		m.failCount--
	}
	result := m.result
	m.mu.Unlock()

	// Add delay if specified (needed for timeout tests)
	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Fail if configured to do so
	if alwaysFail {
		return nil, assert.AnError
	}

	if shouldFail {
		return nil, assert.AnError
	}

	return result, nil
}

func (m *mockActionHandler) Validate(action ActionDefinition) error {
	m.mu.Lock()
	m.validateCount++
	m.validateCalled = true
	m.mu.Unlock()
	return nil
}

// Thread-safe getter methods for test assertions
func (m *mockActionHandler) getExecuteCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.executeCount
}

func (m *mockActionHandler) isExecuteCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.executeCalled
}

func (m *mockActionHandler) isValidateCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.validateCalled
}

func TestStageInstance_StatusTransitions(t *testing.T) {
	stage := &StageInstance{
		Definition: &StageDefinition{
			Name: "test-stage",
		},
		Status: StageStatusPending,
	}

	// Test status transitions
	assert.Equal(t, StageStatusPending, stage.Status)

	stage.Status = StageStatusRunning
	assert.Equal(t, StageStatusRunning, stage.Status)

	stage.Status = StageStatusCompleted
	assert.Equal(t, StageStatusCompleted, stage.Status)
}

func TestActionDefinition_Structure(t *testing.T) {
	action := ActionDefinition{
		Type:     ActionTypeCommand,
		Command:  "echo 'hello'",
		Script:   "#!/bin/bash\necho 'world'",
		Function: "myFunction",
		Timeout:  time.Minute * 5,
		Parameters: map[string]interface{}{
			"env":     map[string]string{"VAR": "value"},
			"workdir": "/tmp",
		},
	}

	assert.Equal(t, ActionTypeCommand, action.Type)
	assert.Equal(t, "echo 'hello'", action.Command)
	assert.Equal(t, "#!/bin/bash\necho 'world'", action.Script)
	assert.Equal(t, "myFunction", action.Function)
	assert.Equal(t, time.Minute*5, action.Timeout)
	assert.Equal(t, "value", action.Parameters["env"].(map[string]string)["VAR"])
}

func TestStageDefinition_Structure(t *testing.T) {
	stageDef := &StageDefinition{
		Name:         "deploy-stage",
		Description:  "Deploy application to production",
		Dependencies: []string{"build-stage", "test-stage"},
		Timeout:      time.Minute * 30,
		Parallel:     false,
		Action: ActionDefinition{
			Type:    ActionTypeCommand,
			Command: "deploy.sh",
		},
		Metadata: map[string]interface{}{
			"environment": "production",
			"replicas":    3,
		},
	}

	assert.Equal(t, "deploy-stage", stageDef.Name)
	assert.Equal(t, "Deploy application to production", stageDef.Description)
	assert.Contains(t, stageDef.Dependencies, "build-stage")
	assert.Contains(t, stageDef.Dependencies, "test-stage")
	assert.Equal(t, time.Minute*30, stageDef.Timeout)
	assert.False(t, stageDef.Parallel)
	assert.Equal(t, ActionTypeCommand, stageDef.Action.Type)
	assert.Equal(t, "production", stageDef.Metadata["environment"])
	assert.Equal(t, 3, stageDef.Metadata["replicas"])
}

func TestStageExecutor_ParallelExecution(t *testing.T) {
	config := EngineConfig{
		MaxConcurrentStages: 3,
		DefaultTimeout:      time.Second * 30,
	}
	executor, err := NewStageExecutor(config)
	require.NoError(t, err)

	// Register mock handler
	mockHandler := &mockActionHandler{
		result: "parallel result",
	}
	executor.RegisterActionHandler(ActionTypeCommand, mockHandler)

	// Create test workflow
	workflow := &WorkflowInstance{
		ID:    "test-workflow",
		State: WorkflowStateRunning,
	}

	// Create multiple stages for parallel execution
	stages := []*StageInstance{
		{
			ID: "stage-1",
			Definition: &StageDefinition{
				Name: "parallel-stage-1",
				Action: ActionDefinition{
					Type:    ActionTypeCommand,
					Command: "echo 'stage1'",
				},
			},
			Status: StageStatusPending,
		},
		{
			ID: "stage-2",
			Definition: &StageDefinition{
				Name: "parallel-stage-2",
				Action: ActionDefinition{
					Type:    ActionTypeCommand,
					Command: "echo 'stage2'",
				},
			},
			Status: StageStatusPending,
		},
	}

	ctx := context.Background()
	err = executor.ExecuteParallelStages(ctx, stages, workflow)

	require.NoError(t, err)

	// All stages should be completed
	for _, stage := range stages {
		assert.Equal(t, StageStatusCompleted, stage.Status)
	}

	// Handler should be called for each stage
	assert.Equal(t, 2, mockHandler.getExecuteCount())
}
