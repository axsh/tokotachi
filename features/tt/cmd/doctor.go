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
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check repository health and configuration",
	Long:  "Diagnose the repository structure, configuration files, and external tool availability.",
	Args:  cobra.NoArgs,
	RunE:  runDoctor,
}

func init() {
	doctorCmd.Flags().StringVar(&doctorFlagFeature, "feature", "", "Check only the specified feature")
	doctorCmd.Flags().BoolVar(&doctorFlagJSON, "json", false, "Output results in JSON format")
	doctorCmd.Flags().BoolVar(&doctorFlagFix, "fix", false, "Auto-fix issues where possible")
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
