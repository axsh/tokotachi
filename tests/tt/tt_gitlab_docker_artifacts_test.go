package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitLabDockerArtifacts_Exist(t *testing.T) {
	root := projectRoot()
	required := []string{
		"docker/gitlab/docker-compose.yml",
		"docker/gitlab/README.md",
		"docker/gitlab/.env.example",
		"docker/gitlab/.gitignore",
		"scripts/dev/gitlab-live.sh",
		"scripts/dev/gitlab-glab-ni.sh",
		"scripts/dev/gitlab-glab-ni.cmd",
		"tests/live-gitlab/.opt-in-only",
	}
	for _, rel := range required {
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		require.NoError(t, err, "missing required artifact: %s", rel)
	}

	compose, err := os.ReadFile(filepath.Join(root, "docker", "gitlab", "docker-compose.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(compose), "gitlab/gitlab-ce:",
		"compose must pin an official GitLab CE image tag")
}

func TestGitLabDockerScripts_Help(t *testing.T) {
	root := projectRoot()
	bash := resolveBash(t)
	script := filepath.Join(root, "scripts", "dev", "gitlab-live.sh")

	for _, args := range [][]string{{"--help"}, {"help"}} {
		cmd := exec.Command(bash, append([]string{script}, args...)...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "gitlab-live.sh %v failed: %s", args, out)
		combined := string(out)
		assert.Contains(t, combined, "Usage", "help output should include Usage: %s", combined)
	}
}

func TestGitLabLiveCategory_IsOptInOnly(t *testing.T) {
	path := filepath.Join(projectRoot(), "tests", "live-gitlab", ".opt-in-only")
	_, err := os.Stat(path)
	require.NoError(t, err, "tests/live-gitlab/.opt-in-only must exist so discover_categories skips it")
}

func resolveBash(t *testing.T) string {
	t.Helper()
	if p, err := exec.LookPath("bash"); err == nil {
		return p
	}
	if runtime.GOOS == "windows" {
		candidates := []string{
			`C:\Program Files\Git\bin\bash.exe`,
			`C:\Program Files\Git\usr\bin\bash.exe`,
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}
	t.Fatalf("bash not found; install Git Bash or ensure bash is on PATH (needed to run scripts/dev/gitlab-live.sh --help)")
	return ""
}
