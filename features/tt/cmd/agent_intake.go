package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/agent/status"
)

var agentIntakeCmd = &cobra.Command{
	Use:   "intake",
	Short: "Manage intake events",
}

var agentIntakeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List intake events",
	RunE:  runAgentIntakeList,
}

var agentIntakeShowCmd = &cobra.Command{
	Use:   "show [event-id]",
	Short: "Show a single intake event",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentIntakeShow,
}

// Filter flags
var (
	intakeListStatus string
	intakeListAgent  string
	intakeListBranch string
	intakeListQuery  string
	intakeListFrom   string
	intakeListTo     string
	intakeListFormat string
	intakeListLimit  int
)

func init() {
	// List command flags
	agentIntakeListCmd.Flags().StringVar(&intakeListStatus, "status", "", "Filter by status (pending, processed, failed, ignored)")
	agentIntakeListCmd.Flags().StringVar(&intakeListAgent, "agent", "", "Filter by agent")
	agentIntakeListCmd.Flags().StringVar(&intakeListBranch, "branch", "", "Filter by branch")
	agentIntakeListCmd.Flags().StringVar(&intakeListQuery, "query", "", "FTS search query")
	agentIntakeListCmd.Flags().StringVar(&intakeListFrom, "from", "", "Filter from date (ISO8601)")
	agentIntakeListCmd.Flags().StringVar(&intakeListTo, "to", "", "Filter to date (ISO8601)")
	agentIntakeListCmd.Flags().StringVar(&intakeListFormat, "format", "json", "Output format (json or table)")
	agentIntakeListCmd.Flags().IntVar(&intakeListLimit, "limit", 50, "Maximum number of results")

	agentIntakeCmd.AddCommand(agentIntakeListCmd)
	agentIntakeCmd.AddCommand(agentIntakeShowCmd)
	agentCmd.AddCommand(agentIntakeCmd)
}

func runAgentIntakeList(cmd *cobra.Command, args []string) error {
	varDir := filepath.Join("prompts", "memory", "var")

	opts := status.ListOptions{
		Status: intakeListStatus,
		Agent:  intakeListAgent,
		Branch: intakeListBranch,
		Query:  intakeListQuery,
		From:   intakeListFrom,
		To:     intakeListTo,
		Format: intakeListFormat,
		Limit:  intakeListLimit,
	}

	items, err := status.List(varDir, opts)
	if err != nil {
		return fmt.Errorf("failed to list events: %w", err)
	}

	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal items: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func runAgentIntakeShow(cmd *cobra.Command, args []string) error {
	varDir := filepath.Join("prompts", "memory", "var")

	event, err := status.Show(varDir, args[0])
	if err != nil {
		return fmt.Errorf("failed to show event: %w", err)
	}

	data, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
