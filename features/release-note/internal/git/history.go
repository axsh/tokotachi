package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

var (
	// Pattern: Merge branch 'branch-name' [into ...]
	mergeBranchRe = regexp.MustCompile(`Merge branch '([^']+)'`)
	// Pattern: Merge pull request #N from user/branch-name
	mergePRRe = regexp.MustCompile(`Merge pull request #\d+ from [^/]+/(.+)`)
)

// Collector gathers Git history information.
type Collector struct {
	repoRoot string
}

// NewCollector creates a new Collector for the given repository root.
func NewCollector(repoRoot string) *Collector {
	return &Collector{repoRoot: repoRoot}
}

// GetLatestReleaseTag returns the latest release tag for a tool-id,
// or empty string if no release exists.
// Uses: gh release list --limit 100 --json tagName ...
func (c *Collector) GetLatestReleaseTag(toolID string) (string, error) {
	jqExpr := fmt.Sprintf(
		`[.[] | select(.tagName | startswith("%s-v"))] | sort_by(.tagName) | last | .tagName // empty`,
		toolID,
	)

	cmd := exec.Command("gh", "release", "list", "--limit", "100",
		"--json", "tagName", "--jq", jqExpr)
	cmd.Dir = c.repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get releases: %s: %w", stderr.String(), err)
	}

	tag := strings.TrimSpace(stdout.String())
	return tag, nil
}

// GetCommitSHA returns the commit SHA for a given tag.
func (c *Collector) GetCommitSHA(tag string) (string, error) {
	cmd := exec.Command("git", "rev-list", "-n", "1", tag)
	cmd.Dir = c.repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get commit SHA for tag %s: %s: %w", tag, stderr.String(), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetBranchNames extracts branch names from merge commits since a given commit.
// If sinceCommit is empty, gets all merge commits.
func (c *Collector) GetBranchNames(sinceCommit string) ([]string, error) {
	args := []string{"log", "--merges", "--oneline"}
	if sinceCommit != "" {
		args = append(args, sinceCommit+"..HEAD")
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = c.repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to get merge commits: %s: %w", stderr.String(), err)
	}

	var branches []string
	for _, line := range strings.Split(stdout.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		branch := ExtractBranchFromMessage(line)
		if branch != "" {
			branches = append(branches, branch)
		}
	}

	return DeduplicateBranches(branches), nil
}

// ExtractBranchFromMessage parses a merge commit message and returns
// the branch name, or empty string if not a merge commit.
func ExtractBranchFromMessage(message string) string {
	// Try merge branch pattern first
	if matches := mergeBranchRe.FindStringSubmatch(message); len(matches) > 1 {
		return matches[1]
	}

	// Try pull request pattern
	if matches := mergePRRe.FindStringSubmatch(message); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	return ""
}

// DeduplicateBranches removes duplicates from a branch name list
// and returns a sorted unique list.
func DeduplicateBranches(branches []string) []string {
	seen := make(map[string]struct{})
	var unique []string

	for _, b := range branches {
		if _, ok := seen[b]; !ok {
			seen[b] = struct{}{}
			unique = append(unique, b)
		}
	}

	sort.Strings(unique)
	return unique
}
