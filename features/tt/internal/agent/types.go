package agent

import "time"

// NotifyPayload represents the input from a coding agent.
type NotifyPayload struct {
	Version         int            `json:"version"`
	SourceType      string         `json:"source_type"`
	Agent           string         `json:"agent"`
	TaskSummary     string         `json:"task_summary"`
	RawNotes        []string       `json:"raw_notes"`
	ChangedPaths    []string       `json:"changed_paths,omitempty"`
	Flags           *Flags         `json:"flags,omitempty"`
	ClientRequestID string         `json:"client_request_id,omitempty"`
	Context         *PayloadContext `json:"context,omitempty"`
	Extra           map[string]any `json:"extra,omitempty"`
}

// Flags represents boolean flags for categorization.
type Flags struct {
	ArchitectureImpact      bool `json:"architecture_impact,omitempty"`
	MemoryRelated           bool `json:"memory_related,omitempty"`
	PromptRelated           bool `json:"prompt_related,omitempty"`
	AgentBehaviorRelated    bool `json:"agent_behavior_related,omitempty"`
	RequiresImmediateAction bool `json:"requires_immediate_action,omitempty"`
}

// PayloadContext holds session/task metadata.
type PayloadContext struct {
	SessionID      string `json:"session_id,omitempty"`
	TaskID         string `json:"task_id,omitempty"`
	WrapperVersion string `json:"wrapper_version,omitempty"`
}

// BranchPackageInfo holds the structured branch package identifier.
type BranchPackageInfo struct {
	Key       string `json:"key"`        // "owner/repo:branch:merge_base"
	ID        string `json:"id"`         // "BR-{branch_slug}-{merge_base_short8}"
	Branch    string `json:"branch"`     // raw branch name
	MergeBase string `json:"merge_base"` // full merge_base hash
}

// IntakeEvent is the full stored event (payload + computed fields).
type IntakeEvent struct {
	NotifyPayload
	EventID               string             `json:"event_id"`
	InstanceID            string             `json:"instance_id"`
	ContentHash           string             `json:"content_hash"`
	ContentID             string             `json:"content_id"`
	Git                   *GitInfo           `json:"git,omitempty"`
	Scope                 string             `json:"scope"`
	BranchPackage         *BranchPackageInfo `json:"branch_package,omitempty"`
	EffectiveChangedPaths []string           `json:"effective_changed_paths,omitempty"`
	Timestamps            Timestamps         `json:"timestamps"`
	Provenance            Provenance         `json:"provenance"`
}

// GitInfo holds git repository state.
type GitInfo struct {
	Branch        string `json:"branch"`
	HeadCommit    string `json:"head_commit"`
	IsDirty       bool   `json:"is_dirty"`
	MergeBase     string `json:"merge_base,omitempty"`
	DefaultBranch string `json:"default_branch,omitempty"`
}

// Timestamps holds creation and storage times.
type Timestamps struct {
	CreatedAt time.Time `json:"created_at"`
	StoredAt  time.Time `json:"stored_at"`
}

// Provenance holds environment info.
type Provenance struct {
	Hostname       string `json:"hostname"`
	User           string `json:"user"`
	Cwd            string `json:"cwd"`
	WrapperVersion string `json:"wrapper_version,omitempty"`
}

// NotifyResult is the command output.
type NotifyResult struct {
	Status      string   `json:"status"`
	Code        string   `json:"code"`
	Mode        string   `json:"mode,omitempty"`
	EventID     string   `json:"event_id,omitempty"`
	InstanceID  string   `json:"instance_id,omitempty"`
	ContentHash string   `json:"content_hash,omitempty"`
	ContentID   string   `json:"content_id,omitempty"`
	StoredAt    string   `json:"stored_at,omitempty"`
	IndexState  string   `json:"index_state,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
	NextAction  string   `json:"next_action,omitempty"`
	Message     string   `json:"message,omitempty"`
}

// Exit code constants for tt agent notify.
const (
	ExitOK                    = 0
	ExitJSONParseError        = 10
	ExitSchemaValidationError = 11
	ExitUnsupportedVersion    = 12
	ExitAgentIDInvalid        = 20
	ExitStorageLockTimeout    = 30
	ExitStorageWriteFailed    = 31
	ExitPermissionDenied      = 40
)

// Result code constants.
const (
	CodeOK                    = "OK"
	CodeIndexDegraded         = "INDEX_DEGRADED"
	CodeNoGitRepository       = "NO_GIT_REPOSITORY"
	CodeJSONParseError        = "JSON_PARSE_ERROR"
	CodeSchemaValidationError = "SCHEMA_VALIDATION_ERROR"
	CodeUnsupportedVersion    = "UNSUPPORTED_SCHEMA_VERSION"
	CodeAgentIDInvalid        = "AGENT_ID_INVALID"
	CodeStorageLockTimeout    = "STORAGE_LOCK_TIMEOUT"
	CodeStorageWriteFailed    = "STORAGE_WRITE_FAILED"
	CodePermissionDenied      = "PERMISSION_DENIED"
)

// ValidAgents is the set of canonical agent names.
var ValidAgents = map[string]bool{
	"codex":        true,
	"claude-code":  true,
	"antigravity":  true,
	"cursor":       true,
	"unknown":      true,
}

// --- Knowledge Atom types ---

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
	Version int                      `json:"version"`
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

// --- Agent Task types ---

// AgentTask represents a task for a coding agent.
type AgentTask struct {
	TaskID           string      `json:"task_id"`
	Version          int         `json:"version"`
	TaskType         string      `json:"task_type"`
	Scope            string      `json:"scope"`
	BranchPackageID  string      `json:"branch_package_id,omitempty"`
	BranchPackageKey string      `json:"branch_package_key,omitempty"`
	Instruction      string      `json:"instruction"`
	Events           []TaskEvent `json:"events"`
	OutputSchemaPath string      `json:"output_schema_path"`
	SubmitCommand    string      `json:"submit_command"`
	Status           string      `json:"status"`
	CreatedAt        string      `json:"created_at"`
}

// TaskEvent is a summary of an intake event embedded in a task.
type TaskEvent struct {
	EventID      string   `json:"event_id"`
	TaskSummary  string   `json:"task_summary"`
	RawNotes     []string `json:"raw_notes"`
	ChangedPaths []string `json:"changed_paths,omitempty"`
	Flags        *Flags   `json:"flags,omitempty"`
}

// --- Assist / Submit result types ---

// AssistResult is the output of tt agent assist.
type AssistResult struct {
	Status      string `json:"status"`
	TaskID      string `json:"task_id,omitempty"`
	TaskType    string `json:"task_type,omitempty"`
	EventCount  int    `json:"event_count,omitempty"`
	TaskFile    string `json:"task_file,omitempty"`
	NextCommand string `json:"next_command,omitempty"`
	Message     string `json:"message,omitempty"`
}

// SubmitResult is the output of tt agent task submit.
type SubmitResult struct {
	Status           string   `json:"status"`
	TaskID           string   `json:"task_id"`
	KnowledgeCreated int      `json:"knowledge_created"`
	KnowledgeFiles   []string `json:"knowledge_files"`
	ProcessedEvents  []string `json:"processed_events"`
	Message          string   `json:"message,omitempty"`
}

// --- Branch Manifest ---

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
