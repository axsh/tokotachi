package assist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/axsh/tokotachi/features/tt/internal/agent/storage"
	"github.com/oklog/ulid/v2"

	"crypto/rand"
)

// Handler orchestrates the assist command.
type Handler struct {
	index     *storage.Index
	fileStore *storage.FileStore
	varDir    string
}

// NewHandler creates a new assist Handler.
func NewHandler(varDir string) (*Handler, error) {
	dbPath := filepath.Join(varDir, "intake", "index.db")
	idx, err := storage.NewIndex(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index: %w", err)
	}
	fs := storage.NewFileStore(filepath.Join(varDir, "intake"))
	return &Handler{
		index:     idx,
		fileStore: fs,
		varDir:    varDir,
	}, nil
}

// Close releases resources.
func (h *Handler) Close() error {
	return h.index.Close()
}

// HandleAssist processes an assist request for the given branch.
// Returns (result, exitCode).
func (h *Handler) HandleAssist(branch string, force bool) (*agent.AssistResult, int) {
	// 1. Get pending events for this branch
	records, err := h.index.ListPendingByBranch(branch)
	if err != nil {
		return &agent.AssistResult{
			Status:  "error",
			Message: fmt.Sprintf("Failed to list pending events: %v", err),
		}, 1
	}
	if len(records) == 0 {
		return &agent.AssistResult{
			Status:  "no_pending_events",
			Message: "No pending intake events found for the current branch.",
		}, 0
	}

	// 2. Check for existing pending task (unless --force)
	if !force {
		existingTask, err := h.findExistingTask(branch)
		if err == nil && existingTask != nil {
			return &agent.AssistResult{
				Status:   "existing_task",
				TaskID:   existingTask.TaskID,
				TaskFile: h.taskFilePath(existingTask.TaskID),
				Message:  fmt.Sprintf("An existing pending task was found. Use 'tt agent task show %s' to view it.", existingTask.TaskID),
			}, 0
		}
	}

	// 3. Build task events from records and extract BranchPackage info
	events, bpID, bpKey := h.buildTaskEventsAndBP(records)

	// 4. Generate task ID
	taskID, err := h.generateTaskID()
	if err != nil {
		return &agent.AssistResult{
			Status:  "error",
			Message: fmt.Sprintf("Failed to generate task ID: %v", err),
		}, 1
	}

	// 5. Build paths
	schemaPath := "prompts/memory/schemas/knowledge-atom-batch.schema.json"
	submitCmd := fmt.Sprintf("tt agent task submit %s --file result.json", taskID)

	// 6. Build instruction
	instruction := buildInstruction(schemaPath, submitCmd)

	// 7. Create task
	task := &agent.AgentTask{
		TaskID:           taskID,
		Version:          1,
		TaskType:         "distill_intake_to_knowledge",
		Scope:            "current-branch",
		BranchPackageID:  bpID,
		BranchPackageKey: bpKey,
		Instruction:      instruction,
		Events:           events,
		OutputSchemaPath: schemaPath,
		SubmitCommand:    submitCmd,
		Status:           "pending",
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	}

	// 9. Write task file
	taskFile := h.taskFilePath(taskID)
	if err := h.writeTask(task, taskFile); err != nil {
		return &agent.AssistResult{
			Status:  "error",
			Message: fmt.Sprintf("Failed to write task file: %v", err),
		}, 1
	}

	return &agent.AssistResult{
		Status:      "requires_agent",
		TaskID:      taskID,
		TaskType:    "distill_intake_to_knowledge",
		EventCount:  len(events),
		TaskFile:    taskFile,
		NextCommand: fmt.Sprintf("tt agent task show %s", taskID),
	}, 0
}

// findExistingTask looks for an existing pending task for the same branch.
func (h *Handler) findExistingTask(branch string) (*agent.AgentTask, error) {
	dir := filepath.Join(h.varDir, "tasks", "pending")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var task agent.AgentTask
		if err := json.Unmarshal(data, &task); err != nil {
			continue
		}
		// Match by branch package ID prefix (branch slug is derived from BPID)
		if task.Status == "pending" && task.BranchPackageID != "" {
			return &task, nil
		}
	}
	return nil, fmt.Errorf("no existing task found")
}

// buildTaskEventsAndBP converts EventRecords to TaskEvents and extracts
// BranchPackageID and BranchPackageKey from the full event files.
func (h *Handler) buildTaskEventsAndBP(records []storage.EventRecord) (events []agent.TaskEvent, bpID, bpKey string) {
	for _, r := range records {
		te := agent.TaskEvent{
			EventID:     r.EventID,
			TaskSummary: r.TaskSummary,
		}

		// Try to read full event for raw_notes, changed_paths, flags, and branch_package
		createdAt, err := time.Parse("2006-01-02T15:04:05Z", r.CreatedAt)
		if err == nil {
			fullEvent, err := h.fileStore.ReadEvent(r.EventID, createdAt)
			if err == nil {
				te.RawNotes = fullEvent.RawNotes
				te.ChangedPaths = fullEvent.EffectiveChangedPaths
				te.Flags = fullEvent.Flags
				// Extract BranchPackage info from first event that has it
				if bpID == "" && fullEvent.BranchPackage != nil {
					bpID = fullEvent.BranchPackage.ID
					bpKey = fullEvent.BranchPackage.Key
				}
			}
		}

		// Ensure RawNotes is not nil
		if te.RawNotes == nil {
			te.RawNotes = []string{}
		}

		events = append(events, te)
	}
	return events, bpID, bpKey
}

// generateTaskID generates a new ULID-based task ID with "T-" prefix.
func (h *Handler) generateTaskID() (string, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now()), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate ULID: %w", err)
	}
	return "T-" + id.String(), nil
}

// taskFilePath returns the path for a task file.
func (h *Handler) taskFilePath(taskID string) string {
	return filepath.Join(h.varDir, "tasks", "pending", taskID+".json")
}

// writeTask writes the task as JSON to the given file path.
func (h *Handler) writeTask(task *agent.AgentTask, filePath string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("failed to create task directory: %w", err)
	}
	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}
	return os.WriteFile(filePath, data, 0644)
}
