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

// testBatchSchema is a relaxed schema for testing (no event_id pattern constraint).
const testBatchSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "knowledge-atom-batch.schema.json",
  "type": "object",
  "required": ["version", "atoms"],
  "properties": {
    "version": { "type": "integer", "const": 1 },
    "atoms": {
      "type": "array", "minItems": 1,
      "items": {
        "type": "object",
        "required": ["type", "title", "body", "importance", "confidence", "activation_hints", "source"],
        "properties": {
          "type": { "type": "string", "enum": ["Fact", "Decision", "Constraint", "Pattern", "Warning", "Skill"] },
          "title": { "type": "string", "minLength": 1 },
          "body": { "type": "string", "minLength": 1 },
          "importance": { "type": "string", "enum": ["low", "medium", "high", "critical"] },
          "confidence": { "type": "number", "minimum": 0.0, "maximum": 1.0 },
          "activation_hints": {
            "type": "object",
            "properties": {
              "positive": { "type": "array", "minItems": 1, "items": { "type": "string" } },
              "negative": { "type": "array", "items": { "type": "string" } }
            },
            "required": ["positive"]
          },
          "source": {
            "type": "object",
            "required": ["event_ids"],
            "properties": {
              "event_ids": { "type": "array", "minItems": 1, "items": { "type": "string" } }
            }
          }
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}`

// setupSubmitTest creates the necessary directory structure and files for submit tests.
func setupSubmitTest(t *testing.T) (memoryRoot, varDir, schemasDir string) {
	t.Helper()
	tmpDir := t.TempDir()
	memoryRoot = filepath.Join(tmpDir, "memory")
	varDir = filepath.Join(memoryRoot, "var")
	schemasDir = filepath.Join(memoryRoot, "schemas")

	// Create directories
	for _, d := range []string{
		filepath.Join(varDir, "intake"),
		filepath.Join(varDir, "tasks", "pending"),
		filepath.Join(varDir, "tasks", "completed"),
		filepath.Join(varDir, "tasks", "failed"),
		schemasDir,
	} {
		require.NoError(t, os.MkdirAll(d, 0755))
	}

	// Use relaxed schema for unit tests (event_id pattern validated by integration tests)
	require.NoError(t, os.WriteFile(
		filepath.Join(schemasDir, "knowledge-atom-batch.schema.json"),
		[]byte(testBatchSchema), 0644))

	return memoryRoot, varDir, schemasDir
}

func createPendingTask(t *testing.T, varDir string, taskID string, eventIDs []string) {
	t.Helper()
	events := make([]agent.TaskEvent, len(eventIDs))
	for i, eid := range eventIDs {
		events[i] = agent.TaskEvent{EventID: eid, TaskSummary: "test event", RawNotes: []string{"note1"}}
	}
	task := agent.AgentTask{
		TaskID:          taskID,
		Version:         1,
		TaskType:        "distill_intake_to_knowledge",
		Scope:           "current-branch",
		BranchPackageID: "BR-test-branch-abcdef12",
		Status:          "pending",
		Events:          events,
		Instruction:     "test instruction",
	}
	data, _ := json.MarshalIndent(task, "", "  ")
	require.NoError(t, os.WriteFile(filepath.Join(varDir, "tasks", "pending", taskID+".json"), data, 0644))
}

func createValidBatch(eventIDs []string) agent.KnowledgeAtomBatch {
	return agent.KnowledgeAtomBatch{
		Version: 1,
		Atoms: []agent.KnowledgeAtomCandidate{
			{
				Type:       agent.KnowledgeTypeFact,
				Title:      "Test Fact",
				Body:       "This is a test fact.",
				Importance: "medium",
				Confidence: 0.8,
				ActivationHints: agent.ActivationHints{
					Positive: []string{"when testing"},
				},
				Source: agent.CandidateSource{EventIDs: eventIDs},
			},
		},
	}
}

func TestHandleSubmit_Success(t *testing.T) {
	memoryRoot, varDir, schemasDir := setupSubmitTest(t)

	taskID := "T-01JABC123456789012345678"
	eventIDs := []string{"E-01JABC123456789012345678"}
	createPendingTask(t, varDir, taskID, eventIDs)

	batch := createValidBatch(eventIDs)
	batchData, _ := json.MarshalIndent(batch, "", "  ")
	resultFile := filepath.Join(t.TempDir(), "result.json")
	require.NoError(t, os.WriteFile(resultFile, batchData, 0644))

	h, err := NewSubmitHandler(memoryRoot, varDir, schemasDir)
	require.NoError(t, err)
	defer h.Close()

	result, exitCode := h.HandleSubmit(taskID, resultFile)

	assert.Equal(t, 0, exitCode, "Submit should succeed, got message: %s", result.Message)
	assert.Equal(t, "completed", result.Status)
	assert.Equal(t, taskID, result.TaskID)
	assert.Equal(t, 1, result.KnowledgeCreated)
	assert.Len(t, result.KnowledgeFiles, 1)

	// Verify knowledge file exists
	knowledgePath := filepath.Join(memoryRoot, "branches", "BR-test-branch-abcdef12", "knowledge")
	entries, err := os.ReadDir(knowledgePath)
	require.NoError(t, err)
	assert.Len(t, entries, 1)

	// Verify task moved to completed
	_, err = os.Stat(filepath.Join(varDir, "tasks", "pending", taskID+".json"))
	assert.True(t, os.IsNotExist(err), "Task should be removed from pending")

	completedData, err := os.ReadFile(filepath.Join(varDir, "tasks", "completed", taskID+".json"))
	require.NoError(t, err)
	var completedTask agent.AgentTask
	require.NoError(t, json.Unmarshal(completedData, &completedTask))
	assert.Equal(t, "completed", completedTask.Status)
}

func TestHandleSubmit_InvalidEventReference(t *testing.T) {
	memoryRoot, varDir, schemasDir := setupSubmitTest(t)

	taskID := "T-01JABC123456789012345678"
	createPendingTask(t, varDir, taskID, []string{"E-01JABC123456789012345678"})

	// Create batch referencing a different event
	batch := createValidBatch([]string{"E-DIFFERENT_EVENT_REFERENCE"})
	batchData, _ := json.MarshalIndent(batch, "", "  ")
	resultFile := filepath.Join(t.TempDir(), "result.json")
	require.NoError(t, os.WriteFile(resultFile, batchData, 0644))

	h, err := NewSubmitHandler(memoryRoot, varDir, schemasDir)
	require.NoError(t, err)
	defer h.Close()

	result, exitCode := h.HandleSubmit(taskID, resultFile)

	assert.NotEqual(t, 0, exitCode)
	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Message, "Invalid event reference")
}

func TestHandleSubmit_TaskNotFound(t *testing.T) {
	memoryRoot, varDir, schemasDir := setupSubmitTest(t)

	h, err := NewSubmitHandler(memoryRoot, varDir, schemasDir)
	require.NoError(t, err)
	defer h.Close()

	result, exitCode := h.HandleSubmit("T-NONEXISTENT", "result.json")

	assert.NotEqual(t, 0, exitCode)
	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Message, "Task not found")
}

func TestHandleSubmit_SchemaValidationError(t *testing.T) {
	memoryRoot, varDir, schemasDir := setupSubmitTest(t)

	taskID := "T-01JABC123456789012345678"
	createPendingTask(t, varDir, taskID, []string{"E-01"})

	// Create invalid batch (missing required fields)
	invalidBatch := `{"version": 1, "atoms": [{"type": "InvalidType"}]}`
	resultFile := filepath.Join(t.TempDir(), "result.json")
	require.NoError(t, os.WriteFile(resultFile, []byte(invalidBatch), 0644))

	h, err := NewSubmitHandler(memoryRoot, varDir, schemasDir)
	require.NoError(t, err)
	defer h.Close()

	result, exitCode := h.HandleSubmit(taskID, resultFile)

	assert.NotEqual(t, 0, exitCode)
	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.Message, "Schema validation failed")
}

func TestHandleSubmit_CompletedTaskResubmit(t *testing.T) {
	memoryRoot, varDir, schemasDir := setupSubmitTest(t)

	taskID := "T-01JABC123456789012345678"
	eventIDs := []string{"E-01JABC123456789012345678"}
	createPendingTask(t, varDir, taskID, eventIDs)

	batch := createValidBatch(eventIDs)
	batchData, _ := json.MarshalIndent(batch, "", "  ")
	resultFile := filepath.Join(t.TempDir(), "result.json")
	require.NoError(t, os.WriteFile(resultFile, batchData, 0644))

	h, err := NewSubmitHandler(memoryRoot, varDir, schemasDir)
	require.NoError(t, err)
	defer h.Close()

	result, exitCode := h.HandleSubmit(taskID, resultFile)
	require.Equal(t, 0, exitCode)
	assert.Equal(t, "completed", result.Status)

	// Second submit (should fail - task no longer in pending)
	h2, err := NewSubmitHandler(memoryRoot, varDir, schemasDir)
	require.NoError(t, err)
	defer h2.Close()

	result2, exitCode2 := h2.HandleSubmit(taskID, resultFile)
	assert.NotEqual(t, 0, exitCode2)
	assert.Equal(t, "error", result2.Status)
	assert.Contains(t, result2.Message, "Task not found")
}
