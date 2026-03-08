package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/devctl/internal/resolve"
	"github.com/axsh/tokotachi/features/devctl/internal/state"
)

var listCmd = &cobra.Command{
	Use:   "list <branch>",
	Short: "List features for a branch",
	Long:  "List all features under the given branch by reading the state file.",
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
	_ = globalCfg

	statePath := state.StatePath(ctx.RepoRoot, branch)
	sf, err := state.Load(statePath)
	if err != nil {
		fmt.Fprintf(os.Stdout, "No state found for branch %q\n", branch)
		ctx.Report.OverallResult = "SUCCESS"
		return nil
	}

	if len(sf.Features) == 0 {
		fmt.Fprintf(os.Stdout, "No features for branch %q\n", branch)
		ctx.Report.OverallResult = "SUCCESS"
		return nil
	}

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

	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
