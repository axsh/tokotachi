package action_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPR_GitHub_UsesGhPrCreate(t *testing.T) {
	env := newTestEnv(t)
	repo := env.RepoRoot
	runGitPR(t, repo, "init")
	runGitPR(t, repo, "remote", "add", "origin", "https://github.com/axsh/tokotachi.git")
	t.Setenv("TT_FORGE", "")

	work := t.TempDir()
	err := env.Runner.PR(repo, work)
	require.NoError(t, err)

	assert.True(t, recorderHas(env, "gh pr create"), "records=%#v", env.Recorder.Records())
}

func TestPR_GitLab_UsesGlabMrCreate(t *testing.T) {
	env := newTestEnv(t)
	repo := env.RepoRoot
	runGitPR(t, repo, "init")
	runGitPR(t, repo, "remote", "add", "origin", "https://git.example.com/g/p.git")
	t.Setenv("TT_FORGE", "gitlab")

	work := t.TempDir()
	err := env.Runner.PR(repo, work)
	require.NoError(t, err)

	assert.True(t, recorderHas(env, "glab mr create"), "records=%#v", env.Recorder.Records())
}

func recorderHas(env *testEnv, needle string) bool {
	for _, rec := range env.Recorder.Records() {
		if strings.Contains(rec.Command, needle) {
			return true
		}
	}
	return false
}

func runGitPR(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
}
