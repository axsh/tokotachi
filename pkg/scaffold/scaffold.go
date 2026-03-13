package scaffold

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/axsh/tokotachi/internal/github"
	"github.com/axsh/tokotachi/internal/log"
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
	SkipDeps        bool              // --skip-deps flag: skip dependency resolution
	Force           bool              // --force flag: ignore download history and re-download all
}

// githubEntryFetcher implements EntryFetcher using the GitHub API.
type githubEntryFetcher struct {
	client *github.Client
}

// FetchEntry retrieves a ScaffoldEntry by computing its shard path.
func (f *githubEntryFetcher) FetchEntry(category, name string) (*ScaffoldEntry, error) {
	shardPath := ShardPath(category, name)
	shardData, err := f.client.FetchFile(shardPath)
	if err != nil {
		return nil, fmt.Errorf("scaffold not found: %s/%s (shard: %s): %w",
			category, name, shardPath, err)
	}

	detailEntries, err := ParseScaffoldDetail(shardData)
	if err != nil {
		return nil, err
	}

	return findEntry(detailEntries, category, name)
}

// Run executes the full scaffold workflow:
// resolve entry -> resolve dependencies -> download templates -> build plan.
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

	// 2. Resolve dependency chain
	entries := []ScaffoldEntry{*entry}
	if !opts.SkipDeps && len(entry.DependsOn) > 0 {
		fetcher := &githubEntryFetcher{client: downloader}
		entries, err = ResolveDependencies(fetcher, entry)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve dependencies: %w", err)
		}
		// Show dependency resolution to user
		fmt.Fprintf(os.Stderr, "Resolving dependencies...\n")
		for i, e := range entries {
			fmt.Fprintf(os.Stderr, "  [%d/%d] %s/%s\n", i+1, len(entries), e.Category, e.Name)
		}
	}

	// 3. Build plans for each scaffold in the dependency chain
	if len(entries) == 1 {
		// Single scaffold (no dependencies) - use original flow
		return buildSinglePlan(downloader, &entries[0], opts, spinner)
	}

	// Multiple scaffolds - build composite plan
	return buildCompositePlan(downloader, entries, opts, spinner)
}

// buildSinglePlan builds a plan for a single scaffold (no dependencies).
func buildSinglePlan(downloader *github.Client, entry *ScaffoldEntry,
	opts RunOptions, spinner *Spinner) (*Plan, error) {

	// Check prerequisites
	if err := CheckRequirements(entry.Requirements, opts.RepoRoot); err != nil {
		return nil, err
	}

	// Download and process template (ZIP or directory)
	templateFiles, placement, err := fetchTemplateAndPlacement(downloader, entry, opts.Lang, opts.Logger, spinner)
	if err != nil {
		return nil, err
	}

	// Collect option values (interactive if needed)
	var optionValues map[string]string
	options := effectiveOptions(entry)
	if len(options) > 0 {
		var err error
		optionValues, err = CollectOptionValues(options, opts.OptionOverrides, opts.Stdin, opts.Stdout, opts.UseDefaults)
		if err != nil {
			return nil, err
		}
	}

	// Merge overrides not defined in template_params (e.g. base_dir variables)
	optionValues = mergeOverrides(optionValues, opts.OptionOverrides)

	// Build execution plan
	plan, err := BuildPlan(templateFiles, placement, opts.RepoRoot, entry.Name, optionValues)
	if err != nil {
		return nil, err
	}
	plan.PostActions = placement.PostActions
	plan.OptionValues = optionValues

	return plan, nil
}

// buildCompositePlan builds a composite plan for a dependency chain.
func buildCompositePlan(downloader *github.Client, entries []ScaffoldEntry,
	opts RunOptions, spinner *Spinner) (*Plan, error) {

	plan := &Plan{
		ScaffoldName: entries[len(entries)-1].Name,
	}

	for i := range entries {
		e := &entries[i]

		// Download and process template
		templateFiles, placement, err := fetchTemplateAndPlacement(downloader, e, opts.Lang, opts.Logger, spinner)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch template for %s/%s: %w", e.Category, e.Name, err)
		}

		// Collect option values for this scaffold
		var optionValues map[string]string
		options := effectiveOptions(e)
		if len(options) > 0 {
			fmt.Fprintf(opts.Stdout, "\nOptions for %s/%s:\n", e.Category, e.Name)
			optionValues, err = CollectOptionValues(options, opts.OptionOverrides, opts.Stdin, opts.Stdout, opts.UseDefaults)
			if err != nil {
				return nil, err
			}
		}

		// Merge overrides not defined in template_params (e.g. base_dir variables)
		optionValues = mergeOverrides(optionValues, opts.OptionOverrides)

		// Build sub-plan
		subPlan, err := BuildPlan(templateFiles, placement, opts.RepoRoot, e.Name, optionValues)
		if err != nil {
			return nil, fmt.Errorf("failed to build plan for %s/%s: %w", e.Category, e.Name, err)
		}

		// Aggregate warnings
		plan.Warnings = append(plan.Warnings, subPlan.Warnings...)

		plan.DependencyPlans = append(plan.DependencyPlans, DependencyPlan{
			Entry:        *e,
			Files:        templateFiles,
			Placement:    placement,
			OptionValues: optionValues,
			SubPlan:      subPlan,
		})
	}

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

	// Handle dependency chain apply
	if len(plan.DependencyPlans) > 0 {
		return applyDependencyChain(plan, opts)
	}

	// Single scaffold apply (original flow)
	return applySingleScaffold(plan, opts)
}

// applySingleScaffold applies a single scaffold (no dependencies).
func applySingleScaffold(plan *Plan, opts RunOptions) error {
	spinner := NewSpinner(os.Stderr)

	// Re-download template files for application
	downloader, entry, _, err := fetchAndResolveEntry(opts, spinner)
	if err != nil {
		return err
	}

	templateFiles, placement, err := fetchTemplateAndPlacement(downloader, entry, opts.Lang, opts.Logger, spinner)
	if err != nil {
		return err
	}

	// Check download history for static scaffolds
	historyStore := NewDownloadHistoryStore(opts.RepoRoot)
	isDynamic := IsDynamic(placement)

	if !isDynamic && !opts.Force && historyStore.IsDownloaded(entry.Category, entry.Name) {
		opts.Logger.Info("Skipping %s/%s (already downloaded)", entry.Category, entry.Name)
		if err := RemoveCheckpoint(opts.RepoRoot); err != nil {
			opts.Logger.Warn("Failed to remove checkpoint: %v", err)
		}
		return nil
	}

	optionValues := plan.OptionValues

	spinner.Start("Applying template files...")
	if err := ApplyFiles(templateFiles, placement, opts.RepoRoot, optionValues); err != nil {
		spinner.Stop()
		return fmt.Errorf("failed to apply files: %w", err)
	}
	spinner.Stop()

	if err := ApplyPostActions(placement.PostActions, opts.RepoRoot, placement.BaseDir); err != nil {
		return fmt.Errorf("failed to apply post-actions: %w", err)
	}

	// Record download for static scaffolds
	if !isDynamic {
		if err := historyStore.RecordDownload(entry.Category, entry.Name); err != nil {
			opts.Logger.Warn("Failed to record download: %v", err)
		}
	}

	if err := RemoveCheckpoint(opts.RepoRoot); err != nil {
		opts.Logger.Warn("Failed to remove checkpoint: %v", err)
	}

	opts.Logger.Info("Scaffold applied successfully!")
	return nil
}

// applyDependencyChain applies all scaffolds in the dependency chain in order.
func applyDependencyChain(plan *Plan, opts RunOptions) error {
	spinner := NewSpinner(os.Stderr)
	total := len(plan.DependencyPlans)
	historyStore := NewDownloadHistoryStore(opts.RepoRoot)

	for i, dp := range plan.DependencyPlans {
		category := dp.Entry.Category
		name := dp.Entry.Name

		// Re-download template files to get placement for IsDynamic check
		downloader, err := github.NewClient(opts.RepoURL)
		if err != nil {
			return fmt.Errorf("failed to create downloader: %w", err)
		}

		templateFiles, placement, err := fetchTemplateAndPlacement(
			downloader, &dp.Entry, opts.Lang, opts.Logger, spinner)
		if err != nil {
			return fmt.Errorf("failed to fetch template for %s/%s: %w",
				category, name, err)
		}

		// Check download history for static scaffolds
		isDynamic := IsDynamic(placement)

		if !isDynamic && !opts.Force && historyStore.IsDownloaded(category, name) {
			opts.Logger.Info("Skipping %s/%s (already downloaded)", category, name)
			continue
		}

		fmt.Fprintf(os.Stderr, "[%d/%d] Applying %s/%s...\n",
			i+1, total, category, name)

		// Apply files
		spinner.Start(fmt.Sprintf("Applying %s/%s...", category, name))
		if err := ApplyFiles(templateFiles, placement, opts.RepoRoot, dp.OptionValues); err != nil {
			spinner.Stop()
			return fmt.Errorf("failed to apply files for %s/%s: %w",
				category, name, err)
		}
		spinner.Stop()

		// Apply post-actions
		if err := ApplyPostActions(placement.PostActions, opts.RepoRoot, placement.BaseDir); err != nil {
			return fmt.Errorf("failed to apply post-actions for %s/%s: %w",
				category, name, err)
		}

		// Record download for static scaffolds
		if !isDynamic {
			if err := historyStore.RecordDownload(category, name); err != nil {
				opts.Logger.Warn("Failed to record download for %s/%s: %v", category, name, err)
			}
		}
	}

	// Remove checkpoint on success
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

// mergeOverrides merges option overrides into optionValues.
// Any keys in overrides that are not already present in optionValues
// are added. This allows --v flags to provide extra variables
// not defined in template_params (e.g. base_dir template variables).
func mergeOverrides(optionValues, overrides map[string]string) map[string]string {
	if len(overrides) == 0 {
		return optionValues
	}
	if optionValues == nil {
		optionValues = make(map[string]string)
	}
	for k, v := range overrides {
		if _, exists := optionValues[k]; !exists {
			optionValues[k] = v
		}
	}
	return optionValues
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
