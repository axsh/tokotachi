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
	"github.com/axsh/tokotachi/features/tt/internal/agent/record"
)

var agentRecordCmd = &cobra.Command{
	Use:   "record",
	Short: "Record a far-knowledge intake event",
	Long: `Record a far-knowledge intake event from a coding agent.

Accepts input via JSON file/stdin or CLI flags. The two modes are mutually exclusive.

JSON mode:
  tt agent record --file payload.json
  cat payload.json | tt agent record --stdin

CLI flag mode:
  tt agent record --agent antigravity --summary "Implemented auth" --note "Added JWT" --note "Updated config" --design-pattern`,
	RunE: runAgentRecord,
}

// CLI flags
var (
	recordFile         string
	recordStdin        bool
	recordAgent        string
	recordSummary      string
	recordSummaryFile  string
	recordNotes        []string
	recordNotesFile    string
	recordChangedPaths []string
	recordFromGit      bool
	// Existing flags
	recordFlagArch   bool
	recordFlagMem    bool
	recordFlagPrompt bool
	recordFlagAgent  bool
	recordFlagUrgent bool
	// Far-knowledge flags (R1)
	recordFlagDesignPattern bool
	recordFlagConvention    bool
	recordFlagLessonLearned bool
	recordFlagPreference    bool
	// Metadata
	recordClientReqID  string
	recordDryRun       bool
	recordPrintPayload bool
)

func init() {
	// JSON input mode
	agentRecordCmd.Flags().StringVar(&recordFile, "file", "", "Path to JSON payload file")
	agentRecordCmd.Flags().BoolVar(&recordStdin, "stdin", false, "Read JSON payload from stdin")

	// CLI flag mode
	agentRecordCmd.Flags().StringVar(&recordAgent, "agent", "", "Agent identifier (codex, claude-code, antigravity, cursor, unknown)")
	agentRecordCmd.Flags().StringVar(&recordSummary, "summary", "", "Task summary (max 500 chars)")
	agentRecordCmd.Flags().StringVar(&recordSummaryFile, "summary-file", "", "Read task summary from file")
	agentRecordCmd.Flags().StringArrayVar(&recordNotes, "note", nil, "Raw note (can be repeated)")
	agentRecordCmd.Flags().StringVar(&recordNotesFile, "notes-file", "", "Read notes from file (one per line)")
	agentRecordCmd.Flags().StringArrayVar(&recordChangedPaths, "changed-path", nil, "Changed file path (can be repeated)")
	agentRecordCmd.Flags().BoolVar(&recordFromGit, "changed-paths-from-git", false, "Collect changed paths from git dirty state")

	// Existing flag shortcuts
	agentRecordCmd.Flags().BoolVar(&recordFlagArch, "architecture-impact", false, "Flag: architecture impact")
	agentRecordCmd.Flags().BoolVar(&recordFlagMem, "memory-related", false, "Flag: memory related")
	agentRecordCmd.Flags().BoolVar(&recordFlagPrompt, "prompt-related", false, "Flag: prompt related")
	agentRecordCmd.Flags().BoolVar(&recordFlagAgent, "agent-behavior-related", false, "Flag: agent behavior related")
	agentRecordCmd.Flags().BoolVar(&recordFlagUrgent, "requires-immediate-action", false, "Flag: requires immediate action")

	// Far-knowledge flag shortcuts (R1)
	agentRecordCmd.Flags().BoolVar(&recordFlagDesignPattern, "design-pattern", false, "Flag: cross-cutting design pattern")
	agentRecordCmd.Flags().BoolVar(&recordFlagConvention, "convention", false, "Flag: convention or style rule")
	agentRecordCmd.Flags().BoolVar(&recordFlagLessonLearned, "lesson-learned", false, "Flag: lesson from past failure or review")
	agentRecordCmd.Flags().BoolVar(&recordFlagPreference, "preference", false, "Flag: engineer preference or quality standard")

	// Metadata
	agentRecordCmd.Flags().StringVar(&recordClientReqID, "client-request-id", "", "Client-side deduplication ID")

	// Output control
	agentRecordCmd.Flags().BoolVar(&recordDryRun, "dry-run", false, "Validate only, do not store")
	agentRecordCmd.Flags().BoolVar(&recordPrintPayload, "print-payload", false, "Print constructed payload and exit")

	agentCmd.AddCommand(agentRecordCmd)
}

func runAgentRecord(cmd *cobra.Command, args []string) error {
	// Determine input mode
	jsonMode := recordFile != "" || recordStdin
	cliMode := recordAgent != "" || recordSummary != "" || recordSummaryFile != "" || len(recordNotes) > 0 || recordNotesFile != ""

	if jsonMode && cliMode {
		return fmt.Errorf("--file/--stdin and CLI flags (--agent, --summary, --note, etc.) are mutually exclusive")
	}
	if !jsonMode && !cliMode {
		return fmt.Errorf("provide either --file/--stdin or CLI flags (--agent, --summary, --note)")
	}

	var inputJSON []byte
	var err error

	if jsonMode {
		inputJSON, err = readRecordJSONInput()
		if err != nil {
			return err
		}
	} else {
		inputJSON, err = buildRecordPayloadFromFlags()
		if err != nil {
			return err
		}
	}

	if recordPrintPayload {
		fmt.Println(string(inputJSON))
		return nil
	}

	// Resolve schemas and var directories
	schemasDir := filepath.Join("prompts", "memory", "schemas")
	varDir := filepath.Join("prompts", "memory", "var")

	if recordDryRun {
		// Validate only
		v, err := record.NewValidator(schemasDir)
		if err != nil {
			return fmt.Errorf("failed to create validator: %w", err)
		}
		if err := v.Validate(inputJSON); err != nil {
			result := &agent.NotifyResult{
				Status:  "rejected",
				Code:    agent.CodeSchemaValidationError,
				Message: err.Error(),
			}
			return outputRecordResult(result, agent.ExitSchemaValidationError)
		}
		result := &agent.NotifyResult{
			Status:  "accepted",
			Code:    agent.CodeOK,
			Message: "Dry-run: validation passed",
		}
		return outputRecordResult(result, agent.ExitOK)
	}

	// Full pipeline
	h, err := record.NewHandler(schemasDir, varDir)
	if err != nil {
		return fmt.Errorf("failed to initialize handler: %w", err)
	}
	defer h.Close()

	result, exitCode := h.HandleNotify(inputJSON, recordFromGit)
	return outputRecordResult(result, exitCode)
}

// readRecordJSONInput reads JSON from file or stdin.
func readRecordJSONInput() ([]byte, error) {
	if recordFile != "" {
		data, err := os.ReadFile(recordFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", recordFile, err)
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

// buildRecordPayloadFromFlags constructs a NotifyPayload from CLI flags and marshals to JSON.
func buildRecordPayloadFromFlags() ([]byte, error) {
	payload := agent.NotifyPayload{
		Version:    1,
		SourceType: "coding_agent",
		Agent:      recordAgent,
	}

	// Summary
	if recordSummaryFile != "" {
		data, err := os.ReadFile(recordSummaryFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read summary file: %w", err)
		}
		payload.TaskSummary = strings.TrimSpace(string(data))
	} else {
		payload.TaskSummary = recordSummary
	}

	// Notes
	if recordNotesFile != "" {
		data, err := os.ReadFile(recordNotesFile)
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
	payload.RawNotes = append(payload.RawNotes, recordNotes...)

	// Changed paths
	payload.ChangedPaths = recordChangedPaths

	// Flags (existing 5 + new 4)
	anyFlag := recordFlagArch || recordFlagMem || recordFlagPrompt || recordFlagAgent || recordFlagUrgent ||
		recordFlagDesignPattern || recordFlagConvention || recordFlagLessonLearned || recordFlagPreference
	if anyFlag {
		payload.Flags = &agent.Flags{
			ArchitectureImpact:      recordFlagArch,
			MemoryRelated:           recordFlagMem,
			PromptRelated:           recordFlagPrompt,
			AgentBehaviorRelated:    recordFlagAgent,
			RequiresImmediateAction: recordFlagUrgent,
			DesignPattern:           recordFlagDesignPattern,
			Convention:              recordFlagConvention,
			LessonLearned:           recordFlagLessonLearned,
			Preference:              recordFlagPreference,
		}
	}

	// Client request ID
	payload.ClientRequestID = recordClientReqID

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}
	return data, nil
}

// outputRecordResult outputs the result as JSON to stdout and sets the exit code.
func outputRecordResult(result *agent.NotifyResult, exitCode int) error {
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
