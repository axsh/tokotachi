package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/internal/report"
	"github.com/axsh/tokotachi/pkg/action"
)

var (
	mergeFlagNoFF bool
	mergeFlagFF   bool
)

var mergeCmd = &cobra.Command{
	Use:   "merge <branch>",
	Short: "Merge branch into its base branch locally",
	Long: "Performs a local git merge of the specified branch into its base branch. " +
		"The base branch is automatically resolved from the state recorded at 'tt open' time. " +
		"By default, uses --ff-only strategy.",
	Args: cobra.ExactArgs(1),
	RunE: runMerge,
}

func init() {
	mergeCmd.Flags().BoolVar(&mergeFlagNoFF, "no-ff", false, "Always create a merge commit")
	mergeCmd.Flags().BoolVar(&mergeFlagFF, "ff", false, "Use git default merge strategy (ff if possible)")
}

func runMerge(cmd *cobra.Command, args []string) error {
	ctx, err := InitContext(args)
	if err != nil {
		return err
	}
	defer finalizeReport(ctx)

	// Determine merge strategy from flags
	strategy := action.MergeStrategyFFOnly // default
	if mergeFlagNoFF {
		strategy = action.MergeStrategyNoFF
	} else if mergeFlagFF {
		strategy = action.MergeStrategyFF
	}

	result, err := ctx.ActionRunner.Merge(action.MergeOptions{
		Branch:   ctx.Branch,
		RepoRoot: ctx.RepoRoot,
		Strategy: strategy,
	})

	if err != nil {
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Merge", Success: false})
		ctx.Report.OverallResult = "FAILED"

		// Provide helpful hint for --ff-only failures
		if strategy == action.MergeStrategyFFOnly && strings.Contains(err.Error(), "git merge failed") {
			ctx.Logger.Info("Hint: If the base branch has new commits, try: tt merge %s --no-ff", ctx.Branch)
			ctx.Logger.Info("      Or use: tt merge %s --ff (to let git decide)", ctx.Branch)
		}

		return fmt.Errorf("merge failed: %w", err)
	}

	ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "Merge", Success: true})
	ctx.Report.OverallResult = "SUCCESS"
	ctx.Logger.Info("Successfully merged %s into %s (strategy: %s)", ctx.Branch, result.BaseBranch, result.Strategy)
	return nil
}
