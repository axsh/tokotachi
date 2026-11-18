package track

import (
	"fmt"
	"os/exec"
	"strings"
)

// CorpusGitDirty reports whether `git status --porcelain -- <corpusPath>` is non-empty.
// corpusPath is relative to workspace (the registry value).
func CorpusGitDirty(workspace, corpusPath string) (bool, error) {
	cmd := exec.Command("git", "-C", workspace, "status", "--porcelain", "--", corpusPath)
	out, err := cmd.Output()
	if err != nil {
		var stderr string
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(ee.Stderr))
		}
		return false, fmt.Errorf("git status for %s failed: %w %s", corpusPath, err, stderr)
	}
	return strings.TrimSpace(string(out)) != "", nil
}
