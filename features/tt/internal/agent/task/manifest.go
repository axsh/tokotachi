package task

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	branchName := ExtractBranchFromBPID(task.BranchPackageID)
	mergeBase := ExtractMergeBaseFromBPID(task.BranchPackageID)

	manifest := agent.BranchManifest{
		ID:            task.BranchPackageID,
		Key:           task.BranchPackageKey,
		Branch:        branchName,
		MergeBase:     mergeBase,
		DefaultBranch: "main",
		CreatedAt:     now.Format(time.RFC3339),
		Status:        "active",
	}

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	return os.WriteFile(manifestPath, data, 0644)
}

// ExtractBranchFromBPID extracts the branch slug from a branch package ID.
// "BR-fix-memory-compiling-4a67ef5a" -> "fix-memory-compiling"
func ExtractBranchFromBPID(bpid string) string {
	// Remove "BR-" prefix
	s := bpid
	if len(s) > 3 && s[:3] == "BR-" {
		s = s[3:]
	}
	// Remove last segment (8-char merge_base hash)
	if idx := strings.LastIndexByte(s, '-'); idx > 0 {
		return s[:idx]
	}
	return s
}

// ExtractMergeBaseFromBPID extracts the short merge base from a branch package ID.
// "BR-fix-memory-compiling-4a67ef5a" -> "4a67ef5a"
func ExtractMergeBaseFromBPID(bpid string) string {
	if idx := strings.LastIndexByte(bpid, '-'); idx > 0 {
		return bpid[idx+1:]
	}
	return ""
}
