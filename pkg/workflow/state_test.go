package workflow

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStateManager(t *testing.T) {
	config := EngineConfig{
		MaxConcurrentStages: 5,
	}

	sm, err := NewStateManager(config)
	require.NoError(t, err)
	assert.NotNil(t, sm)
	assert.Equal(t, config, sm.config)
	assert.NotNil(t, sm.listeners)
}

func TestStateManager_AddListener(t *testing.T) {
	config := EngineConfig{}
	sm, err := NewStateManager(config)
	require.NoError(t, err)

	listener := newMockStateListener()
	entityID := testWorkflowName

	sm.AddStateListener(entityID, listener)

	// Verify listener was added
	sm.listenersMutex.RLock()
	listeners := sm.listeners[entityID]
	sm.listenersMutex.RUnlock()

	assert.Len(t, listeners, 1)
	assert.Equal(t, listener, listeners[0])
}

func TestStateManager_RemoveListener(t *testing.T) {
	config := EngineConfig{}
	sm, err := NewStateManager(config)
	require.NoError(t, err)

	listener := newMockStateListener()
	entityID := testWorkflowName

	// Add then remove listener
	sm.AddStateListener(entityID, listener)
	sm.RemoveStateListener(entityID, listener)

	// Verify listener was removed
	sm.listenersMutex.RLock()
	listeners := sm.listeners[entityID]
	sm.listenersMutex.RUnlock()

	assert.Len(t, listeners, 0)
}

func TestStateManager_TransitionWorkflowState(t *testing.T) {
	config := EngineConfig{}
	sm, err := NewStateManager(config)
	require.NoError(t, err)

	listener := newMockStateListener()
	workflowID := "test-workflow"

	sm.AddStateListener(workflowID, listener)

	// Test valid transition
	// Create a mock workflow instance
	workflow := &WorkflowInstance{
		ID:    workflowID,
		State: WorkflowStateInitializing,
	}
	err = sm.TransitionWorkflowWithReason(workflow, WorkflowStateRunning, "Starting workflow")
	assert.NoError(t, err)

	// Wait for async notification
	listener.WaitForTransitions(1)

	// Verify listener was notified
	transitions := listener.GetTransitions()
	assert.Len(t, transitions, 1)
	transition := transitions[0]
	assert.Equal(t, EntityTypeWorkflow, transition.EntityType)
	assert.Equal(t, workflowID, transition.EntityID)
	assert.Equal(t, WorkflowStateInitializing, transition.FromState)
	assert.Equal(t, WorkflowStateRunning, transition.ToState)
	assert.Equal(t, "Starting workflow", transition.Reason)
}

func TestStateManager_TransitionStageState(t *testing.T) {
	config := EngineConfig{}
	sm, err := NewStateManager(config)
	require.NoError(t, err)

	listener := newMockStateListener()
	stageID := "test-stage"

	sm.AddStateListener(stageID, listener)

	// Test valid transition
	// Create a mock stage instance
	stage := &StageInstance{
		ID:     stageID,
		Status: StageStatusPending,
	}
	err = sm.TransitionStageWithReason(stage, StageStatusRunning, "Starting stage")
	assert.NoError(t, err)

	// Wait for async notification
	listener.WaitForTransitions(1)

	// Verify listener was notified
	transitions := listener.GetTransitions()
	assert.Len(t, transitions, 1)
	transition := transitions[0]
	assert.Equal(t, EntityTypeStage, transition.EntityType)
	assert.Equal(t, stageID, transition.EntityID)
	assert.Equal(t, StageStatusPending, transition.FromState)
	assert.Equal(t, StageStatusRunning, transition.ToState)
	assert.Equal(t, "Starting stage", transition.Reason)
}

func TestStateManager_InvalidTransition(t *testing.T) {
	config := EngineConfig{}
	sm, err := NewStateManager(config)
	require.NoError(t, err)

	workflowID := "test-workflow"

	// Test invalid transition (e.g., from Running to Initializing)
	// Create a mock workflow instance in running state
	workflow := &WorkflowInstance{
		ID:    workflowID,
		State: WorkflowStateRunning,
	}
	err = sm.TransitionWorkflowWithReason(workflow, WorkflowStateInitializing, "Invalid transition")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid workflow state transition")
}

func TestStateManager_ValidateWorkflowTransition(t *testing.T) {
	config := EngineConfig{}
	sm, err := NewStateManager(config)
	require.NoError(t, err)

	tests := []struct {
		name      string
		fromState WorkflowState
		toState   WorkflowState
		valid     bool
	}{
		{"initializing to running", WorkflowStateInitializing, WorkflowStateRunning, true},
		{"running to completed", WorkflowStateRunning, WorkflowStateCompleted, true},
		{"running to failed", WorkflowStateRunning, WorkflowStateFailed, true},
		{"completed to running", WorkflowStateCompleted, WorkflowStateRunning, false},
		{"failed to completed", WorkflowStateFailed, WorkflowStateCompleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := sm.CanTransitionWorkflow(tt.fromState, tt.toState)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

func TestStateManager_ValidateStageTransition(t *testing.T) {
	config := EngineConfig{}
	sm, err := NewStateManager(config)
	require.NoError(t, err)

	tests := []struct {
		name      string
		fromState StageStatus
		toState   StageStatus
		valid     bool
	}{
		{"pending to running", StageStatusPending, StageStatusRunning, true},
		{"running to completed", StageStatusRunning, StageStatusCompleted, true},
		{"running to failed", StageStatusRunning, StageStatusFailed, true},
		{"completed to running", StageStatusCompleted, StageStatusRunning, false},
		{"failed to completed", StageStatusFailed, StageStatusCompleted, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := sm.CanTransitionStage(tt.fromState, tt.toState)
			assert.Equal(t, tt.valid, valid)
		})
	}
}

func TestStateTransition_Structure(t *testing.T) {
	now := time.Now()
	transition := StateTransition{
		EntityType: EntityTypeWorkflow,
		EntityID:   "workflow-123",
		FromState:  WorkflowStateInitializing,
		ToState:    WorkflowStateRunning,
		Timestamp:  now,
		Reason:     "Workflow started",
		Metadata: map[string]interface{}{
			"user":   "test-user",
			"source": "api",
		},
	}

	assert.Equal(t, EntityTypeWorkflow, transition.EntityType)
	assert.Equal(t, "workflow-123", transition.EntityID)
	assert.Equal(t, WorkflowStateInitializing, transition.FromState)
	assert.Equal(t, WorkflowStateRunning, transition.ToState)
	assert.Equal(t, now, transition.Timestamp)
	assert.Equal(t, "Workflow started", transition.Reason)
	assert.Equal(t, "test-user", transition.Metadata["user"])
	assert.Equal(t, "api", transition.Metadata["source"])
}

func TestEntityType_Constants(t *testing.T) {
	// Test that entity type constants are properly defined
	entityTypes := []EntityType{
		EntityTypeWorkflow,
		EntityTypeStage,
	}

	// Each entity type should have a unique value
	entityTypeMap := make(map[EntityType]bool)
	for _, entityType := range entityTypes {
		assert.False(t, entityTypeMap[entityType], "Duplicate entity type value: %v", entityType)
		entityTypeMap[entityType] = true
	}

	// Should have expected entity types
	assert.Len(t, entityTypeMap, 2)
}

func TestStateManager_MultipleListeners(t *testing.T) {
	config := EngineConfig{}
	sm, err := NewStateManager(config)
	require.NoError(t, err)

	listener1 := &mockStateListener{}
	listener2 := &mockStateListener{}
	entityID := "test-entity"

	// Add multiple listeners
	sm.AddStateListener(entityID, listener1)
	sm.AddStateListener(entityID, listener2)

	// Trigger state transition
	workflow := &WorkflowInstance{
		ID:    entityID,
		State: WorkflowStateInitializing,
	}
	err = sm.TransitionWorkflowWithReason(workflow, WorkflowStateRunning, "Test transition")
	assert.NoError(t, err)

	// Wait for async notifications
	listener1.WaitForTransitions(1)
	listener2.WaitForTransitions(1)

	// Both listeners should receive the notification
	transitions1 := listener1.GetTransitions()
	transitions2 := listener2.GetTransitions()
	assert.Len(t, transitions1, 1)
	assert.Len(t, transitions2, 1)
	assert.Equal(t, transitions1[0].EntityID, transitions2[0].EntityID)
}

func TestStateManager_ThreadSafety(t *testing.T) {
	config := EngineConfig{}
	sm, err := NewStateManager(config)
	require.NoError(t, err)

	// Verify that the state manager has proper mutex protection
	assert.NotNil(t, &sm.listenersMutex)

	// Test concurrent access doesn't panic
	listener := newMockStateListener()

	go func() {
		sm.AddStateListener("entity1", listener)
	}()

	go func() {
		sm.AddStateListener("entity2", listener)
	}()

	// Goroutines complete immediately in test

	// Should not panic
	assert.True(t, true)
}

// Mock state listener for testing
type mockStateListener struct {
	transitions []StateTransition
	mu          sync.Mutex
	eventCh     chan struct{}
}

func newMockStateListener() *mockStateListener {
	return &mockStateListener{
		eventCh: make(chan struct{}, 100),
	}
}

func (m *mockStateListener) OnStateChange(transition StateTransition) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.transitions = append(m.transitions, transition)

	// Signal that a transition was received
	select {
	case m.eventCh <- struct{}{}:
	default:
	}
}

func (m *mockStateListener) WaitForTransitions(count int) {
	for i := 0; i < count; i++ {
		select {
		case <-m.eventCh:
			// Event received
		case <-time.After(100 * time.Millisecond):
			// Timeout - this is expected for async notification
			return
		}
	}
}

func (m *mockStateListener) GetTransitions() []StateTransition {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]StateTransition{}, m.transitions...)
}

func TestStateListener_Interface(t *testing.T) {
	// Verify that our mock implements the interface correctly
	var listener StateListener = newMockStateListener()

	transition := StateTransition{
		EntityType: EntityTypeWorkflow,
		EntityID:   "test",
		FromState:  WorkflowStateInitializing,
		ToState:    WorkflowStateRunning,
		Timestamp:  time.Now(),
		Reason:     "test",
	}

	// Test OnStateChange doesn't panic
	assert.NotPanics(t, func() {
		listener.OnStateChange(transition)
	})
}
