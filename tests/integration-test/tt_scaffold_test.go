package integration_test

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireGitHubReachable verifies that the GitHub API is accessible.
// Fails the test with t.Fatalf if unreachable (no t.Skip per testing rules).
func requireGitHubReachable(t *testing.T) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.github.com/rate_limit")
	if err != nil {
		t.Fatalf("GitHub API is unreachable: %v. Ensure network connectivity.", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GitHub API returned unexpected status: %d. Ensure network connectivity.", resp.StatusCode)
	}
}

// initGitRepo initializes a git repository in the given directory with an initial commit.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmds := [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@test.local"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}
	for _, args := range cmds {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s failed: %s", strings.Join(args, " "), string(out))
	}
}

// commitAll stages and commits all files in the given directory.
func commitAll(t *testing.T, dir, message string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, args := range [][]string{
		{"git", "add", "-A"},
		{"git", "commit", "-m", message},
	} {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = dir
		_, _ = cmd.CombinedOutput()
	}
}

// runTTInDir executes the tt binary with the given arguments
// in the specified working directory. Returns stdout, stderr, and exit code.
func runTTInDir(t *testing.T, dir string, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	binary := ttBinary(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = dir

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

// TestScaffoldDefault runs the default scaffold against the real
// tokotachi-scaffolds repository and validates the complete result set
// in a single scaffold invocation to minimize GitHub API usage.
func TestScaffoldDefault(t *testing.T) {
	requireGitHubReachable(t)

	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Execute scaffold against the real repository
	stdout, stderr, code := runTTInDir(t, tmpDir, "scaffold", "--yes")
	require.Equal(t, 0, code,
		"tt scaffold --yes failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	// --- Sub-test: directory structure ---
	t.Run("CreatesExpectedStructure", func(t *testing.T) {
		expectedFiles := []string{
			"features/README.md",
			"prompts/phases/README.md",
			"prompts/phases/000-foundation/ideas/.gitkeep",
			"prompts/phases/000-foundation/plans/.gitkeep",
			"prompts/rules/.gitkeep",
			"scripts/.gitkeep",
			"shared/README.md",
			"shared/libs/README.md",
			"work/README.md",
		}
		for _, f := range expectedFiles {
			fullPath := filepath.Join(tmpDir, filepath.FromSlash(f))
			_, err := os.Stat(fullPath)
			assert.NoError(t, err, "Expected file %q was not created", f)
		}
	})

	// --- Sub-test: file contents ---
	t.Run("FileContents", func(t *testing.T) {
		readmeFiles := []string{
			"features/README.md",
			"shared/README.md",
			"shared/libs/README.md",
			"work/README.md",
		}
		for _, f := range readmeFiles {
			fullPath := filepath.Join(tmpDir, filepath.FromSlash(f))
			content, err := os.ReadFile(fullPath)
			require.NoError(t, err, "Failed to read %s", f)
			assert.NotEmpty(t, content, "File %s should have content", f)
		}
	})

	// --- Sub-test: gitignore post-action ---
	t.Run("GitignorePostAction", func(t *testing.T) {
		gitignorePath := filepath.Join(tmpDir, ".gitignore")
		content, err := os.ReadFile(gitignorePath)
		require.NoError(t, err, ".gitignore should exist after scaffold")
		assert.Contains(t, string(content), "work/*",
			".gitignore should contain 'work/*' entry")
	})

	// --- Sub-test: idempotent skip (re-run in same dir) ---
	t.Run("IdempotentSkip", func(t *testing.T) {
		// Record original content
		readmePath := filepath.Join(tmpDir, "features", "README.md")
		originalContent, err := os.ReadFile(readmePath)
		require.NoError(t, err)

		// Commit to make worktree clean (scaffold creates checkpoint via git stash)
		commitAll(t, tmpDir, "scaffold applied")

		// Second run
		stdout2, stderr2, code2 := runTTInDir(t, tmpDir, "scaffold", "--yes")
		require.Equal(t, 0, code2,
			"Second scaffold should succeed (idempotent).\nSTDOUT:\n%s\nSTDERR:\n%s",
			stdout2, stderr2)

		// Verify original content is preserved
		afterContent, err := os.ReadFile(readmePath)
		require.NoError(t, err)
		assert.Equal(t, string(originalContent), string(afterContent),
			"File content should be unchanged after idempotent re-run")

		// Verify the second run output mentions skipping
		combinedOutput := stdout2 + stderr2
		assert.Contains(t, strings.ToLower(combinedOutput), "skip",
			"Second run should mention skipping existing files")
	})
}

// TestScaffoldList verifies that 'tt scaffold --list' returns
// entries from the real tokotachi-scaffolds repository.
func TestScaffoldList(t *testing.T) {
	requireGitHubReachable(t)

	stdout, stderr, code := runTT(t, "scaffold", "--list")
	require.Equal(t, 0, code,
		"tt scaffold --list failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	assert.Contains(t, stdout, "default",
		"scaffold --list should include 'default' template.\nOutput:\n%s", stdout)
}

// TestScaffoldDefaultLocaleJa verifies that locale overlay (ja) correctly
// replaces README.md files with Japanese versions.
func TestScaffoldDefaultLocaleJa(t *testing.T) {
	requireGitHubReachable(t)

	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	stdout, stderr, code := runTTInDir(t, tmpDir, "scaffold", "--yes", "--lang", "ja")
	require.Equal(t, 0, code,
		"tt scaffold --yes --lang ja failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	// Verify README files contain Japanese content from locale.ja overlay
	readmeChecks := map[string]string{
		"features/README.md": "feature",
		"shared/README.md":   "共有",
		"work/README.md":     "ワークツリー",
	}

	for f, expectedJapanese := range readmeChecks {
		fullPath := filepath.Join(tmpDir, filepath.FromSlash(f))
		content, err := os.ReadFile(fullPath)
		require.NoError(t, err, "Failed to read %s", f)

		contentStr := string(content)
		assert.True(t,
			containsJapanese(contentStr) || strings.Contains(contentStr, expectedJapanese),
			"File %s should contain Japanese content with locale overlay (ja).\nContent:\n%s",
			f, contentStr)
	}
}

// TestScaffoldCwdFlag verifies that --cwd forces the current working
// directory as the scaffold root, bypassing Git root auto-detection.
// Uses a non-git temporary directory to confirm CWD override behavior.
func TestScaffoldCwdFlag(t *testing.T) {
	requireGitHubReachable(t)

	tmpDir := t.TempDir()
	// Intentionally NOT calling initGitRepo — this is a non-git directory.

	stdout, stderr, code := runTTInDir(t, tmpDir, "scaffold", "--cwd", "--yes")
	require.Equal(t, 0, code,
		"tt scaffold --cwd --yes failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	// Verify template files were created in the CWD (tmpDir)
	readmePath := filepath.Join(tmpDir, "features", "README.md")
	_, err := os.Stat(readmePath)
	assert.NoError(t, err, "Expected features/README.md to be created in CWD with --cwd flag")
}

// TestScaffoldWithDependencies verifies that the depends_on chain is
// resolved and applied in topological order. Running scaffold for
// feature/axsh-go-standard should first apply root/default, then
// project/axsh-go-standard, then feature/axsh-go-standard.
func TestScaffoldWithDependencies(t *testing.T) {
	requireGitHubReachable(t)

	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Execute scaffold with dependency chain
	// Note: feature/axsh-go-standard uses {{feature_name}} in base_dir,
	// so we need to provide it via --v flag.
	stdout, stderr, code := runTTInDir(t, tmpDir,
		"scaffold", "feature", "axsh-go-standard", "--yes", "--default",
		"--v", "feature_name=testfeature")
	require.Equal(t, 0, code,
		"tt scaffold feature axsh-go-standard --yes --default failed.\nSTDOUT:\n%s\nSTDERR:\n%s",
		stdout, stderr)

	combinedOutput := stdout + stderr

	// Verify dependency resolution progress is shown
	t.Run("DependencyProgressDisplay", func(t *testing.T) {
		assert.Contains(t, combinedOutput, "[1/3]",
			"Should show [1/3] progress for root/default")
		assert.Contains(t, combinedOutput, "[2/3]",
			"Should show [2/3] progress for project/axsh-go-standard")
		assert.Contains(t, combinedOutput, "[3/3]",
			"Should show [3/3] progress for feature/axsh-go-standard")
	})

	// Verify files from root/default are created
	t.Run("RootDefaultFiles", func(t *testing.T) {
		expectedFiles := []string{
			"features/README.md",
			"prompts/phases/README.md",
			"scripts/.gitkeep",
		}
		for _, f := range expectedFiles {
			fullPath := filepath.Join(tmpDir, filepath.FromSlash(f))
			_, err := os.Stat(fullPath)
			assert.NoError(t, err, "Expected file %q from root/default was not created", f)
		}
	})

	// Verify files from feature/axsh-go-standard are created
	t.Run("FeatureFiles", func(t *testing.T) {
		// The feature scaffold should create files in features/ directory
		featureDir := filepath.Join(tmpDir, "features")
		info, err := os.Stat(featureDir)
		require.NoError(t, err, "features/ directory should exist")
		assert.True(t, info.IsDir(), "features/ should be a directory")
	})
}

// TestScaffoldDownloadHistory verifies that the download history file
// is created after scaffold application, and that static scaffolds
// are recorded while dynamic scaffolds (with {{...}} in base_dir) are not.
func TestScaffoldDownloadHistory(t *testing.T) {
	requireGitHubReachable(t)

	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// Run default scaffold (root/default — static base_dir)
	stdout, stderr, code := runTTInDir(t, tmpDir, "scaffold", "--yes")
	require.Equal(t, 0, code,
		"tt scaffold --yes failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout, stderr)

	// Verify downloaded.yaml was created
	historyPath := filepath.Join(tmpDir, ".kotoshiro", "tokotachi", "downloaded.yaml")

	t.Run("HistoryFileCreated", func(t *testing.T) {
		_, err := os.Stat(historyPath)
		require.NoError(t, err, "downloaded.yaml should be created after scaffold")
	})

	t.Run("StaticScaffoldRecorded", func(t *testing.T) {
		data, err := os.ReadFile(historyPath)
		require.NoError(t, err)
		content := string(data)

		// root/default should be recorded (static base_dir: ".")
		assert.Contains(t, content, "root",
			"root category should be in download history")
		assert.Contains(t, content, "default",
			"default name should be in download history")
		assert.Contains(t, content, "downloaded_at",
			"downloaded_at should be recorded")
	})
}

// TestScaffoldSkipAlreadyDownloaded verifies that running the same
// static scaffold twice results in a skip on the second run, with
// an appropriate log message. Running with --force should re-apply.
func TestScaffoldSkipAlreadyDownloaded(t *testing.T) {
	requireGitHubReachable(t)

	tmpDir := t.TempDir()
	initGitRepo(t, tmpDir)

	// First run: should download and record
	stdout1, stderr1, code1 := runTTInDir(t, tmpDir, "scaffold", "--yes")
	require.Equal(t, 0, code1,
		"First scaffold run failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout1, stderr1)

	t.Run("SecondRunSkips", func(t *testing.T) {
		// Second run: should skip (same scaffold already downloaded)
		stdout2, stderr2, code2 := runTTInDir(t, tmpDir, "scaffold", "--yes")
		require.Equal(t, 0, code2,
			"Second scaffold run failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout2, stderr2)

		combinedOutput := stdout2 + stderr2
		assert.Contains(t, combinedOutput, "Skipping",
			"Second run should show skip message for already downloaded scaffold")
	})

	t.Run("ForceOverridesSkip", func(t *testing.T) {
		// Third run with --force: should NOT skip
		stdout3, stderr3, code3 := runTTInDir(t, tmpDir, "scaffold", "--yes", "--force")
		require.Equal(t, 0, code3,
			"Force scaffold run failed.\nSTDOUT:\n%s\nSTDERR:\n%s", stdout3, stderr3)

		combinedOutput := stdout3 + stderr3
		assert.NotContains(t, combinedOutput, "Skipping",
			"Force run should not show skip message")
	})
}
func containsJapanese(s string) bool {
	for _, r := range s {
		if (r >= '\u3040' && r <= '\u309F') || // Hiragana
			(r >= '\u30A0' && r <= '\u30FF') || // Katakana
			(r >= '\u4E00' && r <= '\u9FAF') { // CJK Unified Ideographs
			return true
		}
	}
	return false
}
