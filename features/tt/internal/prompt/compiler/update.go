package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/axsh/tokotachi/pkg/resolve"
	"gopkg.in/yaml.v3"
)

// UpdateOptions holds options for the update pipeline.
type UpdateOptions struct {
	ProjectPath string
	Paths       *PathConfig
	Target      string // default: "all"
	Force       bool
	DryRun      bool
}

// UpdateResult holds the output of the update pipeline.
type UpdateResult struct {
	TargetResults map[string]*TargetUpdateResult
}

// TargetUpdateResult holds the result for a single target.
type TargetUpdateResult struct {
	Skipped      bool
	Reason       string // e.g., "no changes detected"
	DeployResult *DeployResult
}

// UpdateMetadata holds the last update information per target.
type UpdateMetadata struct {
	UpdatedAt    string `yaml:"updated_at"`
	SourceDigest string `yaml:"source_digest"`
	Target       string `yaml:"target"`
}

// Update executes the update pipeline for the given targets.
// It resolves the target, checks for changes, and runs deploy for each target.
func Update(opts UpdateOptions) (*UpdateResult, error) {
	target := opts.Target
	if target == "" {
		target = "all"
	}

	targets, err := resolve.ResolveTargets(target)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve targets: %w", err)
	}

	paths, err := resolvePathsFromUpdateOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve paths: %w", err)
	}

	rootDir := paths.Workspace

	result := &UpdateResult{
		TargetResults: make(map[string]*TargetUpdateResult),
	}

	for _, t := range targets {
		tr := &TargetUpdateResult{}
		metaDir := filepath.Join(rootDir, resolve.MetaDir(t))

		meta, err := ReadMetadata(metaDir)
		if err != nil {
			return nil, fmt.Errorf("failed to read metadata for target %s: %w", t, err)
		}

		needsUpdate := shouldUpdate(meta, opts.Force)
		if !needsUpdate {
			if meta != nil {
				updatedAt, err := time.Parse(time.RFC3339, meta.UpdatedAt)
				if err == nil {
					gitChanged, _ := CheckForChanges(rootDir, paths.PromptsDir, updatedAt)
					needsUpdate = gitChanged
				}
			}
		}

		if !needsUpdate {
			if CheckDrift(paths, t) {
				needsUpdate = true
			}
		}

		if !needsUpdate {
			tr.Skipped = true
			tr.Reason = "no changes detected"
			result.TargetResults[t] = tr
			continue
		}

		deployResult, err := Deploy(DeployOptions{
			Paths:  paths,
			Target: t,
			Force:  opts.Force,
			DryRun: opts.DryRun,
		})
		if err != nil {
			return nil, fmt.Errorf("deploy failed for target %s: %w", t, err)
		}
		tr.DeployResult = deployResult

		// Check for validation errors from Compile
		if deployResult.CompileResult != nil && len(deployResult.CompileResult.Errors) > 0 {
			for _, e := range deployResult.CompileResult.Errors {
				fmt.Fprintln(os.Stderr, e.Error())
			}
			return nil, fmt.Errorf("deploy failed for target %s: %d validation error(s)",
				t, len(deployResult.CompileResult.Errors))
		}

		// Write metadata (only when not dry-run)
		if !opts.DryRun {
			newMeta := &UpdateMetadata{
				UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
				SourceDigest: deployResult.DigestCurrent,
				Target:       t,
			}
			if err := WriteMetadata(metaDir, newMeta); err != nil {
				return nil, fmt.Errorf("failed to write metadata for target %s: %w", t, err)
			}
		}

		result.TargetResults[t] = tr
	}

	return result, nil
}

// shouldUpdate determines if an update is needed based on metadata and force flag.
func shouldUpdate(meta *UpdateMetadata, force bool) bool {
	if force {
		return true
	}
	if meta == nil {
		return true
	}
	return false
}

// CheckForChanges checks if source files have changed since the given time
// by examining git log for modifications under prompts-dir.
func CheckForChanges(rootDir, promptsDir string, since time.Time) (bool, error) {
	sinceStr := since.Format(time.RFC3339)
	watchPaths := []string{
		filepath.ToSlash(filepath.Join(promptsDir, "manifest")),
		filepath.ToSlash(filepath.Join(promptsDir, "memory")),
	}
	args := append([]string{
		"log", "--since=" + sinceStr,
		"--name-only", "--pretty=format:", "--",
	}, watchPaths...)
	cmd := exec.Command("git", args...)
	cmd.Dir = rootDir
	output, err := cmd.Output()
	if err != nil {
		return true, nil
	}
	trimmed := strings.TrimSpace(string(output))
	return trimmed != "", nil
}

// ReadMetadata reads the last update metadata from the given metaDir.
// Returns nil (not an error) if the metadata file does not exist.
func ReadMetadata(metaDir string) (*UpdateMetadata, error) {
	metaPath := filepath.Join(metaDir, "last_update.yaml")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read metadata file: %w", err)
	}

	var meta UpdateMetadata
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse metadata file: %w", err)
	}
	return &meta, nil
}

// WriteMetadata writes the update metadata to the given metaDir.
func WriteMetadata(metaDir string, meta *UpdateMetadata) error {
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return fmt.Errorf("failed to create metadata directory: %w", err)
	}

	data, err := yaml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	content := "# Auto-generated by tt prompt update. Do not edit.\n" + string(data)
	metaPath := filepath.Join(metaDir, "last_update.yaml")
	return os.WriteFile(metaPath, []byte(content), 0644)
}
