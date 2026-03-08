package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/log"
	"github.com/axsh/tokotachi/features/tt/internal/scaffold"
)

var (
	scaffoldFlagYes      bool
	scaffoldFlagRollback bool
	scaffoldFlagList     bool
	scaffoldFlagRepo     string
	scaffoldFlagLang     string
	scaffoldFlagCwd      bool
)

var scaffoldCmd = &cobra.Command{
	Use:   "scaffold [category] [name]",
	Short: "Generate project structure from templates",
	Long:  "Scaffold creates directory structures and files from predefined templates downloaded from an external repository.",
	Args:  cobra.MaximumNArgs(2),
	RunE:  runScaffold,
}

func init() {
	scaffoldCmd.Flags().BoolVar(&scaffoldFlagYes, "yes", false, "Skip [y/N] confirmation and execute immediately")
	scaffoldCmd.Flags().BoolVar(&scaffoldFlagRollback, "rollback", false, "Rollback the last scaffold operation")
	scaffoldCmd.Flags().BoolVar(&scaffoldFlagList, "list", false, "List available scaffold templates")
	scaffoldCmd.Flags().StringVar(&scaffoldFlagRepo, "repo", "", "Override the default template repository URL")
	scaffoldCmd.Flags().StringVar(&scaffoldFlagLang, "lang", "", "Specify locale for template localization (e.g. ja, en)")
	scaffoldCmd.Flags().BoolVar(&scaffoldFlagCwd, "cwd", false, "Use current working directory as root instead of auto-detecting Git root")
}

func runScaffold(cmd *cobra.Command, args []string) error {
	repoRoot := resolveRepoRoot(scaffoldFlagCwd)

	logger := log.New(os.Stderr, flagVerbose)

	// Handle --rollback
	if scaffoldFlagRollback {
		return scaffold.Rollback(repoRoot, logger)
	}

	// Handle --list
	if scaffoldFlagList {
		entries, err := scaffold.List(scaffoldFlagRepo)
		if err != nil {
			return err
		}
		fmt.Println("Available scaffold templates:")
		for _, entry := range entries {
			fmt.Printf("  %-20s %s [%s]\n", entry.Name, entry.Description, entry.Category)
		}
		return nil
	}

	// Run scaffold
	opts := scaffold.RunOptions{
		Pattern:  args,
		RepoURL:  scaffoldFlagRepo,
		RepoRoot: repoRoot,
		DryRun:   flagDryRun,
		Yes:      scaffoldFlagYes,
		Lang:     scaffoldFlagLang,
		Logger:   logger,
		Stdout:   os.Stdout,
		Stdin:    os.Stdin,
	}

	plan, err := scaffold.Run(opts)
	if err != nil {
		return err
	}

	// plan can be nil if multiple matches were found (category listing)
	if plan == nil {
		return nil
	}

	// Display the execution plan
	scaffold.PrintPlan(plan, os.Stdout)

	// Check for error conflicts
	if len(plan.Warnings) > 0 {
		return fmt.Errorf("cannot proceed due to conflicts (see warnings above)")
	}

	// --dry-run: display only, no confirmation, no execution
	if flagDryRun {
		return nil
	}

	// Confirm with user (unless --yes)
	if !scaffoldFlagYes {
		fmt.Print("\nProceed? [y/N]: ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			response := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if response != "y" && response != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		} else {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Apply
	return scaffold.Apply(plan, opts)
}

// resolveRepoRoot determines the target root directory.
// If useCwd is true, always uses os.Getwd().
// Otherwise, tries "git rev-parse --show-toplevel" first,
// falling back to os.Getwd() on failure.
func resolveRepoRoot(useCwd bool) string {
	if !useCwd {
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		out, err := cmd.Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
