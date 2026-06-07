package notify

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/axsh/tokotachi/features/tt/internal/agent/storage"
)

// Handler orchestrates the full notify pipeline.
type Handler struct {
	validator *Validator
	fileStore *storage.FileStore
	index     *storage.Index
	auditLog  *storage.AuditLog
	executor  GitExecutor
}

// NewHandler creates a new Handler with the given storage backends.
func NewHandler(schemasDir, varDir string) (*Handler, error) {
	v, err := NewValidator(schemasDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}

	intakeDir := filepath.Join(varDir, "intake")
	fs := storage.NewFileStore(intakeDir)

	dbPath := filepath.Join(varDir, "intake", "index.db")
	idx, err := storage.NewIndex(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	al := storage.NewAuditLog(filepath.Join(varDir, "logs"))

	return &Handler{
		validator: v,
		fileStore: fs,
		index:     idx,
		auditLog:  al,
		executor:  &RealGitExecutor{},
	}, nil
}

// NewHandlerWithExecutor creates a Handler with a custom GitExecutor (for testing).
func NewHandlerWithExecutor(schemasDir, varDir string, executor GitExecutor) (*Handler, error) {
	h, err := NewHandler(schemasDir, varDir)
	if err != nil {
		return nil, err
	}
	h.executor = executor
	return h, nil
}

// Close releases resources held by the handler.
func (h *Handler) Close() error {
	if h.index != nil {
		return h.index.Close()
	}
	return nil
}

// HandleNotify processes a notify request end-to-end.
// Returns (result, exitCode).
func (h *Handler) HandleNotify(inputJSON []byte, collectGitPaths bool) (*agent.NotifyResult, int) {
	now := time.Now().UTC()

	// Step 1: Parse JSON
	var payload agent.NotifyPayload
	if err := json.Unmarshal(inputJSON, &payload); err != nil {
		return h.reject(agent.CodeJSONParseError, agent.ExitJSONParseError,
			fmt.Sprintf("Failed to parse JSON: %v", err), nil)
	}

	// Step 2: Schema validation
	if err := h.validator.Validate(inputJSON); err != nil {
		return h.reject(agent.CodeSchemaValidationError, agent.ExitSchemaValidationError,
			fmt.Sprintf("Schema validation failed: %v", err), nil)
	}

	// Step 3: Normalize
	if err := NormalizePayload(&payload); err != nil {
		return h.reject(agent.CodeSchemaValidationError, agent.ExitSchemaValidationError,
			fmt.Sprintf("Normalization failed: %v", err), nil)
	}

	// Step 4: Build IntakeEvent and supplement
	event := &agent.IntakeEvent{
		NotifyPayload: payload,
	}
	warnings := SupplementEnvironment(event, h.executor, collectGitPaths)

	// Step 5: Generate IDs
	eventID, err := GenerateEventID()
	if err != nil {
		return h.reject(agent.CodeStorageWriteFailed, agent.ExitStorageWriteFailed,
			fmt.Sprintf("Failed to generate event ID: %v", err), nil)
	}
	event.EventID = eventID
	event.InstanceID = eventID
	event.ContentHash = ComputeContentHash(event)
	event.ContentID = ComputeContentID(event)

	// Step 6: Set timestamps
	event.Timestamps = agent.Timestamps{
		CreatedAt: now,
		StoredAt:  now,
	}

	// Step 7: Check idempotency and store in index
	indexState := "ok"
	existingID, err := h.index.Store(event)
	if err != nil {
		warnings = append(warnings, "Index storage failed: "+err.Error())
		indexState = "degraded"
	}
	if existingID != "" {
		// Idempotent hit: return the existing event_id
		return &agent.NotifyResult{
			Status:      "accepted",
			Code:        agent.CodeOK,
			Mode:        "deferred",
			EventID:     existingID,
			InstanceID:  existingID,
			ContentHash: event.ContentHash,
			ContentID:   event.ContentID,
			IndexState:  indexState,
			Message:     "Idempotent: event already exists",
		}, agent.ExitOK
	}

	// Step 8: Write file
	storedAt, err := h.fileStore.Write(event)
	if err != nil {
		return h.reject(agent.CodeStorageWriteFailed, agent.ExitStorageWriteFailed,
			fmt.Sprintf("Failed to write event file: %v", err), event)
	}

	// Step 9: Build result
	status := "accepted"
	if len(warnings) > 0 {
		status = "accepted_with_warnings"
	}

	result := &agent.NotifyResult{
		Status:      status,
		Code:        agent.CodeOK,
		Mode:        "deferred",
		EventID:     event.EventID,
		InstanceID:  event.InstanceID,
		ContentHash: event.ContentHash,
		ContentID:   event.ContentID,
		StoredAt:    storedAt,
		IndexState:  indexState,
		Warnings:    warnings,
	}

	// Step 10: Audit log (best effort)
	_ = h.auditLog.Append(event, result)

	return result, agent.ExitOK
}

// reject builds a rejection result and writes to audit log.
func (h *Handler) reject(code string, exitCode int, message string, event *agent.IntakeEvent) (*agent.NotifyResult, int) {
	result := &agent.NotifyResult{
		Status:  "rejected",
		Code:    code,
		Message: message,
	}

	if event != nil {
		_ = h.auditLog.Append(event, result)
	}

	return result, exitCode
}
