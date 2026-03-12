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
	"gopkg.in/yaml.v3"
)

const defaultRepoURL = "https://github.com/axsh/tokotachi-scaffolds"

// RunOptions holds parameters for a scaffold execution.
type RunOptions struct {
	Pattern         []string // Command arguments [category, name]
	RepoURL         string   // Template repository URL
	RepoRoot        string   // Target repository root path
	DryRun          bool
	Yes             bool
	Lang            string // Explicit locale (empty = auto-detect)
	Logger          *log.Logger
	Stdout          io.Writer         // Output writer for plan display
	Stdin           io.Reader         // Input reader for interactive prompts
	OptionOverrides map[string]string // Values from --v key=value flags
	UseDefaults     bool              // --default flag: auto-apply defaults for non-required options
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

	// 1. Download catalog and resolve entry
	downloader, entry, multipleEntries, err := fetchAndResolveEntry(opts, spinner)
	if err != nil {
		return nil, err
	}
	if multipleEntries != nil {
		// Multiple matches (category search) - show list and return
		fmt.Fprintln(opts.Stdout, "Multiple templates found. Please specify a name:")
		for _, e := range multipleEntries {
			fmt.Fprintf(opts.Stdout, "  - %s: %s\n", e.Name, e.Description)
		}
		return nil, nil
	}

	opts.Logger.Info("Selected template: %s (%s)", entry.Name, entry.Description)

	// 2. Check prerequisites
	if err := CheckRequirements(entry.Requirements, opts.RepoRoot); err != nil {
		return nil, err
	}

	// 3. Download and process template (ZIP or directory)
	templateFiles, placement, err := fetchTemplateAndPlacement(downloader, entry, opts.Lang, opts.Logger, spinner)
	if err != nil {
		return nil, err
	}

	// 4. Collect option values (interactive if needed)
	var optionValues map[string]string
	options := effectiveOptions(entry)
	if len(options) > 0 {
		optionValues, err = CollectOptionValues(options, opts.OptionOverrides, opts.Stdin, opts.Stdout, opts.UseDefaults)
		if err != nil {
			return nil, err
		}
	}

	// 5. Build execution plan
	plan, err := BuildPlan(templateFiles, placement, opts.RepoRoot, entry.Name, optionValues)
	if err != nil {
		return nil, err
	}
	plan.PostActions = placement.PostActions
	plan.OptionValues = optionValues

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
	spinner := NewSpinner(os.Stderr)
	downloader, entry, _, err := fetchAndResolveEntry(opts, spinner)
	if err != nil {
		return err
	}

	// 3. Download and process template
	templateFiles, placement, err := fetchTemplateAndPlacement(downloader, entry, opts.Lang, opts.Logger, spinner)
	if err != nil {
		return err
	}

	// Use option values from the plan (collected during Run, not re-prompted)
	optionValues := plan.OptionValues

	// 4. Apply files
	spinner.Start("Applying template files...")
	if err := ApplyFiles(templateFiles, placement, opts.RepoRoot, optionValues); err != nil {
		spinner.Stop()
		return fmt.Errorf("failed to apply files: %w", err)
	}
	spinner.Stop()

	// 5. Apply post-actions
	if err := ApplyPostActions(placement.PostActions, opts.RepoRoot, placement.BaseDir); err != nil {
		return fmt.Errorf("failed to apply post-actions: %w", err)
	}

	// 6. Remove checkpoint on success
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

// MetaYAML represents the meta.yaml file at the repository root.
type MetaYAML struct {
	Version         string `yaml:"version"`
	DefaultScaffold string `yaml:"default_scaffold"`
	UpdatedAt       string `yaml:"updated_at"`
}

// List fetches the catalog and returns all available scaffolds.
// Uses cache (via meta.yaml updated_at) when available.
func List(repoURL string, repoRoot string, filterCategory string) ([]ScaffoldEntry, error) {
	if repoURL == "" {
		repoURL = defaultRepoURL
	}

	downloader, err := github.NewClient(repoURL)
	if err != nil {
		return nil, err
	}

	// Fetch meta.yaml for cache validation
	metaData, err := downloader.FetchFile("meta.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch meta.yaml: %w", err)
	}

	var meta MetaYAML
	if err := yaml.Unmarshal(metaData, &meta); err != nil {
		return nil, fmt.Errorf("failed to parse meta.yaml: %w", err)
	}

	// Try cache
	var catalogData []byte
	if repoRoot != "" {
		cache := NewCacheStore(repoRoot)
		if cache.IsValid(meta.UpdatedAt) {
			cached, err := cache.Load()
			if err == nil && cached != nil {
				catalogData = cached.CatalogData
			}
		}
	}

	// Fetch if not cached
	if catalogData == nil {
		catalogData, err = downloader.FetchFile("catalog.yaml")
		if err != nil {
			return nil, fmt.Errorf("failed to fetch catalog.yaml: %w", err)
		}
		// Save to cache
		if repoRoot != "" {
			cache := NewCacheStore(repoRoot)
			_ = cache.Save(&CachedCatalog{
				UpdatedAt:   meta.UpdatedAt,
				CatalogData: catalogData,
			})
		}
	}

	catalogIdx, err := ParseCatalogIndex(catalogData)
	if err != nil {
		return nil, err
	}

	// Collect all entries
	var allEntries []ScaffoldEntry
	for category, entries := range catalogIdx.Scaffolds {
		if filterCategory != "" && category != filterCategory {
			continue
		}
		for _, ref := range entries {
			detailData, fetchErr := downloader.FetchFile(ref)
			if fetchErr != nil {
				continue
			}
			detailEntries, parseErr := ParseScaffoldDetail(detailData)
			if parseErr != nil {
				continue
			}
			allEntries = append(allEntries, detailEntries...)
		}
	}

	return allEntries, nil
}

// --- Helper functions ---

// fetchAndResolveEntry resolves the pattern to a single scaffold entry
// using Method A (shard path computation) for direct access.
func fetchAndResolveEntry(opts RunOptions, spinner *Spinner) (
	*github.Client, *ScaffoldEntry, []ScaffoldEntry, error) {

	downloader, err := github.NewClient(opts.RepoURL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create downloader: %w", err)
	}

	category, name := resolveArgs(opts.Pattern)

	// Category only without --list → error
	if name == "" {
		return nil, nil, nil, fmt.Errorf(
			"scaffold name is required: tt scaffold %s <name>", category)
	}

	// Method A: compute shard path directly
	shardPath := ShardPath(category, name)

	spinner.Start("Fetching template details...")
	shardData, err := downloader.FetchFile(shardPath)
	if err != nil {
		spinner.Stop()
		return nil, nil, nil, fmt.Errorf("scaffold not found: %s/%s (shard: %s): %w",
			category, name, shardPath, err)
	}
	spinner.Stop()

	detailEntries, err := ParseScaffoldDetail(shardData)
	if err != nil {
		return nil, nil, nil, err
	}

	entry, err := findEntry(detailEntries, category, name)
	if err != nil {
		return nil, nil, nil, err
	}

	return downloader, entry, nil, nil
}

// resolveArgs resolves command arguments into category and name.
// Pattern is positional: [category] [name]
func resolveArgs(pattern []string) (category, name string) {
	switch len(pattern) {
	case 0:
		return DefaultCategory, DefaultName
	case 1:
		return pattern[0], "" // category only
	default:
		return pattern[0], pattern[1]
	}
}

// findEntry finds a scaffold entry matching both category and name
// from a list of entries (which may contain multiple due to hash collisions).
func findEntry(entries []ScaffoldEntry, category, name string) (*ScaffoldEntry, error) {
	for i, e := range entries {
		if e.Category == category && e.Name == name {
			return &entries[i], nil
		}
	}
	return nil, fmt.Errorf("scaffold entry not found: %s/%s", category, name)
}

// ZipScaffoldMeta represents the scaffold.yaml metadata file inside a ZIP template.
type ZipScaffoldMeta struct {
	Name        string    `yaml:"name"`
	Category    string    `yaml:"category"`
	Description string    `yaml:"description"`
	OriginalRef string    `yaml:"original_ref"`
	Placement   Placement `yaml:"placement"`
}

// fetchTemplateAndPlacement downloads template files and extracts placement.
// For ZIP templates, it parses scaffold.yaml from the archive, separates
// base files from locale files, and applies locale overlay.
// For legacy directory-based templates, it uses FetchDirectory + FetchFile.
func fetchTemplateAndPlacement(downloader *github.Client, entry *ScaffoldEntry,
	lang string, logger *log.Logger, spinner *Spinner) ([]DownloadedFile, *Placement, error) {

	if strings.HasSuffix(entry.TemplateRef, ".zip") {
		return fetchZipTemplateAndPlacement(downloader, entry, lang, logger, spinner)
	}
	return fetchLegacyTemplateAndPlacement(downloader, entry, lang, logger, spinner)
}

// fetchZipTemplateAndPlacement handles ZIP-based template archives.
func fetchZipTemplateAndPlacement(downloader *github.Client, entry *ScaffoldEntry,
	lang string, logger *log.Logger, spinner *Spinner) ([]DownloadedFile, *Placement, error) {

	spinner.Start("Downloading template...")
	zipData, err := downloader.FetchFile(entry.TemplateRef)
	if err != nil {
		spinner.Stop()
		return nil, nil, fmt.Errorf("failed to fetch template zip: %w", err)
	}

	allFiles, err := ExtractZip(zipData)
	if err != nil {
		spinner.Stop()
		return nil, nil, fmt.Errorf("failed to extract template zip: %w", err)
	}
	spinner.Stop()

	// Parse scaffold.yaml from the archive for placement info
	placement := defaultPlacement()
	for _, f := range allFiles {
		if f.RelativePath == "scaffold.yaml" {
			var meta ZipScaffoldMeta
			if yamlErr := yaml.Unmarshal(f.Content, &meta); yamlErr == nil {
				placement = &meta.Placement
			}
			break
		}
	}

	// Separate files by prefix: base/, locale.XX/
	var baseFiles []DownloadedFile
	localePrefix := ""
	locale := DetectLocale(lang)
	if locale != "" {
		localePrefix = "locale." + locale + "/"
	}
	var localeFiles []DownloadedFile

	for _, f := range allFiles {
		switch {
		case f.RelativePath == "scaffold.yaml":
			// Skip metadata file
			continue
		case strings.HasPrefix(f.RelativePath, "base/"):
			// Strip "base/" prefix
			stripped := strings.TrimPrefix(f.RelativePath, "base/")
			if stripped != "" {
				baseFiles = append(baseFiles, DownloadedFile{
					RelativePath: stripped,
					Content:      f.Content,
				})
			}
		case localePrefix != "" && strings.HasPrefix(f.RelativePath, localePrefix):
			// Strip locale prefix (e.g. "locale.ja/")
			stripped := strings.TrimPrefix(f.RelativePath, localePrefix)
			if stripped != "" {
				localeFiles = append(localeFiles, DownloadedFile{
					RelativePath: stripped,
					Content:      f.Content,
				})
			}
			// Ignore other locale directories (e.g. locale.en/ when lang=ja)
		}
	}

	// Apply locale overlay if available
	if len(localeFiles) > 0 {
		baseFiles = MergeLocaleFiles(baseFiles, localeFiles)
		logger.Info("Applied locale overlay: %s (%d files)", locale, len(localeFiles))
	}

	return baseFiles, placement, nil
}

// fetchLegacyTemplateAndPlacement handles legacy directory-based templates.
func fetchLegacyTemplateAndPlacement(downloader *github.Client, entry *ScaffoldEntry,
	lang string, logger *log.Logger, spinner *Spinner) ([]DownloadedFile, *Placement, error) {

	// Fetch placement definition
	var placement *Placement
	if entry.PlacementRef != "" {
		spinner.Start("Downloading placement definition...")
		placementData, err := downloader.FetchFile(entry.PlacementRef)
		if err != nil {
			spinner.Stop()
			return nil, nil, fmt.Errorf("failed to fetch placement: %w", err)
		}
		spinner.Stop()

		p, err := ParsePlacement(placementData)
		if err != nil {
			return nil, nil, err
		}
		placement = p
	} else {
		placement = defaultPlacement()
	}

	// Fetch template files
	spinner.Start("Downloading template...")
	baseFiles, err := downloader.FetchDirectory(entry.TemplateRef + "/base")
	if err != nil {
		spinner.Stop()
		return nil, nil, fmt.Errorf("failed to fetch template: %w", err)
	}

	// Detect and apply locale overlay
	locale := DetectLocale(lang)
	var mergedFiles []DownloadedFile
	if locale != "" {
		localePath := fmt.Sprintf("%s/locale.%s", entry.TemplateRef, locale)
		spinner.UpdateMessage(fmt.Sprintf("Downloading locale overlay (%s)...", locale))
		localeFiles, localeErr := downloader.FetchDirectory(localePath)
		if localeErr == nil && len(localeFiles) > 0 {
			mergedFiles = MergeLocaleFiles(baseFiles, localeFiles)
			logger.Info("Applied locale overlay: %s (%d files)", locale, len(localeFiles))
		} else {
			mergedFiles = baseFiles
		}
	} else {
		mergedFiles = baseFiles
	}
	spinner.Stop()

	return mergedFiles, placement, nil
}

// defaultPlacement returns a default Placement for new-format scaffolds.
func defaultPlacement() *Placement {
	return &Placement{
		ConflictPolicy: "skip",
	}
}

// effectiveOptions returns the applicable options for a scaffold entry.
// TemplateParams (new format) take precedence over Options (legacy format).
func effectiveOptions(entry *ScaffoldEntry) []Option {
	if len(entry.TemplateParams) > 0 {
		return entry.TemplateParams
	}
	return entry.Options
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
