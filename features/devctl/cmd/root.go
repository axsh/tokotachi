package cmd

import (
	"os"
	"time"

	"github.com/spf13/cobra"
)

var (
	flagVerbose bool
	flagDryRun  bool
	flagReport  string
)

var rootCmd = &cobra.Command{
	Use:   "devctl",
	Short: "Development environment orchestrator",
	Long:  "devctl manages feature-level development environments with matrix-driven subcommands.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Store start time for report
		cmd.SetContext(cmd.Context())
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Show debug logs")
	rootCmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "Show planned actions without executing")
	rootCmd.PersistentFlags().StringVar(&flagReport, "report", "", "Write execution report to Markdown file")

	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(execCmd)
	rootCmd.AddCommand(prCmd)
	rootCmd.AddCommand(closeCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(scaffoldCmd)
	rootCmd.AddCommand(updateCodeStatusCmd)
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// finalizeReport outputs the execution report.
func finalizeReport(ctx *AppContext) {
	if ctx == nil || ctx.Report == nil {
		return
	}
	ctx.Report.StartTime = time.Now()
	ctx.Report.Print(os.Stdout)
	if ctx.ReportFile != "" {
		if err := ctx.Report.WriteMarkdown(ctx.ReportFile); err != nil {
			ctx.Logger.Error("Failed to write report: %v", err)
		} else {
			ctx.Logger.Info("Report written to %s", ctx.ReportFile)
		}
	}
}
