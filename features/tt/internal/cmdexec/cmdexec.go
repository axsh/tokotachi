package cmdexec

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/axsh/tokotachi/features/tt/internal/log"
)

// ExecRecord stores one command execution history.
type ExecRecord struct {
	Command  string // full command line
	Success  bool
	ExitCode int
	Duration time.Duration
	DryRun   bool
}

// Recorder collects ExecRecords during a session.
type Recorder struct {
	mu      sync.Mutex
	records []ExecRecord
}

// NewRecorder creates a new Recorder.
func NewRecorder() *Recorder {
	return &Recorder{}
}

// Records returns all recorded execution records.
func (r *Recorder) Records() []ExecRecord {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := make([]ExecRecord, len(r.records))
	copy(cp, r.records)
	return cp
}

// Add appends a record.
func (r *Recorder) Add(rec ExecRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records = append(r.records, rec)
}

// RunOption controls logging behavior for command execution.
type RunOption struct {
	FailLevel    log.Level // Log level on failure
	FailLevelSet bool      // If true, use FailLevel; if false, default to LevelError
	FailLabel    string    // Label tag on failure (default: "FAIL")
	QuietCmd     bool      // If true, [CMD] log uses LevelDebug instead of LevelInfo
	Dir          string    // Working directory for command execution (empty = inherit process cwd)
}

// effectiveFailLevel returns the log level to use on failure.
func (o RunOption) effectiveFailLevel() log.Level {
	if o.FailLevelSet {
		return o.FailLevel
	}
	return log.LevelError
}

// effectiveFailLabel returns the label tag to use on failure.
func (o RunOption) effectiveFailLabel() string {
	if o.FailLabel != "" {
		return o.FailLabel
	}
	return "FAIL"
}

// CheckOpt returns a RunOption for condition-check commands.
// Failures are logged at DEBUG level with [SKIP] label.
// Command execution itself is also logged at DEBUG level.
func CheckOpt() RunOption {
	return RunOption{FailLevel: log.LevelDebug, FailLevelSet: true, FailLabel: "SKIP", QuietCmd: true}
}

// ToleratedOpt returns a RunOption for tolerated-failure commands.
// Failures are logged at WARN level.
func ToleratedOpt() RunOption {
	return RunOption{FailLevel: log.LevelWarn, FailLevelSet: true, FailLabel: "FAIL", QuietCmd: false}
}

// Runner executes external commands with logging and recording.
type Runner struct {
	Logger   *log.Logger
	DryRun   bool
	Recorder *Recorder
}

// Run executes the command, logs it, and records the result.
// Returns stdout as string. Stderr is forwarded to os.Stderr.
// Failures are logged at ERROR level.
func (r *Runner) Run(name string, args ...string) (string, error) {
	return r.RunWithOpts(RunOption{}, name, args...)
}

// RunWithOpts executes the command with configurable failure logging.
func (r *Runner) RunWithOpts(opts RunOption, name string, args ...string) (string, error) {
	cmdLine := formatCmdLine(name, args)
	start := time.Now()

	if r.DryRun {
		if opts.Dir != "" {
			r.Logger.Info("[DRY-RUN] (in %s) %s", opts.Dir, cmdLine)
		} else {
			r.Logger.Info("[DRY-RUN] %s", cmdLine)
		}
		r.Recorder.Add(ExecRecord{
			Command:  cmdLine,
			Success:  true,
			Duration: time.Since(start),
			DryRun:   true,
		})
		return "", nil
	}

	if opts.QuietCmd {
		r.Logger.Debug("[CMD] %s", cmdLine)
	} else {
		r.Logger.Info("[CMD] %s", cmdLine)
	}
	cmd := exec.Command(name, args...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	if !opts.QuietCmd {
		cmd.Stderr = os.Stderr
	}
	out, err := cmd.Output()

	exitCode := 0
	success := true
	if err != nil {
		success = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	rec := ExecRecord{
		Command:  cmdLine,
		Success:  success,
		ExitCode: exitCode,
		Duration: time.Since(start),
	}
	r.Recorder.Add(rec)

	if !success {
		failLevel := opts.effectiveFailLevel()
		failLabel := opts.effectiveFailLabel()
		r.Logger.Log(failLevel, "[%s] %s (exit=%d)", failLabel, cmdLine, exitCode)
		return string(out), fmt.Errorf("%s: %w", cmdLine, err)
	}

	r.Logger.Debug("[OK] %s (%.1fs)", cmdLine, rec.Duration.Seconds())
	return strings.TrimRight(string(out), "\n\r"), nil
}

// RunInteractive executes the command with stdin/stdout/stderr attached.
// Used for shell, exec, and editor launch.
// Failures are logged at ERROR level.
func (r *Runner) RunInteractive(name string, args ...string) error {
	return r.RunInteractiveWithOpts(RunOption{}, name, args...)
}

// RunInteractiveWithOpts executes an interactive command with configurable failure logging.
func (r *Runner) RunInteractiveWithOpts(opts RunOption, name string, args ...string) error {
	cmdLine := formatCmdLine(name, args)
	start := time.Now()

	if r.DryRun {
		if opts.Dir != "" {
			r.Logger.Info("[DRY-RUN] (in %s) %s", opts.Dir, cmdLine)
		} else {
			r.Logger.Info("[DRY-RUN] %s", cmdLine)
		}
		r.Recorder.Add(ExecRecord{
			Command:  cmdLine,
			Success:  true,
			Duration: time.Since(start),
			DryRun:   true,
		})
		return nil
	}

	if opts.QuietCmd {
		r.Logger.Debug("[CMD] %s", cmdLine)
	} else {
		r.Logger.Info("[CMD] %s", cmdLine)
	}
	cmd := exec.Command(name, args...)
	if opts.Dir != "" {
		cmd.Dir = opts.Dir
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	exitCode := 0
	success := true
	if err != nil {
		success = false
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	rec := ExecRecord{
		Command:  cmdLine,
		Success:  success,
		ExitCode: exitCode,
		Duration: time.Since(start),
	}
	r.Recorder.Add(rec)

	if !success {
		failLevel := opts.effectiveFailLevel()
		failLabel := opts.effectiveFailLabel()
		r.Logger.Log(failLevel, "[%s] %s (exit=%d)", failLabel, cmdLine, exitCode)
		return fmt.Errorf("%s: %w", cmdLine, err)
	}
	return nil
}

// ResolveCommand returns the env var value or the default command.
func ResolveCommand(envKey, defaultCmd string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return defaultCmd
}

func formatCmdLine(name string, args []string) string {
	if len(args) == 0 {
		return name
	}
	return name + " " + strings.Join(args, " ")
}
