package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTtDoctor_GitLabRemote_ChecksGlab(t *testing.T) {
	dir := setupForgeDoctorRepo(t, "https://gitlab.com/group/proj.git")
	stdout, stderr, code := runTTInDirEnv(t, dir, nil, "doctor", "--json")
	require.Equal(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)

	names := doctorToolNames(t, stdout, stderr)
	assert.Contains(t, names, "glab")
	assert.NotContains(t, names, "gh")
}

func TestTtDoctor_GitHubRemote_ChecksGh(t *testing.T) {
	dir := setupForgeDoctorRepo(t, "https://github.com/axsh/tokotachi.git")
	stdout, stderr, code := runTTInDirEnv(t, dir, nil, "doctor", "--json")
	require.Equal(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)

	names := doctorToolNames(t, stdout, stderr)
	assert.Contains(t, names, "gh")
	assert.NotContains(t, names, "glab")
}

func TestTtDoctor_TTForgeGitlab_OverridesNonGitlabURL(t *testing.T) {
	dir := setupForgeDoctorRepo(t, "https://git.example.com/g/p.git")
	stdout, stderr, code := runTTInDirEnv(t, dir, []string{"TT_FORGE=gitlab"}, "doctor", "--json")
	require.Equal(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)

	names := doctorToolNames(t, stdout, stderr)
	assert.Contains(t, names, "glab")
	assert.NotContains(t, names, "gh")
}

func TestTtPr_Help_MentionsForgeFlagAndPriority(t *testing.T) {
	stdout, stderr, code := runTT(t, "pr", "--help")
	require.Equal(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)
	assertForgeHelpContract(t, stdout+stderr)
}

func TestTtList_Help_MentionsForgeFlagAndPriority(t *testing.T) {
	stdout, stderr, code := runTT(t, "list", "--help")
	require.Equal(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)
	assertForgeHelpContract(t, stdout+stderr)
}

func TestTtDoctor_Help_MentionsForgeFlagAndPriority(t *testing.T) {
	stdout, stderr, code := runTT(t, "doctor", "--help")
	require.Equal(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)
	assertForgeHelpContract(t, stdout+stderr)
}

func TestTtDoctor_ForgeFlag_OverridesGitHubURL(t *testing.T) {
	dir := setupForgeDoctorRepo(t, "https://github.com/axsh/tokotachi.git")
	stdout, stderr, code := runTTInDirEnv(t, dir, nil, "doctor", "--forge", "gitlab", "--json")
	require.Equal(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)

	names := doctorToolNames(t, stdout, stderr)
	assert.Contains(t, names, "glab")
	assert.NotContains(t, names, "gh")
}

func TestTtDoctor_ForgeFlag_BeatsTTForge(t *testing.T) {
	dir := setupForgeDoctorRepo(t, "https://gitlab.com/g/p.git")
	stdout, stderr, code := runTTInDirEnv(t, dir, []string{"TT_FORGE=gitlab"}, "doctor", "--forge", "github", "--json")
	require.Equal(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)

	names := doctorToolNames(t, stdout, stderr)
	assert.Contains(t, names, "gh")
	assert.NotContains(t, names, "glab")
}

func TestTtDoctor_ForgeFlag_Invalid_FailsFast(t *testing.T) {
	dir := setupForgeDoctorRepo(t, "https://github.com/a/b.git")
	stdout, stderr, code := runTTInDirEnv(t, dir, nil, "doctor", "--forge", "bogus", "--json")
	require.NotEqual(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)
	combined := stdout + stderr
	assert.True(t,
		strings.Contains(combined, "--forge") || strings.Contains(combined, "forge") || strings.Contains(combined, "invalid"),
		"expected invalid forge error, got: %s", combined)
}

func TestTtDoctor_ForgeFlag_OverridesLocalhostURL(t *testing.T) {
	dir := setupForgeDoctorRepo(t, "http://localhost:8929/group/proj.git")
	stdout, stderr, code := runTTInDirEnv(t, dir, nil, "doctor", "--forge", "gitlab", "--json")
	require.Equal(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)

	names := doctorToolNames(t, stdout, stderr)
	assert.Contains(t, names, "glab")
	assert.NotContains(t, names, "gh")
}

func TestTtList_ForgeFlag_Invalid_FailsFastWithoutUpdate(t *testing.T) {
	dir := setupForgeDoctorRepo(t, "https://github.com/a/b.git")
	stdout, stderr, code := runTTInDirEnv(t, dir, nil, "list", "--forge", "bogus")
	require.NotEqual(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)
	combined := stdout + stderr
	assert.Contains(t, combined, "--forge")
}

func TestTtPr_ForgeFlag_DryRun_SelectsGitLabHost(t *testing.T) {
	dir, branch := setupForgePRRepo(t, "https://github.com/axsh/tokotachi.git")
	stdout, stderr, code := runTTInDirEnv(t, dir, []string{"TT_FORGE=github"},
		"pr", "--forge", "gitlab", "--dry-run", branch)
	combined := stdout + stderr
	require.Equal(t, 0, code, "stdout=%s stderr=%s", stdout, stderr)
	assert.Contains(t, combined, "Detected forge host: gitlab")
}

func assertForgeHelpContract(t *testing.T, help string) {
	t.Helper()
	assert.Contains(t, help, "--forge")
	assert.Contains(t, help, "TT_FORGE")
	assert.Contains(t, help, "github")
	assert.Contains(t, help, "gitlab")
	assert.Contains(t, help, "TT_CMD_GH")
	assert.Contains(t, help, "TT_CMD_GLAB")
	// Priority: --forge before TT_FORGE in the help text
	forgeIdx := strings.Index(help, "--forge")
	envIdx := strings.Index(help, "TT_FORGE")
	require.GreaterOrEqual(t, forgeIdx, 0)
	require.GreaterOrEqual(t, envIdx, 0)
	assert.Less(t, forgeIdx, envIdx, "help should list --forge before TT_FORGE")
}

func setupForgeDoctorRepo(t *testing.T, originURL string) string {
	t.Helper()
	dir := t.TempDir()
	runGitIn(t, dir, "init")
	runGitIn(t, dir, "remote", "add", "origin", originURL)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "features", "demo"), 0o755))
	return dir
}

func doctorToolNames(t *testing.T, stdout, stderr string) []string {
	t.Helper()
	raw := strings.TrimSpace(stdout)
	if raw == "" {
		raw = strings.TrimSpace(stderr)
	}
	var parsed struct {
		Results []struct {
			Category string `json:"category"`
			Name     string `json:"name"`
		} `json:"results"`
	}
	require.NoError(t, json.Unmarshal([]byte(raw), &parsed), "raw=%s", raw)
	var names []string
	for _, r := range parsed.Results {
		if r.Category == "External Tools" {
			names = append(names, r.Name)
		}
	}
	return names
}

func runGitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
}

func setupForgePRRepo(t *testing.T, originURL string) (repoDir, branch string) {
	t.Helper()
	repoDir = t.TempDir()
	branch = "feat-forge-flag"

	runGitIn(t, repoDir, "init", "-b", "main")
	runGitIn(t, repoDir, "config", "user.email", "test@example.com")
	runGitIn(t, repoDir, "config", "user.name", "test")
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("forge\n"), 0o644))
	runGitIn(t, repoDir, "add", "README.md")
	runGitIn(t, repoDir, "commit", "-m", "init")
	runGitIn(t, repoDir, "remote", "add", "origin", originURL)

	worktreePath := filepath.Join(repoDir, "work", branch)
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, "work"), 0o755))
	runGitIn(t, repoDir, "branch", branch)
	runGitIn(t, repoDir, "worktree", "add", worktreePath, branch)
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, "features", "demo"), 0o755))
	return repoDir, branch
}

func runTTInDirEnv(t *testing.T, dir string, extraEnv []string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	binary := ttBinary(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), extraEnv...)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	stdout = outBuf.String()
	stderr = errBuf.String()
	if err != nil {
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		} else {
			exitCode = -1
		}
	}
	return stdout, stderr, exitCode
}
