package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// FileAction describes a planned or executed file operation.
type FileAction struct {
	Path           string
	Action         string // "create", "skip", "overwrite", "append", "error"
	ConflictPolicy string
	Exists         bool
}

// PermissionAction describes a planned permission change.
type PermissionAction struct {
	Path string
	Mode string // Display format: "0755", "0600", etc.
}

// Plan represents an execution plan for scaffold operations.
type Plan struct {
	ScaffoldName      string
	FilesToCreate     []FileAction
	FilesToSkip       []FileAction
	FilesToModify     []FileAction
	PostActions       PostActions
	PermissionActions []PermissionAction
	Warnings          []string
}

// ApplyFiles places downloaded template files into the target directory.
// It applies the conflict policy and processes template variables.
func ApplyFiles(files []DownloadedFile, placement *Placement,
	repoRoot string, optionValues map[string]string) error {

	baseDir := placement.BaseDir
	if optionValues != nil && strings.Contains(baseDir, "{{") {
		var err error
		baseDir, err = ProcessTemplatePath(baseDir, optionValues)
		if err != nil {
			return fmt.Errorf("failed to process base_dir template: %w", err)
		}
	}

	for _, file := range files {
		relPath := file.RelativePath
		content := file.Content

		// Process template files
		if placement.TemplateConfig.TemplateExtension != "" &&
			strings.HasSuffix(relPath, placement.TemplateConfig.TemplateExtension) {
			if optionValues != nil {
				rendered, err := ProcessTemplate(string(content), optionValues)
				if err != nil {
					return fmt.Errorf("failed to process template %s: %w", relPath, err)
				}
				content = []byte(rendered)
			}
			if placement.TemplateConfig.StripExtension {
				relPath = strings.TrimSuffix(relPath, placement.TemplateConfig.TemplateExtension)
			}
		}

		// Apply file mapping
		for _, fm := range placement.FileMappings {
			if fm.Source == file.RelativePath || fm.Source == relPath {
				relPath = fm.Target
				break
			}
		}

		targetPath := filepath.Join(repoRoot, baseDir, relPath)

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", targetPath, err)
		}

		// Check if file already exists
		_, statErr := os.Stat(targetPath)
		exists := statErr == nil

		if exists {
			switch placement.ConflictPolicy {
			case "skip":
				continue
			case "overwrite":
				// Fall through to write
			case "append":
				existing, err := os.ReadFile(targetPath)
				if err != nil {
					return fmt.Errorf("failed to read existing file %s: %w", targetPath, err)
				}
				content = append(existing, content...)
			case "error":
				return fmt.Errorf("conflict: file already exists: %s", targetPath)
			}
		}

		if err := os.WriteFile(targetPath, content, 0o644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}
	}

	return nil
}

// ApplyPostActions executes post-processing actions.
// baseDir is the placement base directory (relative to repoRoot).
func ApplyPostActions(actions PostActions, repoRoot string, baseDir string) error {
	// Apply gitignore entries
	if err := applyGitignoreEntries(actions.GitignoreEntries, repoRoot); err != nil {
		return err
	}

	// Apply file permissions
	if err := applyFilePermissions(actions.FilePermissions, repoRoot, baseDir); err != nil {
		return err
	}

	return nil
}

// applyGitignoreEntries adds entries to .gitignore, skipping duplicates.
func applyGitignoreEntries(entries []string, repoRoot string) error {
	if len(entries) == 0 {
		return nil
	}

	gitignorePath := filepath.Join(repoRoot, ".gitignore")

	// Read existing content (may not exist)
	existing, _ := os.ReadFile(gitignorePath)
	existingLines := splitLines(string(existing))

	// Build set of existing entries for dedup check
	entrySet := make(map[string]bool, len(existingLines))
	for _, line := range existingLines {
		entrySet[line] = true
	}

	// Add new entries
	var newEntries []string
	for _, entry := range entries {
		if !entrySet[entry] {
			newEntries = append(newEntries, entry)
			entrySet[entry] = true
		}
	}

	if len(newEntries) == 0 {
		return nil
	}

	// Write back
	content := string(existing)
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += strings.Join(newEntries, "\n") + "\n"

	return os.WriteFile(gitignorePath, []byte(content), 0o644)
}

// applyFilePermissions applies file permission settings based on glob patterns.
// It walks the directory tree under repoRoot/baseDir and matches files against
// the configured patterns using doublestar glob matching.
func applyFilePermissions(perms []FilePermission, repoRoot string, baseDir string) error {
	if len(perms) == 0 {
		return nil
	}

	fullBaseDir := filepath.Join(repoRoot, baseDir)

	for _, fp := range perms {
		mode, err := fp.ResolvedMode()
		if err != nil {
			return fmt.Errorf("file_permissions: %w", err)
		}

		// Walk the directory and match files
		err = filepath.WalkDir(fullBaseDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}

			// Get relative path from base dir
			relPath, err := filepath.Rel(fullBaseDir, path)
			if err != nil {
				return err
			}
			// Normalize to forward slashes for glob matching
			relPath = filepath.ToSlash(relPath)

			matched, err := doublestar.Match(fp.Pattern, relPath)
			if err != nil {
				return fmt.Errorf("invalid glob pattern %q: %w", fp.Pattern, err)
			}

			if matched {
				if chmodErr := os.Chmod(path, mode); chmodErr != nil {
					return fmt.Errorf("failed to chmod %s: %w", relPath, chmodErr)
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// buildPermissionActions generates PermissionAction entries for the execution plan.
func buildPermissionActions(perms []FilePermission, repoRoot string, baseDir string) []PermissionAction {
	if len(perms) == 0 {
		return nil
	}

	var actions []PermissionAction
	fullBaseDir := filepath.Join(repoRoot, baseDir)

	for _, fp := range perms {
		mode, err := fp.ResolvedMode()
		if err != nil {
			continue
		}
		modeStr := fmt.Sprintf("%04o", mode)

		_ = filepath.WalkDir(fullBaseDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return walkErr
			}

			relPath, err := filepath.Rel(fullBaseDir, path)
			if err != nil {
				return nil
			}
			relPath = filepath.ToSlash(relPath)

			matched, _ := doublestar.Match(fp.Pattern, relPath)
			if matched {
				displayPath := filepath.ToSlash(filepath.Join(baseDir, relPath))
				actions = append(actions, PermissionAction{
					Path: displayPath,
					Mode: modeStr,
				})
			}
			return nil
		})
	}

	return actions
}

// splitLines splits a string into lines, filtering out empty trailing lines.
func splitLines(s string) []string {
	lines := strings.Split(s, "\n")
	// Remove trailing empty line
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// BuildPlan generates an execution plan without performing any file operations.
func BuildPlan(files []DownloadedFile, placement *Placement,
	repoRoot string, scaffoldName string,
	optionValues map[string]string) (*Plan, error) {

	plan := &Plan{
		ScaffoldName: scaffoldName,
		PostActions:  placement.PostActions,
	}

	baseDir := placement.BaseDir
	if optionValues != nil && strings.Contains(baseDir, "{{") {
		var err error
		baseDir, err = ProcessTemplatePath(baseDir, optionValues)
		if err != nil {
			return nil, fmt.Errorf("failed to process base_dir template: %w", err)
		}
	}

	for _, file := range files {
		relPath := file.RelativePath

		// Strip template extension for display
		if placement.TemplateConfig.StripExtension &&
			strings.HasSuffix(relPath, placement.TemplateConfig.TemplateExtension) {
			relPath = strings.TrimSuffix(relPath, placement.TemplateConfig.TemplateExtension)
		}

		// Apply file mapping
		for _, fm := range placement.FileMappings {
			if fm.Source == file.RelativePath || fm.Source == relPath {
				relPath = fm.Target
				break
			}
		}

		targetPath := filepath.Join(baseDir, relPath)
		fullPath := filepath.Join(repoRoot, targetPath)

		_, statErr := os.Stat(fullPath)
		exists := statErr == nil

		action := FileAction{
			Path:           targetPath,
			Exists:         exists,
			ConflictPolicy: placement.ConflictPolicy,
		}

		if exists {
			switch placement.ConflictPolicy {
			case "skip":
				action.Action = "skip"
				plan.FilesToSkip = append(plan.FilesToSkip, action)
			case "overwrite":
				action.Action = "overwrite"
				plan.FilesToModify = append(plan.FilesToModify, action)
			case "append":
				action.Action = "append"
				plan.FilesToModify = append(plan.FilesToModify, action)
			case "error":
				action.Action = "error"
				plan.Warnings = append(plan.Warnings,
					fmt.Sprintf("conflict: %s already exists (policy: error)", targetPath))
			}
		} else {
			action.Action = "create"
			plan.FilesToCreate = append(plan.FilesToCreate, action)
		}
	}

	// Build permission actions preview
	plan.PermissionActions = buildPermissionActions(
		placement.PostActions.FilePermissions, repoRoot, baseDir)

	return plan, nil
}
