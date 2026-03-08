package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/devctl/internal/resolve"
	"github.com/axsh/tokotachi/features/devctl/internal/state"
	"github.com/axsh/tokotachi/features/devctl/internal/worktree"
)

var listCmd = &cobra.Command{
	Use:   "list <branch>",
	Short: "List features for a branch",
	Long:  "List all feature worktrees under the given branch (scans work/<branch>/features/).",
	Args:  cobra.ExactArgs(1),
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	branch := args[0]
	// list does not need feature, pass branch only
	ctx, err := InitContext([]string{branch})
	if err != nil {
		return err
	}
	defer finalizeReport(ctx)

	globalCfg, _ := resolve.LoadGlobalConfig(ctx.RepoRoot)
	projectName := globalCfg.ProjectName
	if projectName == "" {
		projectName = "devctl"
	}

	wm := &worktree.Manager{CmdRunner: ctx.CmdRunner, RepoRoot: ctx.RepoRoot}
	entries, err := wm.List(branch)
	if err != nil {
		return fmt.Errorf("list failed: %w", err)
	}

	if len(entries) == 0 {
		fmt.Fprintf(os.Stdout, "No feature worktrees found for branch %q\n", branch)
		ctx.Report.OverallResult = "SUCCESS"
		return nil
	}

	// Print table header
	fmt.Fprintf(os.Stdout, "%-20s %-10s %-15s %s\n", "Feature", "Status", "ContainerMode", "CreatedAt")
	fmt.Fprintf(os.Stdout, "%-20s %-10s %-15s %s\n", "-------", "------", "-------------", "---------")

	for _, e := range entries {
		statePath := state.StatePath(ctx.RepoRoot, e.Feature, branch)
		s, err := state.Load(statePath)
		if err != nil {
			// No state file, show minimal info
			fmt.Fprintf(os.Stdout, "%-20s %-10s %-15s %s\n", e.Feature, "unknown", "-", "-")
			continue
		}
		containerName := resolve.ContainerName(projectName, e.Feature)
		_ = containerName
		fmt.Fprintf(os.Stdout, "%-20s %-10s %-15s %s\n",
			e.Feature,
			string(s.Status),
			s.ContainerMode,
			s.CreatedAt.Format("2006-01-02 15:04"),
		)
	}

	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
