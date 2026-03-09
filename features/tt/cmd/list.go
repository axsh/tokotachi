package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/cmdexec"
	"github.com/axsh/tokotachi/features/tt/internal/codestatus"
	"github.com/axsh/tokotachi/features/tt/internal/listing"
	"github.com/axsh/tokotachi/features/tt/internal/log"
	"github.com/axsh/tokotachi/features/tt/internal/report"
	"github.com/axsh/tokotachi/features/tt/internal/resolve"
	"github.com/axsh/tokotachi/features/tt/internal/state"
)

var (
	flagListJSON   bool
	flagListPath   bool
	flagListUpdate bool
	flagListFull   bool
)

var listCmd = &cobra.Command{
	Use:   "list [branch]",
	Short: "List branches or features",
	Long:  "Without arguments, list all worktree branches. With a branch, list features for that branch.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runList,
}

func init() {
	listCmd.Flags().BoolVar(&flagListJSON, "json", false, "Output in JSON format")
	listCmd.Flags().BoolVar(&flagListPath, "path", false, "Show worktree path column")
	listCmd.Flags().BoolVar(&flagListUpdate, "update", false, "Force update code status immediately")
	listCmd.Flags().BoolVar(&flagListFull, "full", false, "Disable column truncation")
}

func runList(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return runListBranches()
	}
	return runListFeatures(args)
}

// runListBranches shows all worktree branches with optional feature info.
func runListBranches() error {
	repoRoot, err := os.Getwd()
	if err != nil {
		repoRoot = "."
	}

	logger := log.New(os.Stderr, flagVerbose)
	rec := cmdexec.NewRecorder()
	runner := &cmdexec.Runner{Logger: logger, DryRun: flagDryRun, Recorder: rec}

	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
	output, err := runner.Run(gitCmd, "worktree", "list", "--porcelain")
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	entries := listing.ParseWorktreeOutput(output)
	states, err := state.ScanStateFiles(repoRoot)
	if err != nil {
		logger.Warn("Failed to scan state files: %v", err)
		states = make(map[string]state.StateFile)
	}

	// Handle --update: force foreground update
	if flagListUpdate {
		ghCmd := cmdexec.ResolveCommand("TT_CMD_GH", "gh")
		checker := &codestatus.Checker{
			GitCmd:   gitCmd,
			GhCmd:    ghCmd,
			RepoRoot: repoRoot,
			Timeout:  30 * time.Second,
		}

		// Collect all non-bare branch names
		var branchNames []string
		for _, e := range entries {
			if !e.Bare {
				branchNames = append(branchNames, e.Branch)
			}
		}

		if len(branchNames) > 0 {
			ctx := context.Background()
			if err := checker.UpdateAll(ctx, branchNames); err != nil {
				logger.Warn("Some branches failed to update: %v", err)
			}
			// Re-scan after update
			states, err = state.ScanStateFiles(repoRoot)
			if err != nil {
				logger.Warn("Failed to re-scan state files: %v", err)
				states = make(map[string]state.StateFile)
			}
		}
	} else {
		// Check if background update needed
		if codestatus.NeedsUpdate(states, time.Now()) {
			exe, err := os.Executable()
			if err == nil {
				if bgErr := codestatus.StartBackground(repoRoot, exe, nil); bgErr != nil {
					logger.Debug("Failed to start background updater: %v", bgErr)
				}
			}
		}
	}

	branches := listing.CollectBranches(entries, states)

	if flagListJSON {
		return listing.FormatJSON(os.Stdout, branches)
	}

	// Read env vars for table formatting
	maxWidth := 40
	if v := os.Getenv("TT_LIST_WIDTH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxWidth = n
		}
	}
	padding := 2
	if v := os.Getenv("TT_LIST_PADDING"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			padding = n
		}
	}

	opts := listing.TableOptions{
		ShowPath: flagListPath,
		Full:     flagListFull,
		MaxWidth: maxWidth,
		Padding:  padding,
	}
	listing.FormatTable(os.Stdout, branches, opts)

	// Show environment variables report if --env is specified
	if flagEnv {
		rep := &report.Report{
			StartTime:   time.Now(),
			Branch:      "(list)",
			EnvVars:     CollectEnvVars(),
			ShowEnvVars: true,
		}
		rep.Print(os.Stderr)
	}

	// Show background process message
	if !flagListUpdate && codestatus.IsRunning(repoRoot) {
		fmt.Fprintln(os.Stderr, "* update process is still running in the background.")
	}

	return nil
}

// runListFeatures shows features for a specific branch (existing behavior).
func runListFeatures(args []string) error {
	branch := args[0]
	// list does not need feature, pass branch only
	ctx, err := InitContext([]string{branch})
	if err != nil {
		return err
	}
	defer finalizeReport(ctx)

	globalCfg, _ := resolve.LoadGlobalConfig(ctx.RepoRoot)
	_ = globalCfg

	statePath := state.StatePath(ctx.RepoRoot, branch)
	sf, err := state.Load(statePath)
	if err != nil {
		if flagListJSON {
			fmt.Fprintln(os.Stdout, "[]")
		} else {
			fmt.Fprintf(os.Stdout, "No state found for branch %q\n", branch)
		}
		ctx.Report.OverallResult = "SUCCESS"
		return nil
	}

	if len(sf.Features) == 0 {
		if flagListJSON {
			fmt.Fprintln(os.Stdout, "[]")
		} else {
			fmt.Fprintf(os.Stdout, "No features for branch %q\n", branch)
		}
		ctx.Report.OverallResult = "SUCCESS"
		return nil
	}

	if flagListJSON {
		type featureJSON struct {
			Name          string `json:"name"`
			Status        string `json:"status"`
			ContainerName string `json:"container_name,omitempty"`
			StartedAt     string `json:"started_at"`
		}
		features := make([]featureJSON, 0, len(sf.Features))
		for name, fs := range sf.Features {
			features = append(features, featureJSON{
				Name:          name,
				Status:        string(fs.Status),
				ContainerName: fs.Connectivity.Docker.ContainerName,
				StartedAt:     fs.StartedAt.Format("2006-01-02 15:04"),
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(features); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}
	} else {
		// Print table header
		fmt.Fprintf(os.Stdout, "%-20s %-10s %-20s %s\n", "Feature", "Status", "Container", "StartedAt")
		fmt.Fprintf(os.Stdout, "%-20s %-10s %-20s %s\n", "-------", "------", "---------", "---------")

		for name, fs := range sf.Features {
			fmt.Fprintf(os.Stdout, "%-20s %-10s %-20s %s\n",
				name,
				string(fs.Status),
				fs.Connectivity.Docker.ContainerName,
				fs.StartedAt.Format("2006-01-02 15:04"),
			)
		}
	}

	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
