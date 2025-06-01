package workflow

import (
	"fmt"
	"sync"
	"time"
)

// StateManager manages workflow and stage state transitions
type StateManager struct {
	config           EngineConfig
	transitions      map[WorkflowState][]WorkflowState
	stageTransitions map[StageStatus][]StageStatus
	listeners        map[string][]StateListener
	listenersMutex   sync.RWMutex
}

// StateListener receives state change notifications
type StateListener interface {
	OnStateChange(transition StateTransition)
}

// StateTransition represents a state change
type StateTransition struct {
	EntityType EntityType
	EntityID   string
	FromState  interface{}
	ToState    interface{}
	Timestamp  time.Time
	Reason     string
	Metadata   map[string]interface{}
}

// EntityType defines what entity had a state change
type EntityType int

const (
	EntityTypeWorkflow EntityType = iota
	EntityTypeStage
)

// NewStateManager creates a new state manager
func NewStateManager(config EngineConfig) (*StateManager, error) {
	sm := &StateManager{
		config:    config,
		listeners: make(map[string][]StateListener),
	}

	// Initialize valid transitions
	sm.initializeTransitions()

	return sm, nil
}

// initializeTransitions sets up valid state transitions
func (sm *StateManager) initializeTransitions() {
	// Workflow state transitions
	sm.transitions = map[WorkflowState][]WorkflowState{
		WorkflowStateInitializing: {
			WorkflowStateRunning,
			WorkflowStateFailed,
			WorkflowStateCancelled,
		},
		WorkflowStateRunning: {
			WorkflowStatePaused,
			WorkflowStateWaitingForInput,
			WorkflowStateCompleted,
			WorkflowStateFailed,
			WorkflowStateCancelled,
			WorkflowStateAborted,
		},
		WorkflowStatePaused: {
			WorkflowStateRunning,
			WorkflowStateCancelled,
			WorkflowStateAborted,
		},
		WorkflowStateWaitingForInput: {
			WorkflowStateRunning,
			WorkflowStateFailed,
			WorkflowStateCancelled,
			WorkflowStateAborted,
		},
		WorkflowStateCompleted: {
			// Terminal state - no transitions
		},
		WorkflowStateFailed: {
			WorkflowStateRunning, // For retry
		},
		WorkflowStateCancelled: {
			// Terminal state - no transitions
		},
		WorkflowStateAborted: {
			// Terminal state - no transitions
		},
	}

	// Stage state transitions
	sm.stageTransitions = map[StageStatus][]StageStatus{
		StageStatusPending: {
			StageStatusRunning,
			StageStatusWaitingForDependencies,
			StageStatusSkipped,
			StageStatusCancelled,
		},
		StageStatusWaitingForDependencies: {
			StageStatusRunning,
			StageStatusSkipped,
			StageStatusCancelled,
		},
		StageStatusRunning: {
			StageStatusCompleted,
			StageStatusFailed,
			StageStatusWaitingForInput,
			StageStatusCancelled,
		},
		StageStatusWaitingForInput: {
			StageStatusRunning,
			StageStatusFailed,
			StageStatusCancelled,
		},
		StageStatusCompleted: {
			// Terminal state - no transitions
		},
		StageStatusFailed: {
			StageStatusRunning, // For retry
			StageStatusSkipped,
		},
		StageStatusSkipped: {
			// Terminal state - no transitions
		},
		StageStatusCancelled: {
			// Terminal state - no transitions
		},
	}
}

// TransitionWorkflow transitions a workflow to a new state
func (sm *StateManager) TransitionWorkflow(workflow *WorkflowInstance, newState WorkflowState) error {
	return sm.TransitionWorkflowWithReason(workflow, newState, "")
}

// TransitionWorkflowWithReason transitions a workflow with a reason
func (sm *StateManager) TransitionWorkflowWithReason(workflow *WorkflowInstance, newState WorkflowState, reason string) error {
	workflow.stateMutex.Lock()
	defer workflow.stateMutex.Unlock()

	oldState := workflow.State

	// Check if transition is valid
	if !sm.isValidWorkflowTransition(oldState, newState) {
		return fmt.Errorf("invalid workflow state transition from %v to %v", oldState, newState)
	}

	// Update state
	workflow.State = newState

	// Set end time for terminal states
	if sm.isTerminalWorkflowState(newState) {
		workflow.EndTime = time.Now()
	}

	// Create transition event
	transition := StateTransition{
		EntityType: EntityTypeWorkflow,
		EntityID:   workflow.ID,
		FromState:  oldState,
		ToState:    newState,
		Timestamp:  time.Now(),
		Reason:     reason,
		Metadata:   make(map[string]interface{}),
	}

	// Add to workflow events
	workflow.eventsMutex.Lock()
	workflow.Events = append(workflow.Events, WorkflowEvent{
		Type:       EventTypeStateChanged,
		WorkflowID: workflow.ID,
		Timestamp:  time.Now(),
		Data: map[string]interface{}{
			"from_state": oldState.String(),
			"to_state":   newState.String(),
			"reason":     reason,
		},
	})
	workflow.eventsMutex.Unlock()

	// Notify listeners
	sm.notifyListeners(transition)

	return nil
}

// TransitionStage transitions a stage to a new status
func (sm *StateManager) TransitionStage(stage *StageInstance, newStatus StageStatus) error {
	return sm.TransitionStageWithReason(stage, newStatus, "")
}

// TransitionStageWithReason transitions a stage with a reason
func (sm *StateManager) TransitionStageWithReason(stage *StageInstance, newStatus StageStatus, reason string) error {
	oldStatus := stage.Status

	// Check if transition is valid
	if !sm.isValidStageTransition(oldStatus, newStatus) {
		return fmt.Errorf("invalid stage state transition from %v to %v", oldStatus, newStatus)
	}

	// Update status
	stage.Status = newStatus

	// Set timestamps
	if newStatus == StageStatusRunning && stage.StartTime.IsZero() {
		stage.StartTime = time.Now()
	}
	if sm.isTerminalStageStatus(newStatus) {
		stage.EndTime = time.Now()
	}

	// Create transition event
	transition := StateTransition{
		EntityType: EntityTypeStage,
		EntityID:   stage.ID,
		FromState:  oldStatus,
		ToState:    newStatus,
		Timestamp:  time.Now(),
		Reason:     reason,
		Metadata:   make(map[string]interface{}),
	}

	// Notify listeners
	sm.notifyListeners(transition)

	return nil
}

// CanTransitionWorkflow checks if a workflow can transition to a new state
func (sm *StateManager) CanTransitionWorkflow(currentState, newState WorkflowState) bool {
	return sm.isValidWorkflowTransition(currentState, newState)
}

// CanTransitionStage checks if a stage can transition to a new status
func (sm *StateManager) CanTransitionStage(currentStatus, newStatus StageStatus) bool {
	return sm.isValidStageTransition(currentStatus, newStatus)
}

// AddStateListener adds a state change listener
func (sm *StateManager) AddStateListener(entityID string, listener StateListener) {
	sm.listenersMutex.Lock()
	defer sm.listenersMutex.Unlock()

	sm.listeners[entityID] = append(sm.listeners[entityID], listener)
}

// RemoveStateListener removes a state change listener
func (sm *StateManager) RemoveStateListener(entityID string, listener StateListener) {
	sm.listenersMutex.Lock()
	defer sm.listenersMutex.Unlock()

	listeners, exists := sm.listeners[entityID]
	if !exists {
		return
	}

	// Remove listener from slice
	for i, l := range listeners {
		if l == listener {
			sm.listeners[entityID] = append(listeners[:i], listeners[i+1:]...)
			break
		}
	}

	// Clean up empty slice
	if len(sm.listeners[entityID]) == 0 {
		delete(sm.listeners, entityID)
	}
}

// GetWorkflowStateHistory returns state change history for a workflow
func (sm *StateManager) GetWorkflowStateHistory(workflow *WorkflowInstance) []WorkflowEvent {
	workflow.eventsMutex.RLock()
	defer workflow.eventsMutex.RUnlock()

	// Filter for state change events
	var stateEvents []WorkflowEvent
	for _, event := range workflow.Events {
		if event.Type == EventTypeStateChanged {
			stateEvents = append(stateEvents, event)
		}
	}

	return stateEvents
}

// IsTerminalState checks if a workflow state is terminal
func (sm *StateManager) IsTerminalState(state WorkflowState) bool {
	return sm.isTerminalWorkflowState(state)
}

// IsTerminalStageStatus checks if a stage status is terminal
func (sm *StateManager) IsTerminalStageStatus(status StageStatus) bool {
	return sm.isTerminalStageStatus(status)
}

// Helper methods

func (sm *StateManager) isValidWorkflowTransition(from, to WorkflowState) bool {
	validTransitions, exists := sm.transitions[from]
	if !exists {
		return false
	}

	for _, validTo := range validTransitions {
		if validTo == to {
			return true
		}
	}

	return false
}

func (sm *StateManager) isValidStageTransition(from, to StageStatus) bool {
	validTransitions, exists := sm.stageTransitions[from]
	if !exists {
		return false
	}

	for _, validTo := range validTransitions {
		if validTo == to {
			return true
		}
	}

	return false
}

func (sm *StateManager) isTerminalWorkflowState(state WorkflowState) bool {
	switch state {
	case WorkflowStateCompleted, WorkflowStateFailed, WorkflowStateCancelled, WorkflowStateAborted:
		return true
	default:
		return false
	}
}

func (sm *StateManager) isTerminalStageStatus(status StageStatus) bool {
	switch status {
	case StageStatusCompleted, StageStatusFailed, StageStatusSkipped, StageStatusCancelled:
		return true
	default:
		return false
	}
}

func (sm *StateManager) notifyListeners(transition StateTransition) {
	sm.listenersMutex.RLock()
	defer sm.listenersMutex.RUnlock()

	// Notify specific entity listeners
	if listeners, exists := sm.listeners[transition.EntityID]; exists {
		for _, listener := range listeners {
			go listener.OnStateChange(transition)
		}
	}

	// Notify global listeners (empty entity ID)
	if listeners, exists := sm.listeners[""]; exists {
		for _, listener := range listeners {
			go listener.OnStateChange(transition)
		}
	}
}

// StateValidator provides state validation utilities
type StateValidator struct {
	stateManager *StateManager
}

// NewStateValidator creates a new state validator
func NewStateValidator(stateManager *StateManager) *StateValidator {
	return &StateValidator{
		stateManager: stateManager,
	}
}

// ValidateWorkflowState validates a workflow's current state
func (sv *StateValidator) ValidateWorkflowState(workflow *WorkflowInstance) error {
	workflow.stateMutex.RLock()
	defer workflow.stateMutex.RUnlock()

	state := workflow.State

	// Check for invalid states based on context
	switch state {
	case WorkflowStateRunning:
		// Should have at least one active stage
		hasActiveStage := false
		for _, stage := range workflow.Stages {
			if stage.Status == StageStatusRunning {
				hasActiveStage = true
				break
			}
		}
		if !hasActiveStage {
			return fmt.Errorf("workflow %s is in running state but has no active stages", workflow.ID)
		}

	case WorkflowStateCompleted:
		// All stages should be completed or skipped
		for _, stage := range workflow.Stages {
			if stage.Status != StageStatusCompleted && stage.Status != StageStatusSkipped {
				return fmt.Errorf("workflow %s is marked complete but stage %s is not finished", workflow.ID, stage.ID)
			}
		}

	case WorkflowStateFailed:
		// Should have at least one failed stage
		hasFailedStage := false
		for _, stage := range workflow.Stages {
			if stage.Status == StageStatusFailed {
				hasFailedStage = true
				break
			}
		}
		if !hasFailedStage {
			return fmt.Errorf("workflow %s is marked failed but has no failed stages", workflow.ID)
		}
	}

	return nil
}

// ValidateStageStatus validates a stage's current status
func (sv *StateValidator) ValidateStageStatus(stage *StageInstance) error {
	status := stage.Status

	// Check for invalid statuses based on context
	switch status {
	case StageStatusRunning:
		if stage.StartTime.IsZero() {
			return fmt.Errorf("stage %s is running but has no start time", stage.ID)
		}

	case StageStatusCompleted:
		if stage.StartTime.IsZero() || stage.EndTime.IsZero() {
			return fmt.Errorf("stage %s is completed but missing start/end time", stage.ID)
		}

	case StageStatusFailed:
		if stage.Error == nil {
			return fmt.Errorf("stage %s is marked failed but has no error", stage.ID)
		}

	case StageStatusWaitingForDependencies:
		// Check if dependencies are actually incomplete
		hasIncompleteDeps := false
		for _, dep := range stage.Dependencies {
			if dep.Status != StageStatusCompleted && dep.Status != StageStatusSkipped {
				hasIncompleteDeps = true
				break
			}
		}
		if !hasIncompleteDeps {
			return fmt.Errorf("stage %s is waiting for dependencies but all dependencies are complete", stage.ID)
		}
	}

	return nil
}

// GetNextValidStates returns valid next states for a workflow
func (sv *StateValidator) GetNextValidStates(currentState WorkflowState) []WorkflowState {
	return sv.stateManager.transitions[currentState]
}

// GetNextValidStatuses returns valid next statuses for a stage
func (sv *StateValidator) GetNextValidStatuses(currentStatus StageStatus) []StageStatus {
	return sv.stateManager.stageTransitions[currentStatus]
}
