package scaffold

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/axsh/tokotachi/features/tt/internal/github"
	"github.com/axsh/tokotachi/features/tt/internal/log"
)

const defaultRepoURL = "https://github.com/axsh/tokotachi-scaffolds"

// RunOptions holds parameters for a scaffold execution.
type RunOptions struct {
	Pattern  []string // Command arguments [category, name]
	RepoURL  string   // Template repository URL
	RepoRoot string   // Target repository root path
	DryRun   bool
	Yes      bool
	Lang     string // Explicit locale (empty = auto-detect)
	Logger   *log.Logger
	Stdout   io.Writer // Output writer for plan display
	Stdin    io.Reader // Input reader for interactive prompts
}

// Run executes the full scaffold workflow:
// catalog fetch -> pattern resolve -> prerequisite check -> download -> plan.
func Run(opts RunOptions) (*Plan, error) {
	if opts.RepoURL == "" {
		opts.RepoURL = defaultRepoURL
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}

	spinner := NewSpinner(os.Stderr)

	// 1. Download catalog
	spinner.Start("Fetching catalog...")
	downloader, err := github.NewClient(opts.RepoURL)
	if err != nil {
		spinner.Stop()
		return nil, fmt.Errorf("failed to create downloader: %w", err)
	}

	catalogData, err := downloader.FetchFile("catalog.yaml")
	if err != nil {
		spinner.Stop()
		return nil, fmt.Errorf("failed to fetch catalog: %w", err)
	}
	spinner.Stop()

	catalog, err := ParseCatalog(catalogData)
	if err != nil {
		return nil, err
	}

	// 2. Resolve pattern
	entries, err := catalog.ResolvePattern(opts.Pattern)
	if err != nil {
		return nil, err
	}

	if len(entries) > 1 {
		// Multiple matches (category search) - show list and return
		fmt.Fprintln(opts.Stdout, "Multiple templates found. Please specify a name:")
		for _, entry := range entries {
			fmt.Fprintf(opts.Stdout, "  - %s: %s\n", entry.Name, entry.Description)
		}
		return nil, nil
	}

	entry := entries[0]
	opts.Logger.Info("Selected template: %s (%s)", entry.Name, entry.Description)

	// 3. Check prerequisites
	if err := CheckRequirements(entry.Requirements, opts.RepoRoot); err != nil {
		return nil, err
	}

	// 4. Download placement definition
	spinner.Start("Downloading placement definition...")
	placementData, err := downloader.FetchFile(entry.PlacementRef)
	if err != nil {
		spinner.Stop()
		return nil, fmt.Errorf("failed to fetch placement: %w", err)
	}
	spinner.Stop()

	placement, err := ParsePlacement(placementData)
	if err != nil {
		return nil, err
	}

	// 5. Collect option values (interactive if needed)
	var optionValues map[string]string
	if len(entry.Options) > 0 {
		optionValues, err = CollectOptionValues(entry.Options, nil, opts.Stdin, opts.Stdout)
		if err != nil {
			return nil, err
		}
	}

	// 6. Download template files (with locale overlay)
	spinner.Start("Downloading template...")
	baseFiles, err := downloader.FetchDirectory(entry.TemplateRef + "/base")
	if err != nil {
		spinner.Stop()
		return nil, fmt.Errorf("failed to fetch template: %w", err)
	}

	// Detect and apply locale overlay
	locale := DetectLocale(opts.Lang)
	var mergedFiles []DownloadedFile
	if locale != "" {
		localePath := fmt.Sprintf("%s/locale.%s", entry.TemplateRef, locale)
		spinner.UpdateMessage(fmt.Sprintf("Downloading locale overlay (%s)...", locale))
		localeFiles, localeErr := downloader.FetchDirectory(localePath)
		if localeErr == nil && len(localeFiles) > 0 {
			mergedFiles = MergeLocaleFiles(baseFiles, localeFiles)
			opts.Logger.Info("Applied locale overlay: %s (%d files)", locale, len(localeFiles))
		} else {
			mergedFiles = baseFiles
		}
	} else {
		mergedFiles = baseFiles
	}
	spinner.Stop()

	// 7. Build execution plan
	plan, err := BuildPlan(mergedFiles, placement, opts.RepoRoot, entry.Name, optionValues)
	if err != nil {
		return nil, err
	}
	plan.PostActions = placement.PostActions

	return plan, nil
}

// Apply executes the plan: checkpoint -> file placement -> post-actions.
func Apply(plan *Plan, opts RunOptions) error {
	if opts.RepoURL == "" {
		opts.RepoURL = defaultRepoURL
	}

	// 1. Create checkpoint
	headCommit := getHeadCommit(opts.RepoRoot)
	stashRef := ""

	// Check for uncommitted changes
	if hasDirtyWorktree(opts.RepoRoot) {
		opts.Logger.Info("Stashing uncommitted changes...")
		out, err := runGitCommand(opts.RepoRoot, "stash", "push", "-m", "tt-scaffold-checkpoint")
		if err != nil {
			opts.Logger.Warn("Failed to stash changes: %v (output: %s)", err, out)
		} else {
			stashRef = "stash@{0}"
		}
	}

	checkpointInfo := BuildCheckpointFromPlan(plan, headCommit, stashRef)
	if err := SaveCheckpoint(opts.RepoRoot, checkpointInfo); err != nil {
		opts.Logger.Warn("Failed to save checkpoint: %v", err)
	}

	// 2. Download template files again for application
	// (we don't store files in the plan, so re-download)
	downloader, err := github.NewClient(opts.RepoURL)
	if err != nil {
		return err
	}

	spinner := NewSpinner(os.Stderr)
	spinner.Start("Downloading template for application...")

	catalogData, err := downloader.FetchFile("catalog.yaml")
	if err != nil {
		spinner.Stop()
		return err
	}
	catalog, err := ParseCatalog(catalogData)
	if err != nil {
		spinner.Stop()
		return err
	}
	entries, _ := catalog.ResolvePattern(opts.Pattern)
	entry := entries[0]

	baseFiles, err := downloader.FetchDirectory(entry.TemplateRef + "/base")
	if err != nil {
		spinner.Stop()
		return err
	}

	locale := DetectLocale(opts.Lang)
	var mergedFiles []DownloadedFile
	if locale != "" {
		localePath := fmt.Sprintf("%s/locale.%s", entry.TemplateRef, locale)
		localeFiles, localeErr := downloader.FetchDirectory(localePath)
		if localeErr == nil && len(localeFiles) > 0 {
			mergedFiles = MergeLocaleFiles(baseFiles, localeFiles)
		} else {
			mergedFiles = baseFiles
		}
	} else {
		mergedFiles = baseFiles
	}
	spinner.Stop()

	placementData, err := downloader.FetchFile(entry.PlacementRef)
	if err != nil {
		return err
	}
	placement, err := ParsePlacement(placementData)
	if err != nil {
		return err
	}

	var optionValues map[string]string
	if len(entry.Options) > 0 {
		optionValues, _ = CollectOptionValues(entry.Options, nil, opts.Stdin, opts.Stdout)
	}

	// 3. Apply files
	spinner.Start("Applying template files...")
	if err := ApplyFiles(mergedFiles, placement, opts.RepoRoot, optionValues); err != nil {
		spinner.Stop()
		return fmt.Errorf("failed to apply files: %w", err)
	}
	spinner.Stop()

	// 4. Apply post-actions
	if err := ApplyPostActions(placement.PostActions, opts.RepoRoot, placement.BaseDir); err != nil {
		return fmt.Errorf("failed to apply post-actions: %w", err)
	}

	// 5. Remove checkpoint on success
	if err := RemoveCheckpoint(opts.RepoRoot); err != nil {
		opts.Logger.Warn("Failed to remove checkpoint: %v", err)
	}

	opts.Logger.Info("Scaffold applied successfully!")
	return nil
}

// Rollback undoes the last scaffold operation.
func Rollback(repoRoot string, logger *log.Logger) error {
	info, err := LoadCheckpoint(repoRoot)
	if err != nil {
		return fmt.Errorf("no scaffold checkpoint found to rollback: %w", err)
	}

	logger.Info("Rolling back scaffold %q...", info.ScaffoldName)

	// Remove created files
	for _, path := range info.FilesCreated {
		fullPath := filepath.Join(repoRoot, path)
		if er := os.Remove(fullPath); er != nil {
			logger.Warn("Failed to remove %s: %v", path, er)
		}
	}

	// Restore stashed changes
	if info.StashRef != "" {
		logger.Info("Restoring stashed changes...")
		if _, err := runGitCommand(repoRoot, "stash", "pop"); err != nil {
			logger.Warn("Failed to restore stash: %v", err)
		}
	}

	// Remove checkpoint file
	if err := RemoveCheckpoint(repoRoot); err != nil {
		logger.Warn("Failed to remove checkpoint file: %v", err)
	}

	logger.Info("Rollback completed!")
	return nil
}

// List fetches the catalog and returns all available scaffolds.
func List(repoURL string) ([]ScaffoldEntry, error) {
	if repoURL == "" {
		repoURL = defaultRepoURL
	}

	downloader, err := github.NewClient(repoURL)
	if err != nil {
		return nil, err
	}

	catalogData, err := downloader.FetchFile("catalog.yaml")
	if err != nil {
		return nil, err
	}

	catalog, err := ParseCatalog(catalogData)
	if err != nil {
		return nil, err
	}

	return catalog.ListScaffolds(), nil
}

// getHeadCommit returns the current HEAD commit hash.
func getHeadCommit(repoRoot string) string {
	out, err := runGitCommand(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// hasDirtyWorktree checks if there are uncommitted changes.
func hasDirtyWorktree(repoRoot string) bool {
	out, err := runGitCommand(repoRoot, "status", "--porcelain")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// runGitCommand executes a git command and returns its output.
func runGitCommand(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	return string(out), err
}
