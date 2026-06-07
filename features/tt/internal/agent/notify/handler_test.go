package notify

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestHandler(t *testing.T) (*Handler, *mockGitExecutor) {
	t.Helper()
	tmpDir := t.TempDir()
	schemasDir := filepath.Join("..", "..", "..", "..", "..", "prompts", "memory", "schemas")
	varDir := filepath.Join(tmpDir, "var")

	mock := newMockGitExecutor()
	// Default: no git repo
	mock.errors[fmt.Sprintf("%v", []string{"rev-parse", "--show-toplevel"})] = fmt.Errorf("not a git repo")

	h, err := NewHandlerWithExecutor(schemasDir, varDir, mock)
	require.NoError(t, err)
	t.Cleanup(func() { h.Close() })

	return h, mock
}

func setupTestHandlerWithGit(t *testing.T) (*Handler, *mockGitExecutor) {
	t.Helper()
	tmpDir := t.TempDir()
	schemasDir := filepath.Join("..", "..", "..", "..", "..", "prompts", "memory", "schemas")
	varDir := filepath.Join(tmpDir, "var")

	mock := newMockGitExecutor()
	mock.responses[fmt.Sprintf("%v", []string{"rev-parse", "--show-toplevel"})] = "/repo"
	mock.responses[fmt.Sprintf("%v", []string{"rev-parse", "--abbrev-ref", "HEAD"})] = "feature/test"
	mock.responses[fmt.Sprintf("%v", []string{"rev-parse", "HEAD"})] = "abc123"
	mock.responses[fmt.Sprintf("%v", []string{"status", "--porcelain"})] = ""
	mock.responses[fmt.Sprintf("%v", []string{"rev-parse", "--verify", "refs/heads/main"})] = "exists"
	mock.responses[fmt.Sprintf("%v", []string{"merge-base", "HEAD", "main"})] = "base123"
	mock.responses[fmt.Sprintf("%v", []string{"remote", "get-url", "origin"})] = "git@github.com:axsh/tokotachi.git"

	h, err := NewHandlerWithExecutor(schemasDir, varDir, mock)
	require.NoError(t, err)
	t.Cleanup(func() { h.Close() })

	return h, mock
}

func validPayloadJSON() []byte {
	return []byte(`{
		"version": 1,
		"source_type": "coding_agent",
		"agent": "antigravity",
		"task_summary": "Implement auth middleware",
		"raw_notes": ["Added JWT validation", "Updated config schema"]
	}`)
}

func TestHandler_HandleNotify_Success(t *testing.T) {
	h, _ := setupTestHandlerWithGit(t)

	result, exitCode := h.HandleNotify(validPayloadJSON(), false)

	assert.Equal(t, agent.ExitOK, exitCode)
	assert.Equal(t, "accepted", result.Status)
	assert.Equal(t, agent.CodeOK, result.Code)
	assert.Equal(t, "deferred", result.Mode)
	assert.NotEmpty(t, result.EventID)
	assert.NotEmpty(t, result.ContentHash)
	assert.NotEmpty(t, result.ContentID)
	assert.NotEmpty(t, result.StoredAt)
	assert.Equal(t, "ok", result.IndexState)
}

func TestHandler_HandleNotify_InvalidJSON(t *testing.T) {
	h, _ := setupTestHandler(t)

	result, exitCode := h.HandleNotify([]byte("not json"), false)

	assert.Equal(t, agent.ExitJSONParseError, exitCode)
	assert.Equal(t, "rejected", result.Status)
	assert.Equal(t, agent.CodeJSONParseError, result.Code)
}

func TestHandler_HandleNotify_SchemaViolation(t *testing.T) {
	h, _ := setupTestHandler(t)

	// Missing required field raw_notes
	input := []byte(`{
		"version": 1,
		"source_type": "coding_agent",
		"agent": "antigravity",
		"task_summary": "task"
	}`)

	result, exitCode := h.HandleNotify(input, false)

	assert.Equal(t, agent.ExitSchemaValidationError, exitCode)
	assert.Equal(t, "rejected", result.Status)
	assert.Equal(t, agent.CodeSchemaValidationError, result.Code)
}

func TestHandler_HandleNotify_Idempotency(t *testing.T) {
	h, _ := setupTestHandlerWithGit(t)

	input := []byte(`{
		"version": 1,
		"source_type": "coding_agent",
		"agent": "antigravity",
		"task_summary": "task",
		"raw_notes": ["note"],
		"client_request_id": "idempotent-001"
	}`)

	// First call
	result1, exitCode1 := h.HandleNotify(input, false)
	assert.Equal(t, agent.ExitOK, exitCode1)
	assert.Equal(t, "accepted", result1.Status)
	firstEventID := result1.EventID

	// Second call with same client_request_id
	result2, exitCode2 := h.HandleNotify(input, false)
	assert.Equal(t, agent.ExitOK, exitCode2)
	assert.Equal(t, "accepted", result2.Status)
	assert.Equal(t, firstEventID, result2.EventID,
		"idempotent call should return same event_id")
}

func TestHandler_HandleNotify_NoGitWarning(t *testing.T) {
	h, _ := setupTestHandler(t) // no git setup

	result, exitCode := h.HandleNotify(validPayloadJSON(), false)

	assert.Equal(t, agent.ExitOK, exitCode)
	assert.Equal(t, "accepted_with_warnings", result.Status)
	assert.Contains(t, result.Warnings, agent.CodeNoGitRepository)
}
