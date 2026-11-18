package integration_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLiveGitLab_PR_CreatesMR(t *testing.T) {
	baseURL, token, _ := requireLiveGitLab(t)
	repoDir, branch, _ := setupLiveRepo(t, baseURL, token)

	require.Equal(t, 0, mrIIDForBranch(t, repoDir, token, branch), "branch should not already have an MR")

	stdout, stderr, code := runTTInDirEnv(t, repoDir, liveEnv(token, glabNIPath(t)), "pr", branch, featureDemoName)
	require.Equal(t, 0, code, "tt pr failed: stdout=%s stderr=%s", stdout, stderr)

	iid := mrIIDForBranch(t, repoDir, token, branch)
	require.Greater(t, iid, 0, "expected MR after tt pr; stdout=%s stderr=%s", stdout, stderr)
}
