package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/escape-dev/devctl/internal/resolve"
	"github.com/escape-dev/devctl/internal/state"
	"github.com/escape-dev/devctl/internal/worktree"
)

var listCmd = &cobra.Command{
	Use:   "list <feature>",
	Short: "List branches for a feature",
	Args:  cobra.ExactArgs(1),
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	feature := args[0]
	// list does not need branch, so manual init
	ctx, err := InitContext([]string{feature, feature})
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
	entries, err := wm.List(feature)
	if err != nil {
		return fmt.Errorf("list failed: %w", err)
	}

	if len(entries) == 0 {
		fmt.Fprintf(os.Stdout, "No worktrees found for feature %q\n", feature)
		ctx.Report.OverallResult = "SUCCESS"
		return nil
	}

	// Print table header
	fmt.Fprintf(os.Stdout, "%-20s %-10s %-15s %s\n", "Branch", "Status", "ContainerMode", "CreatedAt")
	fmt.Fprintf(os.Stdout, "%-20s %-10s %-15s %s\n", "------", "------", "-------------", "---------")

	for _, e := range entries {
		statePath := state.StatePath(ctx.RepoRoot, feature, e.Branch)
		s, err := state.Load(statePath)
		if err != nil {
			// No state file, show minimal info
			fmt.Fprintf(os.Stdout, "%-20s %-10s %-15s %s\n", e.Branch, "unknown", "-", "-")
			continue
		}
		containerName := resolve.ContainerName(projectName, feature)
		_ = containerName
		fmt.Fprintf(os.Stdout, "%-20s %-10s %-15s %s\n",
			e.Branch,
			string(s.Status),
			s.ContainerMode,
			s.CreatedAt.Format("2006-01-02 15:04"),
		)
	}

	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
