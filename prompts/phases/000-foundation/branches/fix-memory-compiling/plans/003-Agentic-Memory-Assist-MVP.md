# 003-Agentic-Memory-Assist-MVP

> **Source Specification**: [003-Agentic-Memory-Assist-MVP.md](../ideas/003-Agentic-Memory-Assist-MVP.md)

## Goal Description

`tt agent notify` で蓄積された Intake Event を、Coding Agent が整理して Knowledge Atom (自己完結した知識の最小単位) に変換するための協調ワークフローを実装する。`tt agent assist` がタスクを生成し、`tt agent task show/submit` でタスクの閲覧・結果提出を行う。

## User Review Required

None.

## Requirement Traceability

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| R1: Knowledge Atom Schema | Proposed Changes > Schema Files > knowledge-atom.schema.json |
| R2: Knowledge Atom Batch Schema | Proposed Changes > Schema Files > knowledge-atom-batch.schema.json |
| R3: Agent Task Schema | Proposed Changes > Schema Files > agent-task.schema.json |
| R4: `tt agent assist --scope current-branch` | Proposed Changes > Command Layer > agent_assist.go, Internal Logic > assist/ |
| R5: `tt agent task show <task-id>` | Proposed Changes > Command Layer > agent_task.go, Internal Logic > task/ |
| R6: `tt agent task submit <task-id> --file` | Proposed Changes > Command Layer > agent_task.go, Internal Logic > task/ |
| R7: Branch Manifest 自動作成 | Proposed Changes > Internal Logic > task/manifest.go |
| R8: Intake Event の processed 移行 | Proposed Changes > Storage Layer > index.go, filestore.go |
| R9: Wrapper スクリプト | Proposed Changes > Wrapper Scripts |
| R10: ディレクトリレイアウト | Step-by-Step > Step 1 |
| R11: Knowledge Atom の status 管理 | Proposed Changes > Internal Logic > task/knowledge.go (status: draft 固定) |

## Proposed Changes

### Schema Files (`prompts/memory/schemas/`)

---

#### [NEW] [knowledge-atom.schema.json](file://prompts/memory/schemas/knowledge-atom.schema.json)

*   **Description**: Knowledge Atom の JSON Schema 定義
*   **Technical Design**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "knowledge-atom.schema.json",
  "title": "Knowledge Atom",
  "description": "Self-contained unit of knowledge derived from intake events",
  "type": "object",
  "required": ["id", "version", "type", "title", "body", "status", "importance", "confidence", "source", "timestamps"],
  "properties": {
    "id": {
      "type": "string",
      "pattern": "^K-[0-9A-Z]{26}$",
      "description": "ULID-based identifier with K- prefix"
    },
    "version": { "type": "integer", "const": 1 },
    "type": {
      "type": "string",
      "enum": ["Fact", "Decision", "Constraint", "Pattern", "Warning", "Skill"]
    },
    "title": { "type": "string", "minLength": 1, "maxLength": 200 },
    "body": { "type": "string", "minLength": 1, "maxLength": 2000 },
    "status": { "type": "string", "enum": ["draft", "active"] },
    "importance": { "type": "string", "enum": ["low", "medium", "high", "critical"] },
    "confidence": { "type": "number", "minimum": 0.0, "maximum": 1.0 },
    "activation_hints": {
      "type": "object",
      "properties": {
        "positive": {
          "type": "array",
          "minItems": 1, "maxItems": 10,
          "items": { "type": "string", "minLength": 1, "maxLength": 200 }
        },
        "negative": {
          "type": "array",
          "maxItems": 10,
          "items": { "type": "string", "minLength": 1, "maxLength": 200 }
        }
      },
      "required": ["positive"]
    },
    "source": {
      "type": "object",
      "required": ["event_ids", "branch_package_id", "agent", "git_branch"],
      "properties": {
        "event_ids": {
          "type": "array",
          "minItems": 1,
          "items": { "type": "string", "pattern": "^E-[0-9A-Z]{26}$" }
        },
        "branch_package_id": { "type": "string" },
        "agent": { "type": "string" },
        "git_branch": { "type": "string" }
      }
    },
    "timestamps": {
      "type": "object",
      "required": ["created_at"],
      "properties": {
        "created_at": { "type": "string", "format": "date-time" }
      }
    }
  },
  "additionalProperties": false
}
```

---

#### [NEW] [knowledge-atom-batch.schema.json](file://prompts/memory/schemas/knowledge-atom-batch.schema.json)

*   **Description**: Coding Agent が submit する Knowledge Atom バッチの JSON Schema
*   **Technical Design**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "knowledge-atom-batch.schema.json",
  "title": "Knowledge Atom Batch",
  "description": "Batch of Knowledge Atom candidates submitted by a coding agent",
  "type": "object",
  "required": ["version", "atoms"],
  "properties": {
    "version": { "type": "integer", "const": 1 },
    "atoms": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["type", "title", "body", "importance", "confidence", "activation_hints", "source"],
        "properties": {
          "type": {
            "type": "string",
            "enum": ["Fact", "Decision", "Constraint", "Pattern", "Warning", "Skill"]
          },
          "title": { "type": "string", "minLength": 1, "maxLength": 200 },
          "body": { "type": "string", "minLength": 1, "maxLength": 2000 },
          "importance": { "type": "string", "enum": ["low", "medium", "high", "critical"] },
          "confidence": { "type": "number", "minimum": 0.0, "maximum": 1.0 },
          "activation_hints": {
            "type": "object",
            "properties": {
              "positive": {
                "type": "array", "minItems": 1, "maxItems": 10,
                "items": { "type": "string", "minLength": 1, "maxLength": 200 }
              },
              "negative": {
                "type": "array", "maxItems": 10,
                "items": { "type": "string", "minLength": 1, "maxLength": 200 }
              }
            },
            "required": ["positive"]
          },
          "source": {
            "type": "object",
            "required": ["event_ids"],
            "properties": {
              "event_ids": {
                "type": "array", "minItems": 1,
                "items": { "type": "string", "pattern": "^E-[0-9A-Z]{26}$" }
              }
            }
          }
        },
        "additionalProperties": false
      }
    }
  },
  "additionalProperties": false
}
```

---

#### [NEW] [agent-task.schema.json](file://prompts/memory/schemas/agent-task.schema.json)

*   **Description**: Agent Task の JSON Schema
*   **Technical Design**:

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "agent-task.schema.json",
  "title": "Agent Task",
  "description": "Task generated by tt agent assist for coding agent processing",
  "type": "object",
  "required": ["task_id", "version", "task_type", "scope", "instruction", "events", "output_schema_path", "submit_command", "status", "created_at"],
  "properties": {
    "task_id": { "type": "string", "pattern": "^T-[0-9A-Z]{26}$" },
    "version": { "type": "integer", "const": 1 },
    "task_type": { "type": "string", "enum": ["distill_intake_to_knowledge"] },
    "scope": { "type": "string", "enum": ["current-branch"] },
    "branch_package_id": { "type": "string" },
    "branch_package_key": { "type": "string" },
    "instruction": { "type": "string", "minLength": 1 },
    "events": {
      "type": "array",
      "minItems": 1,
      "items": {
        "type": "object",
        "required": ["event_id", "task_summary", "raw_notes"],
        "properties": {
          "event_id": { "type": "string" },
          "task_summary": { "type": "string" },
          "raw_notes": { "type": "array", "items": { "type": "string" } },
          "changed_paths": { "type": "array", "items": { "type": "string" } },
          "flags": { "type": "object" }
        }
      }
    },
    "output_schema_path": { "type": "string" },
    "submit_command": { "type": "string" },
    "status": { "type": "string", "enum": ["pending", "completed", "failed"] },
    "created_at": { "type": "string", "format": "date-time" }
  },
  "additionalProperties": false
}
```

---

### Go 型定義 (`features/tt/internal/agent/`)

---

#### [MODIFY] [types.go](file://features/tt/internal/agent/types.go)

*   **Description**: Knowledge Atom, Agent Task, Branch Manifest, Assist/Submit 結果の型を追加
*   **Technical Design**:
    *   既存の型 (NotifyPayload, IntakeEvent, NotifyResult 等) は変更しない
    *   以下の型を末尾に追加する

```go
// KnowledgeAtomType enumerates the allowed knowledge types.
type KnowledgeAtomType string

const (
	KnowledgeTypeFact       KnowledgeAtomType = "Fact"
	KnowledgeTypeDecision   KnowledgeAtomType = "Decision"
	KnowledgeTypeConstraint KnowledgeAtomType = "Constraint"
	KnowledgeTypePattern    KnowledgeAtomType = "Pattern"
	KnowledgeTypeWarning    KnowledgeAtomType = "Warning"
	KnowledgeTypeSkill      KnowledgeAtomType = "Skill"
)

// ValidKnowledgeTypes is the set of valid knowledge atom types.
var ValidKnowledgeTypes = map[KnowledgeAtomType]bool{
	KnowledgeTypeFact: true, KnowledgeTypeDecision: true,
	KnowledgeTypeConstraint: true, KnowledgeTypePattern: true,
	KnowledgeTypeWarning: true, KnowledgeTypeSkill: true,
}

// KnowledgeAtom represents a self-contained unit of knowledge.
type KnowledgeAtom struct {
	ID              string            `json:"id" yaml:"id"`
	Version         int               `json:"version" yaml:"version"`
	Type            KnowledgeAtomType `json:"type" yaml:"type"`
	Title           string            `json:"title" yaml:"title"`
	Body            string            `json:"body" yaml:"body"`
	Status          string            `json:"status" yaml:"status"`
	Importance      string            `json:"importance" yaml:"importance"`
	Confidence      float64           `json:"confidence" yaml:"confidence"`
	ActivationHints ActivationHints   `json:"activation_hints" yaml:"activation_hints"`
	Source          KnowledgeSource   `json:"source" yaml:"source"`
	Timestamps      KnowledgeTS       `json:"timestamps" yaml:"timestamps"`
}

// ActivationHints provides retrieval context for a knowledge atom.
type ActivationHints struct {
	Positive []string `json:"positive" yaml:"positive"`
	Negative []string `json:"negative,omitempty" yaml:"negative,omitempty"`
}

// KnowledgeSource tracks the origin of a knowledge atom.
type KnowledgeSource struct {
	EventIDs        []string `json:"event_ids" yaml:"event_ids"`
	BranchPackageID string   `json:"branch_package_id" yaml:"branch_package_id"`
	Agent           string   `json:"agent" yaml:"agent"`
	GitBranch       string   `json:"git_branch" yaml:"git_branch"`
}

// KnowledgeTS holds creation timestamp for a knowledge atom.
type KnowledgeTS struct {
	CreatedAt time.Time `json:"created_at" yaml:"created_at"`
}

// KnowledgeAtomBatch represents the input from a coding agent.
type KnowledgeAtomBatch struct {
	Version int                     `json:"version"`
	Atoms   []KnowledgeAtomCandidate `json:"atoms"`
}

// KnowledgeAtomCandidate is a single atom in the batch (no id/version/status/timestamps).
type KnowledgeAtomCandidate struct {
	Type            KnowledgeAtomType `json:"type"`
	Title           string            `json:"title"`
	Body            string            `json:"body"`
	Importance      string            `json:"importance"`
	Confidence      float64           `json:"confidence"`
	ActivationHints ActivationHints   `json:"activation_hints"`
	Source          CandidateSource   `json:"source"`
}

// CandidateSource holds only event_ids (other fields auto-filled by tt).
type CandidateSource struct {
	EventIDs []string `json:"event_ids"`
}

// AgentTask represents a task for a coding agent.
type AgentTask struct {
	TaskID           string           `json:"task_id"`
	Version          int              `json:"version"`
	TaskType         string           `json:"task_type"`
	Scope            string           `json:"scope"`
	BranchPackageID  string           `json:"branch_package_id,omitempty"`
	BranchPackageKey string           `json:"branch_package_key,omitempty"`
	Instruction      string           `json:"instruction"`
	Events           []TaskEvent      `json:"events"`
	OutputSchemaPath string           `json:"output_schema_path"`
	SubmitCommand    string           `json:"submit_command"`
	Status           string           `json:"status"`
	CreatedAt        string           `json:"created_at"`
}

// TaskEvent is a summary of an intake event embedded in a task.
type TaskEvent struct {
	EventID     string   `json:"event_id"`
	TaskSummary string   `json:"task_summary"`
	RawNotes    []string `json:"raw_notes"`
	ChangedPaths []string `json:"changed_paths,omitempty"`
	Flags       *Flags   `json:"flags,omitempty"`
}

// AssistResult is the output of tt agent assist.
type AssistResult struct {
	Status     string `json:"status"`
	TaskID     string `json:"task_id,omitempty"`
	TaskType   string `json:"task_type,omitempty"`
	EventCount int    `json:"event_count,omitempty"`
	TaskFile   string `json:"task_file,omitempty"`
	NextCommand string `json:"next_command,omitempty"`
	Message    string `json:"message,omitempty"`
}

// SubmitResult is the output of tt agent task submit.
type SubmitResult struct {
	Status          string   `json:"status"`
	TaskID          string   `json:"task_id"`
	KnowledgeCreated int    `json:"knowledge_created"`
	KnowledgeFiles  []string `json:"knowledge_files"`
	ProcessedEvents []string `json:"processed_events"`
	Message         string   `json:"message,omitempty"`
}

// BranchManifest describes a branch package.
type BranchManifest struct {
	ID            string `yaml:"id"`
	Key           string `yaml:"key"`
	Branch        string `yaml:"branch"`
	MergeBase     string `yaml:"merge_base"`
	DefaultBranch string `yaml:"default_branch"`
	CreatedAt     string `yaml:"created_at"`
	Status        string `yaml:"status"`
}

// Exit codes for agent task commands.
const (
	ExitTaskNotFound         = 1
	ExitTaskNotPending       = 1
	ExitInvalidEventRef      = 1
)

// Result codes for agent task commands.
const (
	CodeTaskNotFound         = "TASK_NOT_FOUND"
	CodeTaskNotPending       = "TASK_NOT_PENDING"
	CodeInvalidEventRef      = "INVALID_EVENT_REFERENCE"
)
```

---

### Storage Layer (`features/tt/internal/agent/storage/`)

---

#### [MODIFY] [index.go](file://features/tt/internal/agent/storage/index.go)

*   **Description**: `UpdateStatus` と `ListPendingByBranch` メソッドを追加
*   **Technical Design**:

```go
// UpdateStatus updates the status of an event in the index.
func (idx *Index) UpdateStatus(eventID, newStatus string) error {
	result, err := idx.db.Exec(
		"UPDATE intake_events SET status = ? WHERE event_id = ?",
		newStatus, eventID,
	)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("event %s not found", eventID)
	}
	return nil
}

// ListPendingByBranch returns all pending events for a given branch.
func (idx *Index) ListPendingByBranch(branch string) ([]EventRecord, error) {
	rows, err := idx.db.Query(`
		SELECT event_id, content_hash, content_id, agent, branch, scope,
		       branch_package, status, client_request_id, task_summary,
		       stored_at, created_at
		FROM intake_events
		WHERE status = 'pending' AND branch = ?
		ORDER BY created_at ASC
	`, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending events: %w", err)
	}
	defer rows.Close()

	var records []EventRecord
	for rows.Next() {
		var r EventRecord
		if err := rows.Scan(
			&r.EventID, &r.ContentHash, &r.ContentID, &r.Agent, &r.Branch,
			&r.Scope, &r.BranchPackage, &r.Status, &r.ClientRequestID,
			&r.TaskSummary, &r.StoredAt, &r.CreatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}
```

---

#### [MODIFY] [filestore.go](file://features/tt/internal/agent/storage/filestore.go)

*   **Description**: `MoveToProcessed` メソッドを追加。pending ファイルを processed に移動する。
*   **Technical Design**:

```go
// MoveToProcessed moves an event file from pending/ to processed/.
// The relative path structure (YYYY/MM/DD/event_id.json) is preserved.
func (fs *FileStore) MoveToProcessed(eventID string, createdAt time.Time) error {
	relPath := filepath.Join(
		createdAt.Format("2006"), createdAt.Format("01"), createdAt.Format("02"),
		eventID+".json")
	srcPath := filepath.Join(fs.baseDir, "pending", relPath)
	dstPath := filepath.Join(fs.baseDir, "processed", relPath)

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("failed to create processed directory: %w", err)
	}

	// Move file
	if err := os.Rename(srcPath, dstPath); err != nil {
		return fmt.Errorf("failed to move event to processed: %w", err)
	}
	return nil
}

// ReadEvent reads and unmarshals an IntakeEvent from the pending directory.
func (fs *FileStore) ReadEvent(eventID string, createdAt time.Time) (*agent.IntakeEvent, error) {
	relPath := filepath.Join(
		createdAt.Format("2006"), createdAt.Format("01"), createdAt.Format("02"),
		eventID+".json")
	filePath := filepath.Join(fs.baseDir, "pending", relPath)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read event file: %w", err)
	}
	var event agent.IntakeEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return &event, nil
}
```

*   **Logic**:
    *   `MoveToProcessed` は pending と同じ YYYY/MM/DD 階層を processed 配下に再現してファイルを rename する
    *   `ReadEvent` は eventID と createdAt からパスを計算して pending ファイルを読み込む

---

### Internal Logic: Assist (`features/tt/internal/agent/assist/`)

---

#### [NEW] [handler.go](file://features/tt/internal/agent/assist/handler.go)

*   **Description**: `tt agent assist --scope current-branch` のコアロジック
*   **Technical Design**:

```go
package assist

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/axsh/tokotachi/features/tt/internal/agent/notify"
	"github.com/axsh/tokotachi/features/tt/internal/agent/storage"
)

// Handler orchestrates the assist command.
type Handler struct {
	index     *storage.Index
	fileStore *storage.FileStore
	varDir    string
	executor  notify.GitExecutor
}

// NewHandler creates a new assist Handler.
func NewHandler(varDir string, executor notify.GitExecutor) (*Handler, error) {
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
		executor:  executor,
	}, nil
}

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

	// 3. Build task events from records
	events := h.buildTaskEvents(records)

	// 4. Generate task ID
	taskID, err := generateTaskID()
	if err != nil {
		return &agent.AssistResult{
			Status:  "error",
			Message: fmt.Sprintf("Failed to generate task ID: %v", err),
		}, 1
	}

	// 5. Derive branch package info from first record
	bpID := records[0].BranchPackage
	bpKey := ""
	// bpKey is derived from the record's branch_package field

	// 6. Build instruction
	instruction := buildInstruction()

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
		OutputSchemaPath: "prompts/memory/schemas/knowledge-atom-batch.schema.json",
		SubmitCommand:    fmt.Sprintf("tt agent task submit %s --file result.json", taskID),
		Status:           "pending",
		CreatedAt:        time.Now().UTC().Format(time.RFC3339),
	}

	// 8. Write task file
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
```

*   **Logic**:
    *   `findExistingTask`: `prompts/memory/var/tasks/pending/` 内のファイルを走査し、同一ブランチ向けの pending task を探す
    *   `buildTaskEvents`: `EventRecord` から `TaskEvent` へ変換。ファイルから raw event を読んで `raw_notes`, `changed_paths`, `flags` を取得
    *   `generateTaskID`: `ulid.New()` で ULID 生成、`T-` prefix
    *   `buildInstruction`: self-contained 原則を含むテンプレート文字列を返す
    *   `writeTask`: JSON を `var/tasks/pending/T-{ULID}.json` に書き出し
    *   `taskFilePath`: `varDir/tasks/pending/T-{ULID}.json` のパスを返す

---

#### [NEW] [instruction.go](file://features/tt/internal/agent/assist/instruction.go)

*   **Description**: Agent Task に埋め込む instruction テンプレート
*   **Technical Design**:

```go
package assist

// instructionTemplate is the fixed instruction text for distill_intake_to_knowledge tasks.
const instructionTemplate = `Convert the following intake events into self-contained Knowledge Atoms.

## Rules

1. Each Knowledge Atom must be self-contained.
   Resolve references such as "this", "that", "the above approach", or "the previous design".
   Do not depend on the original chat context.

2. For each atom, specify:
   - type: one of Fact, Decision, Constraint, Pattern, Warning, Skill
   - title: a concise, self-contained title (max 200 chars)
   - body: a self-contained description (max 2000 chars)
   - importance: one of low, medium, high, critical
   - confidence: a float between 0.0 and 1.0
   - activation_hints.positive: list of situations where this knowledge is relevant
   - activation_hints.negative: list of situations where this knowledge is NOT relevant (optional)
   - source.event_ids: list of intake event IDs this atom was derived from

3. One intake event may produce zero or more Knowledge Atoms.
   Skip events that contain no meaningful long-term knowledge.

4. Output format must conform to the schema at: %s

## Submit

When done, save the result as a JSON file and run:
%s`

// buildInstruction returns the instruction text with paths filled in.
func buildInstruction() string {
	// Paths are filled by the caller via fmt.Sprintf
	return instructionTemplate
}
```

*   **Logic**: `buildInstruction` は `instructionTemplate` に `output_schema_path` と `submit_command` を埋め込んで返す

---

### Internal Logic: Task (`features/tt/internal/agent/task/`)

---

#### [NEW] [show.go](file://features/tt/internal/agent/task/show.go)

*   **Description**: `tt agent task show <task-id>` のロジック
*   **Technical Design**:

```go
package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
)

// Show reads and returns the task with the given ID.
// Searches pending/, then completed/, then failed/.
func Show(varDir string, taskID string) (*agent.AgentTask, error) {
	for _, subdir := range []string{"pending", "completed", "failed"} {
		path := filepath.Join(varDir, "tasks", subdir, taskID+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var task agent.AgentTask
		if err := json.Unmarshal(data, &task); err != nil {
			return nil, fmt.Errorf("failed to parse task file %s: %w", path, err)
		}
		return &task, nil
	}
	return nil, fmt.Errorf("task %s not found in pending, completed, or failed", taskID)
}
```

---

#### [NEW] [submit.go](file://features/tt/internal/agent/task/submit.go)

*   **Description**: `tt agent task submit` のコアロジック
*   **Technical Design**:

```go
package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/axsh/tokotachi/features/tt/internal/agent/storage"
)

// SubmitHandler orchestrates the task submit workflow.
type SubmitHandler struct {
	varDir      string
	memoryRoot  string
	schemasDir  string
	index       *storage.Index
	fileStore   *storage.FileStore
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
		}, agent.ExitTaskNotFound
	}
	var task agent.AgentTask
	if err := json.Unmarshal(taskData, &task); err != nil {
		return h.errorResult(taskID, "Failed to parse task", 1)
	}
	if task.Status != "pending" {
		return &agent.SubmitResult{
			Status:  "error",
			TaskID:  taskID,
			Message: "Task is not in pending status",
		}, agent.ExitTaskNotPending
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
				}, agent.ExitInvalidEventRef
			}
		}
	}

	// 4. Generate Knowledge Atoms with IDs and supplemented fields
	now := time.Now().UTC()
	var atoms []agent.KnowledgeAtom
	var knowledgeFiles []string
	for _, candidate := range batch.Atoms {
		atomID, err := generateKnowledgeID()
		if err != nil {
			return h.errorResult(taskID, fmt.Sprintf("Failed to generate ID: %v", err), 1)
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
				GitBranch:       "", // filled from task context
			},
			Timestamps: agent.KnowledgeTS{CreatedAt: now},
		}
		// Derive git_branch from task's branch_package_id
		if task.BranchPackageID != "" {
			atom.Source.GitBranch = extractBranchFromBPID(task.BranchPackageID)
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
	// Update task status in file
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
			// Log warning but continue
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
```

*   **Logic**:
    *   `validateBatchSchema`: `knowledge-atom-batch.schema.json` を使って JSON Schema バリデーション
    *   `moveTask`: `os.Rename` で `pending/` -> `completed/` (または `failed/`)
    *   `extractBranchFromBPID`: `BR-{slug}-{hash}` から branch 名を推定 (slug 部分を抽出)
    *   `generateKnowledgeID`: ULID 生成、`K-` prefix

---

#### [NEW] [knowledge.go](file://features/tt/internal/agent/task/knowledge.go)

*   **Description**: Knowledge Atom を YAML ファイルとして保存する
*   **Technical Design**:

```go
package task

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"gopkg.in/yaml.v3"
)

// WriteKnowledgeAtom writes a Knowledge Atom as a YAML file.
// Returns the relative path from memoryRoot.
func WriteKnowledgeAtom(memoryRoot, branchPackageID string, atom *agent.KnowledgeAtom) (string, error) {
	dir := filepath.Join(memoryRoot, "branches", branchPackageID, "knowledge")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create knowledge directory: %w", err)
	}

	filename := atom.ID + ".yaml"
	filePath := filepath.Join(dir, filename)

	data, err := yaml.Marshal(atom)
	if err != nil {
		return "", fmt.Errorf("failed to marshal atom to YAML: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write atom file: %w", err)
	}

	// Return relative path from project root
	relPath := filepath.Join("prompts", "memory", "branches", branchPackageID, "knowledge", filename)
	return relPath, nil
}
```

---

#### [NEW] [manifest.go](file://features/tt/internal/agent/task/manifest.go)

*   **Description**: Branch manifest の作成
*   **Technical Design**:

```go
package task

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"gopkg.in/yaml.v3"
)

// EnsureBranchManifest creates a branch manifest if it doesn't exist.
func EnsureBranchManifest(memoryRoot string, task agent.AgentTask, now time.Time) error {
	dir := filepath.Join(memoryRoot, "branches", task.BranchPackageID)
	manifestPath := filepath.Join(dir, "manifest.yaml")

	// Check if manifest already exists
	if _, err := os.Stat(manifestPath); err == nil {
		return nil // already exists
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create branch directory: %w", err)
	}

	defaultBranch := "main"
	branchName := extractBranchFromBPID(task.BranchPackageID)
	mergeBase := extractMergeBaseFromBPID(task.BranchPackageID)

	manifest := agent.BranchManifest{
		ID:            task.BranchPackageID,
		Key:           task.BranchPackageKey,
		Branch:        branchName,
		MergeBase:     mergeBase,
		DefaultBranch: defaultBranch,
		CreatedAt:     now.Format(time.RFC3339),
		Status:        "active",
	}

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	return os.WriteFile(manifestPath, data, 0644)
}

// extractBranchFromBPID extracts the branch slug from a branch package ID.
// "BR-fix-memory-compiling-4a67ef5a" -> "fix-memory-compiling"
func extractBranchFromBPID(bpid string) string {
	// Remove "BR-" prefix
	s := bpid
	if len(s) > 3 && s[:3] == "BR-" {
		s = s[3:]
	}
	// Remove last segment (8-char merge_base hash)
	if idx := lastIndexByte(s, '-'); idx > 0 {
		return s[:idx]
	}
	return s
}

// extractMergeBaseFromBPID extracts the short merge base from a branch package ID.
// "BR-fix-memory-compiling-4a67ef5a" -> "4a67ef5a"
func extractMergeBaseFromBPID(bpid string) string {
	if idx := lastIndexByte(bpid, '-'); idx > 0 {
		return bpid[idx+1:]
	}
	return ""
}

func lastIndexByte(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}
```

---

#### [NEW] [idgen.go](file://features/tt/internal/agent/task/idgen.go)

*   **Description**: Task ID と Knowledge ID の ULID 生成
*   **Technical Design**:

```go
package task

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// generateTaskID generates a new ULID-based task ID with "T-" prefix.
func generateTaskID() (string, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now()), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate ULID: %w", err)
	}
	return "T-" + id.String(), nil
}

// generateKnowledgeID generates a new ULID-based knowledge atom ID with "K-" prefix.
func generateKnowledgeID() (string, error) {
	id, err := ulid.New(ulid.Timestamp(time.Now()), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to generate ULID: %w", err)
	}
	return "K-" + id.String(), nil
}
```

---

### Command Layer (`features/tt/cmd/`)

---

#### [NEW] [agent_assist.go](file://features/tt/cmd/agent_assist.go)

*   **Description**: `tt agent assist` コマンド定義
*   **Technical Design**:

```go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/agent/assist"
	"github.com/axsh/tokotachi/features/tt/internal/agent/notify"
)

var agentAssistCmd = &cobra.Command{
	Use:   "assist",
	Short: "Generate an agent task from pending intake events",
	Long: `Scans pending intake events for the current branch and generates
an Agent Task for a coding agent to process.

This command does NOT perform LLM processing. It only creates a task
description that a coding agent can read and act upon.`,
	RunE: runAgentAssist,
}

var (
	assistScope string
	assistForce bool
)

func init() {
	agentAssistCmd.Flags().StringVar(&assistScope, "scope", "", "Scope (required: current-branch)")
	agentAssistCmd.Flags().BoolVar(&assistForce, "force", false, "Force new task creation even if pending task exists")
	_ = agentAssistCmd.MarkFlagRequired("scope")
	agentCmd.AddCommand(agentAssistCmd)
}

func runAgentAssist(cmd *cobra.Command, args []string) error {
	if assistScope != "current-branch" {
		return fmt.Errorf("unsupported scope: %s (only 'current-branch' is supported)", assistScope)
	}

	branch := getCurrentBranch()
	if branch == "" {
		return fmt.Errorf("failed to detect current git branch")
	}

	varDir := filepath.Join("prompts", "memory", "var")
	h, err := assist.NewHandler(varDir, &notify.RealGitExecutor{})
	if err != nil {
		return fmt.Errorf("failed to initialize assist handler: %w", err)
	}
	defer h.Close()

	result, exitCode := h.HandleAssist(branch, assistForce)

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	fmt.Println(string(data))

	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}
```

---

#### [NEW] [agent_task.go](file://features/tt/cmd/agent_task.go)

*   **Description**: `tt agent task` サブコマンドグループ (`show`, `submit`)
*   **Technical Design**:

```go
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/agent/task"
)

var agentTaskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage agent tasks",
}

var agentTaskShowCmd = &cobra.Command{
	Use:   "show [task-id]",
	Short: "Show an agent task",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentTaskShow,
}

var agentTaskSubmitCmd = &cobra.Command{
	Use:   "submit [task-id]",
	Short: "Submit knowledge atom results for a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentTaskSubmit,
}

var submitFile string

func init() {
	agentTaskSubmitCmd.Flags().StringVar(&submitFile, "file", "", "Path to result JSON file")
	_ = agentTaskSubmitCmd.MarkFlagRequired("file")

	agentTaskCmd.AddCommand(agentTaskShowCmd)
	agentTaskCmd.AddCommand(agentTaskSubmitCmd)
	agentCmd.AddCommand(agentTaskCmd)
}

func runAgentTaskShow(cmd *cobra.Command, args []string) error {
	varDir := filepath.Join("prompts", "memory", "var")
	t, err := task.Show(varDir, args[0])
	if err != nil {
		return fmt.Errorf("failed to show task: %w", err)
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func runAgentTaskSubmit(cmd *cobra.Command, args []string) error {
	memoryRoot := filepath.Join("prompts", "memory")
	varDir := filepath.Join(memoryRoot, "var")
	schemasDir := filepath.Join(memoryRoot, "schemas")

	h, err := task.NewSubmitHandler(memoryRoot, varDir, schemasDir)
	if err != nil {
		return fmt.Errorf("failed to initialize submit handler: %w", err)
	}
	defer h.Close()

	result, exitCode := h.HandleSubmit(args[0], submitFile)

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	fmt.Println(string(data))

	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}
```

---

### Wrapper Scripts (`scripts/code/agent/`)

---

#### [NEW] [assist.sh](file://scripts/code/agent/assist.sh)

*   **Description**: `tt agent assist` の wrapper
*   **Technical Design**:

```bash
#!/usr/bin/env bash
# scripts/code/agent/assist.sh -- tt agent assist wrapper
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../_resolve_tool.sh"

TT_ARGS=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --scope) TT_ARGS+=(--scope "$2"); shift 2 ;;
    --force) TT_ARGS+=(--force);      shift ;;
    *)
      echo "[ERROR] Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

exec "$TOOL" agent assist "${TT_ARGS[@]}"
```

---

#### [NEW] [task.sh](file://scripts/code/agent/task.sh)

*   **Description**: `tt agent task show/submit` の wrapper
*   **Technical Design**:

```bash
#!/usr/bin/env bash
# scripts/code/agent/task.sh -- tt agent task wrapper
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../_resolve_tool.sh"

if [[ $# -lt 1 ]]; then
  echo "[ERROR] Usage: task.sh <show|submit> [args...]" >&2
  exit 1
fi

SUBCMD="$1"
shift

TT_ARGS=()
case "$SUBCMD" in
  show)
    if [[ $# -lt 1 ]]; then
      echo "[ERROR] Usage: task.sh show <task-id>" >&2
      exit 1
    fi
    TT_ARGS+=("$1")
    shift
    ;;
  submit)
    if [[ $# -lt 1 ]]; then
      echo "[ERROR] Usage: task.sh submit <task-id> --file <path>" >&2
      exit 1
    fi
    TT_ARGS+=("$1")
    shift
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --file) TT_ARGS+=(--file "$2"); shift 2 ;;
        *)
          echo "[ERROR] Unknown argument: $1" >&2
          exit 1
          ;;
      esac
    done
    ;;
  *)
    echo "[ERROR] Unknown subcommand: $SUBCMD" >&2
    exit 1
    ;;
esac

exec "$TOOL" agent task "$SUBCMD" "${TT_ARGS[@]}"
```

---

## Step-by-Step Implementation Guide

### Phase 1: スキーマとディレクトリ準備

- [ ] **Step 1: ディレクトリレイアウト準備**
  - `prompts/memory/var/intake/processed/` ディレクトリの `.gitkeep` を作成
  - `prompts/memory/var/tasks/pending/` ディレクトリの `.gitkeep` を作成
  - `prompts/memory/var/tasks/completed/` ディレクトリの `.gitkeep` を作成
  - `prompts/memory/var/tasks/failed/` ディレクトリの `.gitkeep` を作成
  - `prompts/memory/.gitignore` に `var/tasks/` を追加確認 (既に `var/` がある場合は不要)

- [ ] **Step 2: JSON Schema ファイルの作成**
  - `prompts/memory/schemas/knowledge-atom.schema.json` を上記 Proposed Changes の内容で作成
  - `prompts/memory/schemas/knowledge-atom-batch.schema.json` を作成
  - `prompts/memory/schemas/agent-task.schema.json` を作成

### Phase 2: Go 型定義

- [ ] **Step 3: 型定義のテスト作成**
  - `features/tt/internal/agent/types_test.go` に `KnowledgeAtom`, `AgentTask` 等のバリデーションテストを追加
  - テストケース: 型の enum 値の網羅、必須フィールドの有無

- [ ] **Step 4: 型定義の実装**
  - `features/tt/internal/agent/types.go` に上記の全型を追加
  - テストが通ることを確認

### Phase 3: Storage Layer 拡張

- [ ] **Step 5: Storage テストの作成**
  - `features/tt/internal/agent/storage/index_test.go` に `UpdateStatus` と `ListPendingByBranch` のテストを追加
  - `features/tt/internal/agent/storage/filestore_test.go` に `MoveToProcessed` と `ReadEvent` のテストを追加

- [ ] **Step 6: Storage メソッドの実装**
  - `index.go` に `UpdateStatus` と `ListPendingByBranch` を追加
  - `filestore.go` に `MoveToProcessed` と `ReadEvent` を追加
  - テストが通ることを確認

### Phase 4: Assist ロジック

- [ ] **Step 7: Assist テストの作成**
  - `features/tt/internal/agent/assist/handler_test.go` を作成
  - テストケース:
    - pending events 0 件 -> `no_pending_events`
    - pending events あり -> `requires_agent` + task file 作成
    - 既存 task あり -> `existing_task`
    - `--force` で既存 task 無視 -> 新規 task 作成

- [ ] **Step 8: Assist ロジックの実装**
  - `features/tt/internal/agent/assist/handler.go` を作成
  - `features/tt/internal/agent/assist/instruction.go` を作成
  - テストが通ることを確認

### Phase 5: Task ロジック

- [ ] **Step 9: Task テストの作成**
  - `features/tt/internal/agent/task/show_test.go` を作成: pending/completed/failed の検索
  - `features/tt/internal/agent/task/submit_test.go` を作成: batch バリデーション、event_id 検証、Knowledge Atom 採番、status 更新
  - `features/tt/internal/agent/task/knowledge_test.go` を作成: YAML シリアライズ、フィールド補完
  - `features/tt/internal/agent/task/manifest_test.go` を作成: branch manifest の作成、既存 manifest の読み込み

- [ ] **Step 10: Task ロジックの実装**
  - `features/tt/internal/agent/task/show.go` を作成
  - `features/tt/internal/agent/task/submit.go` を作成
  - `features/tt/internal/agent/task/knowledge.go` を作成
  - `features/tt/internal/agent/task/manifest.go` を作成
  - `features/tt/internal/agent/task/idgen.go` を作成
  - テストが通ることを確認

### Phase 6: Command Layer

- [ ] **Step 11: コマンドの実装**
  - `features/tt/cmd/agent_assist.go` を作成
  - `features/tt/cmd/agent_task.go` を作成

### Phase 7: Wrapper Scripts

- [ ] **Step 12: Wrapper の作成**
  - `scripts/code/agent/assist.sh` を作成
  - `scripts/code/agent/task.sh` を作成
  - `chmod +x` を確認

### Phase 8: 統合テスト

- [ ] **Step 13: 統合テストの作成**
  - `tests/tt/tt_agent_assist_test.go` を作成
    - シナリオ 1: notify -> assist -> task 生成
    - シナリオ 2: pending 0 件
    - シナリオ 3: 既存 task 再利用
  - `tests/tt/tt_agent_task_test.go` を作成
    - シナリオ 4: task show
    - シナリオ 5: task submit (正常系)
    - シナリオ 6: event_id validation エラー
    - シナリオ 7: スキーマ違反
    - シナリオ 8: 完了後の再 submit

### Phase 9: ビルド・検証

- [ ] **Step 14: ビルドと単体テスト**
  - `./scripts/process/build.sh --skip-frontend --skip-etc` を実行
  - 全テスト PASS を確認

- [ ] **Step 15: 統合テスト**
  - `./scripts/process/integration_test.sh --categories tt --specify "AgentAssist|AgentTask"` を実行
  - 全テスト PASS を確認

- [ ] **Step 16: 最終検証**
  - `./scripts/process/build.sh` を実行 (フラグなし、全体ビルド)
  - 全テスト PASS を確認

---

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
   ```bash
   ./scripts/process/build.sh
   ```

2. **Integration Tests** (ファイル I/O, DB 操作, CLI サブコマンドを含むため統合テスト必須):
   ```bash
   ./scripts/process/build.sh && ./scripts/process/integration_test.sh --categories tt --specify "AgentAssist|AgentTask"
   ```
   *   **Log Verification**:
       - assist: `requires_agent` status が返ること、task file が存在すること
       - submit: `completed` status が返ること、YAML ファイルが正しく生成されること
       - intake event が processed に移動していること

3. **GUI E2E Tests**: 本実装は GUI 変更を含まないため不要。

### Self-Review of Test Items

#### 観点チェックリスト (testing-rules.md Section 11.3)

| # | 観点 | 対応テスト |
|---|------|------------|
| 1 | 正常系の動作確認 | assist: pending events ありで task 生成。submit: valid batch で knowledge 保存 |
| 2 | 異常系・境界値 | assist: 0 件。submit: invalid event_id, スキーマ違反, 完了済み task への再 submit |
| 3 | 外部連携の実動作 | 統合テスト: 実際の tt binary 経由で CLI -> SQLite -> ファイルシステムの一気通貫 |
| 4 | データの一貫性 | submit 後に YAML を読み戻して内容を検証。index status が processed に更新されていることを確認 |
| 5 | 状態遷移の検証 | task: pending -> completed。intake: pending -> processed |
| 6 | 設定・構成の反映 | branch_package_id が task から knowledge に正しく伝播されることを検証 |
| 7 | 副作用の確認 | submit 後に pending ディレクトリからファイルが消え、processed に移動していることを確認 |

#### セルフレビュー結果 (testing-rules.md Section 11.4)

1. **網羅性の検証**: 全 8 シナリオ (仕様書の検証シナリオ 1-8) をテストとしてカバー。テスト全通過で「assist -> task 生成 -> submit -> knowledge 保存 -> intake 移行」の全ワークフローが動作すると言い切れる。
2. **証拠の十分性**: 各テストで JSON 出力の status, task_id, knowledge_files, processed_events の値を検証。ファイルの存在確認も実施。
3. **迂回・抜け道の排除**: submit の event_id validation テストにより、task に含まれない event を参照できないことを確認。
4. **依存関係の整合性**: ボトムアップ順序 (Storage -> Assist/Task logic -> Command -> Integration) でテストを実行。

### 総合判定プロセス

全テスト完了後、以下の判定基準で合否を確認する:

1. `./scripts/process/build.sh` がエラーなく完了すること
2. `./scripts/process/integration_test.sh --categories tt --specify "AgentAssist|AgentTask"` が全 PASS であること
3. 既存テスト (`./scripts/process/integration_test.sh --categories tt`) にリグレッションがないこと

---

## Documentation

本実装で新規にドキュメントを作成する必要はない。仕様書 [003-Agentic-Memory-Assist-MVP.md](../ideas/003-Agentic-Memory-Assist-MVP.md) が正とする。
