package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/agent/assist"
)

var agentAssistCmd = &cobra.Command{
	Use:   "assist",
	Short: "Generate an agent task from pending intake events",
	Long: `Scans pending intake events for the current branch and generates
an Agent Task for a coding agent to process.

This command does NOT perform LLM processing. It only creates a task
description that a coding agent can read and act upon.`,
	RunE: runAgentAssist,
}

var (
	assistScope string
	assistForce bool
)

func init() {
	agentAssistCmd.Flags().StringVar(&assistScope, "scope", "", "Scope (required: current-branch)")
	agentAssistCmd.Flags().BoolVar(&assistForce, "force", false, "Force new task creation even if pending task exists")
	_ = agentAssistCmd.MarkFlagRequired("scope")
	agentCmd.AddCommand(agentAssistCmd)
}

func runAgentAssist(cmd *cobra.Command, args []string) error {
	if assistScope != "current-branch" {
		return fmt.Errorf("unsupported scope: %s (only 'current-branch' is supported)", assistScope)
	}

	branch := getCurrentBranch()
	if branch == "" {
		return fmt.Errorf("failed to detect current git branch")
	}

	varDir := filepath.Join("prompts", "memory", "var")
	h, err := assist.NewHandler(varDir)
	if err != nil {
		return fmt.Errorf("failed to initialize assist handler: %w", err)
	}
	defer h.Close()

	result, exitCode := h.HandleAssist(branch, assistForce)

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}
	fmt.Println(string(data))

	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}
