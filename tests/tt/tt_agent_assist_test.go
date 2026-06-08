package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupAgentTestDirs ensures the memory directories exist.
func setupAgentTestDirs(t *testing.T) {
	t.Helper()
	root := projectRoot()
	dirs := []string{
		filepath.Join(root, "prompts", "memory", "var", "intake", "pending"),
		filepath.Join(root, "prompts", "memory", "var", "intake", "processed"),
		filepath.Join(root, "prompts", "memory", "var", "tasks", "pending"),
		filepath.Join(root, "prompts", "memory", "var", "tasks", "completed"),
		filepath.Join(root, "prompts", "memory", "var", "tasks", "failed"),
	}
	for _, d := range dirs {
		require.NoError(t, os.MkdirAll(d, 0755))
	}
}

// cleanupAgentTasks removes any pending/completed/failed task files.
func cleanupAgentTasks(t *testing.T) {
	t.Helper()
	root := projectRoot()
	for _, sub := range []string{"pending", "completed", "failed"} {
		dir := filepath.Join(root, "prompts", "memory", "var", "tasks", sub)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.Name() != ".gitkeep" {
				os.Remove(filepath.Join(dir, e.Name()))
			}
		}
	}
}

// TestAgentAssist_NoPendingEvents verifies that assist returns no_pending_events
// when there are no pending events.
func TestAgentAssist_NoPendingEvents(t *testing.T) {
	setupAgentTestDirs(t)
	cleanupAgentTasks(t)

	stdout, stderr, exitCode := runTT(t, "agent", "assist", "--scope", "current-branch")
	_ = stderr

	// Even if the index is empty, exit code should be 0
	assert.Equal(t, 0, exitCode, "Exit code should be 0 for no pending events. stderr: %s", stderr)

	var result map[string]any
	err := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "Should produce valid JSON. stdout: %s", stdout)
	assert.Equal(t, "no_pending_events", result["status"])
}

// TestAgentAssist_CreateTask verifies the full assist flow:
// 1. Create intake events via notify
// 2. Run assist to generate a task
// 3. Verify task file exists
func TestAgentAssist_CreateTask(t *testing.T) {
	setupAgentTestDirs(t)
	cleanupAgentTasks(t)

	// 1. Create an intake event
	payload := `{
		"version": 1,
		"source_type": "coding_agent",
		"agent": "antigravity",
		"task_summary": "Integration test event for assist",
		"raw_notes": ["Test note for assist integration test"]
	}`
	notifyStdout, notifyStderr, notifyExit := runTTWithInput(t, payload, "agent", "notify")
	require.Equal(t, 0, notifyExit, "Notify should succeed. stderr: %s, stdout: %s", notifyStderr, notifyStdout)

	// 2. Run assist
	stdout, stderr, exitCode := runTT(t, "agent", "assist", "--scope", "current-branch", "--force")
	require.Equal(t, 0, exitCode, "Assist should succeed. stderr: %s, stdout: %s", stderr, stdout)

	var result map[string]any
	err := json.Unmarshal([]byte(stdout), &result)
	require.NoError(t, err, "Should produce valid JSON")

	status := result["status"].(string)
	// It should either create a new task or return an existing one
	assert.True(t, status == "requires_agent" || status == "existing_task",
		"Status should be requires_agent or existing_task, got: %s", status)

	if status == "requires_agent" {
		// Verify task file exists
		taskFile, ok := result["task_file"].(string)
		require.True(t, ok, "task_file should be present")
		require.True(t, len(taskFile) > 0, "task_file should not be empty")

		// Verify the task_id was returned
		taskID, ok := result["task_id"].(string)
		require.True(t, ok, "task_id should be present")
		assert.True(t, strings.HasPrefix(taskID, "T-"), "task_id should start with T-")

		// 3. Show the task
		showStdout, showStderr, showExit := runTT(t, "agent", "task", "show", taskID)
		require.Equal(t, 0, showExit, "Task show should succeed. stderr: %s", showStderr)

		var taskData map[string]any
		err = json.Unmarshal([]byte(showStdout), &taskData)
		require.NoError(t, err)
		assert.Equal(t, taskID, taskData["task_id"])
		assert.Equal(t, "pending", taskData["status"])
		assert.Equal(t, "distill_intake_to_knowledge", taskData["task_type"])
	}
}

// TestAgentTask_ShowNotFound verifies show returns error for nonexistent task.
func TestAgentTask_ShowNotFound(t *testing.T) {
	setupAgentTestDirs(t)

	_, stderr, exitCode := runTT(t, "agent", "task", "show", "T-NONEXISTENT00000000000000")
	assert.NotEqual(t, 0, exitCode)
	assert.Contains(t, stderr, "not found")
}

// TestAgentTask_SubmitFullFlow verifies the complete submit workflow:
// 1. notify -> 2. assist -> 3. task show -> 4. task submit -> 5. verify
func TestAgentTask_SubmitFullFlow(t *testing.T) {
	setupAgentTestDirs(t)
	cleanupAgentTasks(t)

	// 1. Create an intake event
	payload := `{
		"version": 1,
		"source_type": "coding_agent",
		"agent": "antigravity",
		"task_summary": "Full flow test event",
		"raw_notes": ["Full flow note 1", "Full flow note 2"]
	}`
	notifyStdout, notifyStderr, notifyExit := runTTWithInput(t, payload, "agent", "notify")
	require.Equal(t, 0, notifyExit, "Notify should succeed. stderr: %s", notifyStderr)

	// Extract event_id from notify result
	var notifyResult map[string]any
	require.NoError(t, json.Unmarshal([]byte(notifyStdout), &notifyResult))
	eventID, ok := notifyResult["event_id"].(string)
	require.True(t, ok && eventID != "", "Should have event_id in notify result")

	// 2. Assist
	assistStdout, assistStderr, assistExit := runTT(t, "agent", "assist", "--scope", "current-branch", "--force")
	require.Equal(t, 0, assistExit, "Assist should succeed. stderr: %s", assistStderr)

	var assistResult map[string]any
	require.NoError(t, json.Unmarshal([]byte(assistStdout), &assistResult))
	require.Equal(t, "requires_agent", assistResult["status"])
	taskID := assistResult["task_id"].(string)

	// 3. Task show
	showStdout, _, showExit := runTT(t, "agent", "task", "show", taskID)
	require.Equal(t, 0, showExit)

	var taskData map[string]any
	require.NoError(t, json.Unmarshal([]byte(showStdout), &taskData))
	assert.Equal(t, "pending", taskData["status"])

	// 4. Create a result file and submit
	batchJSON := `{
		"version": 1,
		"atoms": [{
			"type": "Fact",
			"title": "Integration test fact",
			"body": "This fact was created during integration testing of the full submit flow.",
			"importance": "medium",
			"confidence": 0.9,
			"activation_hints": {"positive": ["during integration testing"]},
			"source": {"event_ids": ["` + eventID + `"]}
		}]
	}`
	resultFile := filepath.Join(projectRoot(), "tmp", "test_submit_result.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(resultFile), 0755))
	require.NoError(t, os.WriteFile(resultFile, []byte(batchJSON), 0644))
	defer os.Remove(resultFile)

	submitStdout, submitStderr, submitExit := runTT(t, "agent", "task", "submit", taskID, "--file", resultFile)
	require.Equal(t, 0, submitExit, "Submit should succeed. stderr: %s, stdout: %s", submitStderr, submitStdout)

	var submitResult map[string]any
	require.NoError(t, json.Unmarshal([]byte(submitStdout), &submitResult))
	assert.Equal(t, "completed", submitResult["status"])

	knowledgeCreated, ok := submitResult["knowledge_created"].(float64)
	require.True(t, ok)
	assert.Equal(t, float64(1), knowledgeCreated)

	// 5. Verify task is now completed
	showStdout2, _, showExit2 := runTT(t, "agent", "task", "show", taskID)
	require.Equal(t, 0, showExit2)

	var taskData2 map[string]any
	require.NoError(t, json.Unmarshal([]byte(showStdout2), &taskData2))
	assert.Equal(t, "completed", taskData2["status"])

	// 6. Verify resubmit fails
	_, _, resubmitExit := runTT(t, "agent", "task", "submit", taskID, "--file", resultFile)
	assert.NotEqual(t, 0, resubmitExit, "Resubmit should fail")
}
