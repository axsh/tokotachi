package task

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/axsh/tokotachi/features/tt/internal/agent/storage"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

// SubmitHandler orchestrates the task submit workflow.
type SubmitHandler struct {
	varDir     string
	memoryRoot string
	schemasDir string
	index      *storage.Index
	fileStore  *storage.FileStore
}

// NewSubmitHandler creates a new SubmitHandler.
func NewSubmitHandler(memoryRoot, varDir, schemasDir string) (*SubmitHandler, error) {
	dbPath := filepath.Join(varDir, "intake", "index.db")
	idx, err := storage.NewIndex(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index: %w", err)
	}
	fs := storage.NewFileStore(filepath.Join(varDir, "intake"))
	return &SubmitHandler{
		varDir:     varDir,
		memoryRoot: memoryRoot,
		schemasDir: schemasDir,
		index:      idx,
		fileStore:  fs,
	}, nil
}

// Close releases resources.
func (h *SubmitHandler) Close() error {
	return h.index.Close()
}

// HandleSubmit processes a task submission.
func (h *SubmitHandler) HandleSubmit(taskID, resultFile string) (*agent.SubmitResult, int) {
	// 1. Load task
	taskPath := filepath.Join(h.varDir, "tasks", "pending", taskID+".json")
	taskData, err := os.ReadFile(taskPath)
	if err != nil {
		return &agent.SubmitResult{
			Status:  "error",
			TaskID:  taskID,
			Message: fmt.Sprintf("Task not found: %v", err),
		}, 1
	}
	var task agent.AgentTask
	if err := json.Unmarshal(taskData, &task); err != nil {
		return h.errorResult(taskID, fmt.Sprintf("Failed to parse task: %v", err), 1)
	}
	if task.Status != "pending" {
		return &agent.SubmitResult{
			Status:  "error",
			TaskID:  taskID,
			Message: "Task is not in pending status",
		}, 1
	}

	// 2. Load and validate result
	resultData, err := os.ReadFile(resultFile)
	if err != nil {
		return h.errorResult(taskID, fmt.Sprintf("Failed to read result file: %v", err), 1)
	}

	// 2a. Schema validation
	if err := h.validateBatchSchema(resultData); err != nil {
		return &agent.SubmitResult{
			Status:  "error",
			TaskID:  taskID,
			Message: fmt.Sprintf("Schema validation failed: %v", err),
		}, agent.ExitSchemaValidationError
	}

	var batch agent.KnowledgeAtomBatch
	if err := json.Unmarshal(resultData, &batch); err != nil {
		return h.errorResult(taskID, fmt.Sprintf("Failed to parse result: %v", err), 1)
	}

	// 3. Validate event references
	validEventIDs := make(map[string]bool)
	for _, e := range task.Events {
		validEventIDs[e.EventID] = true
	}
	for _, atom := range batch.Atoms {
		for _, eid := range atom.Source.EventIDs {
			if !validEventIDs[eid] {
				return &agent.SubmitResult{
					Status:  "error",
					TaskID:  taskID,
					Message: fmt.Sprintf("Invalid event reference: %s is not part of this task", eid),
				}, 1
			}
		}
	}

	// 4. Generate Knowledge Atoms with IDs and supplemented fields
	now := time.Now().UTC()
	var atoms []agent.KnowledgeAtom
	for _, candidate := range batch.Atoms {
		atomID, err := generateKnowledgeID()
		if err != nil {
			return h.errorResult(taskID, fmt.Sprintf("Failed to generate ID: %v", err), 1)
		}
		gitBranch := ""
		if task.BranchPackageID != "" {
			gitBranch = ExtractBranchFromBPID(task.BranchPackageID)
		}
		atom := agent.KnowledgeAtom{
			ID:              atomID,
			Version:         1,
			Type:            candidate.Type,
			Title:           candidate.Title,
			Body:            candidate.Body,
			Status:          "draft",
			Importance:      candidate.Importance,
			Confidence:      candidate.Confidence,
			ActivationHints: candidate.ActivationHints,
			Source: agent.KnowledgeSource{
				EventIDs:        candidate.Source.EventIDs,
				BranchPackageID: task.BranchPackageID,
				Agent:           "coding_agent",
				GitBranch:       gitBranch,
			},
			Timestamps: agent.KnowledgeTS{CreatedAt: now},
		}
		atoms = append(atoms, atom)
	}

	// 5. Ensure branch manifest
	if task.BranchPackageID != "" {
		if err := EnsureBranchManifest(h.memoryRoot, task, now); err != nil {
			return h.errorResult(taskID, fmt.Sprintf("Failed to create manifest: %v", err), 1)
		}
	}

	// 6. Write Knowledge Atom YAML files
	var knowledgeFiles []string
	for _, atom := range atoms {
		filePath, err := WriteKnowledgeAtom(h.memoryRoot, task.BranchPackageID, &atom)
		if err != nil {
			return h.errorResult(taskID, fmt.Sprintf("Failed to write atom: %v", err), agent.ExitStorageWriteFailed)
		}
		knowledgeFiles = append(knowledgeFiles, filePath)
	}

	// 7. Move task to completed
	if err := h.moveTask(taskID, "pending", "completed"); err != nil {
		return h.errorResult(taskID, fmt.Sprintf("Failed to move task: %v", err), 1)
	}
	// Update task status in completed file
	task.Status = "completed"
	completedPath := filepath.Join(h.varDir, "tasks", "completed", taskID+".json")
	taskJSON, _ := json.MarshalIndent(task, "", "  ")
	_ = os.WriteFile(completedPath, taskJSON, 0644)

	// 8. Move intake events to processed
	processedEventIDs := make(map[string]bool)
	for _, atom := range atoms {
		for _, eid := range atom.Source.EventIDs {
			processedEventIDs[eid] = true
		}
	}

	var processedEvents []string
	for eid := range processedEventIDs {
		// Update index status
		if err := h.index.UpdateStatus(eid, "processed"); err != nil {
			continue
		}
		// Move file (need createdAt from index record)
		record, err := h.index.GetByEventID(eid)
		if err == nil {
			createdAt, err := time.Parse("2006-01-02T15:04:05Z", record.CreatedAt)
			if err == nil {
				_ = h.fileStore.MoveToProcessed(eid, createdAt)
			}
		}
		processedEvents = append(processedEvents, eid)
	}

	return &agent.SubmitResult{
		Status:           "completed",
		TaskID:           taskID,
		KnowledgeCreated: len(atoms),
		KnowledgeFiles:   knowledgeFiles,
		ProcessedEvents:  processedEvents,
	}, 0
}

// validateBatchSchema validates the result data against knowledge-atom-batch.schema.json.
func (h *SubmitHandler) validateBatchSchema(data []byte) error {
	schemaPath := filepath.Join(h.schemasDir, "knowledge-atom-batch.schema.json")
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema: %w", err)
	}

	c := jsonschema.NewCompiler()
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaData))
	if err != nil {
		return fmt.Errorf("failed to unmarshal schema: %w", err)
	}
	if err := c.AddResource("knowledge-atom-batch.schema.json", doc); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}
	schema, err := c.Compile("knowledge-atom-batch.schema.json")
	if err != nil {
		return fmt.Errorf("failed to compile schema: %w", err)
	}

	var inst any
	if err := json.Unmarshal(data, &inst); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	return schema.Validate(inst)
}

// moveTask moves a task file from one status directory to another.
func (h *SubmitHandler) moveTask(taskID, fromStatus, toStatus string) error {
	src := filepath.Join(h.varDir, "tasks", fromStatus, taskID+".json")
	dstDir := filepath.Join(h.varDir, "tasks", toStatus)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	dst := filepath.Join(dstDir, taskID+".json")
	return os.Rename(src, dst)
}

// errorResult creates an error SubmitResult.
func (h *SubmitHandler) errorResult(taskID, message string, exitCode int) (*agent.SubmitResult, int) {
	return &agent.SubmitResult{
		Status:  "error",
		TaskID:  taskID,
		Message: message,
	}, exitCode
}
