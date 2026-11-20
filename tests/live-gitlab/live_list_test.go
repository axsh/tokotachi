package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestLiveGitLab_ListUpdate_DetectsOpenMR(t *testing.T) {
	baseURL, token, _ := requireLiveGitLab(t)
	repoDir, branch, _ := setupLiveRepo(t, baseURL, token)

	defaultBranch := strings.TrimSpace(runGitIn(t, repoDir, "symbolic-ref", "refs/remotes/origin/HEAD"))
	defaultBranch = strings.TrimPrefix(defaultBranch, "refs/remotes/origin/")
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	// Create MR via API (avoid glab --fill re-pushing from a dirty main worktree).
	createMergeRequestAPI(t, baseURL, token, branch, defaultBranch)
	iid := mrIIDForBranch(t, repoDir, token, branch)
	require.Greater(t, iid, 0, "expected open MR for branch %s", branch)

	// tt list --update only updates branches that already have a state file.
	seedMinimalStateFile(t, repoDir, branch)

	stdout, stderr, code := runTTInDirEnv(t, repoDir, liveEnv(t, token, glabNIPath(t)), "list", "--update", "--json")
	require.Equal(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)

	status := readCodeStatusFromState(t, repoDir, branch)
	assert.Equal(t, "pr", status, "list stdout=%s stderr=%s", stdout, stderr)

	// Cross-check JSON output mentions the branch.
	assert.Contains(t, stdout, branch)
}

func createMergeRequestAPI(t *testing.T, baseURL, token, source, target string) {
	t.Helper()
	form := url.Values{}
	form.Set("source_branch", source)
	form.Set("target_branch", target)
	form.Set("title", "live list "+source)
	form.Set("remove_source_branch", "false")

	api := fmt.Sprintf("%s/api/v4/projects/%s/merge_requests",
		strings.TrimRight(baseURL, "/"),
		url.PathEscape(defaultProject),
	)
	req, err := http.NewRequest(http.MethodPost, api, bytes.NewBufferString(form.Encode()))
	require.NoError(t, err)
	req.Header.Set("PRIVATE-TOKEN", token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	require.True(t, resp.StatusCode == 201 || resp.StatusCode == 200,
		"create MR HTTP %d: %s", resp.StatusCode, body)

	var parsed struct {
		IID int `json:"iid"`
	}
	require.NoError(t, json.Unmarshal(body, &parsed))
	require.Greater(t, parsed.IID, 0)
}

func seedMinimalStateFile(t *testing.T, repoDir, branch string) {
	t.Helper()
	path := filepath.Join(repoDir, "work", branch+".state.yaml")
	content := fmt.Sprintf("branch: %s\ncreated_at: %s\n", branch, time.Now().UTC().Format(time.RFC3339))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func readCodeStatusFromState(t *testing.T, repoDir, branch string) string {
	t.Helper()
	path := filepath.Join(repoDir, "work", branch+".state.yaml")
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "expected state file after tt list --update")
	var sf struct {
		CodeStatus *struct {
			Status string `yaml:"status"`
		} `yaml:"code_status"`
	}
	require.NoError(t, yaml.Unmarshal(raw, &sf), "state=%s", raw)
	require.NotNil(t, sf.CodeStatus, "code_status missing in %s:\n%s", path, raw)
	return strings.TrimSpace(sf.CodeStatus.Status)
}
