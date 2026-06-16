package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/agent/intake"
	"github.com/axsh/tokotachi/features/tt/internal/agent/status"
	"github.com/axsh/tokotachi/features/tt/internal/agent/storage"
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

var agentIntakeProcessedCmd = &cobra.Command{
	Use:   "processed [event-id]",
	Short: "Move an intake event from pending to processed",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentIntakeProcessed,
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
	intakeShowRedact bool
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

	agentIntakeShowCmd.Flags().BoolVar(&intakeShowRedact, "redact", false, "Redact provenance fields")

	agentIntakeCmd.AddCommand(agentIntakeListCmd)
	agentIntakeCmd.AddCommand(agentIntakeShowCmd)
	agentIntakeCmd.AddCommand(agentIntakeProcessedCmd)
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

	if intakeShowRedact {
		event = status.RedactProvenance(event)
	}

	data, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func runAgentIntakeProcessed(cmd *cobra.Command, args []string) error {
	varDir := filepath.Join("prompts", "memory", "var")
	eventID := args[0]

	moveErr := intake.MoveToProcessed(varDir, eventID)
	if moveErr != nil {
		if !errors.Is(moveErr, intake.ErrNotFoundInPending) {
			return fmt.Errorf("failed to move event to processed: %w", moveErr)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "[WARN] %v (attempting DB-only update)\n", moveErr)
	}

	// Update index.db status
	dbPath := filepath.Join(varDir, "intake", "index.db")
	idx, err := storage.NewIndex(dbPath)
	if err != nil {
		if moveErr != nil {
			return fmt.Errorf("event %s: file not in pending and index unavailable: %w", eventID, err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "[WARN] Index update skipped: %v\n", err)
	} else {
		defer idx.Close()
		if updateErr := idx.UpdateStatus(eventID, "processed"); updateErr != nil {
			if moveErr != nil {
				return fmt.Errorf("event %s: file not in pending and index update failed: %w", eventID, updateErr)
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "[WARN] Index update failed: %v\n", updateErr)
		}
	}

	fmt.Printf("Event %s moved to processed\n", eventID)
	return nil
}
