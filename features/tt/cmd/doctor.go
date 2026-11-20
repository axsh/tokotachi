package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/pkg/doctor"
)

var (
	doctorFlagFeature string
	doctorFlagJSON    bool
	doctorFlagFix     bool
	doctorFlagForge   string
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check repository health and configuration",
	Long: `Diagnose the repository structure, configuration files, and external tool
availability (gh for GitHub or glab for GitLab, depending on resolved forge host).` + forgeHostResolveHelp + `
Examples:
  tt doctor --forge gitlab
  TT_FORGE=gitlab tt doctor --json
`,
	Args: cobra.NoArgs,
	RunE: runDoctor,
}

func init() {
	doctorCmd.Flags().StringVar(&doctorFlagFeature, "feature", "", "Check only the specified feature")
	doctorCmd.Flags().BoolVar(&doctorFlagJSON, "json", false, "Output results in JSON format")
	doctorCmd.Flags().BoolVar(&doctorFlagFix, "fix", false, "Auto-fix issues where possible")
	doctorCmd.Flags().StringVar(&doctorFlagForge, "forge", "",
		"Forge host override: github|gitlab (takes precedence over TT_FORGE; process-local only)")
}

func runDoctor(cmd *cobra.Command, args []string) error {
	repoRoot, err := os.Getwd()
	if err != nil {
		repoRoot = "."
	}

	opts := doctor.Options{
		RepoRoot:      repoRoot,
		FeatureFilter: doctorFlagFeature,
		ToolChecker:   &doctor.DefaultToolChecker{},
		Fix:           doctorFlagFix,
		ForgeOverride: doctorFlagForge,
	}

	report, err := doctor.Run(opts)
	if err != nil {
		return err
	}

	if doctorFlagJSON {
		if err := report.PrintJSON(os.Stdout); err != nil {
			return err
		}
	} else {
		report.PrintText(os.Stdout)
	}

	if report.HasFailures() {
		os.Exit(1)
	}

	return nil
}
