package cmd

import "github.com/spf13/cobra"

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage agent memory and notifications",
}

func init() {
	rootCmd.AddCommand(agentCmd)
}
