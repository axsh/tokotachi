package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestLiveGitLab_ListUpdate_DetectsOpenMR(t *testing.T) {
	baseURL, token, _ := requireLiveGitLab(t)
	repoDir, branch, _ := setupLiveRepo(t, baseURL, token)

	// Create an open MR via non-interactive glab (not via tt pr).
	runGlabNI(t, repoDir, token, "mr", "create", "--yes", "--fill", "--source-branch", branch, "--target-branch", "main")
	iid := mrIIDForBranch(t, repoDir, token, branch)
	require.Greater(t, iid, 0, "expected open MR for branch %s", branch)

	stdout, stderr, code := runTTInDirEnv(t, repoDir, liveEnv(token, glabNIPath(t)), "list", "--update", "--json")
	require.Equal(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)

	status := readCodeStatusFromState(t, repoDir, branch)
	assert.Equal(t, "pr", status, "list stdout=%s stderr=%s", stdout, stderr)

	// Cross-check JSON output mentions the branch.
	assert.Contains(t, stdout, branch)
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
