package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/internal/report"
	"github.com/axsh/tokotachi/pkg/resolve"
	"github.com/axsh/tokotachi/pkg/state"
)

var flagPRForge string

var prCmd = &cobra.Command{
	Use:   "pr <branch> [feature]",
	Short: "Create a Pull/Merge Request on the remote forge",
	Long: `Create a Pull Request (GitHub: gh pr create) or Merge Request (GitLab: glab mr create)
for the worktree of the given branch.` + forgeHostResolveHelp + `
Examples:
  tt pr --forge gitlab <branch> [feature]
  TT_FORGE=gitlab tt pr <branch> [feature]
`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runPR,
}

func init() {
	prCmd.Flags().StringVar(&flagPRForge, "forge", "",
		"Forge host override: github|gitlab (takes precedence over TT_FORGE; process-local only)")
}

func runPR(cmd *cobra.Command, args []string) error {
	ctx, err := InitContext(args)
	if err != nil {
		return err
	}
	defer finalizeReport(ctx)

	worktreePath, err := resolve.Worktree(ctx.RepoRoot, ctx.Branch)
	if err != nil {
		return fmt.Errorf("worktree resolution failed: %w", err)
	}

	ctx.ActionRunner.ForgeOverride = flagPRForge
	ctx.Logger.Debug("pr forgeOverride=%q", flagPRForge)
	if err := ctx.ActionRunner.PR(ctx.RepoRoot, worktreePath); err != nil {
		ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "PR create", Success: false})
		ctx.Report.OverallResult = "FAILED"
		return fmt.Errorf("pr failed: %w", err)
	}

	// Update CodeStatus to PR after successful PR creation
	statePath := state.StatePath(ctx.RepoRoot, ctx.Branch)
	sf, loadErr := state.Load(statePath)
	if loadErr == nil {
		now := time.Now()
		sf.CodeStatus = &state.CodeStatus{
			Status:        state.CodeStatusPR,
			PRCreatedAt:   &now,
			LastCheckedAt: &now,
		}
		if saveErr := state.Save(statePath, sf); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to update code status: %v\n", saveErr)
		}
	}

	ctx.Report.Steps = append(ctx.Report.Steps, report.StepEntry{Name: "PR create", Success: true})
	ctx.Report.OverallResult = "SUCCESS"
	return nil
}
