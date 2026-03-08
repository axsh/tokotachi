package report

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/cmdexec"
)

// EnvVar represents a resolved environment variable.
type EnvVar struct {
	Name    string
	Value   string // actual value or ""
	Default string // fallback default
	WasSet  bool   // true if env var was explicitly set
}

// StepEntry describes one step in the execution.
type StepEntry struct {
	Name    string
	Record  *cmdexec.ExecRecord // nil if no command was executed
	Success bool
}

// Report aggregates execution context and results.
type Report struct {
	StartTime     time.Time
	Feature       string
	Branch        string
	OS            string
	Editor        string
	ContainerMode string
	EnvVars       []EnvVar
	ShowEnvVars   bool // only print env vars section when true
	Steps         []StepEntry
	OverallResult string // "SUCCESS" or "FAILED"
}

// Print writes the report to the given writer.
func (r *Report) Print(w io.Writer) {
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "# tt Execution Report\n")
	fmt.Fprintf(w, "- **Date**: %s\n", r.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "- **Feature**: %s\n", r.Feature)
	fmt.Fprintf(w, "- **Branch**: %s\n", r.Branch)
	fmt.Fprintf(w, "\n")

	// Environment Variables
	if r.ShowEnvVars && len(r.EnvVars) > 0 {
		fmt.Fprintf(w, "## Environment Variables\n")
		fmt.Fprintf(w, "| Variable | Value |\n")
		fmt.Fprintf(w, "|---|---|\n")
		for _, ev := range r.EnvVars {
			if ev.WasSet {
				fmt.Fprintf(w, "| %s | %s |\n", ev.Name, ev.Value)
			} else {
				fmt.Fprintf(w, "| %s | *(not set, default: %s)* |\n", ev.Name, ev.Default)
			}
		}
		fmt.Fprintf(w, "\n")
	}

	// Detected Environment
	if r.OS != "" || r.Editor != "" || r.ContainerMode != "" {
		fmt.Fprintf(w, "## Detected Environment\n")
		fmt.Fprintf(w, "- **OS**: %s, **Editor**: %s, **ContainerMode**: %s\n", r.OS, r.Editor, r.ContainerMode)
		fmt.Fprintf(w, "\n")
	}

	// Steps
	if len(r.Steps) > 0 {
		fmt.Fprintf(w, "## Steps\n")
		for i, step := range r.Steps {
			icon := "✅"
			if !step.Success {
				icon = "❌"
			}
			if step.Record != nil {
				fmt.Fprintf(w, "%d. %s %s: `%s`\n", i+1, icon, step.Name, step.Record.Command)
			} else {
				fmt.Fprintf(w, "%d. %s %s\n", i+1, icon, step.Name)
			}
		}
		fmt.Fprintf(w, "\n")
	}

	// Result
	fmt.Fprintf(w, "## Result: **%s**\n", r.OverallResult)
}

// WriteMarkdown writes the report as a Markdown file.
func (r *Report) WriteMarkdown(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create report file: %w", err)
	}
	defer f.Close()
	r.Print(f)
	return nil
}
