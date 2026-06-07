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
