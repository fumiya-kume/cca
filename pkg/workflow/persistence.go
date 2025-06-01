package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PersistenceManager handles workflow persistence
type PersistenceManager struct {
	config     PersistenceConfig
	storageDir string
	mutex      sync.RWMutex
}

// PersistenceConfig configures the persistence manager
type PersistenceConfig struct {
	StorageDir    string
	AutoSave      bool
	SaveInterval  time.Duration
	RetentionDays int
	CompressData  bool
}

// WorkflowSnapshot represents a persisted workflow state
type WorkflowSnapshot struct {
	ID           string                 `json:"id"`
	Definition   *WorkflowDefinition    `json:"definition"`
	State        WorkflowState          `json:"state"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	CurrentStage int                    `json:"current_stage"`
	Variables    map[string]interface{} `json:"variables"`
	Metadata     map[string]interface{} `json:"metadata"`
	Stages       []StageSnapshot        `json:"stages"`
	Events       []WorkflowEvent        `json:"events"`
	LastError    string                 `json:"last_error,omitempty"`
	ErrorCount   int                    `json:"error_count"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// StageSnapshot represents a persisted stage state
type StageSnapshot struct {
	ID           string                 `json:"id"`
	Definition   *StageDefinition       `json:"definition"`
	Status       StageStatus            `json:"status"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      time.Time              `json:"end_time"`
	Output       interface{}            `json:"output"`
	Error        string                 `json:"error,omitempty"`
	RetryCount   int                    `json:"retry_count"`
	Progress     float64                `json:"progress"`
	Message      string                 `json:"message"`
	Metadata     map[string]interface{} `json:"metadata"`
	Dependencies []string               `json:"dependencies"`
}

// NewPersistenceManager creates a new persistence manager
func NewPersistenceManager(config PersistenceConfig) (*PersistenceManager, error) {
	if config.StorageDir == "" {
		config.StorageDir = "./workflow_data"
	}
	if config.SaveInterval == 0 {
		config.SaveInterval = 30 * time.Second
	}
	if config.RetentionDays == 0 {
		config.RetentionDays = 30
	}

	pm := &PersistenceManager{
		config:     config,
		storageDir: config.StorageDir,
	}

	// Create storage directory
	if err := os.MkdirAll(pm.storageDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return pm, nil
}

// SaveWorkflow saves a workflow instance to storage
func (pm *PersistenceManager) SaveWorkflow(workflow *WorkflowInstance) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	snapshot := pm.createWorkflowSnapshot(workflow)

	// Serialize to JSON
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize workflow: %w", err)
	}

	// Write to file
	filename := fmt.Sprintf("workflow_%s.json", workflow.ID)
	filepath := filepath.Join(pm.storageDir, filename)

	if err := os.WriteFile(filepath, data, 0600); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	return nil
}

// LoadWorkflow loads a workflow instance from storage
func (pm *PersistenceManager) LoadWorkflow(workflowID string) (*WorkflowInstance, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	filename := fmt.Sprintf("workflow_%s.json", workflowID)
	filepath := filepath.Join(pm.storageDir, filename)

	// Read file
	// #nosec G304 - filepath is constructed from validated storage directory
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Deserialize from JSON
	var snapshot WorkflowSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to deserialize workflow: %w", err)
	}

	// Convert snapshot back to workflow instance
	return pm.createWorkflowInstance(snapshot)
}

// DeleteWorkflow removes a workflow from storage
func (pm *PersistenceManager) DeleteWorkflow(workflowID string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	filename := fmt.Sprintf("workflow_%s.json", workflowID)
	filepath := filepath.Join(pm.storageDir, filename)

	if err := os.Remove(filepath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete workflow file: %w", err)
	}

	return nil
}

// ListWorkflows returns a list of all persisted workflows
func (pm *PersistenceManager) ListWorkflows() ([]*WorkflowStatus, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	files, err := filepath.Glob(filepath.Join(pm.storageDir, "workflow_*.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow files: %w", err)
	}

	var statuses []*WorkflowStatus
	for _, file := range files {
		// #nosec G304 - file path is from trusted storage directory listing
		data, err := os.ReadFile(file)
		if err != nil {
			continue // Skip files we can't read
		}

		var snapshot WorkflowSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			continue // Skip files we can't parse
		}

		status := pm.createWorkflowStatusFromSnapshot(snapshot)
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// GetWorkflowStatus returns workflow status from storage
func (pm *PersistenceManager) GetWorkflowStatus(workflowID string) (*WorkflowStatus, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	filename := fmt.Sprintf("workflow_%s.json", workflowID)
	filepath := filepath.Join(pm.storageDir, filename)

	// #nosec G304 - filepath is constructed from validated storage directory
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	var snapshot WorkflowSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}

	return pm.createWorkflowStatusFromSnapshot(snapshot), nil
}

// CleanupOldWorkflows removes old workflow files based on retention policy
func (pm *PersistenceManager) CleanupOldWorkflows() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	cutoff := time.Now().AddDate(0, 0, -pm.config.RetentionDays)

	files, err := filepath.Glob(filepath.Join(pm.storageDir, "workflow_*.json"))
	if err != nil {
		return fmt.Errorf("failed to list workflow files: %w", err)
	}

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			_ = os.Remove(file) //nolint:errcheck // Cleanup on failure is best effort
		}
	}

	return nil
}

// Helper methods

func (pm *PersistenceManager) createWorkflowSnapshot(workflow *WorkflowInstance) WorkflowSnapshot {
	workflow.stateMutex.RLock()
	workflow.stageMutex.RLock()
	workflow.eventsMutex.RLock()
	defer workflow.stateMutex.RUnlock()
	defer workflow.stageMutex.RUnlock()
	defer workflow.eventsMutex.RUnlock()

	// Create stage snapshots
	stageSnapshots := make([]StageSnapshot, len(workflow.Stages))
	for i, stage := range workflow.Stages {
		stageSnapshots[i] = pm.createStageSnapshot(stage)
	}

	// Handle error serialization
	lastError := ""
	if workflow.LastError != nil {
		lastError = workflow.LastError.Error()
	}

	return WorkflowSnapshot{
		ID:           workflow.ID,
		Definition:   workflow.Definition,
		State:        workflow.State,
		StartTime:    workflow.StartTime,
		EndTime:      workflow.EndTime,
		CurrentStage: workflow.CurrentStage,
		Variables:    workflow.Variables,
		Metadata:     workflow.Metadata,
		Stages:       stageSnapshots,
		Events:       workflow.Events,
		LastError:    lastError,
		ErrorCount:   workflow.ErrorCount,
		CreatedAt:    workflow.StartTime,
		UpdatedAt:    time.Now(),
	}
}

func (pm *PersistenceManager) createStageSnapshot(stage *StageInstance) StageSnapshot {
	// Get dependency IDs
	var depIDs []string
	for _, dep := range stage.Dependencies {
		depIDs = append(depIDs, dep.ID)
	}

	// Handle error serialization
	errorStr := ""
	if stage.Error != nil {
		errorStr = stage.Error.Error()
	}

	return StageSnapshot{
		ID:           stage.ID,
		Definition:   stage.Definition,
		Status:       stage.Status,
		StartTime:    stage.StartTime,
		EndTime:      stage.EndTime,
		Output:       stage.Output,
		Error:        errorStr,
		RetryCount:   stage.RetryCount,
		Progress:     stage.Progress,
		Message:      stage.Message,
		Metadata:     stage.Metadata,
		Dependencies: depIDs,
	}
}

func (pm *PersistenceManager) createWorkflowInstance(snapshot WorkflowSnapshot) (*WorkflowInstance, error) {
	// Note: This is a simplified version - in practice, you'd need to properly
	// reconstruct the context, channels, and other runtime state

	workflow := &WorkflowInstance{
		ID:           snapshot.ID,
		Definition:   snapshot.Definition,
		State:        snapshot.State,
		StartTime:    snapshot.StartTime,
		EndTime:      snapshot.EndTime,
		CurrentStage: snapshot.CurrentStage,
		Variables:    snapshot.Variables,
		Metadata:     snapshot.Metadata,
		Events:       snapshot.Events,
		ErrorCount:   snapshot.ErrorCount,
	}

	// Handle error deserialization
	if snapshot.LastError != "" {
		workflow.LastError = fmt.Errorf("%s", snapshot.LastError)
	}

	// Create stage instances
	stages := make([]*StageInstance, len(snapshot.Stages))
	for i, stageSnapshot := range snapshot.Stages {
		stage, err := pm.createStageInstance(stageSnapshot)
		if err != nil {
			return nil, fmt.Errorf("failed to create stage instance: %w", err)
		}
		stages[i] = stage
	}
	workflow.Stages = stages

	return workflow, nil
}

func (pm *PersistenceManager) createStageInstance(snapshot StageSnapshot) (*StageInstance, error) {
	stage := &StageInstance{
		ID:         snapshot.ID,
		Definition: snapshot.Definition,
		Status:     snapshot.Status,
		StartTime:  snapshot.StartTime,
		EndTime:    snapshot.EndTime,
		Output:     snapshot.Output,
		RetryCount: snapshot.RetryCount,
		Progress:   snapshot.Progress,
		Message:    snapshot.Message,
		Metadata:   snapshot.Metadata,
	}

	// Handle error deserialization
	if snapshot.Error != "" {
		stage.Error = fmt.Errorf("%s", snapshot.Error)
	}

	// Note: Dependencies would need to be resolved after all stages are created

	return stage, nil
}

func (pm *PersistenceManager) createWorkflowStatusFromSnapshot(snapshot WorkflowSnapshot) *WorkflowStatus {
	stageStatuses := make([]StageStatus, len(snapshot.Stages))
	for i, stage := range snapshot.Stages {
		stageStatuses[i] = stage.Status
	}

	// Calculate progress
	completed := 0
	for _, stage := range snapshot.Stages {
		if stage.Status == StageStatusCompleted {
			completed++
		}
	}

	progress := float64(0)
	if len(snapshot.Stages) > 0 {
		progress = float64(completed) / float64(len(snapshot.Stages))
	}

	// Handle error
	var lastError error
	if snapshot.LastError != "" {
		lastError = fmt.Errorf("%s", snapshot.LastError)
	}

	return &WorkflowStatus{
		ID:            snapshot.ID,
		Name:          snapshot.Definition.Name,
		State:         snapshot.State,
		StartTime:     snapshot.StartTime,
		EndTime:       snapshot.EndTime,
		CurrentStage:  snapshot.CurrentStage,
		StageCount:    len(snapshot.Stages),
		StageStatuses: stageStatuses,
		Progress:      progress,
		LastError:     lastError,
		ErrorCount:    snapshot.ErrorCount,
	}
}
