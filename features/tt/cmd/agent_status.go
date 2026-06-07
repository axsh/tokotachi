package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/agent/status"
)

var agentStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show agent intake status summary",
	RunE:  runAgentStatus,
}

var statusFormat string

func init() {
	agentStatusCmd.Flags().StringVar(&statusFormat, "format", "json", "Output format (json or table)")
	agentCmd.AddCommand(agentStatusCmd)
}

func runAgentStatus(cmd *cobra.Command, args []string) error {
	varDir := filepath.Join("prompts", "memory", "var")

	report, err := status.GetStatus(varDir)
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
