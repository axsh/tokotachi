package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/axsh/tokotachi/features/tt/internal/agent"
	"github.com/axsh/tokotachi/features/tt/internal/agent/notify"
)

var agentNotifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Submit a memory intake notification",
	Long: `Submit a memory intake notification from a coding agent.

Accepts input via JSON file/stdin or CLI flags. The two modes are mutually exclusive.

JSON mode:
  tt agent notify --file payload.json
  cat payload.json | tt agent notify --stdin

CLI flag mode:
  tt agent notify --agent antigravity --summary "Implemented auth" --note "Added JWT" --note "Updated config"`,
	RunE: runAgentNotify,
}

// CLI flags
var (
	notifyFile         string
	notifyStdin        bool
	notifyAgent        string
	notifySummary      string
	notifySummaryFile  string
	notifyNotes        []string
	notifyNotesFile    string
	notifyChangedPaths []string
	notifyFromGit      bool
	notifyFlagArch     bool
	notifyFlagMem      bool
	notifyFlagPrompt   bool
	notifyFlagAgent    bool
	notifyFlagUrgent   bool
	notifyClientReqID  string
	notifyDryRun       bool
	notifyPrintPayload bool
)

func init() {
	// JSON input mode
	agentNotifyCmd.Flags().StringVar(&notifyFile, "file", "", "Path to JSON payload file")
	agentNotifyCmd.Flags().BoolVar(&notifyStdin, "stdin", false, "Read JSON payload from stdin")

	// CLI flag mode
	agentNotifyCmd.Flags().StringVar(&notifyAgent, "agent", "", "Agent identifier (codex, claude-code, antigravity, cursor, unknown)")
	agentNotifyCmd.Flags().StringVar(&notifySummary, "summary", "", "Task summary (max 500 chars)")
	agentNotifyCmd.Flags().StringVar(&notifySummaryFile, "summary-file", "", "Read task summary from file")
	agentNotifyCmd.Flags().StringArrayVar(&notifyNotes, "note", nil, "Raw note (can be repeated)")
	agentNotifyCmd.Flags().StringVar(&notifyNotesFile, "notes-file", "", "Read notes from file (one per line)")
	agentNotifyCmd.Flags().StringArrayVar(&notifyChangedPaths, "changed-path", nil, "Changed file path (can be repeated)")
	agentNotifyCmd.Flags().BoolVar(&notifyFromGit, "changed-paths-from-git", false, "Collect changed paths from git dirty state")

	// Flag shortcuts
	agentNotifyCmd.Flags().BoolVar(&notifyFlagArch, "architecture-impact", false, "Flag: architecture impact")
	agentNotifyCmd.Flags().BoolVar(&notifyFlagMem, "memory-related", false, "Flag: memory related")
	agentNotifyCmd.Flags().BoolVar(&notifyFlagPrompt, "prompt-related", false, "Flag: prompt related")
	agentNotifyCmd.Flags().BoolVar(&notifyFlagAgent, "agent-behavior-related", false, "Flag: agent behavior related")
	agentNotifyCmd.Flags().BoolVar(&notifyFlagUrgent, "requires-immediate-action", false, "Flag: requires immediate action")

	// Metadata
	agentNotifyCmd.Flags().StringVar(&notifyClientReqID, "client-request-id", "", "Client-side deduplication ID")

	// Output control
	agentNotifyCmd.Flags().BoolVar(&notifyDryRun, "dry-run", false, "Validate only, do not store")
	agentNotifyCmd.Flags().BoolVar(&notifyPrintPayload, "print-payload", false, "Print constructed payload and exit")

	agentCmd.AddCommand(agentNotifyCmd)
}

func runAgentNotify(cmd *cobra.Command, args []string) error {
	// Determine input mode
	jsonMode := notifyFile != "" || notifyStdin
	cliMode := notifyAgent != "" || notifySummary != "" || notifySummaryFile != "" || len(notifyNotes) > 0 || notifyNotesFile != ""

	if jsonMode && cliMode {
		return fmt.Errorf("--file/--stdin and CLI flags (--agent, --summary, --note, etc.) are mutually exclusive")
	}
	if !jsonMode && !cliMode {
		return fmt.Errorf("provide either --file/--stdin or CLI flags (--agent, --summary, --note)")
	}

	var inputJSON []byte
	var err error

	if jsonMode {
		inputJSON, err = readJSONInput()
		if err != nil {
			return err
		}
	} else {
		inputJSON, err = buildPayloadFromFlags()
		if err != nil {
			return err
		}
	}

	if notifyPrintPayload {
		fmt.Println(string(inputJSON))
		return nil
	}

	// Resolve schemas and var directories
	schemasDir := filepath.Join("prompts", "memory", "schemas")
	varDir := filepath.Join("prompts", "memory", "var")

	if notifyDryRun {
		// Validate only
		v, err := notify.NewValidator(schemasDir)
		if err != nil {
			return fmt.Errorf("failed to create validator: %w", err)
		}
		if err := v.Validate(inputJSON); err != nil {
			result := &agent.NotifyResult{
				Status:  "rejected",
				Code:    agent.CodeSchemaValidationError,
				Message: err.Error(),
			}
			return outputResult(result, agent.ExitSchemaValidationError)
		}
		result := &agent.NotifyResult{
			Status:  "accepted",
			Code:    agent.CodeOK,
			Message: "Dry-run: validation passed",
		}
		return outputResult(result, agent.ExitOK)
	}

	// Full pipeline
	h, err := notify.NewHandler(schemasDir, varDir)
	if err != nil {
		return fmt.Errorf("failed to initialize handler: %w", err)
	}
	defer h.Close()

	result, exitCode := h.HandleNotify(inputJSON, notifyFromGit)
	return outputResult(result, exitCode)
}

// readJSONInput reads JSON from file or stdin.
func readJSONInput() ([]byte, error) {
	if notifyFile != "" {
		data, err := os.ReadFile(notifyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", notifyFile, err)
		}
		return data, nil
	}
	// stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdin: %w", err)
	}
	return data, nil
}

// buildPayloadFromFlags constructs a NotifyPayload from CLI flags and marshals to JSON.
func buildPayloadFromFlags() ([]byte, error) {
	payload := agent.NotifyPayload{
		Version:    1,
		SourceType: "coding_agent",
		Agent:      notifyAgent,
	}

	// Summary
	if notifySummaryFile != "" {
		data, err := os.ReadFile(notifySummaryFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read summary file: %w", err)
		}
		payload.TaskSummary = strings.TrimSpace(string(data))
	} else {
		payload.TaskSummary = notifySummary
	}

	// Notes
	if notifyNotesFile != "" {
		data, err := os.ReadFile(notifyNotesFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read notes file: %w", err)
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				payload.RawNotes = append(payload.RawNotes, line)
			}
		}
	}
	payload.RawNotes = append(payload.RawNotes, notifyNotes...)

	// Changed paths
	payload.ChangedPaths = notifyChangedPaths

	// Flags
	if notifyFlagArch || notifyFlagMem || notifyFlagPrompt || notifyFlagAgent || notifyFlagUrgent {
		payload.Flags = &agent.Flags{
			ArchitectureImpact:      notifyFlagArch,
			MemoryRelated:           notifyFlagMem,
			PromptRelated:           notifyFlagPrompt,
			AgentBehaviorRelated:    notifyFlagAgent,
			RequiresImmediateAction: notifyFlagUrgent,
		}
	}

	// Client request ID
	payload.ClientRequestID = notifyClientReqID

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	return data, nil
}

// outputResult outputs the result as JSON to stdout and sets the exit code.
func outputResult(result *agent.NotifyResult, exitCode int) error {
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
