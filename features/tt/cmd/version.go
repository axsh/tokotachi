package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the tt product version",
	Long:  "Print the embedded product version string (no network, no workspace writes).",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.OutOrStdout(), "tt %s\n", version.Version)
		return nil
	},
}
