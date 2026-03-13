package resolve

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GitWorktreeInfo holds resolved git worktree path information.
type GitWorktreeInfo struct {
	IsWorktree     bool   // true if .git is a file (worktree config)
	WorktreeGitDir string // absolute path to .git/worktrees/<name>/
	MainGitDir     string // absolute path to parent repo's .git/
}

// DetectGitWorktree inspects the given path's .git entry.
// If .git is a file containing "gitdir: ...", it resolves
// the worktree metadata dir and the main .git dir.
// Returns GitWorktreeInfo with IsWorktree=false if .git is a directory or absent.
func DetectGitWorktree(worktreePath string) (GitWorktreeInfo, error) {
	gitPath := filepath.Join(worktreePath, ".git")

	info, err := os.Stat(gitPath)
	if err != nil {
		// .git does not exist — not a git repo at all
		return GitWorktreeInfo{}, nil
	}

	if info.IsDir() {
		// .git is a directory — regular git repo, not a worktree
		return GitWorktreeInfo{}, nil
	}

	// .git is a file — this is a worktree
	data, err := os.ReadFile(gitPath)
	if err != nil {
		return GitWorktreeInfo{}, fmt.Errorf("failed to read .git file: %w", err)
	}

	content := strings.TrimSpace(string(data))
	if !strings.HasPrefix(content, "gitdir: ") {
		return GitWorktreeInfo{}, fmt.Errorf("unexpected .git file content: %s", content)
	}

	gitdirValue := strings.TrimPrefix(content, "gitdir: ")
	gitdirValue = strings.TrimSpace(gitdirValue)

	// Resolve to absolute path
	worktreeGitDir := gitdirValue
	if !filepath.IsAbs(worktreeGitDir) {
		worktreeGitDir = filepath.Join(worktreePath, worktreeGitDir)
	}
	worktreeGitDir, err = filepath.Abs(worktreeGitDir)
	if err != nil {
		return GitWorktreeInfo{}, fmt.Errorf("failed to resolve worktree git dir: %w", err)
	}
	worktreeGitDir = filepath.Clean(worktreeGitDir)

	// Check if the resolved path exists on the host.
	// If not, the .git file was previously rewritten for container use.
	// Try to restore from .git.tt-backup.
	if _, statErr := os.Stat(worktreeGitDir); statErr != nil {
		backupPath := filepath.Join(worktreePath, ".git.tt-backup")
		backupData, backupErr := os.ReadFile(backupPath)
		if backupErr != nil {
			return GitWorktreeInfo{}, fmt.Errorf("gitdir path %s not found and no backup available: %w", worktreeGitDir, statErr)
		}

		// Restore the .git file from backup
		if restoreErr := os.WriteFile(gitPath, backupData, 0644); restoreErr != nil {
			return GitWorktreeInfo{}, fmt.Errorf("failed to restore .git from backup: %w", restoreErr)
		}

		// Re-parse the restored content
		content = strings.TrimSpace(string(backupData))
		if !strings.HasPrefix(content, "gitdir: ") {
			return GitWorktreeInfo{}, fmt.Errorf("unexpected .git backup content: %s", content)
		}
		gitdirValue = strings.TrimPrefix(content, "gitdir: ")
		gitdirValue = strings.TrimSpace(gitdirValue)

		worktreeGitDir = gitdirValue
		if !filepath.IsAbs(worktreeGitDir) {
			worktreeGitDir = filepath.Join(worktreePath, worktreeGitDir)
		}
		worktreeGitDir, err = filepath.Abs(worktreeGitDir)
		if err != nil {
			return GitWorktreeInfo{}, fmt.Errorf("failed to resolve restored worktree git dir: %w", err)
		}
		worktreeGitDir = filepath.Clean(worktreeGitDir)
	}

	// Read commondir to find the main .git directory
	commondirPath := filepath.Join(worktreeGitDir, "commondir")
	commondirData, err := os.ReadFile(commondirPath)
	if err != nil {
		return GitWorktreeInfo{}, fmt.Errorf("failed to read commondir: %w", err)
	}

	commondirValue := strings.TrimSpace(string(commondirData))
	mainGitDir := filepath.Join(worktreeGitDir, commondirValue)
	mainGitDir, err = filepath.Abs(mainGitDir)
	if err != nil {
		return GitWorktreeInfo{}, fmt.Errorf("failed to resolve main git dir: %w", err)
	}
	mainGitDir = filepath.Clean(mainGitDir)

	return GitWorktreeInfo{
		IsWorktree:     true,
		WorktreeGitDir: worktreeGitDir,
		MainGitDir:     mainGitDir,
	}, nil
}

// CreateContainerGitFile creates a temporary file on the host containing
// the container-internal gitdir path, for use as an override mount.
// The file content is "gitdir: /worktree-git\n" which points to the
// writable copy of worktree metadata inside the container.
// Returns the path to the created file.
func CreateContainerGitFile(tempDir string) (string, error) {
	filePath := filepath.Join(tempDir, "dot-git-override")
	content := []byte("gitdir: /worktree-git\n")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to create container git override file: %w", err)
	}
	return filePath, nil
}
