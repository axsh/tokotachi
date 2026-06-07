package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

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
	memoryRoot := filepath.Join("prompts", "memory")
	varDir := filepath.Join(memoryRoot, "var")
	branch := getCurrentBranch()

	report, err := status.GetStatus(memoryRoot, varDir, branch)
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

// getCurrentBranch returns the current git branch name.
func getCurrentBranch() string {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
