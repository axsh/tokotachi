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
