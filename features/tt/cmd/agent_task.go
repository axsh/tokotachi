package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/agent/task"
)

var agentTaskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage agent tasks",
}

var agentTaskShowCmd = &cobra.Command{
	Use:   "show [task-id]",
	Short: "Show an agent task",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentTaskShow,
}

var agentTaskSubmitCmd = &cobra.Command{
	Use:   "submit [task-id]",
	Short: "Submit knowledge atom results for a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runAgentTaskSubmit,
}

var submitFile string

func init() {
	agentTaskSubmitCmd.Flags().StringVar(&submitFile, "file", "", "Path to result JSON file")
	_ = agentTaskSubmitCmd.MarkFlagRequired("file")

	agentTaskCmd.AddCommand(agentTaskShowCmd)
	agentTaskCmd.AddCommand(agentTaskSubmitCmd)
	agentCmd.AddCommand(agentTaskCmd)
}

func runAgentTaskShow(cmd *cobra.Command, args []string) error {
	varDir := filepath.Join("prompts", "memory", "var")
	t, err := task.Show(varDir, args[0])
	if err != nil {
		return fmt.Errorf("failed to show task: %w", err)
	}

	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func runAgentTaskSubmit(cmd *cobra.Command, args []string) error {
	memoryRoot := filepath.Join("prompts", "memory")
	varDir := filepath.Join(memoryRoot, "var")
	schemasDir := filepath.Join(memoryRoot, "schemas")

	h, err := task.NewSubmitHandler(memoryRoot, varDir, schemasDir)
	if err != nil {
		return fmt.Errorf("failed to initialize submit handler: %w", err)
	}
	defer h.Close()

	result, exitCode := h.HandleSubmit(args[0], submitFile)

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
