package action

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/axsh/tokotachi/internal/cmdexec"
	pkglog "github.com/axsh/tokotachi/pkg/log"
)

const maxDisplayLines = 10

// PendingChanges holds categorized pending changes in a worktree.
type PendingChanges struct {
	UntrackedFiles  []string // git ls-files --others --exclude-standard
	UnstagedChanges []string // git diff --name-only
	StagedChanges   []string // git diff --cached --name-only
	UnpushedCommits []string // git log @{upstream}..HEAD --oneline
}

// TotalCount returns total number of pending items across all categories.
func (p PendingChanges) TotalCount() int {
	return len(p.UntrackedFiles) + len(p.UnstagedChanges) +
		len(p.StagedChanges) + len(p.UnpushedCommits)
}

// ParseLinesFromOutput splits git command output into non-empty trimmed lines.
func ParseLinesFromOutput(output string) []string {
	if output == "" {
		return nil
	}
	raw := strings.Split(output, "\n")
	var result []string
	for _, line := range raw {
		trimmed := strings.TrimRight(line, "\r")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// FormatCategory formats a single category's items with optional truncation.
// Returns formatted lines as a string slice.
func FormatCategory(header string, items []string, verbose bool) []string {
	headerLine := fmt.Sprintf("%s (%d):", header, len(items))
	lines := []string{headerLine}

	if len(items) == 0 {
		lines = append(lines, "  (none)")
		return lines
	}

	limit := len(items)
	if !verbose && limit > maxDisplayLines {
		limit = maxDisplayLines
	}

	for i := range limit {
		lines = append(lines, "  "+items[i])
	}

	if !verbose && len(items) > maxDisplayLines {
		remaining := len(items) - maxDisplayLines
		lines = append(lines, fmt.Sprintf("  ... and %d more (%d total)", remaining, len(items)))
	}

	return lines
}

// collectPendingChanges runs git commands in the worktree directory
// to detect all 4 categories of pending changes.
func collectPendingChanges(cmdRunner *cmdexec.Runner, worktreePath string) PendingChanges {
	gitCmd := cmdexec.ResolveCommand("TT_CMD_GIT", "git")
	opts := cmdexec.RunOption{
		Dir:          worktreePath,
		QuietCmd:     true,
		FailLevelSet: true,
		FailLevel:    pkglog.LevelDebug,
		FailLabel:    "SKIP",
	}

	var changes PendingChanges

	// Untracked files
	if out, err := cmdRunner.RunWithOpts(opts, gitCmd, "ls-files", "--others", "--exclude-standard"); err == nil {
		changes.UntrackedFiles = ParseLinesFromOutput(out)
	}

	// Unstaged changes
	if out, err := cmdRunner.RunWithOpts(opts, gitCmd, "diff", "--name-only"); err == nil {
		changes.UnstagedChanges = ParseLinesFromOutput(out)
	}

	// Staged but uncommitted
	if out, err := cmdRunner.RunWithOpts(opts, gitCmd, "diff", "--cached", "--name-only"); err == nil {
		changes.StagedChanges = ParseLinesFromOutput(out)
	}

	// Unpushed commits (may fail if no upstream is set)
	if out, err := cmdRunner.RunWithOpts(opts, gitCmd, "log", "@{upstream}..HEAD", "--oneline"); err == nil {
		changes.UnpushedCommits = ParseLinesFromOutput(out)
	}

	return changes
}

// displayPendingChanges formats and prints pending changes via logger.
func displayPendingChanges(logger pkglog.Logger, changes PendingChanges, verbose bool) {
	logger.Info("== Pending changes in worktree ==")

	categories := []struct {
		header string
		items  []string
	}{
		{"Untracked files", changes.UntrackedFiles},
		{"Unstaged changes", changes.UnstagedChanges},
		{"Staged changes", changes.StagedChanges},
		{"Unpushed commits", changes.UnpushedCommits},
	}

	for _, cat := range categories {
		for _, line := range FormatCategory(cat.header, cat.items, verbose) {
			logger.Info("%s", line)
		}
	}
}

// checkPendingChangesAndConfirm checks for pending changes and prompts
// for confirmation. Returns true if the user confirms or there are no
// pending changes or --yes is set. Returns false if user aborts.
func (r *Runner) checkPendingChangesAndConfirm(opts CloseOptions, worktreePath string) bool {
	if opts.Yes {
		return true
	}

	changes := collectPendingChanges(r.CmdRunner, worktreePath)
	total := changes.TotalCount()
	if total == 0 {
		return true
	}

	displayPendingChanges(r.Logger, changes, opts.Verbose)

	fmt.Fprintf(os.Stderr, "WARNING: Found %d pending change(s) in worktree. Are you sure you want to delete? [y/N]: ", total)

	if opts.Stdin == nil {
		r.Logger.Info("Aborted (no input source for confirmation).")
		return false
	}

	scanner := bufio.NewScanner(opts.Stdin)
	if scanner.Scan() {
		response := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if response == "y" || response == "yes" {
			return true
		}
	}
	return false
}
