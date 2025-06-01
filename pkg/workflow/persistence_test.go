package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPersistenceManager(t *testing.T) {
	t.Run("with default configuration", func(t *testing.T) {
		config := PersistenceConfig{}
		pm, err := NewPersistenceManager(config)

		require.NoError(t, err)
		assert.NotNil(t, pm)
		assert.Equal(t, "./workflow_data", pm.config.StorageDir)
		assert.Equal(t, 30*time.Second, pm.config.SaveInterval)
		assert.Equal(t, 30, pm.config.RetentionDays)

		// Clean up
		_ = os.RemoveAll(pm.storageDir)
	})

	t.Run("with custom configuration", func(t *testing.T) {
		tempDir := t.TempDir()
		config := PersistenceConfig{
			StorageDir:    tempDir,
			AutoSave:      true,
			SaveInterval:  60 * time.Second,
			RetentionDays: 7,
			CompressData:  true,
		}
		pm, err := NewPersistenceManager(config)

		require.NoError(t, err)
		assert.NotNil(t, pm)
		assert.Equal(t, tempDir, pm.config.StorageDir)
		assert.Equal(t, 60*time.Second, pm.config.SaveInterval)
		assert.Equal(t, 7, pm.config.RetentionDays)
		assert.True(t, pm.config.AutoSave)
		assert.True(t, pm.config.CompressData)
	})

	t.Run("creates storage directory", func(t *testing.T) {
		tempDir := t.TempDir()
		storageDir := filepath.Join(tempDir, "custom_workflow_storage")
		config := PersistenceConfig{
			StorageDir: storageDir,
		}
		pm, err := NewPersistenceManager(config)

		require.NoError(t, err)
		assert.NotNil(t, pm)

		// Verify directory was created
		_, err = os.Stat(storageDir)
		assert.NoError(t, err)
	})

	t.Run("fails with invalid directory", func(t *testing.T) {
		// Use a path that can't be created (root on most systems)
		config := PersistenceConfig{
			StorageDir: "/invalid/path/that/cannot/be/created",
		}
		pm, err := NewPersistenceManager(config)

		assert.Error(t, err)
		assert.Nil(t, pm)
		assert.Contains(t, err.Error(), "failed to create storage directory")
	})
}

func TestPersistenceManager_SaveWorkflow(t *testing.T) {
	pm := createTestPersistenceManager(t)

	t.Run("saves workflow successfully", func(t *testing.T) {
		workflow := createTestWorkflowInstance("test-workflow-1")

		err := pm.SaveWorkflow(workflow)

		require.NoError(t, err)

		// Verify file was created
		filename := fmt.Sprintf("workflow_%s.json", workflow.ID)
		filepath := filepath.Join(pm.storageDir, filename)
		_, err = os.Stat(filepath)
		assert.NoError(t, err)

		// Verify file content
		// #nosec G304 - filepath is from validated test directory
		data, err := os.ReadFile(filepath)
		require.NoError(t, err)

		var snapshot WorkflowSnapshot
		err = json.Unmarshal(data, &snapshot)
		require.NoError(t, err)

		assert.Equal(t, workflow.ID, snapshot.ID)
		assert.Equal(t, workflow.State, snapshot.State)
	})

	t.Run("overwrites existing workflow", func(t *testing.T) {
		workflow := createTestWorkflowInstance("test-workflow-2")

		// Save first time
		err := pm.SaveWorkflow(workflow)
		require.NoError(t, err)

		// Modify workflow
		workflow.State = WorkflowStateCompleted
		workflow.ErrorCount = 5

		// Save again
		err = pm.SaveWorkflow(workflow)
		require.NoError(t, err)

		// Verify updated content
		filename := fmt.Sprintf("workflow_%s.json", workflow.ID)
		filepath := filepath.Join(pm.storageDir, filename)
		data, err := os.ReadFile(filepath)
		require.NoError(t, err)

		var snapshot WorkflowSnapshot
		err = json.Unmarshal(data, &snapshot)
		require.NoError(t, err)

		assert.Equal(t, WorkflowStateCompleted, snapshot.State)
		assert.Equal(t, 5, snapshot.ErrorCount)
	})

	t.Run("handles concurrent saves", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				workflow := createTestWorkflowInstance(fmt.Sprintf("concurrent-workflow-%d", id))
				err := pm.SaveWorkflow(workflow)
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent save failed: %v", err)
		}
	})

	t.Run("handles workflow with complex data", func(t *testing.T) {
		workflow := createComplexWorkflowInstance("complex-workflow")

		err := pm.SaveWorkflow(workflow)

		require.NoError(t, err)

		// Verify complex data was serialized correctly
		filename := fmt.Sprintf("workflow_%s.json", workflow.ID)
		filepath := filepath.Join(pm.storageDir, filename)
		data, err := os.ReadFile(filepath)
		require.NoError(t, err)

		var snapshot WorkflowSnapshot
		err = json.Unmarshal(data, &snapshot)
		require.NoError(t, err)

		assert.Equal(t, len(workflow.Stages), len(snapshot.Stages))
		assert.Equal(t, len(workflow.Events), len(snapshot.Events))
		assert.NotEmpty(t, snapshot.Variables)
		assert.NotEmpty(t, snapshot.Metadata)
	})
}

func TestPersistenceManager_LoadWorkflow(t *testing.T) {
	pm := createTestPersistenceManager(t)

	t.Run("loads workflow successfully", func(t *testing.T) {
		// Save a workflow first
		originalWorkflow := createTestWorkflowInstance("load-test-1")
		err := pm.SaveWorkflow(originalWorkflow)
		require.NoError(t, err)

		// Load the workflow
		loadedWorkflow, err := pm.LoadWorkflow(originalWorkflow.ID)

		require.NoError(t, err)
		assert.NotNil(t, loadedWorkflow)
		assert.Equal(t, originalWorkflow.ID, loadedWorkflow.ID)
		assert.Equal(t, originalWorkflow.State, loadedWorkflow.State)
		assert.Equal(t, originalWorkflow.CurrentStage, loadedWorkflow.CurrentStage)
		assert.Equal(t, originalWorkflow.ErrorCount, loadedWorkflow.ErrorCount)
	})

	t.Run("fails to load non-existent workflow", func(t *testing.T) {
		workflow, err := pm.LoadWorkflow("non-existent-workflow")

		assert.Error(t, err)
		assert.Nil(t, workflow)
		assert.Contains(t, err.Error(), "failed to read workflow file")
	})

	t.Run("fails to load corrupted workflow file", func(t *testing.T) {
		// Create corrupted file
		filename := "workflow_corrupted.json"
		filepath := filepath.Join(pm.storageDir, filename)
		err := os.WriteFile(filepath, []byte("invalid json"), 0600)
		require.NoError(t, err)

		workflow, err := pm.LoadWorkflow("corrupted")

		assert.Error(t, err)
		assert.Nil(t, workflow)
		assert.Contains(t, err.Error(), "failed to deserialize workflow")
	})

	t.Run("handles workflow with error state", func(t *testing.T) {
		originalWorkflow := createTestWorkflowInstance("error-workflow")
		originalWorkflow.LastError = fmt.Errorf("test error message")
		err := pm.SaveWorkflow(originalWorkflow)
		require.NoError(t, err)

		loadedWorkflow, err := pm.LoadWorkflow(originalWorkflow.ID)

		require.NoError(t, err)
		assert.NotNil(t, loadedWorkflow.LastError)
		assert.Equal(t, "test error message", loadedWorkflow.LastError.Error())
	})
}

func TestPersistenceManager_DeleteWorkflow(t *testing.T) {
	pm := createTestPersistenceManager(t)

	t.Run("deletes existing workflow", func(t *testing.T) {
		// Save a workflow first
		workflow := createTestWorkflowInstance("delete-test-1")
		err := pm.SaveWorkflow(workflow)
		require.NoError(t, err)

		// Verify file exists
		filename := fmt.Sprintf("workflow_%s.json", workflow.ID)
		filepath := filepath.Join(pm.storageDir, filename)
		_, err = os.Stat(filepath)
		require.NoError(t, err)

		// Delete the workflow
		err = pm.DeleteWorkflow(workflow.ID)

		require.NoError(t, err)

		// Verify file is gone
		_, err = os.Stat(filepath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("handles deletion of non-existent workflow", func(t *testing.T) {
		err := pm.DeleteWorkflow("non-existent-workflow")

		// Should not return error for non-existent file
		assert.NoError(t, err)
	})

	t.Run("handles concurrent deletions", func(t *testing.T) {
		// Save multiple workflows
		workflows := make([]*WorkflowInstance, 5)
		for i := 0; i < 5; i++ {
			workflows[i] = createTestWorkflowInstance(fmt.Sprintf("concurrent-delete-%d", i))
			err := pm.SaveWorkflow(workflows[i])
			require.NoError(t, err)
		}

		var wg sync.WaitGroup
		errors := make(chan error, 5)

		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				err := pm.DeleteWorkflow(workflows[id].ID)
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent delete failed: %v", err)
		}
	})
}

func TestPersistenceManager_ListWorkflows(t *testing.T) {
	pm := createTestPersistenceManager(t)

	t.Run("lists empty workflows", func(t *testing.T) {
		statuses, err := pm.ListWorkflows()

		require.NoError(t, err)
		assert.Empty(t, statuses)
	})

	t.Run("lists multiple workflows", func(t *testing.T) {
		// Save multiple workflows
		workflows := []*WorkflowInstance{
			createTestWorkflowInstance("list-test-1"),
			createTestWorkflowInstance("list-test-2"),
			createTestWorkflowInstance("list-test-3"),
		}

		for _, workflow := range workflows {
			err := pm.SaveWorkflow(workflow)
			require.NoError(t, err)
		}

		statuses, err := pm.ListWorkflows()

		require.NoError(t, err)
		assert.Len(t, statuses, 3)

		// Verify all workflows are present
		ids := make(map[string]bool)
		for _, status := range statuses {
			ids[status.ID] = true
		}

		for _, workflow := range workflows {
			assert.True(t, ids[workflow.ID], "Workflow %s not found in list", workflow.ID)
		}
	})

	t.Run("skips corrupted files", func(t *testing.T) {
		// Create a fresh persistence manager for this test
		freshPM := createTestPersistenceManager(t)

		// Save a valid workflow
		workflow := createTestWorkflowInstance("valid-workflow")
		err := freshPM.SaveWorkflow(workflow)
		require.NoError(t, err)

		// Create a corrupted file
		corruptedFile := filepath.Join(freshPM.storageDir, "workflow_corrupted.json")
		err = os.WriteFile(corruptedFile, []byte("invalid json"), 0600)
		require.NoError(t, err)

		// Create a non-workflow file (should be ignored)
		otherFile := filepath.Join(freshPM.storageDir, "other_file.json")
		err = os.WriteFile(otherFile, []byte("{}"), 0600)
		require.NoError(t, err)

		statuses, err := freshPM.ListWorkflows()

		require.NoError(t, err)
		assert.Len(t, statuses, 1) // Only the valid workflow
		assert.Equal(t, workflow.ID, statuses[0].ID)
	})
}

func TestPersistenceManager_GetWorkflowStatus(t *testing.T) {
	pm := createTestPersistenceManager(t)

	t.Run("gets workflow status successfully", func(t *testing.T) {
		workflow := createTestWorkflowInstance("status-test-1")
		workflow.State = WorkflowStateRunning
		err := pm.SaveWorkflow(workflow)
		require.NoError(t, err)

		status, err := pm.GetWorkflowStatus(workflow.ID)

		require.NoError(t, err)
		assert.NotNil(t, status)
		assert.Equal(t, workflow.ID, status.ID)
		assert.Equal(t, WorkflowStateRunning, status.State)
		assert.Equal(t, len(workflow.Stages), status.StageCount)
	})

	t.Run("fails for non-existent workflow", func(t *testing.T) {
		status, err := pm.GetWorkflowStatus("non-existent")

		assert.Error(t, err)
		assert.Nil(t, status)
		assert.Contains(t, err.Error(), "workflow not found")
	})

	t.Run("calculates progress correctly", func(t *testing.T) {
		workflow := createComplexWorkflowInstance("progress-test")
		// Set some stages as completed
		workflow.Stages[0].Status = StageStatusCompleted
		workflow.Stages[1].Status = StageStatusCompleted
		workflow.Stages[2].Status = StageStatusRunning

		err := pm.SaveWorkflow(workflow)
		require.NoError(t, err)

		status, err := pm.GetWorkflowStatus(workflow.ID)

		require.NoError(t, err)
		expectedProgress := 2.0 / float64(len(workflow.Stages)) // 2 completed out of total
		assert.InDelta(t, expectedProgress, status.Progress, 0.01)
	})
}

func TestPersistenceManager_CleanupOldWorkflows(t *testing.T) {
	pm := createTestPersistenceManager(t)

	t.Run("cleans up old workflows", func(t *testing.T) {
		// Create some workflow files with different ages
		oldWorkflow := createTestWorkflowInstance("old-workflow")
		recentWorkflow := createTestWorkflowInstance("recent-workflow")

		err := pm.SaveWorkflow(oldWorkflow)
		require.NoError(t, err)
		err = pm.SaveWorkflow(recentWorkflow)
		require.NoError(t, err)

		// Manually set the modification time of old workflow file
		oldFilename := fmt.Sprintf("workflow_%s.json", oldWorkflow.ID)
		oldFilepath := filepath.Join(pm.storageDir, oldFilename)

		// Set modification time to 40 days ago (older than retention policy)
		oldTime := time.Now().AddDate(0, 0, -40)
		err = os.Chtimes(oldFilepath, oldTime, oldTime)
		require.NoError(t, err)

		// Run cleanup
		err = pm.CleanupOldWorkflows()
		require.NoError(t, err)

		// Verify old workflow is gone
		_, err = os.Stat(oldFilepath)
		assert.True(t, os.IsNotExist(err))

		// Verify recent workflow still exists
		recentFilename := fmt.Sprintf("workflow_%s.json", recentWorkflow.ID)
		recentFilepath := filepath.Join(pm.storageDir, recentFilename)
		_, err = os.Stat(recentFilepath)
		assert.NoError(t, err)
	})

	t.Run("handles empty directory", func(t *testing.T) {
		err := pm.CleanupOldWorkflows()
		assert.NoError(t, err)
	})

	t.Run("handles custom retention days", func(t *testing.T) {
		// Create PM with short retention
		config := PersistenceConfig{
			StorageDir:    t.TempDir(),
			RetentionDays: 1,
		}
		shortPM, err := NewPersistenceManager(config)
		require.NoError(t, err)

		workflow := createTestWorkflowInstance("short-retention")
		err = shortPM.SaveWorkflow(workflow)
		require.NoError(t, err)

		// Set file time to 2 days ago
		filename := fmt.Sprintf("workflow_%s.json", workflow.ID)
		filepath := filepath.Join(shortPM.storageDir, filename)
		oldTime := time.Now().AddDate(0, 0, -2)
		err = os.Chtimes(filepath, oldTime, oldTime)
		require.NoError(t, err)

		err = shortPM.CleanupOldWorkflows()
		require.NoError(t, err)

		// File should be deleted
		_, err = os.Stat(filepath)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestPersistenceManager_ThreadSafety(t *testing.T) {
	pm := createTestPersistenceManager(t)

	t.Run("handles concurrent operations", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 50)

		// Concurrent saves
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				workflow := createTestWorkflowInstance(fmt.Sprintf("concurrent-save-%d", id))
				err := pm.SaveWorkflow(workflow)
				if err != nil {
					errors <- err
				}
			}(i)
		}

		// Concurrent loads (will fail for non-existent workflows, but shouldn't crash)
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				_, err := pm.LoadWorkflow(fmt.Sprintf("load-test-%d", id))
				// We expect errors here since workflows might not exist
				_ = err
			}(i)
		}

		// Concurrent lists
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := pm.ListWorkflows()
				if err != nil {
					errors <- err
				}
			}()
		}

		wg.Wait()
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent operation failed: %v", err)
		}
	})
}

// Test helper functions

func createTestPersistenceManager(t *testing.T) *PersistenceManager {
	tempDir := t.TempDir()
	config := PersistenceConfig{
		StorageDir:    tempDir,
		AutoSave:      false,
		SaveInterval:  30 * time.Second,
		RetentionDays: 30,
		CompressData:  false,
	}
	pm, err := NewPersistenceManager(config)
	require.NoError(t, err)
	return pm
}

func createTestWorkflowInstance(id string) *WorkflowInstance {
	return &WorkflowInstance{
		ID: id,
		Definition: &WorkflowDefinition{
			Name:        "Test Workflow",
			Description: "Test workflow for persistence testing",
			Version:     "1.0.0",
		},
		State:        WorkflowStateRunning,
		StartTime:    time.Now(),
		CurrentStage: 0,
		Variables: map[string]interface{}{
			"test_var": "test_value",
			"number":   42,
		},
		Metadata: map[string]interface{}{
			"test_meta": "meta_value",
		},
		Stages: []*StageInstance{
			{
				ID: "stage-1",
				Definition: &StageDefinition{
					Name:        "Test Stage 1",
					Description: "First test stage",
				},
				Status:   StageStatusPending,
				Progress: 0.0,
				Metadata: map[string]interface{}{
					"stage_meta": "stage_value",
				},
			},
		},
		Events: []WorkflowEvent{
			{
				Type:       EventTypeWorkflowStarted,
				WorkflowID: id,
				Timestamp:  time.Now(),
				Data: map[string]interface{}{
					"message": "Workflow started",
				},
			},
		},
		ErrorCount: 0,
	}
}

func createComplexWorkflowInstance(id string) *WorkflowInstance {
	workflow := createTestWorkflowInstance(id)

	// Add more stages
	workflow.Stages = append(workflow.Stages,
		&StageInstance{
			ID: "stage-2",
			Definition: &StageDefinition{
				Name:        "Test Stage 2",
				Description: "Second test stage",
			},
			Status:   StageStatusPending,
			Progress: 0.0,
		},
		&StageInstance{
			ID: "stage-3",
			Definition: &StageDefinition{
				Name:        "Test Stage 3",
				Description: "Third test stage",
			},
			Status:   StageStatusPending,
			Progress: 0.0,
		},
	)

	// Add more events
	workflow.Events = append(workflow.Events,
		WorkflowEvent{
			Type:       EventTypeStageStarted,
			WorkflowID: id,
			StageID:    "stage-2",
			Timestamp:  time.Now(),
			Data: map[string]interface{}{
				"message": "Stage started",
			},
		},
		WorkflowEvent{
			Type:       EventTypeStageCompleted,
			WorkflowID: id,
			StageID:    "stage-2",
			Timestamp:  time.Now(),
			Data: map[string]interface{}{
				"message": "Stage completed",
			},
		},
	)

	// Add more complex variables and metadata
	workflow.Variables["complex_data"] = map[string]interface{}{
		"nested": map[string]interface{}{
			"value": "nested_value",
			"array": []string{"item1", "item2", "item3"},
		},
	}

	workflow.Metadata["performance"] = map[string]interface{}{
		"start_time": time.Now().Unix(),
		"metrics":    []float64{1.1, 2.2, 3.3},
	}

	return workflow
}

// Test Data Validation

func TestWorkflowSnapshot_Serialization(t *testing.T) {
	t.Run("serializes and deserializes correctly", func(t *testing.T) {
		workflow := createComplexWorkflowInstance("serialization-test")
		pm := createTestPersistenceManager(t)

		// Create snapshot
		snapshot := pm.createWorkflowSnapshot(workflow)

		// Serialize to JSON
		data, err := json.MarshalIndent(snapshot, "", "  ")
		require.NoError(t, err)

		// Deserialize back
		var deserializedSnapshot WorkflowSnapshot
		err = json.Unmarshal(data, &deserializedSnapshot)
		require.NoError(t, err)

		// Verify data integrity
		assert.Equal(t, snapshot.ID, deserializedSnapshot.ID)
		assert.Equal(t, snapshot.State, deserializedSnapshot.State)
		assert.Equal(t, len(snapshot.Stages), len(deserializedSnapshot.Stages))
		assert.Equal(t, len(snapshot.Events), len(deserializedSnapshot.Events))

		// Verify key fields in Variables (JSON converts types so we check presence)
		assert.Contains(t, deserializedSnapshot.Variables, "test_var")
		assert.Contains(t, deserializedSnapshot.Variables, "number")
		assert.Contains(t, deserializedSnapshot.Variables, "complex_data")

		// Verify key fields in Metadata
		assert.Contains(t, deserializedSnapshot.Metadata, "test_meta")
		assert.Contains(t, deserializedSnapshot.Metadata, "performance")
	})

	t.Run("handles nil values correctly", func(t *testing.T) {
		workflow := createTestWorkflowInstance("nil-test")
		workflow.LastError = nil
		workflow.Variables = nil
		workflow.Metadata = nil

		pm := createTestPersistenceManager(t)
		snapshot := pm.createWorkflowSnapshot(workflow)

		assert.Empty(t, snapshot.LastError)
		// Variables and Metadata should be preserved as nil/empty
	})
}

func TestStageSnapshot_Creation(t *testing.T) {
	t.Run("creates stage snapshot correctly", func(t *testing.T) {
		stage := &StageInstance{
			ID: "test-stage",
			Definition: &StageDefinition{
				Name:        "Test Stage",
				Description: "Test stage description",
			},
			Status:     StageStatusRunning,
			StartTime:  time.Now(),
			EndTime:    time.Time{},
			Output:     "test output",
			Error:      fmt.Errorf("test error"),
			RetryCount: 3,
			Progress:   0.75,
			Message:    "Stage in progress",
			Metadata: map[string]interface{}{
				"key": "value",
			},
			Dependencies: []*StageInstance{
				{ID: "dep-1"},
				{ID: "dep-2"},
			},
		}

		pm := createTestPersistenceManager(t)
		snapshot := pm.createStageSnapshot(stage)

		assert.Equal(t, stage.ID, snapshot.ID)
		assert.Equal(t, stage.Status, snapshot.Status)
		assert.Equal(t, stage.Output, snapshot.Output)
		assert.Equal(t, "test error", snapshot.Error)
		assert.Equal(t, stage.RetryCount, snapshot.RetryCount)
		assert.Equal(t, stage.Progress, snapshot.Progress)
		assert.Equal(t, stage.Message, snapshot.Message)
		assert.Equal(t, stage.Metadata, snapshot.Metadata)
		assert.Len(t, snapshot.Dependencies, 2)
		assert.Contains(t, snapshot.Dependencies, "dep-1")
		assert.Contains(t, snapshot.Dependencies, "dep-2")
	})
}

func TestWorkflowStatus_Creation(t *testing.T) {
	t.Run("creates workflow status from snapshot", func(t *testing.T) {
		snapshot := WorkflowSnapshot{
			ID: "status-test",
			Definition: &WorkflowDefinition{
				Name:        "Status Test Workflow",
				Description: "Test workflow for status creation",
			},
			State:     WorkflowStateCompleted,
			StartTime: time.Now().Add(-1 * time.Hour),
			EndTime:   time.Now(),
			Stages: []StageSnapshot{
				{Status: StageStatusCompleted},
				{Status: StageStatusCompleted},
				{Status: StageStatusFailed},
			},
			LastError:  "test error",
			ErrorCount: 1,
		}

		pm := createTestPersistenceManager(t)
		status := pm.createWorkflowStatusFromSnapshot(snapshot)

		assert.Equal(t, snapshot.ID, status.ID)
		assert.Equal(t, snapshot.Definition.Name, status.Name)
		assert.Equal(t, snapshot.State, status.State)
		assert.Equal(t, snapshot.StartTime, status.StartTime)
		assert.Equal(t, snapshot.EndTime, status.EndTime)
		assert.Equal(t, 3, status.StageCount)
		assert.Len(t, status.StageStatuses, 3)
		assert.NotNil(t, status.LastError)
		assert.Equal(t, "test error", status.LastError.Error())
		assert.Equal(t, 1, status.ErrorCount)

		// Progress should be 2/3 completed stages
		expectedProgress := 2.0 / 3.0
		assert.InDelta(t, expectedProgress, status.Progress, 0.01)
	})

	t.Run("handles empty stages", func(t *testing.T) {
		snapshot := WorkflowSnapshot{
			ID: "empty-stages-test",
			Definition: &WorkflowDefinition{
				Name: "Empty Stages Test",
			},
			State:  WorkflowStateRunning,
			Stages: []StageSnapshot{},
		}

		pm := createTestPersistenceManager(t)
		status := pm.createWorkflowStatusFromSnapshot(snapshot)

		assert.Equal(t, 0, status.StageCount)
		assert.Empty(t, status.StageStatuses)
		assert.Equal(t, 0.0, status.Progress)
	})
}
