package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/devctl/internal/cmdexec"
	"github.com/axsh/tokotachi/features/devctl/internal/listing"
	"github.com/axsh/tokotachi/features/devctl/internal/log"
	"github.com/axsh/tokotachi/features/devctl/internal/resolve"
	"github.com/axsh/tokotachi/features/devctl/internal/state"
)

var (
	flagListJSON bool
	flagListPath bool
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

	gitCmd := cmdexec.ResolveCommand("DEVCTL_CMD_GIT", "git")
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

	branches := listing.CollectBranches(entries, states)

	if flagListJSON {
		return listing.FormatJSON(os.Stdout, branches)
	}
	listing.FormatTable(os.Stdout, branches, flagListPath)
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
