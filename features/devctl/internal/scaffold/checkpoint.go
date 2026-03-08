package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// CheckpointInfo stores checkpoint metadata for rollback support.
type CheckpointInfo struct {
	CreatedAt          string             `yaml:"created_at"`
	ScaffoldName       string             `yaml:"scaffold_name"`
	HeadCommit         string             `yaml:"head_commit"`
	StashRef           string             `yaml:"stash_ref,omitempty"`
	FilesCreated       []string           `yaml:"files_created"`
	FilesModified      []ModifiedFile     `yaml:"files_modified,omitempty"`
	PermissionsApplied []PermissionRecord `yaml:"permissions_applied,omitempty"`
}

// ModifiedFile records a modified file for rollback.
type ModifiedFile struct {
	Path                string `yaml:"path"`
	Action              string `yaml:"action"`
	OriginalContentHash string `yaml:"original_content_hash,omitempty"`
}

// PermissionRecord records a permission change for checkpoint tracking.
type PermissionRecord struct {
	Path string `yaml:"path"`
	Mode string `yaml:"mode"`
}

// CheckpointFileName is the name of the checkpoint file.
const CheckpointFileName = ".devctl-scaffold-checkpoint"

// SaveCheckpoint writes checkpoint info to disk.
func SaveCheckpoint(repoRoot string, info *CheckpointInfo) error {
	data, err := yaml.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal checkpoint: %w", err)
	}

	path := filepath.Join(repoRoot, CheckpointFileName)
	return os.WriteFile(path, data, 0o644)
}

// LoadCheckpoint reads checkpoint info from disk.
func LoadCheckpoint(repoRoot string) (*CheckpointInfo, error) {
	path := filepath.Join(repoRoot, CheckpointFileName)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("no checkpoint found: %w", err)
	}

	var info CheckpointInfo
	if err := yaml.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse checkpoint: %w", err)
	}

	return &info, nil
}

// RemoveCheckpoint deletes the checkpoint file.
func RemoveCheckpoint(repoRoot string) error {
	path := filepath.Join(repoRoot, CheckpointFileName)
	return os.Remove(path)
}

// BuildCheckpointFromPlan creates a CheckpointInfo from an execution plan.
func BuildCheckpointFromPlan(plan *Plan, headCommit, stashRef string) *CheckpointInfo {
	info := &CheckpointInfo{
		CreatedAt:    time.Now().Format(time.RFC3339),
		ScaffoldName: plan.ScaffoldName,
		HeadCommit:   headCommit,
		StashRef:     stashRef,
	}

	for _, f := range plan.FilesToCreate {
		info.FilesCreated = append(info.FilesCreated, f.Path)
	}

	for _, f := range plan.FilesToModify {
		info.FilesModified = append(info.FilesModified, ModifiedFile{
			Path:   f.Path,
			Action: f.Action,
		})
	}

	for _, pa := range plan.PermissionActions {
		info.PermissionsApplied = append(info.PermissionsApplied, PermissionRecord{
			Path: pa.Path,
			Mode: pa.Mode,
		})
	}

	return info
}
