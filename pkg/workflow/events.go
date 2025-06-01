package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EventBus manages workflow events
type EventBus struct {
	events      chan WorkflowEvent
	subscribers map[EventType][]EventSubscriber
	mutex       sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup

	// Goroutine tracking for subscriber notifications
	notificationWG             sync.WaitGroup
	maxConcurrentNotifications int
	notificationSemaphore      chan struct{}
}

// EventSubscriber receives workflow events
type EventSubscriber interface {
	OnEvent(event WorkflowEvent)
	GetEventTypes() []EventType
}

// WorkflowEvent represents an event in the workflow system
type WorkflowEvent struct {
	Type       EventType
	WorkflowID string
	StageID    string
	Timestamp  time.Time
	Data       map[string]interface{}
	Metadata   map[string]interface{}
}

// EventType defines different types of workflow events
type EventType int

const (
	EventTypeWorkflowStarted EventType = iota
	EventTypeWorkflowCompleted
	EventTypeWorkflowFailed
	EventTypeWorkflowPaused
	EventTypeWorkflowResumed
	EventTypeWorkflowStopped
	EventTypeWorkflowCancelled
	EventTypeStageStarted
	EventTypeStageCompleted
	EventTypeStageFailed
	EventTypeStageSkipped
	EventTypeStageRetried
	EventTypeStateChanged
	EventTypeUserInputRequired
	EventTypeUserInputProvided
	EventTypeMetricsUpdated
	EventTypeErrorOccurred
	EventTypeWarningIssued
	EventTypeProgressUpdated
)

// NewEventBus creates a new event bus
func NewEventBus(bufferSize int) (*EventBus, error) {
	ctx, cancel := context.WithCancel(context.Background())

	maxNotifications := 100 // Limit concurrent notification goroutines
	if bufferSize > 0 && bufferSize < maxNotifications {
		maxNotifications = bufferSize
	}

	bus := &EventBus{
		events:                     make(chan WorkflowEvent, bufferSize),
		subscribers:                make(map[EventType][]EventSubscriber),
		ctx:                        ctx,
		cancel:                     cancel,
		maxConcurrentNotifications: maxNotifications,
		notificationSemaphore:      make(chan struct{}, maxNotifications),
	}

	// Start event processing
	bus.wg.Add(1)
	go bus.processEvents()

	return bus, nil
}

// Subscribe adds an event subscriber
func (eb *EventBus) Subscribe(subscriber EventSubscriber) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	for _, eventType := range subscriber.GetEventTypes() {
		eb.subscribers[eventType] = append(eb.subscribers[eventType], subscriber)
	}
}

// Unsubscribe removes an event subscriber
func (eb *EventBus) Unsubscribe(subscriber EventSubscriber) {
	eb.mutex.Lock()
	defer eb.mutex.Unlock()

	for _, eventType := range subscriber.GetEventTypes() {
		subscribers := eb.subscribers[eventType]
		for i, sub := range subscribers {
			if sub == subscriber {
				eb.subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)
				break
			}
		}
	}
}

// Publish publishes an event to the bus
func (eb *EventBus) Publish(event WorkflowEvent) {
	select {
	case eb.events <- event:
	case <-eb.ctx.Done():
		// Event bus is shutting down
	default:
		// Channel is full, drop event to prevent blocking
	}
}

// Shutdown gracefully shuts down the event bus
func (eb *EventBus) Shutdown() {
	eb.cancel()

	// Wait for main event processing goroutine
	eb.wg.Wait()

	// Wait for all notification goroutines to complete
	eb.notificationWG.Wait()

	close(eb.events)
}

// processEvents processes events from the channel
func (eb *EventBus) processEvents() {
	defer eb.wg.Done()

	for {
		select {
		case event := <-eb.events:
			eb.notifySubscribers(event)
		case <-eb.ctx.Done():
			// Drain remaining events
			for {
				select {
				case event := <-eb.events:
					eb.notifySubscribers(event)
				default:
					return
				}
			}
		}
	}
}

// notifySubscribers notifies all subscribers of an event
func (eb *EventBus) notifySubscribers(event WorkflowEvent) {
	eb.mutex.RLock()
	subscribers := eb.subscribers[event.Type]
	eb.mutex.RUnlock()

	for _, subscriber := range subscribers {
		// Use semaphore to limit concurrent notifications
		select {
		case eb.notificationSemaphore <- struct{}{}:
			eb.notificationWG.Add(1)
			go func(sub EventSubscriber, evt WorkflowEvent) {
				defer func() {
					<-eb.notificationSemaphore
					eb.notificationWG.Done()
					// Recover from subscriber panics to prevent bus crash
					if r := recover(); r != nil {
						// Log panic but continue processing
						// TODO: Add proper logging when logger is available
						_ = r // Acknowledge recovery for now
					}
				}()

				// Check context before processing
				select {
				case <-eb.ctx.Done():
					return
				default:
					sub.OnEvent(evt)
				}
			}(subscriber, event)
		case <-eb.ctx.Done():
			// Bus is shutting down, skip remaining notifications
			return
		default:
			// Semaphore full, drop notification to prevent blocking
			// In production, consider logging this event
		}
	}
}

// DependencyGraph manages stage dependencies
type DependencyGraph struct {
	nodes map[string]*DependencyNode
	mutex sync.RWMutex
}

// DependencyNode represents a node in the dependency graph
type DependencyNode struct {
	StageID      string
	Dependencies []string
	Dependents   []string
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*DependencyNode),
	}
}

// AddStage adds a stage to the dependency graph
func (dg *DependencyGraph) AddStage(stageID string, dependencies []string) {
	dg.mutex.Lock()
	defer dg.mutex.Unlock()

	// Create node if it doesn't exist
	if _, exists := dg.nodes[stageID]; !exists {
		dg.nodes[stageID] = &DependencyNode{
			StageID:      stageID,
			Dependencies: make([]string, 0),
			Dependents:   make([]string, 0),
		}
	}

	node := dg.nodes[stageID]
	node.Dependencies = dependencies

	// Update dependents for dependency nodes
	for _, depID := range dependencies {
		if _, exists := dg.nodes[depID]; !exists {
			dg.nodes[depID] = &DependencyNode{
				StageID:      depID,
				Dependencies: make([]string, 0),
				Dependents:   make([]string, 0),
			}
		}

		// Add this stage as a dependent
		depNode := dg.nodes[depID]
		depNode.Dependents = append(depNode.Dependents, stageID)
	}
}

// GetExecutionOrder returns stages in dependency-resolved execution order
func (dg *DependencyGraph) GetExecutionOrder() ([][]string, error) {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()

	// Topological sort to determine execution order
	var result [][]string
	inDegree := make(map[string]int)

	// Calculate in-degrees
	for stageID, node := range dg.nodes {
		inDegree[stageID] = len(node.Dependencies)
	}

	// Find stages with no dependencies
	for len(inDegree) > 0 {
		var currentLevel []string

		// Find all stages with in-degree 0
		for stageID, degree := range inDegree {
			if degree == 0 {
				currentLevel = append(currentLevel, stageID)
			}
		}

		if len(currentLevel) == 0 {
			return nil, fmt.Errorf("circular dependency detected")
		}

		result = append(result, currentLevel)

		// Remove current level stages and update in-degrees
		for _, stageID := range currentLevel {
			delete(inDegree, stageID)

			// Reduce in-degree for dependent stages
			if node, exists := dg.nodes[stageID]; exists {
				for _, depID := range node.Dependents {
					if _, exists := inDegree[depID]; exists {
						inDegree[depID]--
					}
				}
			}
		}
	}

	return result, nil
}

// GetDependencies returns the dependencies of a stage
func (dg *DependencyGraph) GetDependencies(stageID string) []string {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()

	if node, exists := dg.nodes[stageID]; exists {
		return append([]string{}, node.Dependencies...) // Return copy
	}
	return []string{}
}

// GetDependents returns the dependents of a stage
func (dg *DependencyGraph) GetDependents(stageID string) []string {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()

	if node, exists := dg.nodes[stageID]; exists {
		return append([]string{}, node.Dependents...) // Return copy
	}
	return []string{}
}

// WorkerPool manages concurrent stage execution
type WorkerPool struct {
	workers   int
	semaphore chan struct{}
	ctx       context.Context
	cancel    context.CancelFunc

	// Goroutine tracking for proper cleanup
	activeWorkers sync.WaitGroup
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(maxWorkers int) (*WorkerPool, error) {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workers:   maxWorkers,
		semaphore: make(chan struct{}, maxWorkers),
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// Submit submits work to the pool
func (wp *WorkerPool) Submit(work func()) error {
	select {
	case wp.semaphore <- struct{}{}:
		wp.activeWorkers.Add(1)
		go func() {
			defer func() {
				<-wp.semaphore
				wp.activeWorkers.Done()
				// Recover from worker panics to prevent pool crash
				if r := recover(); r != nil {
					// Log panic but continue
					// TODO: Add proper logging when logger is available
					_ = r // Acknowledge recovery for now
				}
			}()

			// Check context before executing work
			select {
			case <-wp.ctx.Done():
				return
			default:
				work()
			}
		}()
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	default:
		// Pool is full, return error instead of blocking
		return fmt.Errorf("worker pool is full")
	}
}

// Shutdown shuts down the worker pool
func (wp *WorkerPool) Shutdown() {
	wp.cancel()

	// Wait for all active workers to complete
	wp.activeWorkers.Wait()
}

// ShutdownWithTimeout shuts down the worker pool with a timeout
func (wp *WorkerPool) ShutdownWithTimeout(timeout time.Duration) error {
	wp.cancel()

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		wp.activeWorkers.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("worker pool shutdown timeout after %v", timeout)
	}
}

// MetricsCollector collects workflow metrics
type MetricsCollector struct {
	metrics map[string]interface{}
	mutex   sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]interface{}),
	}
}

// RecordMetric records a metric value
func (mc *MetricsCollector) RecordMetric(name string, value interface{}) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	mc.metrics[name] = value
}

// GetMetric gets a metric value
func (mc *MetricsCollector) GetMetric(name string) (interface{}, bool) {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()
	value, exists := mc.metrics[name]
	return value, exists
}

// GetAllMetrics gets all metrics
func (mc *MetricsCollector) GetAllMetrics() map[string]interface{} {
	mc.mutex.RLock()
	defer mc.mutex.RUnlock()

	result := make(map[string]interface{})
	for k, v := range mc.metrics {
		result[k] = v
	}
	return result
}

// eventTypeNames maps event types to their string representations
var eventTypeNames = map[EventType]string{
	EventTypeWorkflowStarted:    "workflow_started",
	EventTypeWorkflowCompleted:  "workflow_completed",
	EventTypeWorkflowFailed:     "workflow_failed",
	EventTypeWorkflowPaused:     "workflow_paused",
	EventTypeWorkflowResumed:    "workflow_resumed",
	EventTypeWorkflowStopped:    "workflow_stopped",
	EventTypeWorkflowCancelled:  "workflow_canceled",
	EventTypeStageStarted:       "stage_started",
	EventTypeStageCompleted:     "stage_completed",
	EventTypeStageFailed:        "stage_failed",
	EventTypeStageSkipped:       "stage_skipped",
	EventTypeStageRetried:       "stage_retried",
	EventTypeStateChanged:       "state_changed",
	EventTypeUserInputRequired:  "user_input_required",
	EventTypeUserInputProvided:  "user_input_provided",
	EventTypeMetricsUpdated:     "metrics_updated",
	EventTypeErrorOccurred:      "error_occurred",
	EventTypeWarningIssued:      "warning_issued",
	EventTypeProgressUpdated:    "progress_updated",
}

func (et EventType) String() string {
	if name, exists := eventTypeNames[et]; exists {
		return name
	}
	return "unknown"
}
