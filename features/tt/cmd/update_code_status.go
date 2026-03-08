package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/cmdexec"
	"github.com/axsh/tokotachi/features/tt/internal/codestatus"
	"github.com/axsh/tokotachi/features/tt/internal/log"
	"github.com/axsh/tokotachi/features/tt/internal/state"
)

var flagUpdateRepoRoot string

var updateCodeStatusCmd = &cobra.Command{
	Use:    "_update-code-status",
	Short:  "Internal: update code hosting status (background)",
	Hidden: true,
	RunE:   runUpdateCodeStatus,
}

func init() {
	updateCodeStatusCmd.Flags().StringVar(&flagUpdateRepoRoot, "repo-root", "", "Repository root path")
}

func runUpdateCodeStatus(cmd *cobra.Command, args []string) error {
	logger := log.New(os.Stderr, flagVerbose)

	repoRoot := flagUpdateRepoRoot
	if repoRoot == "" {
		var err error
		repoRoot, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Acquire lock
	if err := codestatus.AcquireLock(repoRoot); err != nil {
		logger.Debug("Lock already held, exiting: %v", err)
		return nil
	}
	defer codestatus.ReleaseLock(repoRoot)

	// Set up timeout context
	ctx, cancel := context.WithTimeout(context.Background(), codestatus.ProcessTimeout)
	defer cancel()

	// Scan state files
	states, err := state.ScanStateFiles(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to scan state files: %w", err)
	}

	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
	ghCmd := cmdexec.ResolveCommand("TT_CMD_GH", "gh")

	checker := &codestatus.Checker{
		GitCmd:   gitCmd,
		GhCmd:    ghCmd,
		RepoRoot: repoRoot,
		Timeout:  30 * time.Second,
	}

	branchNames := codestatus.BranchesNeedingUpdate(states, time.Now())
	if len(branchNames) == 0 {
		logger.Debug("No branches need updating")
		return nil
	}

	logger.Debug("Updating code status for %d branches...", len(branchNames))
	if err := checker.UpdateAll(ctx, branchNames); err != nil {
		logger.Warn("Some branches failed to update: %v", err)
	}

	return nil
}
