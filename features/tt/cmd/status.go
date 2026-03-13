package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/pkg/resolve"
	"github.com/axsh/tokotachi/pkg/worktree"
)

var statusCmd = &cobra.Command{
	Use:   "status <branch> [feature]",
	Short: "Show environment status",
	Long:  "Show worktree and container status. If feature is omitted, only shows worktree status.",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx, err := InitContext(args)
	if err != nil {
		return err
	}
	defer finalizeReport(ctx)

	globalCfg, _ := resolve.LoadGlobalConfig(ctx.RepoRoot)
	projectName := globalCfg.ProjectName
	if projectName == "" {
		projectName = "tt"
	}

	wm := &worktree.Manager{CmdRunner: ctx.CmdRunner, RepoRoot: ctx.RepoRoot}
	worktreePath := wm.Path(ctx.Branch)

	if ctx.HasFeature() {
		// Full status: worktree + container
		containerName := resolve.ContainerName(projectName, ctx.Feature)
		resolvedPath, _ := resolve.Worktree(ctx.RepoRoot, ctx.Branch)
		if resolvedPath != "" {
			worktreePath = resolvedPath
		}
		ctx.ActionRunner.PrintStatus(ctx.Feature, containerName, worktreePath)
	} else {
		// Worktree only status
		exists := wm.Exists(ctx.Branch)
		if exists {
			fmt.Printf("📁 Branch: %s\n", ctx.Branch)
			fmt.Printf("   Worktree: %s\n", worktreePath)
			fmt.Println("   Status: WORKTREE_ONLY (no feature specified)")
		} else {
			fmt.Printf("❌ Branch: %s — worktree not found\n", ctx.Branch)
		}
	}

	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
