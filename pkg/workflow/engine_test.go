package workflow

import (
	"context"
	"testing"
	"time"
)

// Test constants
const (
	testWorkflowName = "test-workflow"
)

func TestNewEngine(t *testing.T) {
	config := EngineConfig{
		MaxConcurrentWorkflows: 5,
		MaxConcurrentStages:    10,
		DefaultTimeout:         30 * time.Second,
		RetryAttempts:          2,
		RetryDelay:             1 * time.Second,
		PersistenceEnabled:     false,
		MetricsEnabled:         true,
		EventBufferSize:        100,
	}

	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer func() { _ = engine.Shutdown(context.Background()) }()

	if engine.config.MaxConcurrentWorkflows != 5 {
		t.Errorf("Expected max concurrent workflows 5, got %d", engine.config.MaxConcurrentWorkflows)
	}

	if engine.config.MetricsEnabled != true {
		t.Error("Expected metrics to be enabled")
	}
}

func TestEngineDefaults(t *testing.T) {
	config := EngineConfig{}
	engine, err := NewEngine(config)
	if err != nil {
		t.Fatalf("Failed to create engine with defaults: %v", err)
	}
	defer func() { _ = engine.Shutdown(context.Background()) }()

	if engine.config.MaxConcurrentWorkflows != DefaultMaxConcurrentWorkflows {
		t.Errorf("Expected default max concurrent workflows %d, got %d",
			DefaultMaxConcurrentWorkflows, engine.config.MaxConcurrentWorkflows)
	}

	if engine.config.DefaultTimeout != DefaultWorkflowTimeout {
		t.Errorf("Expected default timeout %v, got %v",
			DefaultWorkflowTimeout, engine.config.DefaultTimeout)
	}
}

func TestWorkflowLifecycle(t *testing.T) {
	engine, err := NewEngine(EngineConfig{
		MaxConcurrentWorkflows: 2,
		PersistenceEnabled:     false,
		MetricsEnabled:         false,
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer func() { _ = engine.Shutdown(context.Background()) }()

	// Create a simple workflow definition
	definition := &WorkflowDefinition{
		Name:        testWorkflowName,
		Version:     "1.0",
		Description: "A test workflow",
		Stages: []StageDefinition{
			{
				Name: "stage1",
				Type: StageTypeAction,
				Action: ActionDefinition{
					Type:    ActionTypeCommand,
					Command: "echo 'Hello World'",
				},
			},
		},
	}

	// Start workflow
	instance, err := engine.StartWorkflow(context.Background(), definition, nil)
	if err != nil {
		t.Fatalf("Failed to start workflow: %v", err)
	}

	if instance.Definition.Name != testWorkflowName {
		t.Errorf("Expected workflow name 'test-workflow', got '%s'", instance.Definition.Name)
	}

	if instance.State != WorkflowStateInitializing {
		t.Errorf("Expected initial state %v, got %v", WorkflowStateInitializing, instance.State)
	}

	// Get workflow status immediately (while it's still active)
	status, err := engine.GetWorkflowStatus(instance.ID)
	if err != nil {
		t.Fatalf("Failed to get workflow status: %v", err)
	}

	if status.ID != instance.ID {
		t.Errorf("Expected status ID %s, got %s", instance.ID, status.ID)
	}

	// Stop workflow
	err = engine.StopWorkflow(instance.ID, "test completion")
	if err != nil {
		t.Fatalf("Failed to stop workflow: %v", err)
	}
}

func TestWorkflowStates(t *testing.T) {
	tests := []struct {
		state    WorkflowState
		expected string
	}{
		{WorkflowStateInitializing, "initializing"},
		{WorkflowStateRunning, "running"},
		{WorkflowStatePaused, "paused"},
		{WorkflowStateCompleted, "completed"},
		{WorkflowStateFailed, "failed"},
		{WorkflowStateCancelled, "canceled"},
	}

	for _, test := range tests {
		if test.state.String() != test.expected {
			t.Errorf("Expected state %s, got %s", test.expected, test.state.String())
		}
	}
}

func TestStageStatuses(t *testing.T) {
	tests := []struct {
		status   StageStatus
		expected string
	}{
		{StageStatusPending, "pending"},
		{StageStatusRunning, "running"},
		{StageStatusCompleted, "completed"},
		{StageStatusFailed, "failed"},
		{StageStatusSkipped, "skipped"},
		{StageStatusCancelled, "canceled"},
	}

	for _, test := range tests {
		if test.status.String() != test.expected {
			t.Errorf("Expected status %s, got %s", test.expected, test.status.String())
		}
	}
}

func TestWorkflowConcurrencyLimit(t *testing.T) {
	engine, err := NewEngine(EngineConfig{
		MaxConcurrentWorkflows: 1, // Limit to 1 concurrent workflow
		PersistenceEnabled:     false,
		MetricsEnabled:         false,
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer func() { _ = engine.Shutdown(context.Background()) }()

	definition := &WorkflowDefinition{
		Name: "concurrent-test",
		Stages: []StageDefinition{
			{
				Name: "long-running-stage",
				Type: StageTypeAction,
				Action: ActionDefinition{
					Type:    ActionTypeCommand,
					Command: "sleep 1",
				},
			},
		},
	}

	// Start first workflow
	_, err = engine.StartWorkflow(context.Background(), definition, nil)
	if err != nil {
		t.Fatalf("Failed to start first workflow: %v", err)
	}

	// Try to start second workflow - should fail due to limit
	_, err = engine.StartWorkflow(context.Background(), definition, nil)
	if err == nil {
		t.Error("Expected error when exceeding concurrent workflow limit")
	}
}

func TestGenerateWorkflowID(t *testing.T) {
	id1 := generateWorkflowID()
	// Generate second workflow with different name for uniqueness
	id2 := generateWorkflowID()

	if id1 == id2 {
		t.Error("Generated workflow IDs should be unique")
	}

	if id1 == "" || id2 == "" {
		t.Error("Generated workflow IDs should not be empty")
	}
}

func TestListActiveWorkflows(t *testing.T) {
	engine, err := NewEngine(EngineConfig{
		MaxConcurrentWorkflows: 5,
		PersistenceEnabled:     false,
		MetricsEnabled:         false,
	})
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	defer func() { _ = engine.Shutdown(context.Background()) }()

	// Initially should have no active workflows
	workflows := engine.ListActiveWorkflows()
	if len(workflows) != 0 {
		t.Errorf("Expected 0 active workflows, got %d", len(workflows))
	}

	// Start a workflow
	definition := &WorkflowDefinition{
		Name: "list-test",
		Stages: []StageDefinition{
			{
				Name: "test-stage",
				Type: StageTypeAction,
				Action: ActionDefinition{
					Type:    ActionTypeCommand,
					Command: "echo 'test'",
				},
			},
		},
	}

	_, err = engine.StartWorkflow(context.Background(), definition, nil)
	if err != nil {
		t.Fatalf("Failed to start workflow: %v", err)
	}

	// Should now have 1 active workflow
	workflows = engine.ListActiveWorkflows()
	if len(workflows) != 1 {
		t.Errorf("Expected 1 active workflow, got %d", len(workflows))
	}

	if workflows[0].Name != "list-test" {
		t.Errorf("Expected workflow name 'list-test', got '%s'", workflows[0].Name)
	}
}
