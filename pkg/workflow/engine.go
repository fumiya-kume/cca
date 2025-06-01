// Package workflow provides workflow orchestration and stage-based execution for ccAgents
package workflow

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fumiya-kume/cca/pkg/clock"
)

// Status constants
const (
	statusUnknown   = "unknown"
	statusCompleted = "completed"
)

// Engine manages workflow execution and orchestration
type Engine struct {
	// Core components
	stateManager    *StateManager
	stageExecutor   *StageExecutor
	dependencyGraph *DependencyGraph
	persistence     *PersistenceManager
	metrics         *MetricsCollector
	_ clock.Clock // TODO: implement time-based workflow features

	// Configuration
	config EngineConfig

	// Runtime state
	activeWorkflows map[string]*WorkflowInstance
	workflowsMutex  sync.RWMutex

	// Event handling
	eventBus      *EventBus
	eventHandlers map[EventType][]EventSubscriber
	_ sync.RWMutex // TODO: implement event handler synchronization

	// Worker pools
	stageWorkers *WorkerPool

	// Shutdown handling
	ctx        context.Context
	cancel     context.CancelFunc
	shutdownWG sync.WaitGroup
}

// EngineConfig configures the workflow engine
type EngineConfig struct {
	MaxConcurrentWorkflows int
	MaxConcurrentStages    int
	DefaultTimeout         time.Duration
	RetryAttempts          int
	RetryDelay             time.Duration
	PersistenceEnabled     bool
	MetricsEnabled         bool
	EventBufferSize        int
}

// WorkflowInstance represents a running workflow
type WorkflowInstance struct {
	ID           string
	Definition   *WorkflowDefinition
	State        WorkflowState
	Context      context.Context
	Cancel       context.CancelFunc
	StartTime    time.Time
	EndTime      time.Time
	CurrentStage int
	Stages       []*StageInstance
	Variables    map[string]interface{}
	Metadata     map[string]interface{}

	// Event tracking
	Events      []WorkflowEvent
	eventsMutex sync.RWMutex

	// Error handling
	LastError  error
	ErrorCount int

	// Synchronization
	stageMutex sync.RWMutex
	stateMutex sync.RWMutex
}

// WorkflowDefinition defines a workflow template
type WorkflowDefinition struct {
	Name         string
	Version      string
	Description  string
	Stages       []StageDefinition
	Dependencies map[string][]string
	Variables    map[string]VariableDefinition
	Timeouts     TimeoutConfiguration
	RetryPolicy  RetryPolicy
	Triggers     []TriggerDefinition
}

// StageDefinition defines a workflow stage template
type StageDefinition struct {
	Name         string
	Type         StageType
	Description  string
	Action       ActionDefinition
	Dependencies []string
	Conditions   []ConditionDefinition
	Timeout      time.Duration
	RetryPolicy  *RetryPolicy
	Parallel     bool
	Optional     bool
	Metadata     map[string]interface{}
}

// StageInstance represents a running stage
type StageInstance struct {
	ID           string
	Definition   *StageDefinition
	Status       StageStatus
	StartTime    time.Time
	EndTime      time.Time
	Output       interface{}
	Error        error
	RetryCount   int
	Context      context.Context
	Cancel       context.CancelFunc
	Dependencies []*StageInstance

	// Progress tracking
	Progress float64
	Message  string

	// Metadata
	Metadata map[string]interface{}

	// Synchronization
	mutex sync.RWMutex
}

// WorkflowState represents the state of a workflow
type WorkflowState int

const (
	WorkflowStateInitializing WorkflowState = iota
	WorkflowStateRunning
	WorkflowStatePaused
	WorkflowStateWaitingForInput
	WorkflowStateCompleted
	WorkflowStateFailed
	WorkflowStateCancelled
	WorkflowStateAborted
)

// StageStatus represents the status of a stage
type StageStatus int

const (
	StageStatusPending StageStatus = iota
	StageStatusRunning
	StageStatusCompleted
	StageStatusFailed
	StageStatusSkipped
	StageStatusCancelled
	StageStatusWaitingForDependencies
	StageStatusWaitingForInput
)

// StageType defines different types of stages
type StageType int

const (
	StageTypeAction StageType = iota
	StageTypeParallel
	StageTypeSequential
)

// ActionDefinition defines what action a stage performs
type ActionDefinition struct {
	Type       ActionType
	Command    string
	Parameters map[string]interface{}
	Script     string
	Function   string
	Timeout    time.Duration
}

// ActionType defines types of actions
type ActionType int

const (
	ActionTypeCommand ActionType = iota
	ActionTypeScript
	ActionTypeFunction
	ActionTypeHTTP
	ActionTypeClaudeCode
	ActionTypeGitOperation
	ActionTypeFileOperation
)

// ConditionDefinition defines a conditional check
type ConditionDefinition struct {
	Type       ConditionType
	Expression string
	Variable   string
	Operator   string
	Value      interface{}
}

// ConditionType defines types of conditions
type ConditionType int

const (
	ConditionTypeExpression ConditionType = iota
	ConditionTypeVariable
	ConditionTypeFileExists
	ConditionTypeCommandSuccess
)

// VariableDefinition defines a workflow variable
type VariableDefinition struct {
	Name         string
	Type         VariableType
	DefaultValue interface{}
	Required     bool
	Description  string
}

// VariableType defines variable types
type VariableType int

// VariableType constants removed - currently unused

// TimeoutConfiguration defines timeout settings
type TimeoutConfiguration struct {
	Workflow time.Duration
	Stage    time.Duration
	Action   time.Duration
}

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Conditions   []RetryCondition
}

// RetryCondition defines when to retry
type RetryCondition struct {
	Type    RetryConditionType
	Pattern string
}

// RetryConditionType defines retry condition types
type RetryConditionType int

// RetryConditionType constants removed - currently unused

// TriggerDefinition defines workflow triggers
type TriggerDefinition struct {
	Type       TriggerType
	Event      string
	Condition  string
	Parameters map[string]interface{}
}

// TriggerType defines trigger types
type TriggerType int

// TriggerType constants removed - currently unused

// Default configuration values
const (
	DefaultMaxConcurrentWorkflows = 10
	DefaultMaxConcurrentStages    = 20
	DefaultWorkflowTimeout        = 60 * time.Minute
	DefaultRetryAttempts          = 3
	DefaultRetryDelay             = 5 * time.Second
	DefaultEventBufferSize        = 1000
)

// NewEngine creates a new workflow engine
func NewEngine(config EngineConfig) (*Engine, error) {
	// Set defaults
	if config.MaxConcurrentWorkflows == 0 {
		config.MaxConcurrentWorkflows = DefaultMaxConcurrentWorkflows
	}
	if config.MaxConcurrentStages == 0 {
		config.MaxConcurrentStages = DefaultMaxConcurrentStages
	}
	if config.DefaultTimeout == 0 {
		config.DefaultTimeout = DefaultWorkflowTimeout
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = DefaultRetryAttempts
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = DefaultRetryDelay
	}
	if config.EventBufferSize == 0 {
		config.EventBufferSize = DefaultEventBufferSize
	}

	ctx, cancel := context.WithCancel(context.Background())

	engine := &Engine{
		config:          config,
		activeWorkflows: make(map[string]*WorkflowInstance),
		eventHandlers:   make(map[EventType][]EventSubscriber),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Initialize components
	if err := engine.initializeComponents(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize engine components: %w", err)
	}

	// Start background workers
	engine.startBackgroundWorkers()

	return engine, nil
}

// initializeComponents initializes all engine components
func (e *Engine) initializeComponents() error {
	var err error

	// Initialize state manager
	e.stateManager, err = NewStateManager(e.config)
	if err != nil {
		return fmt.Errorf("failed to create state manager: %w", err)
	}

	// Initialize stage executor
	e.stageExecutor, err = NewStageExecutor(e.config)
	if err != nil {
		return fmt.Errorf("failed to create stage executor: %w", err)
	}

	// Initialize dependency graph
	e.dependencyGraph = NewDependencyGraph()

	// Initialize event bus
	e.eventBus, err = NewEventBus(e.config.EventBufferSize)
	if err != nil {
		return fmt.Errorf("failed to create event bus: %w", err)
	}

	// Initialize worker pool
	e.stageWorkers, err = NewWorkerPool(e.config.MaxConcurrentStages)
	if err != nil {
		return fmt.Errorf("failed to create worker pool: %w", err)
	}

	// Initialize persistence if enabled
	if e.config.PersistenceEnabled {
		e.persistence, err = NewPersistenceManager(PersistenceConfig{})
		if err != nil {
			return fmt.Errorf("failed to create persistence manager: %w", err)
		}
	}

	// Initialize metrics if enabled
	if e.config.MetricsEnabled {
		e.metrics = NewMetricsCollector()
	}

	return nil
}

// startBackgroundWorkers starts background processing workers
func (e *Engine) startBackgroundWorkers() {
	// Start event processor
	e.shutdownWG.Add(1)
	go e.processEvents()

	// Start workflow monitor
	e.shutdownWG.Add(1)
	go e.monitorWorkflows()

	// Start metrics collector if enabled
	if e.config.MetricsEnabled {
		e.shutdownWG.Add(1)
		go e.collectMetrics()
	}
}

// StartWorkflow starts a new workflow instance
func (e *Engine) StartWorkflow(ctx context.Context, definition *WorkflowDefinition, variables map[string]interface{}) (*WorkflowInstance, error) {
	// Check concurrency limits
	e.workflowsMutex.RLock()
	if len(e.activeWorkflows) >= e.config.MaxConcurrentWorkflows {
		e.workflowsMutex.RUnlock()
		return nil, fmt.Errorf("maximum concurrent workflows (%d) reached", e.config.MaxConcurrentWorkflows)
	}
	e.workflowsMutex.RUnlock()

	// Create workflow instance
	instance, err := e.createWorkflowInstance(ctx, definition, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to create workflow instance: %w", err)
	}

	// Register workflow
	e.workflowsMutex.Lock()
	e.activeWorkflows[instance.ID] = instance
	e.workflowsMutex.Unlock()

	// Persist workflow if enabled
	if e.persistence != nil {
		if err := e.persistence.SaveWorkflow(instance); err != nil {
			// Log error but don't fail workflow start
			e.logError(fmt.Errorf("failed to persist workflow %s: %w", instance.ID, err))
		}
	}

	// Emit workflow started event
	e.emitEvent(WorkflowEvent{
		Type:       EventTypeWorkflowStarted,
		WorkflowID: instance.ID,
		Timestamp:  time.Now(),
		Data:       map[string]interface{}{"definition": definition.Name},
	})

	// Start workflow execution with proper tracking
	e.shutdownWG.Add(1)
	go func() {
		defer e.shutdownWG.Done()
		e.executeWorkflow(instance)
	}()

	return instance, nil
}

// StopWorkflow stops a running workflow
func (e *Engine) StopWorkflow(workflowID string, reason string) error {
	e.workflowsMutex.Lock()
	defer e.workflowsMutex.Unlock()

	instance, exists := e.activeWorkflows[workflowID]
	if !exists {
		return fmt.Errorf("workflow %s not found", workflowID)
	}

	// Cancel workflow context
	instance.Cancel()

	// Update state
	_ = e.stateManager.TransitionWorkflow(instance, WorkflowStateCancelled) //nolint:errcheck // State transition errors are logged internally

	// Emit workflow stopped event
	e.emitEvent(WorkflowEvent{
		Type:       EventTypeWorkflowStopped,
		WorkflowID: workflowID,
		Timestamp:  time.Now(),
		Data:       map[string]interface{}{"reason": reason},
	})

	return nil
}

// PauseWorkflow pauses a running workflow
func (e *Engine) PauseWorkflow(workflowID string) error {
	e.workflowsMutex.Lock()
	defer e.workflowsMutex.Unlock()

	instance, exists := e.activeWorkflows[workflowID]
	if !exists {
		return fmt.Errorf("workflow %s not found", workflowID)
	}

	instance.stateMutex.RLock()
	currentState := instance.State
	instance.stateMutex.RUnlock()

	if currentState != WorkflowStateRunning {
		return fmt.Errorf("workflow %s is not running (state: %v)", workflowID, currentState)
	}

	// Transition to paused state
	_ = e.stateManager.TransitionWorkflow(instance, WorkflowStatePaused) //nolint:errcheck // State transition errors are logged internally

	// Emit workflow paused event
	e.emitEvent(WorkflowEvent{
		Type:       EventTypeWorkflowPaused,
		WorkflowID: workflowID,
		Timestamp:  time.Now(),
	})

	return nil
}

// ResumeWorkflow resumes a paused workflow
func (e *Engine) ResumeWorkflow(workflowID string) error {
	e.workflowsMutex.Lock()
	defer e.workflowsMutex.Unlock()

	instance, exists := e.activeWorkflows[workflowID]
	if !exists {
		return fmt.Errorf("workflow %s not found", workflowID)
	}

	instance.stateMutex.RLock()
	currentState := instance.State
	instance.stateMutex.RUnlock()

	if currentState != WorkflowStatePaused {
		return fmt.Errorf("workflow %s is not paused (state: %v)", workflowID, currentState)
	}

	// Transition to running state
	_ = e.stateManager.TransitionWorkflow(instance, WorkflowStateRunning) //nolint:errcheck // State transition errors are logged internally

	// Emit workflow resumed event
	e.emitEvent(WorkflowEvent{
		Type:       EventTypeWorkflowResumed,
		WorkflowID: workflowID,
		Timestamp:  time.Now(),
	})

	return nil
}

// GetWorkflowStatus returns the current status of a workflow
func (e *Engine) GetWorkflowStatus(workflowID string) (*WorkflowStatus, error) {
	e.workflowsMutex.RLock()
	instance, exists := e.activeWorkflows[workflowID]
	e.workflowsMutex.RUnlock()

	if !exists {
		// Check persistence for completed workflows
		if e.persistence != nil {
			return e.persistence.GetWorkflowStatus(workflowID)
		}
		return nil, fmt.Errorf("workflow %s not found", workflowID)
	}

	return e.buildWorkflowStatus(instance), nil
}

// ListActiveWorkflows returns all currently active workflows
func (e *Engine) ListActiveWorkflows() []*WorkflowStatus {
	e.workflowsMutex.RLock()
	defer e.workflowsMutex.RUnlock()

	statuses := make([]*WorkflowStatus, 0, len(e.activeWorkflows))
	for _, instance := range e.activeWorkflows {
		statuses = append(statuses, e.buildWorkflowStatus(instance))
	}

	return statuses
}

// Shutdown gracefully shuts down the engine
func (e *Engine) Shutdown(ctx context.Context) error {
	// Cancel all workflows
	e.workflowsMutex.RLock()
	for _, instance := range e.activeWorkflows {
		instance.Cancel()
	}
	e.workflowsMutex.RUnlock()

	// Cancel engine context
	e.cancel()

	// Wait for workers to finish with timeout
	done := make(chan struct{})
	go func() {
		defer close(done)
		e.shutdownWG.Wait()
	}()

	// Use a timeout context if the provided one doesn't have a deadline
	timeoutCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		timeoutCtx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	select {
	case <-done:
		return nil
	case <-timeoutCtx.Done():
		return fmt.Errorf("shutdown timeout: %w", timeoutCtx.Err())
	}
}

// Helper methods

func (e *Engine) createWorkflowInstance(ctx context.Context, definition *WorkflowDefinition, variables map[string]interface{}) (*WorkflowInstance, error) {
	workflowCtx, cancel := context.WithTimeout(ctx, e.config.DefaultTimeout)

	instance := &WorkflowInstance{
		ID:         generateWorkflowID(),
		Definition: definition,
		State:      WorkflowStateInitializing,
		Context:    workflowCtx,
		Cancel:     cancel,
		StartTime:  time.Now(),
		Variables:  variables,
		Metadata:   make(map[string]interface{}),
		Events:     []WorkflowEvent{},
	}

	// Initialize stages
	stages, err := e.createStageInstances(instance, definition.Stages)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stage instances: %w", err)
	}
	instance.Stages = stages

	return instance, nil
}

func (e *Engine) createStageInstances(workflow *WorkflowInstance, definitions []StageDefinition) ([]*StageInstance, error) {
	stages := make([]*StageInstance, len(definitions))

	for i, def := range definitions {
		stageCtx, cancel := context.WithCancel(workflow.Context)

		stage := &StageInstance{
			ID:         fmt.Sprintf("%s-stage-%d", workflow.ID, i),
			Definition: &def,
			Status:     StageStatusPending,
			Context:    stageCtx,
			Cancel:     cancel,
			Metadata:   make(map[string]interface{}),
		}

		stages[i] = stage
	}

	// Resolve dependencies
	if err := e.resolveStageDependencies(stages, workflow.Definition.Dependencies); err != nil {
		return nil, fmt.Errorf("failed to resolve stage dependencies: %w", err)
	}

	return stages, nil
}

func (e *Engine) resolveStageDependencies(stages []*StageInstance, dependencies map[string][]string) error {
	stageMap := make(map[string]*StageInstance)
	for _, stage := range stages {
		stageMap[stage.Definition.Name] = stage
	}

	for stageName, deps := range dependencies {
		stage, exists := stageMap[stageName]
		if !exists {
			return fmt.Errorf("stage %s not found", stageName)
		}

		for _, depName := range deps {
			depStage, exists := stageMap[depName]
			if !exists {
				return fmt.Errorf("dependency stage %s not found", depName)
			}
			stage.Dependencies = append(stage.Dependencies, depStage)
		}
	}

	return nil
}

func (e *Engine) buildWorkflowStatus(instance *WorkflowInstance) *WorkflowStatus {
	instance.stateMutex.RLock()
	instance.stageMutex.RLock()
	defer instance.stateMutex.RUnlock()
	defer instance.stageMutex.RUnlock()

	stageStatuses := make([]StageStatus, len(instance.Stages))
	for i, stage := range instance.Stages {
		stageStatuses[i] = stage.Status
	}

	return &WorkflowStatus{
		ID:            instance.ID,
		Name:          instance.Definition.Name,
		State:         instance.State,
		StartTime:     instance.StartTime,
		EndTime:       instance.EndTime,
		CurrentStage:  instance.CurrentStage,
		StageCount:    len(instance.Stages),
		StageStatuses: stageStatuses,
		Progress:      e.calculateWorkflowProgress(instance),
		LastError:     instance.LastError,
		ErrorCount:    instance.ErrorCount,
	}
}

func (e *Engine) calculateWorkflowProgress(instance *WorkflowInstance) float64 {
	if len(instance.Stages) == 0 {
		return 0
	}

	completed := 0
	for _, stage := range instance.Stages {
		if stage.Status == StageStatusCompleted {
			completed++
		}
	}

	return float64(completed) / float64(len(instance.Stages))
}

var workflowIDCounter int64

func generateWorkflowID() string {
	counter := atomic.AddInt64(&workflowIDCounter, 1)
	return fmt.Sprintf("workflow_%d_%d", time.Now().UnixNano(), counter)
}

func (e *Engine) logError(err error) {
	// TODO: Implement proper logging
	fmt.Printf("Engine error: %v\n", err)
}

// executeWorkflow executes a workflow instance
func (e *Engine) executeWorkflow(instance *WorkflowInstance) {
	defer func() {
		// Remove from active workflows when complete
		e.workflowsMutex.Lock()
		delete(e.activeWorkflows, instance.ID)
		e.workflowsMutex.Unlock()

		// Persist final state
		if e.persistence != nil {
			_ = e.persistence.SaveWorkflow(instance) //nolint:errcheck // Persistence errors are not critical for workflow execution
		}
	}()

	// Transition to running state
	_ = e.stateManager.TransitionWorkflow(instance, WorkflowStateRunning) //nolint:errcheck // State transition errors are logged internally

	// Get execution order from dependency graph
	executionOrder, err := e.planExecution(instance)
	if err != nil {
		_ = e.stateManager.TransitionWorkflow(instance, WorkflowStateFailed) //nolint:errcheck // State transition errors are logged internally
		instance.LastError = err
		e.emitEvent(WorkflowEvent{
			Type:       EventTypeWorkflowFailed,
			WorkflowID: instance.ID,
			Timestamp:  time.Now(),
			Data:       map[string]interface{}{"error": err.Error()},
		})
		return
	}

	// Execute stages in order
	for levelIndex, stageLevel := range executionOrder {
		// Check if workflow should continue
		instance.stateMutex.RLock()
		currentState := instance.State
		instance.stateMutex.RUnlock()

		if currentState != WorkflowStateRunning {
			break
		}

		instance.stateMutex.Lock()
		instance.CurrentStage = levelIndex
		instance.stateMutex.Unlock()

		// Execute stages in current level (potentially in parallel)
		if err := e.executeStageLevel(instance.Context, stageLevel, instance); err != nil {
			_ = e.stateManager.TransitionWorkflow(instance, WorkflowStateFailed) //nolint:errcheck // State transition errors are logged internally
			instance.LastError = err
			instance.ErrorCount++
			e.emitEvent(WorkflowEvent{
				Type:       EventTypeWorkflowFailed,
				WorkflowID: instance.ID,
				Timestamp:  time.Now(),
				Data:       map[string]interface{}{"error": err.Error()},
			})
			return
		}
	}

	// Workflow completed successfully
	instance.stateMutex.RLock()
	currentState := instance.State
	instance.stateMutex.RUnlock()

	if currentState == WorkflowStateRunning {
		_ = e.stateManager.TransitionWorkflow(instance, WorkflowStateCompleted) //nolint:errcheck // State transition errors are logged internally
		e.emitEvent(WorkflowEvent{
			Type:       EventTypeWorkflowCompleted,
			WorkflowID: instance.ID,
			Timestamp:  time.Now(),
		})
	}
}

// planExecution creates an execution plan for the workflow
func (e *Engine) planExecution(instance *WorkflowInstance) ([][]string, error) {
	// Build dependency graph
	e.dependencyGraph = NewDependencyGraph()

	for _, stage := range instance.Stages {
		var depIDs []string
		for _, dep := range stage.Dependencies {
			depIDs = append(depIDs, dep.ID)
		}
		e.dependencyGraph.AddStage(stage.ID, depIDs)
	}

	return e.dependencyGraph.GetExecutionOrder()
}

// executeStageLevel executes all stages in a level
func (e *Engine) executeStageLevel(ctx context.Context, stageIDs []string, workflow *WorkflowInstance) error {
	// Find stage instances
	var stages []*StageInstance
	stageMap := make(map[string]*StageInstance)
	for _, stage := range workflow.Stages {
		stageMap[stage.ID] = stage
	}

	for _, stageID := range stageIDs {
		if stage, exists := stageMap[stageID]; exists {
			stages = append(stages, stage)
		}
	}

	// Check if any stages should run in parallel
	hasParallel := false
	for _, stage := range stages {
		if stage.Definition.Parallel {
			hasParallel = true
			break
		}
	}

	if hasParallel && len(stages) > 1 {
		// Execute in parallel
		return e.stageExecutor.ExecuteParallelStages(ctx, stages, workflow)
	} else {
		// Execute sequentially
		for _, stage := range stages {
			if err := e.stageExecutor.ExecuteStage(ctx, stage, workflow); err != nil {
				return err
			}
		}
		return nil
	}
}

// processEvents processes workflow events
func (e *Engine) processEvents() {
	defer e.shutdownWG.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			// Event processing would be more sophisticated in practice
			// In a real implementation, this would process actual events
		}
	}
}

// monitorWorkflows monitors active workflows
func (e *Engine) monitorWorkflows() {
	defer e.shutdownWG.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.performHealthCheck()
		}
	}
}

// collectMetrics collects workflow metrics
func (e *Engine) collectMetrics() {
	defer e.shutdownWG.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.updateMetrics()
		}
	}
}

// performHealthCheck performs health checks on active workflows
func (e *Engine) performHealthCheck() {
	e.workflowsMutex.RLock()
	defer e.workflowsMutex.RUnlock()

	for _, workflow := range e.activeWorkflows {
		// Check for timeouts
		if time.Since(workflow.StartTime) > e.config.DefaultTimeout {
			_ = e.stateManager.TransitionWorkflow(workflow, WorkflowStateAborted) //nolint:errcheck // State transition errors are logged internally
			workflow.Cancel()
		}

		// Check for stuck stages
		for _, stage := range workflow.Stages {
			if stage.Status == StageStatusRunning {
				timeout := stage.Definition.Timeout
				if timeout == 0 {
					timeout = e.config.DefaultTimeout
				}
				if time.Since(stage.StartTime) > timeout {
					_ = e.stateManager.TransitionStage(stage, StageStatusFailed) //nolint:errcheck // State transition errors are logged internally
					stage.Error = fmt.Errorf("stage timeout after %v", timeout)
				}
			}
		}
	}
}

// updateMetrics updates workflow metrics
func (e *Engine) updateMetrics() {
	if e.metrics == nil {
		return
	}

	e.workflowsMutex.RLock()
	activeCount := len(e.activeWorkflows)
	e.workflowsMutex.RUnlock()

	e.metrics.RecordMetric("active_workflows", activeCount)
	e.metrics.RecordMetric("last_update", time.Now())
}

// emitEvent emits a workflow event
func (e *Engine) emitEvent(event WorkflowEvent) {
	if e.eventBus != nil {
		e.eventBus.Publish(event)
	}
}

// WorkflowStatus represents the current status of a workflow
type WorkflowStatus struct {
	ID            string
	Name          string
	State         WorkflowState
	StartTime     time.Time
	EndTime       time.Time
	CurrentStage  int
	StageCount    int
	StageStatuses []StageStatus
	Progress      float64
	LastError     error
	ErrorCount    int
}

func (ws WorkflowState) String() string {
	switch ws {
	case WorkflowStateInitializing:
		return "initializing"
	case WorkflowStateRunning:
		return "running"
	case WorkflowStatePaused:
		return "paused"
	case WorkflowStateWaitingForInput:
		return "waiting_for_input"
	case WorkflowStateCompleted:
		return statusCompleted
	case WorkflowStateFailed:
		return "failed"
	case WorkflowStateCancelled:
		return "canceled"
	case WorkflowStateAborted:
		return "aborted"
	default:
		return statusUnknown
	}
}

func (ss StageStatus) String() string {
	switch ss {
	case StageStatusPending:
		return "pending"
	case StageStatusRunning:
		return "running"
	case StageStatusCompleted:
		return statusCompleted
	case StageStatusFailed:
		return "failed"
	case StageStatusSkipped:
		return "skipped"
	case StageStatusCancelled:
		return "canceled"
	case StageStatusWaitingForDependencies:
		return "waiting_for_dependencies"
	case StageStatusWaitingForInput:
		return "waiting_for_input"
	default:
		return statusUnknown
	}
}
