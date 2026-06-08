package task

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShow_PendingTask(t *testing.T) {
	tmpDir := t.TempDir()
	varDir := filepath.Join(tmpDir, "var")

	// Create pending task
	task := agent.AgentTask{
		TaskID:   "T-01JABC123456789012345678",
		Version:  1,
		TaskType: "distill_intake_to_knowledge",
		Scope:    "current-branch",
		Status:   "pending",
		Events:   []agent.TaskEvent{{EventID: "E-01", TaskSummary: "test", RawNotes: []string{}}},
	}

	pendingDir := filepath.Join(varDir, "tasks", "pending")
	require.NoError(t, os.MkdirAll(pendingDir, 0755))
	data, _ := json.MarshalIndent(task, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(pendingDir, task.TaskID+".json"), data, 0644))

	// Show
	result, err := Show(varDir, task.TaskID)
	require.NoError(t, err)
	assert.Equal(t, "T-01JABC123456789012345678", result.TaskID)
	assert.Equal(t, "pending", result.Status)
}

func TestShow_CompletedTask(t *testing.T) {
	tmpDir := t.TempDir()
	varDir := filepath.Join(tmpDir, "var")

	task := agent.AgentTask{
		TaskID: "T-01JABC123456789012345678",
		Status: "completed",
		Events: []agent.TaskEvent{{EventID: "E-01", TaskSummary: "test", RawNotes: []string{}}},
	}

	completedDir := filepath.Join(varDir, "tasks", "completed")
	require.NoError(t, os.MkdirAll(completedDir, 0755))
	data, _ := json.MarshalIndent(task, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(completedDir, task.TaskID+".json"), data, 0644))

	result, err := Show(varDir, task.TaskID)
	require.NoError(t, err)
	assert.Equal(t, "completed", result.Status)
}

func TestShow_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	varDir := filepath.Join(tmpDir, "var")

	// Create empty directories
	for _, sub := range []string{"pending", "completed", "failed"} {
		require.NoError(t, os.MkdirAll(filepath.Join(varDir, "tasks", sub), 0755))
	}

	_, err := Show(varDir, "T-NONEXISTENT")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
