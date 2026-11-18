package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	envLiveGitLab   = "TT_LIVE_GITLAB"
	defaultHTTPURL  = "http://localhost:8929"
	defaultProject  = "root/tokotachi-live"
	tokenRelPath    = "tmp/gitlab-live/token"
	readinessPath   = "/-/readiness"
	featureDemoName = "demo"
)

func projectRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to get caller information")
	}
	root, err := filepath.Abs(filepath.Join(filepath.Dir(filename), "..", ".."))
	if err != nil {
		panic(fmt.Sprintf("failed to resolve project root: %v", err))
	}
	return root
}

func ttBinary(t *testing.T) string {
	t.Helper()
	exePath := filepath.Join(projectRoot(), "bin", "tt.exe")
	noExtPath := filepath.Join(projectRoot(), "bin", "tt")
	if _, err := os.Stat(exePath); err == nil {
		return exePath
	}
	if _, err := os.Stat(noExtPath); err == nil {
		if runtime.GOOS == "windows" {
			data, err := os.ReadFile(noExtPath)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(exePath, data, 0o755))
			return exePath
		}
		return noExtPath
	}
	t.Fatalf("tt binary not found at %s or %s. Run ./scripts/process/build.sh first.", exePath, noExtPath)
	return ""
}

func glabNIPath(t *testing.T) string {
	t.Helper()
	name := "gitlab-glab-ni.sh"
	if runtime.GOOS == "windows" {
		name = "gitlab-glab-ni.cmd"
	}
	p := filepath.Join(projectRoot(), "scripts", "dev", name)
	_, err := os.Stat(p)
	require.NoError(t, err, "missing %s", p)
	return p
}

func requireLiveGitLab(t *testing.T) (baseURL, token, tokenFile string) {
	t.Helper()
	if os.Getenv(envLiveGitLab) != "1" {
		t.Fatalf("%s=1 is required. Run: TT_LIVE_GITLAB=1 ./scripts/dev/gitlab-live.sh smoke", envLiveGitLab)
	}

	tokenFile = filepath.Join(projectRoot(), filepath.FromSlash(tokenRelPath))
	raw, err := os.ReadFile(tokenFile)
	if err != nil || len(strings.TrimSpace(string(raw))) == 0 {
		t.Fatalf("token file missing or empty at %s; run ./scripts/dev/gitlab-live.sh seed", tokenFile)
	}
	token = strings.TrimSpace(string(raw))

	baseURL = strings.TrimRight(os.Getenv("GITLAB_HTTP_URL"), "/")
	if baseURL == "" {
		baseURL = strings.TrimRight(os.Getenv("GITLAB_HOST"), "/")
	}
	if baseURL == "" {
		baseURL = defaultHTTPURL
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(baseURL + readinessPath)
	if err != nil {
		t.Fatalf("GitLab readiness check failed for %s: %v (run gitlab-live.sh up && wait-ready)", baseURL, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("GitLab readiness returned HTTP %d for %s", resp.StatusCode, baseURL+readinessPath)
	}

	realGlab := os.Getenv("TT_CMD_GLAB_REAL")
	if realGlab == "" {
		realGlab = "glab"
	}
	if _, err := exec.LookPath(realGlab); err != nil {
		t.Fatalf("%s not found on PATH: %v", realGlab, err)
	}
	return baseURL, token, tokenFile
}

func liveEnv(token, glabPath string) []string {
	return []string{
		"TT_FORGE=gitlab",
		"GITLAB_TOKEN=" + token,
		"GITLAB_HOST=" + defaultHTTPURL,
		"TT_CMD_GLAB=" + glabPath,
		"GIT_TERMINAL_PROMPT=0",
		"GLAB_FORCE_TTY=0",
	}
}

func runTTInDirEnv(t *testing.T, dir string, extraEnv []string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	binary := ttBinary(t)
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
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

func runGitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
	return string(out)
}

func runGlabNI(t *testing.T, dir, token string, args ...string) string {
	t.Helper()
	bash := resolveBash(t)
	ni := glabNIPath(t)
	cmd := exec.Command(bash, append([]string{ni}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GITLAB_TOKEN="+token,
		"GITLAB_HOST="+defaultHTTPURL,
		"GIT_TERMINAL_PROMPT=0",
	)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "glab-ni %v: %s", args, out)
	return string(out)
}

func resolveBash(t *testing.T) string {
	t.Helper()
	if p, err := exec.LookPath("bash"); err == nil {
		return p
	}
	if runtime.GOOS == "windows" {
		for _, c := range []string{
			`C:\Program Files\Git\bin\bash.exe`,
			`C:\Program Files\Git\usr\bin\bash.exe`,
		} {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}
	t.Fatalf("bash not found (needed to invoke scripts/dev/gitlab-glab-ni.sh)")
	return ""
}

// setupLiveRepo clones the seeded project and creates a pushed feature branch + worktree layout for tt.
func setupLiveRepo(t *testing.T, baseURL, token string) (repoDir, branch, worktreePath string) {
	t.Helper()
	repoDir = t.TempDir()
	branch = fmt.Sprintf("live-gitlab-%d", time.Now().UnixNano())

	cloneURL := fmt.Sprintf("%s/%s.git", strings.TrimRight(baseURL, "/"), defaultProject)
	authURL := strings.Replace(cloneURL, "://", "://oauth2:"+token+"@", 1)

	runGitIn(t, repoDir, "clone", authURL, ".")
	runGitIn(t, repoDir, "config", "user.email", "tokotachi-live@example.com")
	runGitIn(t, repoDir, "config", "user.name", "tokotachi-live")
	// Keep origin without embedding token in saved remote for logs; re-set with token for push.
	runGitIn(t, repoDir, "remote", "set-url", "origin", authURL)

	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, "features", featureDemoName), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(repoDir, "features", featureDemoName, "README.md"),
		[]byte("# demo\n"), 0o644,
	))

	worktreePath = filepath.Join(repoDir, "work", branch)
	runGitIn(t, repoDir, "worktree", "add", "-b", branch, worktreePath)
	require.NoError(t, os.WriteFile(filepath.Join(worktreePath, "live.txt"), []byte(branch+"\n"), 0o644))
	runGitIn(t, worktreePath, "add", "live.txt")
	runGitIn(t, worktreePath, "commit", "-m", "live gitlab test "+branch)
	runGitIn(t, worktreePath, "push", "-u", "origin", branch)

	return repoDir, branch, worktreePath
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

func mrIIDForBranch(t *testing.T, repoDir, token, branch string) int {
	t.Helper()
	out := runGlabNI(t, repoDir, token, "mr", "list", "--source-branch", branch, "--output", "json")
	out = strings.TrimSpace(out)
	if out == "" || out == "[]" {
		return 0
	}
	var rows []struct {
		IID int `json:"iid"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &rows), "mr list json=%s", out)
	if len(rows) == 0 {
		return 0
	}
	return rows[0].IID
}
